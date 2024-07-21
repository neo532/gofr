package zap

/*
 * @abstract zap
 * @mail neo532@126.com
 * @date 2023-08-13
 */
import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/neo532/gofr/logger"
)

var _ logger.ILogger = (*Logger)(nil)

type Logger struct {
	err          error
	paramGlobal  []interface{}
	paramContext []logger.ILoggerArgs

	syncerConf *lumberjack.Logger
	logger     *zap.Logger

	encoder      zapcore.Encoder
	writeSyncer  zapcore.WriteSyncer
	levelEnabler zapcore.LevelEnabler
	core         zapcore.EncoderConfig

	opts []zap.Option
	Sync func() error
}

func New(opts ...Option) (l *Logger, err error) {
	l = &Logger{
		paramGlobal:  make([]interface{}, 0, 2),
		paramContext: make([]logger.ILoggerArgs, 0, 2),
		syncerConf:   &lumberjack.Logger{},
		core: zapcore.EncoderConfig{
			LevelKey:       "level",
			TimeKey:        "time",
			CallerKey:      "source",
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.RFC3339TimeEncoder,
			EncodeDuration: zapcore.NanosDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			LineEnding:     zapcore.DefaultLineEnding,
		},
	}
	for _, o := range opts {
		o(l)
	}
	if err = l.err; err != nil {
		return
	}

	if l.logger != nil {
		return
	}

	l.logger = zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(l.core),
			zapcore.AddSync(l.syncerConf),
			l.levelEnabler,
		),
		l.opts...)
	l.Sync = l.logger.Sync
	return
}

func (l *Logger) Log(c context.Context, level logger.Level, message string, p ...interface{}) (err error) {

	ls := len(p)
	ps := make([]zap.Field, 0, ls/2+len(l.paramContext))

	for i := 0; i < ls; i += 2 {
		s, _ := p[i].(string)
		ps = append(ps, zap.Any(s, p[i+1]))
	}

	for _, fn := range l.paramContext {
		ps = append(ps, zap.Any(fn(c)))
	}

	switch level {
	case logger.LevelDebug:
		l.logger.Log(zapcore.DebugLevel, message, ps...)
	case logger.LevelInfo:
		l.logger.Log(zapcore.InfoLevel, message, ps...)
	case logger.LevelWarn:
		l.logger.Log(zapcore.WarnLevel, message, ps...)
	case logger.LevelError:
		l.logger.Log(zapcore.ErrorLevel, message, ps...)
	case logger.LevelFatal:
		l.logger.Log(zapcore.FatalLevel, message, ps...)
	}
	return nil
}
