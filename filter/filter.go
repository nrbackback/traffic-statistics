package filter

import (
	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
)

type buildFilterFunc func(map[interface{}]interface{}) topology.Filter

var registeredFilter map[string]buildFilterFunc = make(map[string]buildFilterFunc)

func register(filterType string, bf buildFilterFunc) {
	if _, ok := registeredFilter[filterType]; ok {
		log.Errorf("(%s) has been registered, ignore (%T)", filterType, bf)
		return
	}
	registeredFilter[filterType] = bf
}

func BuildFilter(filterType string, config map[interface{}]interface{}) topology.Filter {
	if v, ok := registeredFilter[filterType]; ok {
		return v(config)
	}
	log.Infow("could not build filter", "type", filterType)
	return nil
}
