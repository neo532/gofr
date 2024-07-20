package slog

import (
	"context"
	"fmt"
	"testing"

	"github.com/neo532/gofr/logger"
)

func createLog() (h *logger.Logger) {
	cp := func(c context.Context) (key string, value interface{}) {
		return "aa", "bbbbbbbbb"
	}
	sp := func(c context.Context) (key string, value interface{}) {
		fn, file, line := logger.GetSourceByFunctionName(
			0,
			20,
			[]string{"github.com/neo532/gofr/logger/slog"},
			[]string{
				"github.com/neo532/gofr/logger/slog.GetSourceByFunctionName",
				"github.com/neo532/gofr/logger/slog.createLog.func2",
				"github.com/neo532/gofr/logger/slog.(*Logger).Log",
				"github.com/neo532/gofr/logger.(*Logger).Errorf",
				"github.com/neo532/gofr/logger.(*Logger).Error",
				"github.com/neo532/gofr/logger/slog.(*PrettyHandler).Handle",
			},
		)
		return "source", fmt.Sprintf("%s,%s,%d", fn, file, line)
	}

	l, err := New(
		WithFilename("./test.log"),
		WithMaxBackups(2),
		WithMaxSize(2),
		WithLevel("debug"),
		WithGlobalParam("a", "b", "1", "2"),
		WithContextParam(cp, sp),
		WithReplaceAttr(func() (k string, v interface{}) { return "msg", nil }),
		WithHandler(nil),
	)
	if err != nil {
		fmt.Println(fmt.Sprintf("err:\t%+v", err))
	}
	return logger.NewLogger(l)
}
func TestLogger(t *testing.T) {

	c := context.Background()
	h := createLog()
	for i := 0; i < 1; i++ {
		h.WithArgs(logger.LogkeyModule, "m1").Error(c, "kkkk", "vvvv", "cc")
		h.WithArgs(logger.LogkeyModule, "m2").Errorf(c, "kkkk%s", "cc")
	}
}
