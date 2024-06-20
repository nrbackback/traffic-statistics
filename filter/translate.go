package filter

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"

	"traffic-statistics/pkg/log"
	"traffic-statistics/topology"
	"traffic-statistics/value_render"
)

func init() {
	register("Translate", newTranslateFilter)
}

type translateFilter struct {
	config         map[interface{}]interface{}
	source         string // 源字段
	target         string // 目标字段
	sourceVR       value_render.ValueRender
	dictionaryPath string                      // 字典目录
	dict           map[interface{}]interface{} // 字典
}

func newTranslateFilter(config map[interface{}]interface{}) topology.Filter {
	plugin := &translateFilter{
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
	if dictionaryPath, ok := config["dictionary_path"]; ok {
		plugin.dictionaryPath = dictionaryPath.(string)
	} else {
		log.Fatal("dictionary_path must be set in translate filter plugin")
	}
	if err := plugin.parseDict(); err != nil {
		log.Fatalf("could not parse (%s):(%s)", plugin.dictionaryPath, err)
	}
	return plugin
}

func (f *translateFilter) Filter(event map[string]interface{}) map[string]interface{} {
	o := f.sourceVR.Render(event)
	if o == nil {
		return event
	}
	if targetValue, ok := f.dict[o]; ok {
		event[f.target] = targetValue
		return event
	}
	return event
}

func (f *translateFilter) parseDict() error {
	configFile, err := os.Open(f.dictionaryPath)
	if err != nil {
		return err
	}
	fi, _ := configFile.Stat()
	if fi.Size() == 0 {
		return fmt.Errorf("config file (%s) is empty", f.dictionaryPath)
	}
	buffer := make([]byte, fi.Size())
	if _, err := configFile.Read(buffer); err != nil {
		return err
	}
	dict := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(buffer, &dict); err != nil {
		return err
	}
	f.dict = dict
	return nil
}
