package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
)

// 定义一些颜色
var colors = []string{
	"\x1b[31m", // 红色
	"\x1b[32m", // 绿色
	"\x1b[33m", // 黄色
	"\x1b[34m", // 蓝色
	"\x1b[35m", // 紫色
	"\x1b[36m", // 青色
	"\x1b[91m", // 浅红色
	"\x1b[92m", // 浅绿色
	"\x1b[93m", // 浅黄色
	"\x1b[94m", // 浅蓝色
	"\x1b[95m", // 浅紫色
}

// CustomFormatter 根据 entry 中 id 的值，
type CustomFormatter struct {
	logrus.TextFormatter
	isTerminal   bool
	terminalInit sync.Once
}

type levelStyle struct {
	label string
	color string
}

var levelStyles = map[logrus.Level]levelStyle{
	logrus.PanicLevel: {label: "PANIC", color: "\x1b[31m"},
	logrus.FatalLevel: {label: "FATAL", color: "\x1b[31m"},
	logrus.ErrorLevel: {label: "ERROR", color: "\x1b[31m"},
	logrus.WarnLevel:  {label: "WARN ", color: "\x1b[33m"},
	logrus.InfoLevel:  {label: "INFO ", color: "\x1b[32m"},
	logrus.DebugLevel: {label: "DEBUG", color: "\x1b[36m"},
	logrus.TraceLevel: {label: "TRACE", color: "\x1b[35m"},
}

// checkIfTerminal 检查 writer 是否为终端
func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		// 检查是否为标准输出/错误输出
		return v == os.Stdout || v == os.Stderr
	default:
		return false
	}
}

// init 初始化格式化器，检测输出是否为终端
func (f *CustomFormatter) init(entry *logrus.Entry) {
	if entry.Logger != nil {
		f.isTerminal = checkIfTerminal(entry.Logger.Out)
	}
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// 初始化终端检测（只执行一次）
	f.terminalInit.Do(func() { f.init(entry) })

	var intID int64
	hasID := false
	if id, ok := entry.Data["id"]; ok {
		hasID = true
		switch v := id.(type) {
		case int:
			intID = int64(v)
		case int64:
			intID = v
		default:
			hasID = false
		}
	}

	// 构建自定义格式的日志
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// 格式: 19:25:59.650  INFO [MODULE]   : message
	timestamp := entry.Time.Format(f.TextFormatter.TimestampFormat)
	style := levelStyles[entry.Level]

	// 写入时间戳
	b.WriteString(timestamp)
	b.WriteString(" ")

	// 写入级别（根据 ForceColors 或终端状态决定是否着色）
	shouldColor := (f.TextFormatter.ForceColors || f.isTerminal) && !f.TextFormatter.DisableColors
	if shouldColor {
		// 终端输出或强制彩色：使用颜色
		b.WriteString(style.color)
		b.WriteString(style.label)
		b.WriteString("\x1b[0m")
	} else {
		// 文件输出：不使用颜色
		b.WriteString(style.label)
	}
	b.WriteString(" ")

	// 写入模块标签
	if module, ok := entry.Data["module"]; ok {
		b.WriteString("[")
		b.WriteString(module.(string))
		b.WriteString("]")
		b.WriteString(" ")
	}

	// 写入消息
	b.WriteString(entry.Message)

	if len(entry.Data) > 0 {

		// 写入分隔符
		b.WriteString(" |")

		// 写入其他字段（排除 id 和 module）
		for k, v := range entry.Data {
			if k != "id" && k != "module" {
				fmt.Fprintf(b, " %s=%v", k, v)
			}
		}

		// 如果有 id，单独为 id 字段着色
		if hasID {
			if shouldColor {
				// 终端输出或强制彩色：为 id 着色
				color := colors[int(intID)%len(colors)]
				b.WriteString(" ")
				b.WriteString(color)
				b.WriteString("id=")
				b.WriteString(strconv.FormatInt(intID, 10))
				b.WriteString("\x1b[0m")
			} else {
				// 文件输出：不着色
				b.WriteString(" id=")
				b.WriteString(strconv.FormatInt(intID, 10))
			}
		}

		b.WriteString(" |")
		b.WriteString("\n")

	}

	return b.Bytes(), nil
}
