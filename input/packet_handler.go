package input

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"traffic-statistics/id"
	"traffic-statistics/input/netdata"
	"traffic-statistics/pkg/log"
)

// PacketHandler 抓取模块和上传模块同步信息使用，同步方式包括文件或者数据包
type PacketHandler struct {
	fileToHandle chan string // 上传方式为文件的时候使用
	packet       chan netdata.NetData

	UploadSource string

	IDGenerater id.IDGenerater
}

const (
	// 上传的数据来源为实时更新的文件
	uploadSourceFile = "file"
	// 上传的数据来源为抓取模块实时捕获到的数据包
	uploadSourceLivePacket = "live_packet"
)

func newPacketHandler(c packetHandlerConfig, uploadSource string, enableUpload bool) (*PacketHandler, error) {
	idFile := c.IDFile
	if idFile == "" {
		idFile = "id.bin"
	}
	var d time.Duration
	if c.FlushInterval != "" {
		var err error
		if d, err = time.ParseDuration(c.FlushInterval); err != nil {
			log.Fatalw("invalid flush_interval in id config, failed to parse to duration", "error", err)
		}
	} else {
		d = 5 * time.Second
	}
	f := PacketHandler{
		UploadSource: uploadSource,
		IDGenerater:  id.BuildIDGenerater(idFile, d),
	}
	if uploadSource == uploadSourceFile {
		f.fileToHandle = make(chan string, c.ChannelSize)
	}
	if uploadSource == uploadSourceLivePacket {
		f.packet = make(chan netdata.NetData, c.ChannelSize)
	}
	return &f, nil
}

func (f *PacketHandler) deviceByFileName(filename string) (string, error) {
	_, file := filepath.Split(filename)
	splits := strings.Split(file, "-")
	if len(splits) < 1 {
		return "", fmt.Errorf("wrong filename: (%s)", filename)
	}
	return splits[0], nil
}
