package ingest

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Helpers that reproduce the Python semantics of the reference on values from encoding/json
// (map[string]any, []any, string, float64, bool, nil).

func get(v any, k string) any {
	if m, ok := v.(map[string]any); ok {
		return m[k]
	}
	return nil
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

// or returns the first truthy value, else the last one (Python `a or b or c`).
func or(vs ...any) any {
	for _, v := range vs {
		if truthy(v) {
			return v
		}
	}
	return vs[len(vs)-1]
}

// pyStr is Python str(v) for scalars (used in f-strings such as f'mcp:{server}:{tool}').
func pyStr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case string:
		return x
	case bool:
		if x {
			return "True"
		}
		return "False"
	case float64:
		return fmtNum(x)
	}
	return pyDumps(v)
}

func fmtNum(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e16 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

// pyDumps is json.dumps(v, ensure_ascii=False) with Python separators. Map keys are sorted
// (encoding/json does not keep key order); this only changes the stored text of a few tool calls.
func pyDumps(v any) string {
	var b strings.Builder
	dump(&b, v)
	return b.String()
}

func dump(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case float64:
		b.WriteString(fmtNum(x))
	case string:
		dumpStr(b, x)
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			dump(b, e)
		}
		b.WriteByte(']')
	case []string:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			dumpStr(b, e)
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			dumpStr(b, k)
			b.WriteString(": ")
			dump(b, x[k])
		}
		b.WriteByte('}')
	}
}

func dumpStr(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				b.WriteString(`\u00`)
				b.WriteString(strconv.FormatInt(int64(r)>>4, 16))
				b.WriteString(strconv.FormatInt(int64(r)&15, 16))
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

// tr ports build_event_store.tr: non-strings are JSON-dumped, then cut to n code points plus '…'.
// The bool result is false when v is None (the Python returns None).
func tr(v any, n int) (string, bool) {
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		s = pyDumps(v)
	}
	if len(s) <= n || rules.RuneLen(s) <= n {
		return s, true
	}
	return rules.Trunc(s, n) + "…", true
}

func trs(v any, n int) string { s, _ := tr(v, n); return s }

// toInt ports Python int(x) for exit codes; ok=false when Python would raise or x is None.
func toInt(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		return int64(x), true
	case bool:
		if x {
			return 1, true
		}
		return 0, true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// isoMs ports iso_ms: epoch milliseconds to 2006-01-02T15:04:05.000Z (UTC).
func isoMs(ms float64) string {
	us := math.Round(ms * 1000)
	t := time.UnixMicro(int64(us)).UTC()
	return t.Format("2006-01-02T15:04:05.000Z")
}

// blockText ports block_text.
func blockText(c any) string {
	switch x := c.(type) {
	case string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, b := range x {
			switch bb := b.(type) {
			case map[string]any:
				switch str(bb["type"]) {
				case "text", "input_text", "output_text":
					out = append(out, str(bb["text"]))
				case "tool_result":
					out = append(out, blockText(bb["content"]))
				}
			case string:
				out = append(out, bb)
			}
		}
		return strings.Join(out, "\n")
	}
	return ""
}
