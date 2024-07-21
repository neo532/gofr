package zap

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
	cp(context.Background())

	l, err := New(
		WithFilename("./test.log"),
		WithMaxBackups(2),
		WithMaxSize(2),
		WithLevel("debug"),
		WithGlobalParam("a", "b", "1", "2"),
		WithContextParam(cp),
		WithCallerSkip(2),
		//WithPrettyLogger(nil),
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

	a(c, h)
}

func a(c context.Context, h *logger.Logger) {
	h.WithArgs(logger.LogkeyModule, "m3").Errorf(c, "kkkk%s", "cc")
}
