package testfixture

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

type Ev struct {
	Agent, Sid, Project, Ts, Type, Tool, Text, Out string
	Err                                            bool
}

// EventsDB writes synthetic events into a new store (the ingest schema) and returns its path.
// open is ingest.OpenStore (passed in to avoid an import cycle).
func EventsDB(t *testing.T, open func(string) (*sql.DB, error), evs []Ev) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "events.db")
	db, err := open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i, e := range evs {
		isErr, sig := 0, any(nil)
		if e.Err {
			isErr = 1
			src := e.Out
			if src == "" {
				src = e.Text
			}
			sig = rules.Signature(src)
		}
		if e.Tool == "" && e.Type != "user_msg" && e.Type != "assistant_msg" {
			e.Tool = "Bash"
		}
		cwd := "/Users/x/work/projects/" + e.Project
		_, err := db.Exec(`insert into events (source, agent, session_id, project, cwd, ts, event_type, tool, text, output,
			exit_code, is_error, sig, side, raw_file, raw_line, dkey) values (?,?,?,?,?,?,?,?,?,?,?,?,?,0,?,?,?)`,
			e.Agent+"_test", e.Agent, e.Sid, e.Project, cwd, e.Ts, e.Type, nz(e.Tool), e.Text, nz(e.Out), isErr, isErr, sig,
			"/Users/x/."+e.Agent+"/sessions/"+e.Sid+".jsonl", i+1, []byte(e.Ts+e.Text))
		if err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func nz(s string) any {
	if s == "" {
		return nil
	}
	return s
}
