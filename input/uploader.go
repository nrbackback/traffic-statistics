package input

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"

	"traffic-statistics/cursor"
	"traffic-statistics/input/netdata"
	"traffic-statistics/pkg/log"
)

type uploader interface {
	Startup()
	Stop()
	ReadNetData() *netdata.NetData
}

func newUploader(c uploadConfig, packetHandler *PacketHandler, enableCapture bool) uploader {
	var u *fileUploader
	if c.UploadSource == uploadSourceFile {
		u = &fileUploader{
			Config:        c,
			PacketHandler: packetHandler,
			message:       make(chan netdata.NetData),
			PcapCursor:    cursor.BuildPcapCursor(c.CursorType, c.CursorConfig),
			PcapDir:       c.PcapDir,
		}
		if enableCapture {
			return &fileUploaderWithCapture{
				baseUploader: u,
			}
		}
		return &fileUploaderWithoutCapture{
			baseUploader: u,
		}
	}
	if c.UploadSource == uploadSourceLivePacket {
		return &livePacketUploader{
			PacketHandler: packetHandler,
		}
	}
	log.Fatal("invalid upload source", "source", c.UploadSource)
	return nil
}

type fileUploader struct {
	Config        uploadConfig
	PacketHandler *PacketHandler
	PcapCursor    cursor.PcapCursor
	PcapDir       string

	lock         sync.Mutex
	closer       func()
	wg           sync.WaitGroup
	deviceToFile map[string]chan string
	ctx          context.Context

	message chan netdata.NetData
}

func (u *fileUploader) Startup() {
	f := u.PacketHandler
	files, err := os.ReadDir(u.PcapDir)
	if err != nil {
		log.Errorw("read pcap dir", "dir", u.PcapDir, "error", err)
	}
	if err := u.PacketHandler.IDGenerater.Start(); err != nil {
		log.Errorw("start id generator error", "error", err)
	}
	for _, file := range files {
		// 已经处理过的文件不再处理
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".pcap") {
			uploaded, err := u.PcapCursor.IsFileUploaded(file.Name())
			if err != nil {
				log.Errorw("check completed status error", "file", file.Name(), "error", err)
			} else if !uploaded {
				f.fileToHandle <- filepath.Join(u.PcapDir, file.Name())
			}
		}
	}
}

func (u *fileUploader) Stop() {
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.closer != nil {
		u.closer()
		u.closer = nil
	}
	u.PcapCursor.Stop()
	if err := u.PacketHandler.IDGenerater.Stop(); err != nil {
		log.Errorw("save id error", "error", err)
	}
}

type fileUploaderWithCapture struct {
	lock         sync.Mutex
	baseUploader *fileUploader
}

type fileUploaderWithoutCapture struct {
	lock         sync.Mutex
	baseUploader *fileUploader
	pcapDir      string
}

type livePacketUploader struct {
	lock          sync.Mutex
	PacketHandler *PacketHandler
}

func (u *livePacketUploader) Startup() {
	u.lock.Lock()
	defer u.lock.Unlock()
	if err := u.PacketHandler.IDGenerater.Start(); err != nil {
		log.Errorw("start id generator error", "error", err)
	}
}

func (u *livePacketUploader) Stop() {
	u.lock.Lock()
	defer u.lock.Unlock()
	close(u.PacketHandler.packet)
	if err := u.PacketHandler.IDGenerater.Stop(); err != nil {
		log.Fatalw("stop id generator error", "error", err)
	}
}

func (u *livePacketUploader) ReadNetData() *netdata.NetData {
	msg, ok := <-u.PacketHandler.packet
	if ok {
		return &msg
	}
	return nil
}

func (u *fileUploaderWithoutCapture) Startup() {
	u.lock.Lock()
	defer u.lock.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	u.baseUploader.ctx = ctx
	u.baseUploader.Startup()
	close(u.baseUploader.PacketHandler.fileToHandle)

	for file := range u.baseUploader.PacketHandler.fileToHandle {
		u.baseUploader.wg.Add(1)
		go u.baseUploader.upload(file)
	}
	u.baseUploader.closer = func() {
		u.baseUploader.wg.Wait()
		cancel()
	}
	u.baseUploader.Stop()
	close(u.baseUploader.message)
	log.Info("all file data has been uploaded.")
	return
}

