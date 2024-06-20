package codec

import (
	"reflect"

	"traffic-statistics/pkg/log"
)

type structTagDecoder struct {
	tag string
}

func (d *structTagDecoder) Decode(i interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		log.Errorf("toMap only accepts structs, got (%T)", v)
		return nil
	}
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fi := typ.Field(i)
		if tagv := fi.Tag.Get(d.tag); tagv != "" {
			field := v.Field(i).Interface()
			if field != reflect.Zero(reflect.TypeOf(field)).Interface() {
				out[tagv] = v.Field(i).Interface()
			}
		}
	}
	return out
}
