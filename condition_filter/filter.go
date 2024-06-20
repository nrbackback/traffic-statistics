package condition_filter

import (
	"regexp"
	"strings"

	"traffic-statistics/value_render"
)

type Filter interface {
	Pass(event map[string]interface{}) bool
}

type ConditionFilter struct {
	conditions []Filter
}

func NewConditionFilter(config map[interface{}]interface{}) *ConditionFilter {
	f := &ConditionFilter{}
	if v, ok := config["if"]; ok {
		f.conditions = make([]Filter, len(v.([]interface{})))
		for i, c := range v.([]interface{}) {
			f.conditions[i] = newCondition(c.(string))
		}
	} else {
		f.conditions = nil
	}
	return f
}

func (f *ConditionFilter) Pass(event map[string]interface{}) bool {
	if f.conditions == nil {
		return true
	}
	for _, c := range f.conditions {
		if !c.Pass(event) {
			return false
		}
	}
	return true
}

func newCondition(c string) Filter {
	c = strings.Trim(c, " ")
	if matched, _ := regexp.MatchString(`^{{.*}}$`, c); matched {
		return newTemplateConditionFilter(c)
	}
	return nil
}

type templateCondition struct {
	ifCondition value_render.ValueRender
	ifResult    string
}

func newTemplateConditionFilter(condition string) *templateCondition {
	return &templateCondition{
		ifCondition: value_render.GetValueRender(condition),
		ifResult:    "y",
	}
}

func (c *templateCondition) Pass(event map[string]interface{}) bool {
	r := c.ifCondition.Render(event)
	if r == nil || r.(string) != c.ifResult {
		return false
	}
	return true
}
