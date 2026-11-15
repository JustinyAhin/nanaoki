package activity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNotesRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "notes.jsonl")
	a := Note{ID: NoteID("2026-11-15T10:00:00Z", "noise"), Ts: "2026-11-15T10:00:00Z", Project: "demo", Agent: "claude",
		Session: "s1", Lesson: "cause:x", Tag: "noise", Text: "shown twice, did not help"}
	b := Note{ID: NoteID("2026-11-15T11:00:00Z", "x"), Ts: "2026-11-15T11:00:00Z", Project: "demo", Tag: "missing", Text: "line one\nline \"two\""}
	if err := Append(p, a); err != nil {
		t.Fatal(err)
	}
	if err := Append(p, b); err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString("not json\n")
	f.Close()
	got, err := Read[Note](p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].V != a || got[1].V != b || got[0].N != 1 || got[1].N != 2 {
		t.Fatalf("got %+v", got)
	}
	if a.ID == b.ID {
		t.Fatal("note ids collide")
	}
	if none, err := Read[Note](filepath.Join(t.TempDir(), "missing.jsonl")); err != nil || none != nil {
		t.Fatalf("missing file: %v %v", none, err)
	}
}
