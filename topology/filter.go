package topology

import (
	"traffic-statistics/condition_filter"
)

type Filter interface {
	Filter(map[string]interface{}) map[string]interface{}
}

type FilterBox struct {
	Filter          Filter
	conditionFilter *condition_filter.ConditionFilter
	config          map[interface{}]interface{}
}

func NewFilterBox(config map[interface{}]interface{}) *FilterBox {
	f := FilterBox{
		config:          config,
		conditionFilter: condition_filter.NewConditionFilter(config),
	}
	return &f
}

func (b *FilterBox) Process(event map[string]interface{}) map[string]interface{} {
	// 只对满足条件的数据进行过滤，不满足则不处理直接返回原数据
	if b.conditionFilter.Pass(event) {
		return b.Filter.Filter(event)
	}
	return event
}
