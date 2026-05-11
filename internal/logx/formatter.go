package logx

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type Formatter struct {
	ColorEnabled bool
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	level := strings.ToUpper(entry.Level.String())
	ts := entry.Time.Format("2006-01-02 15:04:05")
	if !f.ColorEnabled {
		return []byte(fmt.Sprintf("%s [%s] %s\n", ts, level, entry.Message)), nil
	}
	return []byte(fmt.Sprintf("%s [%s%s\033[0m] %s\n", ts, levelColor(entry.Level), level, entry.Message)), nil
}

func levelColor(level logrus.Level) string {
	switch level {
	case logrus.DebugLevel:
		return "\033[36m"
	case logrus.InfoLevel:
		return "\033[32m"
	case logrus.WarnLevel:
		return "\033[33m"
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return "\033[31m"
	default:
		return ""
	}
}

func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
