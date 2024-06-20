package id

import (
	"encoding/binary"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"traffic-statistics/pkg/log"
)

type idFile struct {
	FileName      string
	FlushInterval time.Duration

	id         uint64
	lock       sync.Mutex
	exitSignal chan struct{}
	flush      atomic.Value
}

func (f *idFile) Start() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	_, err := os.Stat(f.FileName)
	if err != nil {
		if os.IsNotExist(err) {
			f.id = 0
			file, err := os.Create(f.FileName)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			return binary.Write(file, binary.LittleEndian, &f.id)
		} else {
			return err
		}
	}
	file, err := os.Open(f.FileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	var integer uint64
	err = binary.Read(file, binary.LittleEndian, &integer)
	if err != nil {
		return err
	}
	f.id = integer

	ticker := time.NewTicker(f.FlushInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				f.flush.Store(true)
			case <-f.exitSignal:
				return
			}
		}
	}()
	return nil
}

func (f *idFile) GenerateID() uint64 {
	f.lock.Lock()
	defer f.lock.Unlock()
	atomic.AddUint64(&f.id, 1)
	currentValue := atomic.LoadUint64(&f.id)
	v, ok := f.flush.Load().(bool)
	if !(ok && v) {
		return currentValue
	}

	f.flush.Store(false)
	_, err := os.Stat(f.FileName)
	if err != nil && os.IsNotExist(err) {
		f.id = 1
		currentValue = 1
		file, err := os.Create(f.FileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		if err := binary.Write(file, binary.LittleEndian, currentValue); err != nil {
			log.Errorw("write id error", "error", v)
		}
	} else {
		file, err := os.Create(f.FileName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		if err := binary.Write(file, binary.LittleEndian, currentValue); err != nil {
			log.Errorw("write id error", "error", v)
		}
	}
	return currentValue
}

func (f *idFile) Stop() error {
	f.exitSignal <- struct{}{}
	if f.id == 0 {
		return nil
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	file, err := os.Create(f.FileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	return binary.Write(file, binary.LittleEndian, &f.id)
}
