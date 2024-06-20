package field_setter

type oneLevelFieldSetter struct {
	field string
}

func newOneLevelFieldSetter(field string) *oneLevelFieldSetter {
	r := &oneLevelFieldSetter{
		field: field,
	}
	return r
}

func (fs *oneLevelFieldSetter) SetField(event map[string]interface{}, value interface{}, field string, overwrite bool) map[string]interface{} {
	if _, ok := event[fs.field]; !ok || overwrite {
		event[fs.field] = value
	}
	return event
}
