package logging

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/pkg/utils"
)

var logging *logrus.Logger

func Setup(cfgPath string) {
	fcfg, err := config.NewConfig(cfgPath, 0)
	var cfg *config.LogConfig
	if err == nil {
		cfg = fcfg.Log
	} else {
		// for testing
		cfg = &config.LogConfig{
			Level:  "debug",
			ToFile: false,
		}
	}
	level, err := logrus.ParseLevel(cfg.Level)
	utils.PanicOnError(err)
	var out io.Writer
	if cfg.ToFile {
		filename := fmt.Sprintf(cfg.Filename, time.Now().Format("2006-0102-150405"))
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
		utils.PanicOnError(err)
		out = io.MultiWriter(os.Stdout, file)
	} else {
		out = os.Stdout
	}
	logging = &logrus.Logger{
		Out: out,
		Formatter: &CustomFormatter{ // 使用自定义 Formatter
			TextFormatter: logrus.TextFormatter{
				ForceColors:     true,
				TimestampFormat: "20060102-150405.000",
				FullTimestamp:   true,
			},
		},
		Level: level,
	}
}

// should be called after InitLog
func GetLogger() *logrus.Logger {
	return logging
}

func DebugF(format string, args ...interface{}) {
	logging.Debugf(format, args...)
}

func InfoF(format string, args ...interface{}) {
	logging.Infof(format, args...)
}

func WarnF(format string, args ...interface{}) {
	logging.Warnf(format, args...)
}

func ErrorF(format string, args ...interface{}) {
	logging.Errorf(format, args...)
}

func FatalF(format string, args ...interface{}) {
	logging.Fatalf(format, args...)
}

func Debug(args ...interface{}) {
	logging.Debug(args...)
}

func Info(args ...interface{}) {
	logging.Infoln(args...)
}

func Warn(args ...interface{}) {

	logging.Warnln(args...)
}
func Error(args ...interface{}) {
	logging.Errorln(args...)
}

func Fatal(args ...interface{}) {
	logging.Fatalln(args...)
}

func TraceF(format string, args ...interface{}) {
	logging.Tracef(format, args...)
}

func Trace(args ...interface{}) {
	logging.Traceln(args...)
}
