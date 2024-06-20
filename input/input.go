package input

import (
	"fmt"
	"plugin"

	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
)

type buildInputFunc func(map[interface{}]interface{}) topology.InputWorker

var registeredInput map[string]buildInputFunc = make(map[string]buildInputFunc)

func register(inputType string, bf buildInputFunc) {
	if _, ok := registeredInput[inputType]; ok {
		log.Errorf("(%s) has been registered, ignore (%T)", inputType, bf)
		return
	}
	registeredInput[inputType] = bf
}

func GetInput(inputType string, config map[interface{}]interface{}) topology.InputWorker {
	if v, ok := registeredInput[inputType]; ok {
		return v(config)
	}
	pluginPath := inputType
	output, err := getInputFromPlugin(pluginPath, config)
	if err != nil {
		log.Errorf("could not load (%s), error (%v)", pluginPath, err)
		return nil
	}
	return output
}

func getInputFromPlugin(pluginPath string, config map[interface{}]interface{}) (topology.InputWorker, error) {
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %v", pluginPath, err)
	}
	newFunc, err := p.Lookup("New")
	if err != nil {
		return nil, fmt.Errorf("could not find `New` function in %s: %s", pluginPath, err)
	}
	f, ok := newFunc.(func(map[interface{}]interface{}) interface{})
	if !ok {
		return nil, fmt.Errorf("`New` func in %s format error", pluginPath)
	}
	rst := f(config)
	input, ok := rst.(topology.InputWorker)
	if !ok {
		return nil, fmt.Errorf("`New` func in %s dose not return Input Interface", pluginPath)
	}
	return input, nil
}
