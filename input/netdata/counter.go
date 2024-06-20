package netdata

import "sync/atomic"

type AtomicCounter struct {
	number uint64
}

var Counter *AtomicCounter

func Init() {
	Counter = &AtomicCounter{0}
}

func Add(num uint64) {
	atomic.AddUint64(&Counter.number, num)
}

func Read() uint64 {
	return atomic.LoadUint64(&Counter.number)
}

// ..........................

type AtomicSizeCounter struct {
	number uint64
}

var SizeCounter *AtomicSizeCounter

func InitSize() {
	SizeCounter = &AtomicSizeCounter{0}
}

func AddSize(num uint64) {
	atomic.AddUint64(&SizeCounter.number, num)
}

func ReadSize() uint64 {
	return atomic.LoadUint64(&SizeCounter.number)
}
