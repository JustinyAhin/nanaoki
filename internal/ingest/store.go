package ingest

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Cols are the event columns, in the order of the Python store.
var Cols = []string{"source", "agent", "session_id", "project", "cwd", "ts", "event_type", "tool", "text", "output",
	"exit_code", "is_error", "sig", "side", "raw_file", "raw_line"}

const schema = `
create table if not exists events (
  id integer primary key, source text, agent text, session_id text, project text, cwd text, ts text,
  event_type text, tool text, text text, output text, exit_code integer, is_error integer, sig text,
  side integer, raw_file text, raw_line integer, file_id integer, dkey blob, cause text);
create table if not exists files (
  id integer primary key, path text unique, kind text, size integer, mtime_ns integer, rows integer,
  present integer default 1);
create table if not exists meta (k text primary key, v text);
`

var indexes = []string{
	`create index if not exists ix_type on events(event_type)`,
	`create index if not exists ix_sess on events(agent, session_id, ts)`,
	`create index if not exists ix_sig on events(sig)`,
	`create index if not exists ix_ts on events(ts)`,
	`create index if not exists ix_file on events(file_id)`,
	`create index if not exists ix_src on events(source)`,
}

// DefaultDB is ~/.local/share/nanaoki/events.db.
func DefaultDB() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nanaoki", "events.db")
}

// OpenStore opens (and creates) the event store.
func OpenStore(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{"pragma journal_mode=wal", "pragma synchronous=normal", "pragma temp_store=memory",
		"pragma cache_size=-400000", "pragma mmap_size=4294967296"} {
		if _, err := db.Exec(q); err != nil {
			db.Close()
			return nil, err
		}
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	// stores from before the cause column
	var n int
	if err := db.QueryRow(`select count(*) from pragma_table_info('events') where name='cause'`).Scan(&n); err == nil && n == 0 {
		if _, err := db.Exec(`alter table events add column cause text`); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

// OpenStoreRO opens an existing store read-only.
func OpenStoreRO(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("no event store at %s (run `nanaoki ingest` first)", path)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{"pragma cache_size=-400000", "pragma mmap_size=4294967296", "pragma temp_store=memory"} {
		if _, err := db.Exec(q); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

// dkey is the dedupe key of the reference: md5(agent|ts|event_type|text[:300]).
func dkey(r Row) [16]byte {
	ts := r.Ts
	if ts == "" {
		ts = "None"
	}
	return md5.Sum([]byte(r.Agent + "|" + ts + "|" + r.EventType + "|" + rules.Trunc(r.Text, 300)))
}

func ns(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func ni(n NInt) any {
	if !n.Valid {
		return nil
	}
	return n.V
}

// inserter batches rows into multi-row INSERT statements.
type inserter struct {
	tx    *sql.Tx
	stmts map[int]*sql.Stmt
	buf   []any
	n     int
	total int
}

const batchRows = 64
const colsPerRow = 18

func newInserter(tx *sql.Tx) *inserter { return &inserter{tx: tx, stmts: map[int]*sql.Stmt{}} }

func (in *inserter) stmt(n int) (*sql.Stmt, error) {
	if s, ok := in.stmts[n]; ok {
		return s, nil
	}
	ph := "(" + strings.TrimSuffix(strings.Repeat("?,", colsPerRow), ",") + ")"
	q := "insert into events (" + strings.Join(Cols, ",") + ",file_id,dkey) values " +
		strings.TrimSuffix(strings.Repeat(ph+",", n), ",")
	s, err := in.tx.Prepare(q)
	if err != nil {
		return nil, err
	}
	in.stmts[n] = s
	return s, nil
}

func (in *inserter) add(r Row, fileID any, key []byte) error {
	text := ns(r.Text)
	if r.TextSet && r.Text == "" {
		text = ""
	}
	out := ns(r.Output)
	if r.OutputSet && r.Output == "" {
		out = ""
	}
	var k any
	if key != nil {
		k = key
	}
	in.buf = append(in.buf, r.Source, r.Agent, ns(r.SessionID), ns(r.Project), ns(r.Cwd), ns(r.Ts), r.EventType,
		ns(r.Tool), text, out, ni(r.ExitCode), ni(r.IsError), ns(r.Sig), r.Side, ns(r.RawFile), ni(r.RawLine), fileID, k)
	in.n++
	in.total++
	if in.n == batchRows {
		return in.flush()
	}
	return nil
}

func (in *inserter) flush() error {
	if in.n == 0 {
		return nil
	}
	s, err := in.stmt(in.n)
	if err != nil {
		return err
	}
	_, err = s.Exec(in.buf...)
	in.buf = in.buf[:0]
	in.n = 0
	return err
}

func (in *inserter) close() error {
	err := in.flush()
	for _, s := range in.stmts {
		s.Close()
	}
	return err
}
