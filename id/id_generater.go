package id

import (
	"sync/atomic"
	"time"
)

type IDGenerater interface {
	Start() error
	GenerateID() uint64
	Stop() error
}

func BuildIDGenerater(file string, flushInterval time.Duration) *idFile {
	var f atomic.Value
	f.Store(false)
	return &idFile{
		FileName:      file,
		FlushInterval: flushInterval,
		exitSignal:    make(chan struct{}, 1),
		flush:         f,
	}
}
