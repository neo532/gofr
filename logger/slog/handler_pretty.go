package slog

import (
	"context"
	"encoding/json"
	"io"
	"log"

	"golang.org/x/exp/slog"

	"github.com/fatih/color"
	"github.com/neo532/gofr/logger"
)

type PrettyHandler struct {
	slog.Handler
	l            *log.Logger
	contextParam []logger.ILoggerArgs
}

func NewPrettyHandler(
	out io.Writer,
	opts *slog.HandlerOptions,
	contextParam []logger.ILoggerArgs,
) *PrettyHandler {
	h := &PrettyHandler{
		Handler:      slog.NewJSONHandler(out, opts),
		l:            log.New(out, "", 0),
		contextParam: contextParam,
	}

	return h
}

func (h *PrettyHandler) Handle(c context.Context, r slog.Record) error {
	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}

	fields := make(map[string]interface{}, r.NumAttrs())
	for _, fn := range h.contextParam {
		r.AddAttrs(slog.Any(fn(c)))
	}
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()

		return true
	})

	b, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return err
	}

	timeStr := r.Time.Format("[15:05:05.000]")
	msg := color.CyanString(r.Message)

	h.l.Println(timeStr, level, msg, color.WhiteString(string(b)))

	return nil
}

func (h *PrettyHandler) Enabled(_ context.Context, l slog.Level) bool {
	return true
}

func (h *PrettyHandler) WithAttrs(as []slog.Attr) slog.Handler {
	return &PrettyHandler{h.Handler, h.l, h.contextParam}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{h.Handler, h.l, h.contextParam}
}
