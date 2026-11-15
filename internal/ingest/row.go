package ingest

import (
	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Limits from build_event_store.py.
const (
	okOut   = 400
	errOut  = 3000
	maxCmd  = 2000
	maxMsg  = 20000
	maxAsst = 4000
)

// NInt is a nullable integer column.
type NInt struct {
	V     int64
	Valid bool
}

func nint(v int64) NInt { return NInt{v, true} }

func boolInt(b bool) NInt {
	if b {
		return NInt{1, true}
	}
	return NInt{0, true}
}

// Row is one event. Empty strings are stored as NULL, except Text when TextSet is true.
type Row struct {
	Source, Agent, SessionID, Project, Cwd, Ts, EventType, Tool string
	Text, Output, Sig                                           string
	TextSet, OutputSet                                          bool
	ExitCode, IsError                                           NInt
	Side                                                        int
	RawFile                                                     string
	RawLine                                                     NInt
}

// base holds the fields every row of one source line shares.
type base struct {
	Source, Agent, SessionID, Cwd, Ts string
	Side                              int
	RawFile                           string
	RawLine                           NInt
}

func (b base) row(eventType string) Row {
	return Row{Source: b.Source, Agent: b.Agent, SessionID: b.SessionID, Cwd: b.Cwd, Ts: b.Ts, EventType: eventType,
		Side: b.Side, RawFile: b.RawFile, RawLine: b.RawLine}
}

// finish ports build_event_store.row(): project from cwd, signature for errors.
func (r Row) finish() Row {
	if r.Cwd != "" && r.Project == "" {
		r.Project = rules.Project(r.Cwd)
	}
	if r.IsError.Valid && r.IsError.V != 0 && r.Sig == "" {
		src := r.Output
		if src == "" {
			src = r.Text
		}
		r.Sig = rules.Signature(src)
	}
	return r
}

func withText(r Row, s string, ok bool) Row {
	r.Text, r.TextSet = s, ok
	return r
}

func withOutput(r Row, s string, ok bool) Row {
	r.Output, r.OutputSet = s, ok
	return r
}

// pasteRows ports paste_error_rows.
func pasteRows(b base, text string) []Row {
	t, sig, ok := rules.PasteError(text)
	if !ok {
		return nil
	}
	r := b.row("user_error_paste")
	r = withText(r, trs(t, 1500), true)
	r.Sig = sig
	r.IsError = nint(1)
	return []Row{r.finish()}
}

func userMsg(b base, text string) []Row {
	r := withText(b.row("user_msg"), trs(text, maxMsg), true).finish()
	return append([]Row{r}, pasteRows(b, text)...)
}

func asstMsg(b base, text string) Row {
	return withText(b.row("assistant_msg"), trs(text, maxAsst), true).finish()
}
