package request

/*
 * @abstract logger for request
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

import (
	"context"
	"fmt"
	"time"
)

var logger Logger = &LoggerDefault{}

// Logger is a interface for Log.
type Logger interface {
	// Log's situation:
	// Timeout,if cost>limit,
	// StatusCode is bad,if statusCode!=http.StatusOK
	Log(c context.Context, statusCode int, curl string, limit time.Duration, cost time.Duration, resp []byte, err error)
}

// RegLogger is a register at starup.
func RegLogger(l Logger) {
	logger = l
}

// LoggerDefault is a default value for logger.
type LoggerDefault struct {
}

// Log is a default value to logger for showing.
func (l *LoggerDefault) Log(c context.Context, statusCode int, curl string, limit time.Duration, cost time.Duration, resp []byte, err error) {
	var logMsg = fmt.Sprintf("[%s] [code:%+v] [limit:%+v] [cost:%+v] [%+v]",
		curl,
		statusCode,
		limit,
		cost,
		string(resp),
	)
	fmt.Println(logMsg)
	return
}
