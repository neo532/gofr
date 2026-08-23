package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/neo532/gofr/transport"
	"github.com/neo532/gokit/errorx"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Codec represents a pair of request decoder and response encoder for a content type.
type Codec struct {
	ContentType string // e.g. "application/json"
	Decode      func([]byte, any) error
	Encode      func(any) ([]byte, error)
}

var (
	codecsMu sync.RWMutex
	codecs   = map[string]*Codec{
		"json": {
			ContentType: "application/json",
			Decode:      json.Unmarshal,
			Encode: func(v any) ([]byte, error) {
				if pm, ok := v.(proto.Message); ok {
					return protojson.MarshalOptions{UseEnumNumbers: true}.Marshal(pm)
				}
				return json.Marshal(v)
			},
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
type DecodeRequestFunc func(*http.Request, any) error

// EncodeResponseFunc encodes a value into an HTTP response.
type EncodeResponseFunc func(http.ResponseWriter, *http.Request, any) error

// EncodeErrorFunc encodes an error into an HTTP response.
type EncodeErrorFunc func(http.ResponseWriter, *http.Request, error)

// decodeQuery converts URL query params into JSON and decodes them into v.
// Repeated fields must be emitted as JSON arrays (encoding/json does not
// coerce a single scalar into a slice), so list fields are detected via the
// proto descriptor; scalar fields get the single value.
func decodeQuery(q url.Values, v any) error {
	if len(q) == 0 {
		return nil
	}
	m := make(map[string]any, len(q))
	for k, vs := range q {
		if isRepeatedField(v, k) {
			m[k] = vs
		} else if len(vs) == 1 {
			m[k] = vs[0]
		} else {
			m[k] = vs
		}
	}
	bt, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(bt, v)
}

// isRepeatedField reports whether v is a proto message whose field matching
// name (proto snake_case or JSON camelCase) is a list field. Non-proto values
// have no field schema, so decodeQuery falls back to its array-vs-scalar
// heuristic for them.
func isRepeatedField(v any, name string) bool {
	pm, ok := v.(proto.Message)
	if !ok {
		return false
	}
	fields := pm.ProtoReflect().Descriptor().Fields()
	if fd := fields.ByName(protoreflect.Name(name)); fd != nil {
		return fd.IsList()
	}
	if fd := fields.ByJSONName(name); fd != nil {
		return fd.IsList()
	}
	return false
}

// DefaultRequestDecoder decodes request body based on Content-Type. When the
// body is empty (typical GET), it falls back to decoding URL query params, so
// ?key=a&key=b reaches handlers instead of being silently dropped.
func DefaultRequestDecoder(r *http.Request, v any) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return decodeQuery(r.URL.Query(), v)
	}

	c := matchCodec(r.Header.Get("Content-Type"))
	return c.Decode(data, v)
}

// DefaultResponseEncoder encodes response based on Accept header.
func DefaultResponseEncoder(w http.ResponseWriter, r *http.Request, v any) error {
	c := matchCodec(r.Header.Get("Accept"))
	w.Header().Set("Content-Type", c.ContentType)
	if v == nil {
		w.WriteHeader(http.StatusNoContent)
		_, _ = w.Write(nil)
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
	json.NewEncoder(w).Encode(map[string]any{
		"error": err.Error(),
	})
}

// DefaultPanicHandler logs the panic stack via the Transporter's App logger
// when available, then responds through the server's ErrorEncoder so the
// client gets the standard error response.
func (s *Server) DefaultPanicHandler(w http.ResponseWriter, r *http.Request, v any) {
	if tr, ok := transport.FromServerContext(r.Context()); ok {
		if app := tr.App(); app != nil {
			app.Logger().Error(r.Context(), "panic", "value", v, "stack", string(debug.Stack()))
		}
	}
	s.ene(w, r, errorx.New("panic"))
}

// Parse parses a single string as T. Supported T: string and the numeric/bool
// scalars (int64/int32/uint64/uint32/float64/float32/bool). The type dispatch
// is one interface box + one pointer-compare per call; for string it degrades
// to a direct assignment. Callers pass the raw value, e.g. a path param
// (Parse[int64](ps.ByName("id"))) or a query param
// (Parse[string](r.URL.Query().Get("key"))).
func Parse[T any](s string) (T, error) {
	var zero T
	switch p := any(&zero).(type) {
	case *string:
		*p = s
	case *int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return zero, err
		}
		*p = n
	case *int32:
		n, err := strconv.ParseInt(s, 10, 32)
		if err != nil {
			return zero, err
		}
		*p = int32(n)
	case *uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return zero, err
		}
		*p = n
	case *uint32:
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return zero, err
		}
		*p = uint32(n)
	case *float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return zero, err
		}
		*p = n
	case *float32:
		n, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return zero, err
		}
		*p = float32(n)
	case *bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return zero, err
		}
		*p = b
	default:
		return zero, fmt.Errorf("unsupported type %T", &zero)
	}
	return zero, nil
}
