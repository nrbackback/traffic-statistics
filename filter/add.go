package filter

import (
	"traffic-statistics/field_setter"
	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
	"traffic-statistics/value_render"
)

type addFilter struct {
	config    map[interface{}]interface{}
	fields    map[field_setter.FieldSetter]value_render.ValueRender
	overwrite bool
}

func init() {
	register("Add", newAddFilter)
}

func newAddFilter(config map[interface{}]interface{}) topology.Filter {
	plugin := &addFilter{
		config:    config,
		fields:    make(map[field_setter.FieldSetter]value_render.ValueRender),
		overwrite: true,
	}
	if overwrite, ok := config["overwrite"]; ok {
		plugin.overwrite = overwrite.(bool)
	}
	if fieldsValue, ok := config["fields"]; ok {
		for f, v := range fieldsValue.(map[interface{}]interface{}) {
			fieldSetter := field_setter.NewFieldSetter(f.(string))
			if fieldSetter == nil {
				log.Fatalf("could build field setter from %s", f.(string))
			}
			if value, ok := v.(bool); ok {
				plugin.fields[fieldSetter] = value_render.GetBoolValueRender(value)
			}
		}
	} else {
		log.Fatal("fields must be set in add filter plugin")
	}
	return plugin
}

func (plugin *addFilter) Filter(event map[string]interface{}) map[string]interface{} {
	for fs, v := range plugin.fields {
		event = fs.SetField(event, v.Render(event), "", plugin.overwrite)
	}
	return event
}
