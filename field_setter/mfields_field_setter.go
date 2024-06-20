package field_setter

import "reflect"

type multiLevelFieldSetter struct {
	preFields []string
	lastField string
}

func newMultiLevelFieldSetter(fields []string) *multiLevelFieldSetter {
	fieldsLength := len(fields)
	preFields := make([]string, fieldsLength-1)
	for i := range preFields {
		preFields[i] = fields[i]
	}
	return &multiLevelFieldSetter{
		preFields: preFields,
		lastField: fields[fieldsLength-1],
	}
}

func (fs *multiLevelFieldSetter) SetField(event map[string]interface{}, value interface{}, field string, overwrite bool) map[string]interface{} {
	current := event
	for _, field := range fs.preFields {
		if value, ok := current[field]; ok {
			if reflect.TypeOf(value).Kind() == reflect.Map {
				current = value.(map[string]interface{})
			}
		} else {
			a := make(map[string]interface{})
			current[field] = a
			current = a
		}
	}
	current[fs.lastField] = value
	return event
}
