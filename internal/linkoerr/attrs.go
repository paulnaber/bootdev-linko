package linkoerr

import (
	"errors"
	"log/slog"
)

type errWithAttrs struct {
	error
	attrs []slog.Attr
}

// WithAttrs wraps err with structured logging attributes for later extraction via Attrs.
func WithAttrs(err error, args ...any) error {
	if err == nil {
		return nil
	}
	return &errWithAttrs{
		error: err,
		attrs: argsToAttr(args),
	}
}

func (e *errWithAttrs) Unwrap() error {
	return e.error
}

func (e *errWithAttrs) Attrs() []slog.Attr {
	return e.attrs
}

type attrError interface {
	Attrs() []slog.Attr
}

// Attrs recursively extracts all logging attributes from an error chain. In the
// case of duplicate keys, the outermost value takes precedence.
func Attrs(err error) []slog.Attr {
	var raw []slog.Attr
	for err != nil {
		if ae, ok := err.(attrError); ok {
			raw = append(raw, ae.Attrs()...)
		}
		err = errors.Unwrap(err)
	}
	return dedupeOuterWins(raw)
}

func dedupeOuterWins(attrs []slog.Attr) []slog.Attr {
	seen := make(map[string]struct{}, len(attrs))
	out := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		if _, ok := seen[a.Key]; ok {
			continue
		}
		seen[a.Key] = struct{}{}
		out = append(out, a)
	}
	return out
}

// argsToAttr turns a list of typed or untyped values into a slice of [slog.Attr].
// args[i] is treated as a key if it is a string or an [slog.Attr]; otherwise, it
// is treated as a value with key "!BADKEY".
func argsToAttr(args []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(args))
	for i := 0; i < len(args); {
		switch key := args[i].(type) {
		case slog.Attr:
			attrs = append(attrs, key)
			i++
		case string:
			if i+1 >= len(args) {
				attrs = append(attrs, slog.String("!BADKEY", key))
				i++
			} else {
				attrs = append(attrs, slog.Any(key, args[i+1]))
				i += 2
			}
		default:
			attrs = append(attrs, slog.Any("!BADKEY", args[i]))
			i++
		}
	}
	return attrs
}
