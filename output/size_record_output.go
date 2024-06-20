package output

import (
	"fmt"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"

	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
)

func init() {
	register("SizeRecord", newSizeRecordOutput)
}

type sizeRecordConfig struct {
	CKAddr     string `mapstructure:"addr"`
	CKDatabase string `mapstructure:"database"`
	CKUsername string `mapstructure:"username"`
	CKPassword string `mapstructure:"password"`
	CKTable    string `mapstructure:"table"`
	Interval   string `mapstructure:"interval"`
	Timeout    string `mapstructure:"timeout"`
	RetryTimes int    `mapstructure:"retry_times"`
}

/*
CREATE TABLE interval_traffic
(

	    `device` String,
			`start_time` Int64,
			`end_time` Int64,
			`src_ip` String,
			`dst_ip` String,
			`packet_size` Int64,
			`packet_count` Int64,
			`create_time` Int64

)
ENGINE = MergeTree
PARTITION BY toYYYYMM(toDateTime(create_time))
ORDER BY create_time
SETTINGS index_granularity = 8192
*/
type intervalPacketSizeDB struct {
	// 时间统计规则为左闭又开，该范围内的数据包的时间大于等于StartTime且小于EndTime
	Device      string
	StartTime   int64
	EndTime     int64
	SrcIP       string
	DstIP       string
	PacketSize  int64
	PacketCount int64
	CreateTime  int64
}

type sizeRecordOuput struct {
	db         *gorm.DB
	interval   int64 // 时间间隔，单位为秒
	mux        sync.Mutex
	config     sizeRecordConfig
	idxSizeMap map[int64]map[string]*intervalPacketSizeDB
	startTime  int64 // 程序启动时刻的时间戳
	timeout    int64 // 超时时间
	exit       chan struct{}
}

func newSizeRecordOutput(config map[interface{}]interface{}) topology.OutputWorker {
	// return nil
	var c sizeRecordConfig
	if err := mapstructure.Decode(config, &c); err != nil {
		log.Fatal("decode clickhouse config failed")
	}
	if c.Interval == "" {
		c.Interval = "1m"
	}
	interval, err := time.ParseDuration(c.Interval)
	if err != nil {
		log.Fatalw("parse duration error", "duration", c.Interval, "error", err)
	}
	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		log.Fatalw("parse timeout error", "timeout", c.Timeout, "error", err)
	}
	connStr := fmt.Sprintf(
		"clickhouse://%s:%s@%s/%s?read_timeout=10s&write_timeout=20s",
		c.CKUsername,
		c.CKPassword,
		c.CKAddr,
		c.CKDatabase,
	)
	db, err := gorm.Open(clickhouse.Open(connStr), &gorm.Config{})
	if err != nil {
		log.Fatalw("open clickhouse client error", "error", err)
	}
	e := &sizeRecordOuput{
		db:         db,
		interval:   int64(interval.Seconds()),
		config:     c,
		idxSizeMap: make(map[int64]map[string]*intervalPacketSizeDB, 0),
		startTime:  time.Now().Unix(),
		exit:       make(chan struct{}),
		timeout:    int64(timeout.Seconds()),
	}
	go e.witeToDB()
	return e
}

