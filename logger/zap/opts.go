package zap

/*
 * @abstract zap's option
 * @mail neo532@126.com
 * @date 2023-08-13
 */
import (
	"io"
	"os"

	"github.com/neo532/gofr/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Option func(opt *Logger)

func WithPrettyLogger(w io.Writer) Option {
	return func(l *Logger) {
		if w == nil {
			w = os.Stdout
		}
		l.logger = zap.New(
			zapcore.NewCore(
				zapcore.NewJSONEncoder(l.core),
				zapcore.AddSync(w),
				l.levelEnabler,
			),
			l.opts...)
		l.Sync = l.logger.Sync
		return
	}
}

func WithCallerSkip(skip int) Option {
	return func(l *Logger) {
		l.opts = append(l.opts, zap.WithCaller(true))
		l.opts = append(l.opts, zap.AddCallerSkip(skip))
	}
}

func WithContextParam(fns ...logger.ILoggerArgs) Option {
	return func(l *Logger) {
		l.paramContext = fns
	}
}

func WithGlobalParam(kvs ...interface{}) Option {
	return func(l *Logger) {
		ls := len(kvs)
		ps := make([]zap.Field, 0, ls/2)
		for i := 0; i < ls; i += 2 {
			k, _ := kvs[i].(string)
			ps = append(ps, zap.Any(k, kvs[i+1]))
		}
		l.opts = append(l.opts, zap.Fields(ps...))
	}
}

func WithLevel(lv string) Option {
	return func(l *Logger) {
		var err error
		if l.levelEnabler, err = zapcore.ParseLevel(lv); err != nil {
			l.err = err
		}
	}
}

func WithFilename(s string) Option {
	return func(l *Logger) {
		l.syncerConf.Filename = s
	}
}

func WithMaxSize(i int) Option {
	return func(l *Logger) {
		l.syncerConf.MaxSize = i
	}
}

func WithMaxAge(i int) Option {
	return func(l *Logger) {
		l.syncerConf.MaxAge = i
	}
}

func WithMaxBackups(i int) Option {
	return func(l *Logger) {
		l.syncerConf.MaxBackups = i
	}
}

func WithCompress(b bool) Option {
	return func(l *Logger) {
		l.syncerConf.Compress = b
	}
}
