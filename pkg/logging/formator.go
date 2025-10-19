package logging

import (
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

// CustomFormatter 根据 entry 中 id 的值，为整条日志设置不同颜色
type CustomFormatter struct {
	logrus.TextFormatter
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var intID int64
	if id, ok := entry.Data["id"]; ok {
		switch v := id.(type) {
		case int:
			intID = int64(v)
		case int64:
			intID = v
		default:
			intID = 0
		}
	}
	// 先调用内置 TextFormatter 生成日志文本
	text, err := f.TextFormatter.Format(entry)
	if err != nil {
		return nil, err
	}
	// 根据 id 值选取一种颜色
	color := colors[int(intID)%len(colors)]
	// 用选择的颜色包装整条日志文本
	return []byte(color + string(text) + "\x1b[0m"), nil
}
