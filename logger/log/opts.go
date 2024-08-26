package log

/*
 * @abstract zap's option
 * @mail neo532@126.com
 * @date 2023-08-13
 */
import (
	"log/syslog"

	"github.com/neo532/gofr/logger"
)

type Option func(opt *Logger)

func WithLogger(log *syslog.Writer) Option {
	return func(l *Logger) {
		l.logger = log
	}
}

func WithContextParam(fns ...logger.ILoggerArgs) Option {
	return func(l *Logger) {
		l.paramContext = fns
	}
}

func WithGlobalParam(kvs ...interface{}) Option {
	return func(l *Logger) {
		l.paramGlobal = kvs
	}
}

func WithLevel(lv syslog.Priority) Option {
	return func(l *Logger) {
	}
}
