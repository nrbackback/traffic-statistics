package filter

import (
	"net"

	"github.com/mitchellh/mapstructure"

	"traffic-statistics/pkg/log"
	"traffic-statistics/pkg/utils"
	"traffic-statistics/topology"
	"traffic-statistics/value_render"
)

func init() {
	register("FlowDirection", newFlowDirectionFilter)
}

type flowDirectionFilter struct {
	config          map[interface{}]interface{}
	servicePublicIP []string
	target          string // 目标字段
	srcIPVR         value_render.ValueRender
	dstIPVR         value_render.ValueRender
}

func newFlowDirectionFilter(config map[interface{}]interface{}) topology.Filter {
	plugin := &flowDirectionFilter{
		config: config,
	}
	if ipList, ok := config["service_public_ip"]; ok {
		err := mapstructure.Decode(ipList, &plugin.servicePublicIP)
		if err != nil {
			log.Fatal("wrong config of service_public_ip in flow direction filter plugin", "error", err)
		}
	} else {
		log.Fatal("service_public_ip must be set in flow direction filter plugin")
	}
	plugin.srcIPVR = value_render.GetValueRender2("src_ip")
	plugin.dstIPVR = value_render.GetValueRender2("dst_ip")
	if target, ok := config["target"]; ok {
		plugin.target, ok = target.(string)
		if !ok {
			log.Fatal("wrong config of target in flow direction filter plugin")
		}
	} else {
		log.Fatal("target must be set in flow direction filter plugin")
	}
	return plugin
}

func (f *flowDirectionFilter) Filter(event map[string]interface{}) map[string]interface{} {
	var setTraget = func(o interface{}, value string) {
		if v, ok := o.(string); ok {
			ip := net.ParseIP(v)
			if ip == nil {
				log.Errorw("parse input to ip error", "input", v)
				return
			}
			if utils.IsPublicIP(ip) && !utils.StringInArray(v, f.servicePublicIP) && event[f.target] == nil {
				event[f.target] = value
			}
		}
	}
	if o := f.srcIPVR.Render(event); o != nil {
		setTraget(o, "in")
	}
	if o := f.dstIPVR.Render(event); o != nil {
		setTraget(o, "out")
	}
	return event
}
