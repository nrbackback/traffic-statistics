package cursor

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"

	"traffic-statistics/pkg/log"
)

const FileType = "file"

type fileConfig struct {
	File          string `mapstructure:"file"`
	FlushInterval string `mapstructure:"flush_interval"`
}

type fileHandler struct {
	File       string
	content    FileContent
	exitSignal chan struct{}
	lock       sync.Mutex
}

type FileContent struct {
	UploadedFile   map[string]int64 `json:"uploaded_file"`
	UploadProgress map[string]int64 `json:"upload_progress"`
}

func newFileHandler(config map[string]interface{}) *fileHandler {
	var c fileConfig
	if err := mapstructure.Decode(config, &c); err != nil {
		log.Fatalw("decode file config failed", "error", err)
	}
	d, err := time.ParseDuration(c.FlushInterval)
	if err != nil {
		log.Fatalw("invalid flush_interval in file config, failed to parse to duration", "error", err)
	}
	file, err := os.OpenFile(c.File, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Fatalw("open file failed", "file", c.File, "error", err)
	}
	f, _ := file.Stat()
	content := FileContent{
		UploadedFile:   make(map[string]int64, 0),
		UploadProgress: make(map[string]int64, 0),
	}
	fh := &fileHandler{
		File:       c.File,
		content:    content,
		exitSignal: make(chan struct{}),
	}
	if f.Size() != 0 {
		buffer := make([]byte, f.Size())
		_, err = file.Read(buffer)
		if err != nil {
			log.Fatalw("read file failed", "file", c.File, "error", err)
		}
		err := json.Unmarshal(buffer, &fh.content)
		if err != nil {
			log.Fatalw("invalid file content, failed to unmarshal", "file", c.File, "error", err)
		}
	}
	file.Close()
	ticker := time.NewTicker(d)
	go func() {
		for {
			select {
			case <-fh.exitSignal:
				return
			case <-ticker.C:
				fh.flushToFile()
			}
		}
	}()
	return fh
}

func (h *fileHandler) flushToFile() {
	h.lock.Lock()
	defer h.lock.Unlock()
	v, _ := json.Marshal(h.content)
	if err := os.WriteFile(h.File, v, 0644); err != nil {
		log.Fatalw("write content to file error", "file", h.File, "error", err)
	}
}

func (h *fileHandler) AddOneUploadedRecord(filename string) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	if _, ok := h.content.UploadedFile[filename]; ok {
		h.content.UploadedFile[filename]++
		return nil
	}
	if _, ok := h.content.UploadProgress[filename]; ok {
		h.content.UploadProgress[filename]++
	} else {
		h.content.UploadProgress[filename] = 1
	}
	return nil
}

func (h *fileHandler) MarkFileAsUploaded(filename string) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	count := h.content.UploadProgress[filename]
	delete(h.content.UploadProgress, filename)
	h.content.UploadedFile[filename] = count
	return nil
}

func (h *fileHandler) IsFileUploaded(filename string) (bool, error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	_, ok := h.content.UploadedFile[filename]
	if ok {
		return true, nil
	}
	return false, nil
}

func (h *fileHandler) UploadedRecordCount(filename string) (int64, error) {
	h.lock.Lock()
	defer h.lock.Unlock()
	v, ok := h.content.UploadProgress[filename]
	if !ok {
		v = h.content.UploadedFile[filename]
	}
	return v, nil
}

func (h *fileHandler) Stop() {
	h.exitSignal <- struct{}{}
	h.flushToFile()
}