func (o *sizeRecordOuput) Emit(event map[string]interface{}) {
	// return
	rawCreateTime := event["create_time"]
	createTime, ok := rawCreateTime.(time.Time)
	if !ok {
		log.Errorw("failed to parse create_time from event", "create_time", rawCreateTime)
		return
	}
	createTimeUnix := createTime.Unix()
	var device, srcIP, dstIP string
	device = event["device"].(string)        // 因为input部分已经确保了device不为空且为string类型，故该处直接转换
	packetSize := event["pack_size"].(int32) // 因为input部分已经确保了pack_size不为空且为int32类型，故该处直接转换
	ip := event["src_ip"]
	if ip != nil {
		srcIP, ok = ip.(string)
		if !ok {
			log.Errorw("failed to parse src_ip from event", "src_ip", ip)
			return
		}
	}
	ip = event["dst_ip"]
	if ip != nil {
		dstIP, ok = ip.(string)
		if !ok {
			log.Errorw("failed to parse dst_ip from event", "dst_ip", ip)
			return
		}
	}
	if srcIP == "" && dstIP == "" {
		log.Warn("src_ip and dst_ip are all empty")
		return
	}
	if (srcIP == "" && dstIP != "") || (srcIP != "" && dstIP == "") {
		log.Errorw("src_ip and dst_ip should be empty or not empty at the same time", "src_ip", srcIP, "dst_ip", dstIP)
		return
	}
	if createTimeUnix < time.Now().Unix()-o.timeout {
		log.Errorw("data is outdated", "create_time", createTimeUnix, "now", time.Now())
		return
	}

	idx := periodIdx(createTimeUnix, o.startTime, o.interval)
	direction := fmt.Sprintf("%s_%s_%s", device, srcIP, dstIP)
	s := &intervalPacketSizeDB{
		Device:      device,
		StartTime:   startTimeOfIdx(o.startTime, idx, o.interval),
		EndTime:     endTimeOfIdx(o.startTime, idx, o.interval),
		SrcIP:       srcIP,
		DstIP:       dstIP,
		PacketSize:  int64(packetSize),
		PacketCount: 1,
	}
	o.mux.Lock()
	defer o.mux.Unlock()
	if o.idxSizeMap[idx] == nil {
		o.idxSizeMap[idx] = map[string]*intervalPacketSizeDB{
			direction: s,
		}
	} else {
		if o.idxSizeMap[idx][direction] == nil {
			o.idxSizeMap[idx][direction] = s
		} else {
			o.idxSizeMap[idx][direction].PacketSize += int64(packetSize)
			o.idxSizeMap[idx][direction].PacketCount++
		}
	}
}

func (o *sizeRecordOuput) Shutdown() {
	// return
	o.exit <- struct{}{}
	listToCreate := make([]intervalPacketSizeDB, 0)
	o.mux.Lock()
	defer o.mux.Unlock()
	for idx := range o.idxSizeMap {
		for _, directionToSize := range o.idxSizeMap[idx] {
			directionToSize.CreateTime = time.Now().Unix()
			listToCreate = append(listToCreate, *directionToSize)
		}
		delete(o.idxSizeMap, idx)
	}
	if err := o.db.Table(o.config.CKTable).Create(&listToCreate).Error; err != nil {
		log.Errorw("clickhouse batch write error", "error", err)
	}
}

func (o *sizeRecordOuput) witeToDB() {
	// return
	// fmt.Println("...........<-ticker.C:......", time.Duration(o.interval*time.Second.Nanoseconds()))
	ticker := time.NewTicker(time.Duration(o.interval * time.Second.Nanoseconds()))
	// time.Sleep(time.Duration(o.timeout * time.Second.Nanoseconds()))
	for {
		select {
		case <-o.exit:
			fmt.Println("......o.exit:.....")
			return
		case <-ticker.C:
			// fmt.Println("...........<-ticker.C:......")
			t := time.Now().Unix() - o.timeout
			idxList := o.allIdxToWrite(t)
			listToCreate := make([]intervalPacketSizeDB, 0)
			o.mux.Lock()
			for _, idx := range idxList {
				for _, directionToSize := range o.idxSizeMap[idx] {
					directionToSize.CreateTime = time.Now().Unix()
					listToCreate = append(listToCreate, *directionToSize)
				}
				delete(o.idxSizeMap, idx)
			}
			o.mux.Unlock()
			// fmt.Println("......listToCreate........", listToCreate)
			if o.config.RetryTimes == 0 {
				if err := o.db.Table(o.config.CKTable).Create(&listToCreate).Error; err != nil {
					log.Errorw("clickhouse batch write error", "error", err)
				}
			} else {
				var err error
				for i := 0; i < o.config.RetryTimes; i++ {
					err = o.db.Table(o.config.CKTable).Create(&listToCreate).Error
					if err == nil {
						break
					}
					// 间隔1秒后重试
					time.Sleep(time.Second)
				}
				if err != nil {
					log.Errorw("clickhouse batch write error", "error", err, "tried_times", o.config.RetryTimes)
				}
			}
		}
	}
}

func (o *sizeRecordOuput) allIdxToWrite(t int64) []int64 {
	o.mux.Lock()
	defer o.mux.Unlock()
	idx := periodIdx(t, o.startTime, o.interval)
	var r = make([]int64, 0)
	for k := range o.idxSizeMap {
		if k <= idx {
			r = append(r, k)
		}
	}
	return r
}

func periodIdx(t, startTime, interval int64) int64 {
	if t < startTime {
		return 0
	}
	sub := t - startTime
	return sub/interval + 1
}

func startTimeOfIdx(startTime, idx, interval int64) int64 {
	return startTime + (idx-1)*interval
}

func endTimeOfIdx(startTime, idx, interval int64) int64 {
	return startTime + idx*interval
}
