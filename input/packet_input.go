package input

import (
	"github.com/mitchellh/mapstructure"

	"traffic-statistics/codec"
	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
)

type livePacketChannel struct {
	ChannelSize int32 `mapstructure:"channel_size"`
}

type packetHandlerConfig struct {
	// 上传方式为文件或者实时数据包时，用于将捕获到的信息上传到输出端的管道的大小
	ChannelSize int32 `mapstructure:"channel_size"`

	IDFile        string `mapstructure:"id_file,omitemtpy"`
	FlushInterval string `mapstructure:"flush_interval,omitemtpy"`
}

type outputFile struct {
	NewFileInterval string `mapstructure:"new_file_interval"`
	PcapDir         string `mapstructure:"pcap_dir"`
}

type captureConfig struct {
	Enabled     bool       `mapstructure:"enabled"`
	DeviceType  string     `mapstructure:"device_type"`
	Device      []string   `mapstructure:"device"`
	SnapshotLen int32      `mapstructure:"snapshot_len"`
	Promiscuous bool       `mapstructure:"promiscuous"`
	Output      string     `mapstructure:"output"`
	OutputFile  outputFile `mapstructure:"output_file"`
}

type uploadConfig struct {
	Enabled      bool                   `mapstructure:"enabled"`
	UploadSource string                 `mapstructure:"source"`
	PcapDir      string                 `mapstructure:"pcap_dir,omitemty"` // 当上传来源为目录时使用
	CursorType   string                 `mapstructure:"cursor_type,omitemty"`
	CursorConfig map[string]interface{} `mapstructure:"cursor_config,omitemty"`
	SavePcapFile bool                   `mapstructure:"save_pcap_file"`
}

type Config struct {
	Capture captureConfig `mapstructure:"capture"`
	// Handler为Capture和Upload两者之间传输信息工具
	Handler packetHandlerConfig `mapstructure:"handler"`
	Upload  uploadConfig        `mapstructure:"upload"`
}

type packetInput struct {
	capturer Capturer
	uploader uploader
	decoder  codec.Decoder
	stop     bool
}

func init() {
	register("Packet", newPacketInput)
}

func newPacketInput(config map[interface{}]interface{}) topology.InputWorker {
	var c Config
	if err := mapstructure.Decode(config, &c); err != nil {
		log.Fatalw("decode packet config failed", "error", err)
	}
	enableCapture := c.Capture.Enabled
	enableUpload := c.Upload.Enabled
	if !enableCapture && !enableUpload {
		log.Fatal("please enable at least one of capture module and capture module")
	}
	packetHandler, err := newPacketHandler(c.Handler, c.Upload.UploadSource, enableUpload)
	if err != nil {
		log.Fatalw("new file handler error", "error", err)
	}
	var capturer Capturer
	var uploader uploader
	if enableCapture {
		capturer, err = newCapturer(c.Capture, packetHandler)
		if err != nil {
			log.Fatalw("new capturer error", "error", err)
		}
		go capturer.Startup()
	}
	if enableUpload {
		uploader = newUploader(c.Upload, packetHandler, enableCapture)
		go uploader.Startup()
	}
	return &packetInput{
		capturer: capturer,
		uploader: uploader,
		decoder:  codec.NewDecoder("json_tag"),
	}
}

func (p *packetInput) ReadOneEvent() map[string]interface{} {
	if p.uploader == nil {
		return nil
	}
	msg := p.uploader.ReadNetData()
	if msg != nil {
		return p.decoder.Decode(msg)
	}
	return nil
}

func (p *packetInput) Shutdown() {
	if p.capturer != nil {
		p.capturer.Stop()
	}
	if p.uploader != nil {
		p.uploader.Stop()
	}
}
