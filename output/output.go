package output

import (
	"traffic-statistics/condition_filter"
	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
)

type buildOutputFunc func(map[interface{}]interface{}) topology.OutputWorker

var registeredOutput map[string]buildOutputFunc = make(map[string]buildOutputFunc)

func register(outputType string, bf buildOutputFunc) {
	if _, ok := registeredOutput[outputType]; ok {
		log.Error("(%s) has been registered, ignore (%T)", outputType, bf)
		return
	}
	registeredOutput[outputType] = bf
}

func BuildOutput(outputType string, config map[interface{}]interface{}) *topology.OutputBox {
	var output topology.OutputWorker
	if v, ok := registeredOutput[outputType]; ok {
		output = v(config)
	}
	return &topology.OutputBox{
		OutputWorker:    output,
		ConditionFilter: condition_filter.NewConditionFilter(config),
	}
}
