package topology

import (
	"traffic-statistics/condition_filter"
	"traffic-statistics/pkg/log"
)

type OutputWorker interface {
	Emit(map[string]interface{})
	Shutdown()
}

type OutputBox struct {
	OutputWorker
	*condition_filter.ConditionFilter
}

func (p *OutputBox) Process(event map[string]interface{}) map[string]interface{} {
	if p.Pass(event) {
		p.Emit(event)
	}
	return nil
}

type buildOutputFunc func(outputType string, config map[interface{}]interface{}) *OutputBox

func BuildOutputs(config map[string]interface{}, buildOutput buildOutputFunc) []*OutputBox {
	rst := make([]*OutputBox, 0)
	// 支持多个输出
	for _, outputs := range config["outputs"].([]interface{}) {
		for outputType, outputConfig := range outputs.(map[interface{}]interface{}) {
			outputType := outputType.(string)
			log.Infow("output type", "type", outputType)
			outputConfig := outputConfig.(map[interface{}]interface{})
			output := buildOutput(outputType, outputConfig)
			rst = append(rst, output)
		}
	}
	return rst
}

type OutputsProcessor []*OutputBox

func (p OutputsProcessor) Process(event map[string]interface{}) map[string]interface{} {
	for _, o := range ([]*OutputBox)(p) {
		if o.Pass(event) {
			o.Emit(event)
		}
	}
	return nil
}

type buildFilterFunc func(filterType string, config map[interface{}]interface{}) Filter

func BuildFilterBoxes(config map[string]interface{}, buildFilter buildFilterFunc) []*FilterBox {
	if _, ok := config["filters"]; !ok {
		return nil
	}
	filtersI := config["filters"].([]interface{})
	filters := make([]Filter, len(filtersI))
	for i := 0; i < len(filters); i++ {
		for filterTypeI, filterConfigI := range filtersI[i].(map[interface{}]interface{}) {
			filterType := filterTypeI.(string)
			log.Infof("filter type: (%s)", filterType)
			filterConfig := filterConfigI.(map[interface{}]interface{})
			log.Infof("filter config: (%v)", filterConfig)
			filterPlugin := buildFilter(filterType, filterConfig)
			filters[i] = filterPlugin
		}
	}
	boxes := make([]*FilterBox, len(filters))
	for i := 0; i < len(filters); i++ {
		for _, cfg := range filtersI[i].(map[interface{}]interface{}) {
			boxes[i] = NewFilterBox(cfg.(map[interface{}]interface{}))
			boxes[i].Filter = filters[i]
		}
	}

	return boxes
}
