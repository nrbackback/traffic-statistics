package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	// _ "go.uber.org/automaxprocs"

	"traffic-statistics/config"
	"traffic-statistics/pkg/log"
)

var configFile = flag.String("c", "config/config.yml", "config file")

// exitWhenNil 只需要在只开启上传模块的时候配置为 true
var exitWhenNil = flag.Bool("exit-when-nil", false, "triger gohangout to exit when receive a nil event")

var mainThreadExitChan chan struct{} = make(chan struct{}, 0)

func main() {
	runtime.GOMAXPROCS(1)              // 限制 CPU 使用数，避免过载
	runtime.SetMutexProfileFraction(1) // 开启对锁调用的跟踪
	runtime.SetBlockProfileRate(1)     // 开启对阻塞操作的跟踪
	go func() {
		// 启动一个 http server，注意 pprof 相关的 handler 已经自动注册过了
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	flag.Parse()
	config, err := config.ParseConfig(*configFile)
	if err != nil {
		msg := fmt.Sprintf("load config file error: (%v)", err)
		panic(msg)
	}
	// if v, ok := config["go_max_procs"]; ok {
	// if procs, ok := v.(int); ok {
	// runtime.GOMAXPROCS(procs)
	// }
	// }

	log.NewLogger(config)

	boxes, err := buildPluginLink(config)
	if err != nil {
		log.Fatalw("build plugin link error", "error", err)
	}
	inputs := Inputs(boxes)
	go inputs.start()
	go listenSignal()

	go func() {
		for cfg := range configChannel {
			inputs.stop()
			// 配置变更后运用新配置
			log.NewLogger(cfg)
			boxes, err := buildPluginLink(cfg)
			if err == nil {
				inputs = Inputs(boxes)
				go inputs.start()
			} else {
				log.Errorw("build plugin link error", "error", err)
				mainThreadExitChan <- struct{}{}
			}
		}
	}()
	// 默认监听配置变更
	if err := watchConfig(*configFile, configChannel); err != nil {
		log.Fatalw("watch config fail", "error", err)
	}

	<-mainThreadExitChan
	inputs.stop()
	log.Info("goodbye.")
	log.Sync()
}

func listenSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	log.Infow("got signal", "signal", sig)
	log.Info("shutting down")
	mainThreadExitChan <- struct{}{}
}
