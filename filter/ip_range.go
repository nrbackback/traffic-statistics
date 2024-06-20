package filter

import (
	"net"

	"traffic-statistics/pkg/log"
	"traffic-statistics/pkg/utils"
	"traffic-statistics/topology"
	"traffic-statistics/value_render"
)

func init() {
	register("IPRange", newIPRangeFilter)
}

type ipRangeFilter struct {
	config        map[interface{}]interface{}
	source        string // 源字段
	target        string // 目标字段
	sourceVR      value_render.ValueRender
	netConfigFile string // IP范围配置的文件
}

func newIPRangeFilter(config map[interface{}]interface{}) topology.Filter {
	plugin := &ipRangeFilter{
		config: config,
	}
	if source, ok := config["source"]; ok {
		plugin.source = source.(string)
	} else {
		log.Fatal("source must be set in translate filter plugin")
	}
	plugin.sourceVR = value_render.GetValueRender2(plugin.source)
	if target, ok := config["target"]; ok {
		plugin.target = target.(string)
	} else {
		log.Fatal("target must be set in translate filter plugin")
	}
	if path, ok := config["net_path"]; ok {
		plugin.netConfigFile = path.(string)
	} else {
		log.Fatal("dictionary_path must be set in translate filter plugin")
	}
	return plugin
}

func (f *ipRangeFilter) Filter(event map[string]interface{}) map[string]interface{} {
	o := f.sourceVR.Render(event)
	if o == nil {
		return event
	}
	if v, ok := o.(string); ok {
		ip := net.ParseIP(v)
		if ip == nil {
			log.Errorw("parse input to ip error", "input", v)
			return event
		}
		if utils.IsPublicIP(ip) {
			event[f.target] = "public"
		}
	}
	// TODO: 根据netConfigFilePath判断source是上海还是常州
	return event
}
