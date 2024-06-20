package log

import (
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.SugaredLogger
)

func newCore(logDir string) zapcore.Core {
	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.InfoLevel
	})
	warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.WarnLevel
	})
	errorLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.ErrorLevel
	})
	debugLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.DebugLevel
	})
	fatalLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl == zapcore.FatalLevel
	})
	info := newZapCore(logDir, "info.log", infoLevel)
	warn := newZapCore(logDir, "warn.log", warnLevel)
	error := newZapCore(logDir, "error.log", errorLevel)
	debug := newZapCore(logDir, "debug.log", debugLevel)
	fatal := newZapCore(logDir, "fatal.log", fatalLevel)
	return zapcore.NewTee(info, warn, error, debug, fatal)
}

func newWriteSyncer(logDir, logName string) zapcore.WriteSyncer {
	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(logDir, logName),
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     7,
		LocalTime:  true,
		Compress:   true,
	})
}

func newEncoderConfig() zapcore.EncoderConfig {
	custom := zap.NewProductionEncoderConfig()
	custom.TimeKey = "timestamp"
	custom.MessageKey = "message"
	custom.LevelKey = zapcore.OmitKey
	custom.EncodeLevel = zapcore.CapitalLevelEncoder
	custom.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	return custom
}

func newZapCore(logDir, logName string, level zapcore.LevelEnabler) zapcore.Core {
	writer := newWriteSyncer(logDir, logName)
	custom := newEncoderConfig()
	return zapcore.NewCore(
		zapcore.NewJSONEncoder(custom),
		writer,
		level,
	)
}

type config struct {
	LogDir string `yaml:"log_dir"`
}

func NewLogger(c map[string]interface{}) {
	if v, ok := c["log"].(map[interface{}]interface{}); ok {
		if logDir, ok := v["log_dir"].(string); ok {
			core := newCore(logDir)
			l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
			logger = l.Sugar()
		}
	}
}

func Sync() {
	logger.Sync()
}
