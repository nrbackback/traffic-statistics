package main

import (
	"fmt"
	"runtime"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"traffic-statistics/config"
	"traffic-statistics/pkg/log"
)

type Watcher interface {
	watch(filename string, configChannel chan<- map[string]interface{}) error
}

func watchConfig(filename string, configChannel chan<- map[string]interface{}) error {
	var watcher Watcher
	watcher = &fileWatcher{}

	return watcher.watch(filename, configChannel)
}

type fileWatcher struct{}

var configChannel = make(chan map[string]interface{})

func (f fileWatcher) watch(filename string, configChannel chan<- map[string]interface{}) error {
	vp := viper.New()
	vp.SetConfigFile(filename)
	vp.WatchConfig()
	vp.OnConfigChange(func(e fsnotify.Event) {
		log.Info("config file changed")
		config, err := config.ParseConfig(*configFile)
		if err != nil {
			msg := fmt.Sprintf("load config file error: (%v)", err)
			panic(msg)
		}
		if v, ok := config["go_max_procs"]; ok {
			if procs, ok := v.(int); ok {
				runtime.GOMAXPROCS(procs)
			}
		}
		configChannel <- config
	})
	return nil
}
