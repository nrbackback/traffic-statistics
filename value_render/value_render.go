package value_render

import (
	"regexp"
)

type ValueRender interface {
	Render(map[string]interface{}) interface{}
}

func GetValueRender2(template string) ValueRender {
	return NewOneLevelValueRender(template)
}

// GetValueRender template 需按照 text/template 里的格式
func GetValueRender(template string) ValueRender {
	r := getValueRender(template)
	if r != nil {
		return r
	}
	return nil
}

var matchGoTemp, _ = regexp.Compile(`{{.*}}`)

func getValueRender(template string) ValueRender {
	if matchGoTemp.Match([]byte(template)) {
		return newTemplateValueRender(template)
	}
	return nil
}

func GetBoolValueRender(template bool) ValueRender {
	return NewBoolValueRender(template)
}
