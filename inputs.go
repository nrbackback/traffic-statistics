package main

import (
	"fmt"
	"sync"

	"traffic-statistics/input"
	"traffic-statistics/topology"
)

// Inputs 用于 input 启动和停止时使用
type Inputs []*input.InputBox

func buildPluginLink(config map[string]interface{}) (boxes []*input.InputBox, err error) {
	boxes = make([]*input.InputBox, 0)
	for _, inputI := range config["inputs"].([]interface{}) {
		var inputPlugin topology.InputWorker
		i := inputI.(map[interface{}]interface{})
		for inputTypeI, inputConfigI := range i {
			inputType := inputTypeI.(string)
			inputConfig := inputConfigI.(map[interface{}]interface{})
			inputPlugin = input.GetInput(inputType, inputConfig)
			if inputPlugin == nil {
				err = fmt.Errorf("invalid input plugin")
				return
			}
			// input 配置的每一项都对应一个 *InputBox
			box := input.NewInputBox(inputPlugin, inputConfig, config, mainThreadExitChan)
			if box == nil {
				err = fmt.Errorf("new input box fail")
				return
			}
			box.SetShutdownWhenNil(*exitWhenNil)
			boxes = append(boxes, box)
		}
	}
	return
}

func (inputs Inputs) start() {
	// 每个 input 都对应一个 goroutine
	var wg sync.WaitGroup
	wg.Add(len(inputs))
	for i := range inputs {
		go func(i int) {
			defer wg.Done()
			inputs[i].Beat()
		}(i)
	}
	wg.Wait()
}

func (inputs Inputs) stop() {
	boxes := ([]*input.InputBox)(inputs)
	for _, box := range boxes {
		box.Shutdown()
	}
}
