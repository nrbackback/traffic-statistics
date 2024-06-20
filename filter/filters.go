package filter

import (
	"reflect"

	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
)

type filtersFilter struct {
	config        map[interface{}]interface{}
	processorNode *topology.ProcessorNode
	filterBoxes   []*topology.FilterBox
}

func init() {
	register("Filters", newFiltersFilter)
}

func newFiltersFilter(config map[interface{}]interface{}) topology.Filter {
	f := &filtersFilter{
		config: config,
	}
	_config := make(map[string]interface{})
	for k, v := range config {
		_config[k.(string)] = v
	}
	f.filterBoxes = topology.BuildFilterBoxes(_config, BuildFilter)
	if len(f.filterBoxes) == 0 {
		log.Fatal("no filters configured in Filters")
	}
	for _, b := range f.filterBoxes {
		f.processorNode = topology.AppendProcessorsToLink(f.processorNode, b)
	}
	return f
}

func (f *filtersFilter) Filter(event map[string]interface{}) map[string]interface{} {
	return f.processorNode.Process(event)
}

func (f *filtersFilter) setBelongTo(next topology.Processor) {
	var b *topology.FilterBox = f.filterBoxes[len(f.filterBoxes)-1]
	v := reflect.ValueOf(b.Filter)
	fun := v.MethodByName("SetBelongTo")
	if fun.IsValid() {
		fun.Call([]reflect.Value{reflect.ValueOf(next)})
	}
}
