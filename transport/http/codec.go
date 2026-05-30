package http

import (
	"encoding/json"
	"io"
	"net/http"
)

// DecodeRequestFunc decodes an HTTP request into the given value.
type DecodeRequestFunc func(*http.Request, interface{}) error

// EncodeResponseFunc encodes a value into an HTTP response.
type EncodeResponseFunc func(http.ResponseWriter, *http.Request, interface{}) error

// EncodeErrorFunc encodes an error into an HTTP response.
type EncodeErrorFunc func(http.ResponseWriter, *http.Request, error)

// DefaultRequestDecoder decodes request body as JSON.
func DefaultRequestDecoder(r *http.Request, v interface{}) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// DefaultResponseEncoder encodes response as JSON.
func DefaultResponseEncoder(w http.ResponseWriter, _ *http.Request, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if v == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	return json.NewEncoder(w).Encode(v)
}

// DefaultErrorEncoder encodes error as JSON with status code.
func DefaultErrorEncoder(w http.ResponseWriter, r *http.Request, err error) {
	// try to extract HTTP status from the error
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
