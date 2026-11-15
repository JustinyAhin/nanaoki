// Package search is full-text search over the event store: prompts, assistant messages, and tool calls with
// their results. SQLite FTS5 with BM25 ranking, in a separate file next to the event store (search.db). The
// index syncs from the event store by event id and content key, so it follows the ingest's deletes and inserts.
package search

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DefaultPath is search.db next to the event store.
func DefaultPath(eventsDB string) string { return filepath.Join(filepath.Dir(eventsDB), "search.db") }

const schema = `
create table if not exists docs (id integer primary key, dkey blob, agent text, session_id text, ts text, project text,
  event_type text, raw_file text, raw_line integer);
create index if not exists ix_docs_ts on docs(ts);
create virtual table if not exists fts using fts5(body, tokenize = 'porter unicode61 remove_diacritics 2');
create table if not exists meta (k text primary key, v text);
`

// Indexed event types: what the user asked, what the agent said, and what the tools did.
var docTypes = []string{"user_msg", "assistant_msg", "tool_call", "tool_error", "test_result", "build_result", "deploy",
	"commit", "push", "pr", "file_change", "user_error_paste"}

// Open opens (and creates) the search index.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	for _, q := range []string{"pragma journal_mode=wal", "pragma synchronous=normal", "pragma temp_store=memory"} {
		if _, err := db.Exec(q); err != nil {
			db.Close()
			return nil, err
		}
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// SyncStats reports one sync.
type SyncStats struct {
	Removed, Added int
	Elapsed        time.Duration
}

// Body is the indexed text of one event: the text (prompt, message or command) and the output.
func Body(text, output string) string {
	if output == "" {
		return text
	}
	return text + "\n" + output
}

