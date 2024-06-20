package input

import (
	"sync"

	"traffic-statistics/filter"
	"traffic-statistics/output"
	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
)

type InputBox struct {
	config             map[string]interface{}
	input              topology.InputWorker
	outputsInAllWorker []*topology.OutputBox
	stop               bool
	once               sync.Once
	shutdownChan       chan bool

	shutdownWhenNil    bool
	mainThreadExitChan chan struct{}
}

func (box *InputBox) Beat() {
	go box.beat()
	<-box.shutdownChan
}

func (box *InputBox) beat() {
	var firstNode *topology.ProcessorNode = box.buildTopology()
	var (
		event map[string]interface{}
	)
	for !box.stop {
		event = box.input.ReadOneEvent()
		if event == nil {
			if box.stop {
				break
			}
			if box.shutdownWhenNil {
				log.Info("received nil message. shutdown...")
				box.mainThreadExitChan <- struct{}{}
				break
			} else {
				continue
			}
		}
		firstNode.Process(event)
	}
}

// buildTopology 读取所有 filter 和 output，构建 pipeline
func (box *InputBox) buildTopology() *topology.ProcessorNode {
	outputs := topology.BuildOutputs(box.config, output.BuildOutput)
	// 在 box 的 shutdown 中使用停止 outputsInAllWorker
	box.outputsInAllWorker = outputs
	var outputProcessor topology.Processor
	if len(outputs) == 1 {
		outputProcessor = outputs[0]
	} else {
		// 多个输出的时候使用 OutputsProcessor，类似 Inputs
		outputProcessor = (topology.OutputsProcessor)(outputs)
	}
	filterBoxes := topology.BuildFilterBoxes(box.config, filter.BuildFilter)
	var firstNode *topology.ProcessorNode
	for _, b := range filterBoxes {
		firstNode = topology.AppendProcessorsToLink(firstNode, b)
	}
	firstNode = topology.AppendProcessorsToLink(firstNode, outputProcessor)
	return firstNode
}

func (box *InputBox) shutdown() {
	box.once.Do(func() {
		log.Infow("try to shutdown input")
		box.input.Shutdown()
		for _, outputs := range box.outputsInAllWorker {
			log.Infow("try to shutdown output", "output", box.input)
			outputs.OutputWorker.Shutdown()
		}
	})
	box.shutdownChan <- true
}

func (box *InputBox) Shutdown() {
	box.shutdown()
	box.stop = true
}

func (box *InputBox) SetShutdownWhenNil(shutdownWhenNil bool) {
	box.shutdownWhenNil = shutdownWhenNil
}

func NewInputBox(input topology.InputWorker, inputConfig map[interface{}]interface{},
	config map[string]interface{}, mainThreadExitChan chan struct{}) *InputBox {
	b := &InputBox{
		input:              input,
		config:             config,
		stop:               false,
		shutdownChan:       make(chan bool, 1),
		mainThreadExitChan: mainThreadExitChan,
	}
	return b
}
