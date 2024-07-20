package slog

import (
	"os"

	"golang.org/x/exp/slog"

	"github.com/neo532/gofr/logger"
)

type Option func(opt *Logger)

func WithHandler(handler slog.Handler) Option {
	return func(l *Logger) {
		if handler == nil {
			l.logger = slog.New(
				NewPrettyHandler(os.Stdout, l.opts, l.paramContext),
			).With(l.paramGlobal...)
			return
		}
		l.logger = slog.New(handler).With(l.paramGlobal...)
		return
	}
}

func WithReplaceAttr(fns ...func() (k string, v interface{})) Option {
	return func(l *Logger) {
		l.opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			for _, fn := range fns {
				k, v := fn()
				if k == a.Key {
					if v == nil {
						a.Key = k
						break
					}
					a = slog.Any(k, v)
					break
				}
			}
			return a
		}
	}
}

func WithContextParam(fns ...logger.ILoggerArgs) Option {
	return func(l *Logger) {
		l.paramContext = fns
	}
}

func WithGlobalParam(vs ...interface{}) Option {
	return func(l *Logger) {
		l.paramGlobal = vs
	}
}

func WithLevel(lv string) Option {
	return func(l *Logger) {
		lvl := (&slog.LevelVar{})
		if err := lvl.UnmarshalText([]byte(lv)); err != nil && l.err == nil {
			l.err = err
			return
		}
		l.opts.Level = lvl
	}
}

func WithFilename(s string) Option {
	return func(l *Logger) {
		l.syncerConf.Filename = s
	}
}

// 日志的最大大小（M）
func WithMaxSize(i int) Option {
	return func(l *Logger) {
		l.syncerConf.MaxSize = i
	}
}

// 日志文件存储最大天数
func WithMaxAge(i int) Option {
	return func(l *Logger) {
		l.syncerConf.MaxAge = i
	}
}

// 日志的最大保存数量
func WithMaxBackups(i int) Option {
	return func(l *Logger) {
		l.syncerConf.MaxBackups = i
	}
}

// 是否执行压缩
func WithCompress(b bool) Option {
	return func(l *Logger) {
		l.syncerConf.Compress = b
	}
}
