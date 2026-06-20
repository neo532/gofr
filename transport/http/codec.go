package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
)

// Codec represents a pair of request decoder and response encoder for a content type.
type Codec struct {
	ContentType string               // e.g. "application/json"
	Decode      func([]byte, interface{}) error
	Encode      func(interface{}) ([]byte, error)
}

var (
	codecsMu sync.RWMutex
	codecs   = map[string]*Codec{
		"json": {
			ContentType: "application/json",
			Decode:      json.Unmarshal,
			Encode:      json.Marshal,
		},
	}
)

// RegisterCodec registers a codec for a content subtype (e.g. "xml", "yaml").
// It is used by DefaultRequestDecoder and DefaultResponseEncoder to
// select the codec based on the Content-Type / Accept header.
// Built-in: "json".
func RegisterCodec(subType string, c *Codec) {
	codecsMu.Lock()
	codecs[subType] = c
	codecsMu.Unlock()
}

func matchCodec(ct string) *Codec {
	codecsMu.RLock()
	defer codecsMu.RUnlock()

	// extract subtype from Content-Type: "application/xml;charset=utf-8" -> "xml"
	after := ct
	if idx := strings.IndexByte(ct, '/'); idx >= 0 {
		after = ct[idx+1:]
	}
	if idx := strings.IndexAny(after, "; "); idx >= 0 {
		after = after[:idx]
	}
	sub := strings.TrimSpace(strings.ToLower(after))

	if c, ok := codecs[sub]; ok {
		return c
	}
	// fallback to json
	return codecs["json"]
}

// DecodeRequestFunc decodes an HTTP request into the given value.
type DecodeRequestFunc func(*http.Request, interface{}) error

// EncodeResponseFunc encodes a value into an HTTP response.
type EncodeResponseFunc func(http.ResponseWriter, *http.Request, interface{}) error

// EncodeErrorFunc encodes an error into an HTTP response.
type EncodeErrorFunc func(http.ResponseWriter, *http.Request, error)

// DefaultRequestDecoder decodes request body based on Content-Type.
func DefaultRequestDecoder(r *http.Request, v interface{}) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	c := matchCodec(r.Header.Get("Content-Type"))
	return c.Decode(data, v)
}

// DefaultResponseEncoder encodes response based on Accept header.
func DefaultResponseEncoder(w http.ResponseWriter, r *http.Request, v interface{}) error {
	c := matchCodec(r.Header.Get("Accept"))
	w.Header().Set("Content-Type", c.ContentType)
	if v == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	data, err := c.Encode(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// DefaultErrorEncoder encodes error as JSON with status code.
func DefaultErrorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	code := http.StatusInternalServerError
	if sc, ok := err.(interface{ StatusCode() int }); ok {
		code = sc.StatusCode()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
