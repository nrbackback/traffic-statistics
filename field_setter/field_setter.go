package field_setter

import (
	"regexp"
)

type FieldSetter interface {
	SetField(event map[string]interface{}, value interface{}, fieldName string, overwrite bool) map[string]interface{}
}

func NewFieldSetter(template string) FieldSetter {
	matchp, _ := regexp.Compile(`(\[.*?\])+`)
	findp, _ := regexp.Compile(`(\[(.*?)\])`)
	if matchp.Match([]byte(template)) {
		fields := make([]string, 0)
		for _, v := range findp.FindAllStringSubmatch(template, -1) {
			fields = append(fields, v[2])
		}
		if len(fields) == 1 {
			return newOneLevelFieldSetter(fields[0])
		}
		return newMultiLevelFieldSetter(fields)
	} else {
		return newOneLevelFieldSetter(template)
	}
}
