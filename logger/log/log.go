package log

/*
 * @abstract zap
 * @mail neo532@126.com
 * @date 2023-08-13
 */
import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"

	"github.com/neo532/gofr/logger"
)

var _ logger.ILogger = (*Logger)(nil)

type Logger struct {
	err          error
	paramGlobal  []interface{}
	paramContext []logger.ILoggerArgs

	logger *syslog.Writer
}

func New(opts ...Option) (l *Logger) {
	l = &Logger{
		paramGlobal:  make([]interface{}, 0, 2),
		paramContext: make([]logger.ILoggerArgs, 0, 2),
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

	io.MultiWriter(os.Stdout)
	return
}

func (l *Logger) Log(c context.Context, level logger.Level, message string, p ...interface{}) (err error) {

	var b bytes.Buffer
	b.WriteString(message)

	lg := len(l.paramGlobal)
	for i := 0; i < lg; i += 2 {
		b.WriteString(fmt.Sprintf(" %+v:%+v", l.paramGlobal[i], l.paramGlobal[i+1]))
	}

	lp := len(p)
	for i := 0; i < lp; i += 2 {
		b.WriteString(fmt.Sprintf(" %+v:%+v", p[i], p[i+1]))
	}

	for _, fn := range l.paramContext {
		k, v := fn(c)
		b.WriteString(fmt.Sprintf(" %+v:%+v", k, v))
	}

	switch level {
	case logger.LevelDebug:
		log.Print(b.String())
	case logger.LevelInfo:
		log.Print(b.String())
	case logger.LevelWarn:
		log.Print(b.String())
	case logger.LevelError:
		log.Print(b.String())
	case logger.LevelFatal:
		log.Print(b.String())
	}
	return
}

func (l *Logger) Close() (err error) {
	return
}

func (l *Logger) Err() (err error) {
	return l.err
}