func (u *fileUploaderWithoutCapture) Stop() {
	u.lock.Lock()
	defer u.lock.Unlock()
}

func (u *fileUploaderWithoutCapture) ReadNetData() *netdata.NetData {
	msg, ok := <-u.baseUploader.message
	if ok {
		return &msg
	}
	return nil
}

func (u *fileUploaderWithCapture) Startup() {
	u.lock.Lock()
	defer u.lock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	u.baseUploader.ctx = ctx
	u.baseUploader.Startup()

	u.baseUploader.closer = func() {
		cancel()
		close(u.baseUploader.PacketHandler.fileToHandle)
		if len(u.baseUploader.PacketHandler.fileToHandle) > 0 {
			for file := range u.baseUploader.PacketHandler.fileToHandle {
				if file != "" {
					u.baseUploader.wg.Add(1)
					go u.baseUploader.upload(file)
				}
			}
		}
		u.baseUploader.wg.Wait()
		close(u.baseUploader.message)
	}
	go func() {
		for {
			select {
			case <-u.baseUploader.ctx.Done():
				return
			case file := <-u.baseUploader.PacketHandler.fileToHandle:
				if file != "" {
					u.baseUploader.wg.Add(1)
					go u.baseUploader.upload(file)
				}
			}
		}
	}()
}

func (u *fileUploaderWithCapture) Stop() {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.baseUploader.Stop()
}

func (u *fileUploaderWithCapture) ReadNetData() *netdata.NetData {
	msg, ok := <-u.baseUploader.message
	if ok {
		return &msg
	}
	return nil
}

func (u *fileUploader) upload(file string) {
	device, err := u.PacketHandler.deviceByFileName(file)
	if err != nil {
		log.Errorw("wrong file name", "file", file)
		return
	}
	if u.deviceToFile == nil {
		u.deviceToFile = make(map[string]chan string)
	}
	if u.deviceToFile[device] == nil {
		u.deviceToFile[device] = make(chan string)
		go u.uploadFileByDevice(device)
	}
	u.deviceToFile[device] <- file
}

func (u *fileUploader) uploadFileByDevice(device string) {
	for {
		select {
		case <-u.ctx.Done():
			return
		case file := <-u.deviceToFile[device]:
			u.uploadFile(file)
		}
	}
}

func (u *fileUploader) uploadFile(file string) {
	defer u.wg.Done()
	handle, err := pcap.OpenOffline(file)
	if err != nil {
		log.Errorw("open file error", "file", file, "error", err)
		return
	}
	defer handle.Close()
	log.Infow("start uploading file data", "file", file)
	device, err := u.PacketHandler.deviceByFileName(file)
	if err != nil {
		log.Errorw("wrong file name", "file", file)
		return
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetSource.DecodeOptions = gopacket.DecodeOptions{Lazy: true, NoCopy: true,
		DecodeStreamsAsDatagrams: true}
	_, filename := filepath.Split(file)
	countUploadedBefore, err := u.PcapCursor.UploadedRecordCount(filename)
	if err != nil {
		log.Errorw("read finished_count error", "file", filename, "error", err)
		return
	}
	var uploadedCount = 0
	var firstIndex, lastIndex = countUploadedBefore + 1, countUploadedBefore
	var index int64
	for packet := range packetSource.Packets() {
		index++
		if index >= firstIndex {
			netData := netdata.NetDataFromPacket(device, u.PacketHandler.IDGenerater.GenerateID(), packet)
			if netData.ID != 0 {
				u.message <- netData
			}
			if err := u.PcapCursor.AddOneUploadedRecord(filename); err != nil {
				log.Errorw("add one uploaded record error", "error", err)
			}
			uploadedCount++
			lastIndex++
		}
	}
	if err := u.PcapCursor.MarkFileAsUploaded(filename); err != nil {
		log.Errorw("mark file as uploaded error", "file", filename, "error", err)
	}
	log.Infow("file upload complete", "file", file, "from", firstIndex, "to", lastIndex, "number of uploaded data", uploadedCount)
}
