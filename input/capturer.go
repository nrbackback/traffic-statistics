package input

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"

	"traffic-statistics/input/netdata"
	"traffic-statistics/pkg/log"
)

type Capturer interface {
	Startup()
	Stop()
}

const (
	captureOutputFile = "file"
)

func newCapturer(c captureConfig, packetHandler *PacketHandler) (Capturer, error) {
	devicesToCapture, err := deviceToCapture(c.DeviceType, c.Device)
	if err != nil {
		return nil, err
	}
	if len(devicesToCapture) == 0 {
		return nil, errors.New("no device to capture")
	}
	f := &packetCapturer{
		SnapshotLen:     c.SnapshotLen,
		Promiscuous:     c.Promiscuous,
		PacketHandler:   packetHandler,
		deviceToCapture: devicesToCapture,
	}
	if c.Output == captureOutputFile {
		fg := c.OutputFile
		d, err := time.ParseDuration(fg.NewFileInterval)
		if err != nil {
			return nil, fmt.Errorf("parse interval to time duration error (%v)", err)
		}
		if d == 0 || time.Hour%d != 0 || d%time.Minute != 0 {
			return nil, errors.New("time interval should be in minutes")
		}
		if _, err := os.Stat(fg.PcapDir); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			if err := os.Mkdir(fg.PcapDir, os.ModePerm); err != nil {
				return nil, err
			}
		}
		f.FileOutput = &fileOutput{
			PcapDir:         fg.PcapDir,
			NewFileInterval: fg.NewFileInterval,
		}
	}
	return f, nil
}

type fileOutput struct {
	PcapDir         string
	NewFileInterval string
}

type packetCapturer struct {
	SnapshotLen int32
	Promiscuous bool
	FileOutput  *fileOutput

	PacketHandler *PacketHandler

	wg              sync.WaitGroup
	deviceToCapture []string
	closer          func()
	lock            sync.Mutex
}

func (c *packetCapturer) Startup() {
	c.lock.Lock()
	defer c.lock.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	c.closer = func() {
		cancel()
		c.wg.Wait()
	}
	for _, v := range c.deviceToCapture {
		c.wg.Add(1)
		go c.captureByDevice(ctx, v)
	}
}

func (c *packetCapturer) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.closer != nil {
		c.closer()
		c.closer = nil
	}
}

func (c *packetCapturer) captureByDevice(ctx context.Context, device string) {
	defer c.wg.Done()
	handle, err := pcap.OpenLive(device, c.SnapshotLen, c.Promiscuous, -1*time.Second)
	if err != nil {
		log.Errorw("open device error", "device", device, "error", err)
		return
	}
	defer handle.Close()
	log.Infow("start capture packet", "device", device)
	d := &deviceCapturer{
		packetCapturer: c,
		device:         device,
		handle:         handle,
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetSource.DecodeOptions = gopacket.DecodeOptions{Lazy: true, NoCopy: true,
		DecodeStreamsAsDatagrams: true}
	for {
		select {
		case <-ctx.Done():
			d.stop()
			return
		case packet := <-packetSource.Packets():
			d.sendPacketToOutput(packet)
		}
	}
}

var lastMinute = int64(0)

var lock sync.Mutex

func (d *deviceCapturer) sendPacketToOutput(packet gopacket.Packet) {
	lock.Lock()
	defer lock.Unlock()
	var newFile bool
	f := d.currentFile
	if d.packetCapturer.FileOutput != nil {
		newFile = d.updateWriter(packet.Metadata().Timestamp)
		d.writer.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
	}
	if d.packetCapturer.PacketHandler.UploadSource == uploadSourceFile && newFile {
		// 上传方式为文件且新文件已经产生，则将旧文件传递到上传模块
		d.packetCapturer.PacketHandler.fileToHandle <- f
	}
	if d.packetCapturer.PacketHandler.UploadSource == uploadSourceLivePacket {
		now := time.Now()
		if now.Unix()/60 != int64(lastMinute) {
			lastMinute = now.Unix() / 60
			formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d",
				now.Year(), now.Month(), now.Day(),
				now.Hour(), now.Minute())
			log.Infof("start uploading data in the minute of %s to upload module", formatted)
		}
		d.packetCapturer.PacketHandler.packet <- netdata.NetDataFromPacket(d.device,
			d.packetCapturer.PacketHandler.IDGenerater.GenerateID(), packet)
	}
}

func (d *deviceCapturer) stop() {
	if d.packetCapturer.PacketHandler.UploadSource == uploadSourceFile {
		d.packetCapturer.PacketHandler.fileToHandle <- d.currentFile
		d.file.Close()
	}
	d.handle.Close()
}

type deviceCapturer struct {
	device         string
	packetCapturer *packetCapturer
	handle         *pcap.Handle
	writer         *pcapgo.Writer
	file           *os.File
	currentFile    string
}

func (c *deviceCapturer) currentPcapFileName(t time.Time) string {
	interval := c.packetCapturer.FileOutput.NewFileInterval
	d, _ := time.ParseDuration(interval)
	periodMinute := int(d / time.Minute)
	m := t.Minute() / periodMinute
	nowForFile := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), m*periodMinute, 0, 0, t.Location())
	filename := fmt.Sprintf("%s-%s-%d.%s", c.device, nowForFile.Format("2006-01-02-15"), nowForFile.Minute(), "pcap")
	return filepath.Join(c.packetCapturer.FileOutput.PcapDir, filename)
}

func (d *deviceCapturer) updateWriter(t time.Time) bool {
	currentFile := d.currentPcapFileName(t)
	if currentFile == d.currentFile {
		return false
	}
	if d.currentFile != "" {
		d.file.Close()
	}
	d.currentFile = currentFile
	file, err := os.Create(currentFile)
	if err != nil {
		log.Errorw("create file error", "file", currentFile, "error", err)
		return false
	}
	w := pcapgo.NewWriter(file)
	w.WriteFileHeader(uint32(d.packetCapturer.SnapshotLen), d.handle.LinkType())
	d.file = file
	d.writer = w
	return true
}

const (
	// deviceTypeWhite 设备为白名单
	deviceTypeWhite = "white"
	// deviceTypeBlack 设备为黑名单
	deviceTypeBlack = "black"
)

func deviceToCapture(deviceType string, devices []string) ([]string, error) {
	allDevices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	devicesToCapture := make([]string, 0, len(allDevices))
	if deviceType == deviceTypeWhite {
		allDevicesMap := make(map[string]bool, len(allDevices))
		for _, v := range allDevices {
			allDevicesMap[v.Name] = true
		}
		for _, v := range devices {
			if _, ok := allDevicesMap[v]; ok {
				devicesToCapture = append(devicesToCapture, v)
			}
		}
	}
	if deviceType == deviceTypeBlack {
		devicesMap := make(map[string]bool, len(devices))
		for _, v := range devices {
			devicesMap[v] = true
		}
		for _, v := range allDevices {
			if _, ok := devicesMap[v.Name]; !ok {
				devicesToCapture = append(devicesToCapture, v.Name)
			}
		}
	}
	return devicesToCapture, nil
}
