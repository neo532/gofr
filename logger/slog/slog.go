package slog

import (
	"context"

	"golang.org/x/exp/slog"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/neo532/gofr/logger"
)

var _ logger.ILogger = (*Logger)(nil)

type Logger struct {
	err          error
	paramGlobal  []interface{}
	paramContext []logger.ILoggerArgs

	syncerConf *lumberjack.Logger
	logger     *slog.Logger
	opts       *slog.HandlerOptions
}

func New(opts ...Option) (l *Logger) {

	l = &Logger{
		paramGlobal:  make([]interface{}, 0, 2),
		paramContext: make([]logger.ILoggerArgs, 0, 2),
		syncerConf:   &lumberjack.Logger{},
		opts:         &slog.HandlerOptions{},
	}
	for _, o := range opts {
		o(l)
	}
	if l.err != nil {
		return
	}

	if l.logger != nil {
		return
	}

	l.logger = slog.New(
		slog.NewJSONHandler(l.syncerConf, l.opts),
	).With(l.paramGlobal...)
	return
}

func (l *Logger) Opts() *slog.HandlerOptions {
	return l.opts
}

func (l *Logger) Close() (err error) {
	return l.syncerConf.Close()
}

func (l *Logger) ParamContext() []logger.ILoggerArgs {
	return l.paramContext
}

func (l *Logger) Log(c context.Context, level logger.Level, message string, p ...interface{}) (err error) {

	for _, fn := range l.paramContext {
		p = append(p, slog.Any(fn(c)))
	}

	switch level {
	case logger.LevelDebug:
		l.logger.Log(c, slog.LevelDebug, message, p...)
	case logger.LevelInfo:
		l.logger.Log(c, slog.LevelInfo, message, p...)
	case logger.LevelWarn:
		l.logger.Log(c, slog.LevelWarn, message, p...)
	case logger.LevelError:
		l.logger.Log(c, slog.LevelError, message, p...)
	case logger.LevelFatal:
		l.logger.Log(c, slog.LevelError, message, p...)
	}

	return
}

func (l *Logger) Err() (err error) {
	return l.err
}