// Sync updates the index from the event store: rows gone or changed (other content key) are removed, new rows
// are added.
func Sync(db *sql.DB, eventsDB string) (SyncStats, error) {
	t0 := time.Now()
	var st SyncStats
	if _, err := db.Exec(`attach database ? as ev`, "file:"+eventsDB+"?mode=ro"); err != nil {
		return st, err
	}
	defer db.Exec(`detach database ev`)
	tx, err := db.Begin()
	if err != nil {
		return st, err
	}
	defer tx.Rollback()
	in := "'" + strings.Join(docTypes, "','") + "'"
	if _, err := tx.Exec(`create temp table if not exists gone as select id from docs where 0`); err != nil {
		return st, err
	}
	tx.Exec(`delete from temp.gone`)
	if _, err := tx.Exec(`insert into temp.gone select d.id from docs d left join ev.events e on e.id = d.id
		where e.id is null or e.dkey is not d.dkey`); err != nil {
		return st, err
	}
	res, err := tx.Exec(`delete from fts where rowid in (select id from temp.gone)`)
	if err != nil {
		return st, err
	}
	n, _ := res.RowsAffected()
	st.Removed = int(n)
	if _, err := tx.Exec(`delete from docs where id in (select id from temp.gone)`); err != nil {
		return st, err
	}
	var maxID int64
	if err := tx.QueryRow(`select coalesce(max(id), 0) from docs`).Scan(&maxID); err != nil {
		return st, err
	}
	rs, err := tx.Query(`select e.id, e.dkey, e.agent, e.session_id, e.ts, e.project, e.event_type, e.raw_file, e.raw_line,
		e.text, e.output from ev.events e where e.event_type in (`+in+`) and e.ts is not null
		and e.id > ?
		union all
		select e.id, e.dkey, e.agent, e.session_id, e.ts, e.project, e.event_type, e.raw_file, e.raw_line, e.text, e.output
		from ev.events e where e.id in (select id from temp.gone) and e.id <= ? and e.event_type in (`+in+`)
		and e.ts is not null`, maxID, maxID)
	if err != nil {
		return st, err
	}
	type row struct {
		id                                      int64
		dkey                                    []byte
		agent, sid, ts, proj, et, rf, text, out sql.NullString
		rl                                      sql.NullInt64
	}
	var rows []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.dkey, &r.agent, &r.sid, &r.ts, &r.proj, &r.et, &r.rf, &r.rl, &r.text, &r.out); err != nil {
			rs.Close()
			return st, err
		}
		rows = append(rows, r)
	}
	rs.Close()
	insDoc, err := tx.Prepare(`insert or replace into docs values (?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return st, err
	}
	insFTS, err := tx.Prepare(`insert into fts(rowid, body) values (?, ?)`)
	if err != nil {
		return st, err
	}
	for _, r := range rows {
		if _, err := insDoc.Exec(r.id, r.dkey, r.agent, r.sid, r.ts, r.proj, r.et, r.rf, r.rl); err != nil {
			return st, err
		}
		if _, err := insFTS.Exec(r.id, Body(r.text.String, r.out.String)); err != nil {
			return st, err
		}
	}
	st.Added = len(rows)
	if err := tx.Commit(); err != nil {
		return st, err
	}
	st.Elapsed = time.Since(t0)
	return st, nil
}

// Query selects results.
type Query struct {
	Text            string
	Raw             bool            // Text is FTS5 syntax as is
	Project         string          // only this project
	Agent           string          // claude or codex
	Since, Until    string          // dates YYYY-MM-DD (until inclusive)
	Before          string          // only events with ts < Before (eval: the trigger time)
	ExcludeSession  string          // leave out this session (eval: the trigger's own session)
	ExcludeProjects map[string]bool // leave these projects out
	Limit           int             // results (default 8)
	PerSession      int             // max results per session (default 2)
	SnippetTokens   int             // default and max 64
}

// Result is one hit with its citation.
type Result struct {
	Agent     string  `json:"agent"`
	Session   string  `json:"session"`
	Ts        string  `json:"ts"`
	Project   string  `json:"project"`
	EventType string  `json:"event_type"`
	Snippet   string  `json:"snippet"`
	Score     float64 `json:"score"` // bm25: lower is better
	Pointer   string  `json:"pointer,omitempty"`
}

var termRx = regexp.MustCompile(`[\p{L}\p{N}_]+`)

var stopwords = map[string]bool{}

func init() {
	for _, w := range strings.Fields("a an the and or of to in on at for with by from is are was were be been it this that these those " +
		"what which who whom how why when where did do does done have has had we i you my our your me us can could should would " +
		"there their them they he she his her its not no yes any all some about into than then also just so if as but up out " +
		"find check look search session sessions conversation conversations past earlier previous before last again tell show") {
		stopwords[w] = true
	}
}

// MatchExpr turns free text into an FTS5 OR query of its words (quoted, stop words left out).
func MatchExpr(text string) string {
	seen := map[string]bool{}
	var terms []string
	for _, w := range termRx.FindAllString(strings.ToLower(text), -1) {
		if len(w) < 2 || stopwords[w] || seen[w] {
			continue
		}
		seen[w] = true
		terms = append(terms, `"`+w+`"`)
	}
	return strings.Join(terms, " OR ")
}

func homeTilde(p string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "" && strings.HasPrefix(p, h+"/") {
		return "~" + p[len(h):]
	}
	return p
}

// Search runs a query.
func Search(db *sql.DB, q Query) ([]Result, error) {
	if q.Limit <= 0 {
		q.Limit = 8
	}
	if q.PerSession <= 0 {
		q.PerSession = 2
	}
	if q.SnippetTokens <= 0 {
		q.SnippetTokens = 64
	}
	if q.SnippetTokens > 64 { // FTS5 limit
		q.SnippetTokens = 64
	}
	m := q.Text
	if !q.Raw {
		m = MatchExpr(q.Text)
	}
	if m == "" {
		return nil, fmt.Errorf("empty query")
	}
	where := []string{"fts match ?"}
	args := []any{m}
	add := func(cond string, v any) {
		where = append(where, cond)
		args = append(args, v)
	}
	if q.Project != "" {
		add("d.project = ?", q.Project)
	}
	if q.Agent != "" {
		add("d.agent = ?", q.Agent)
	}
	if q.Since != "" {
		add("d.ts >= ?", q.Since)
	}
	if q.Until != "" {
		add("d.ts < ?", q.Until+"T99")
	}
	if q.Before != "" {
		add("d.ts < ?", q.Before)
	}
	if q.ExcludeSession != "" {
		add("d.session_id is not ?", q.ExcludeSession)
	}
	for p := range q.ExcludeProjects {
		add("d.project is not ?", p)
	}
	// read more than Limit: duplicates (the same prompt in a history file and a transcript) and the per-session
	// cap drop some
	sqlq := fmt.Sprintf(`select d.agent, d.session_id, d.ts, d.project, d.event_type, d.raw_file, d.raw_line,
		snippet(fts, 0, '[', ']', '…', %d), bm25(fts) from fts join docs d on d.id = fts.rowid
		where %s order by bm25(fts) limit ?`, q.SnippetTokens, strings.Join(where, " and "))
	args = append(args, q.Limit*20)
	rs, err := db.Query(sqlq, args...)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []Result
	perSess := map[string]int{}
	seen := map[string]bool{}
	for rs.Next() {
		var r Result
		var agent, sid, ts, proj, et, rf sql.NullString
		var rl sql.NullInt64
		if err := rs.Scan(&agent, &sid, &ts, &proj, &et, &rf, &rl, &r.Snippet, &r.Score); err != nil {
			return nil, err
		}
		r.Agent, r.Session, r.Ts, r.Project, r.EventType = agent.String, sid.String, ts.String, proj.String, et.String
		if rf.String != "" {
			r.Pointer = homeTilde(rf.String) + ":" + fmt.Sprint(rl.Int64)
		}
		key := r.Session + "\x00" + strings.Join(strings.Fields(r.Snippet), " ")
		if seen[key] || perSess[r.Session] >= q.PerSession {
			continue
		}
		seen[key] = true
		perSess[r.Session]++
		out = append(out, r)
		if len(out) == q.Limit {
			break
		}
	}
	return out, rs.Err()
}
