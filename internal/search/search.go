// Package search is SQLite FTS5 search over the event store, in search.db, synced by event id and content key.
package search

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

type Query struct {
	Text            string
	Raw             bool            // Text is FTS5 syntax as is
	Project         string          // only this project
	Agent           string          // claude or codex
	Since, Until    string          // dates YYYY-MM-DD (until inclusive)
	Before          string          // only events with ts < Before (eval: the trigger time)
	ExcludeSession  string          // leave out this session (eval: the trigger's own session)
	Exclude         []string        // leave out these sessions
	ExcludeProjects map[string]bool // leave these projects out
	Limit           int             // results (default 8)
	PerSession      int             // results per session (default 1; the other hits of the session are counted in Hits)
	SnippetTokens   int             // default and max 64
}

type Result struct {
	Agent     string  `json:"agent"`
	Session   string  `json:"session"`
	Ts        string  `json:"ts"`
	Project   string  `json:"project"`
	EventType string  `json:"event_type"`
	Snippet   string  `json:"snippet"` // matches in [brackets]
	Score     float64 `json:"score"`   // bm25 times the event weight: lower is better
	Hits      int     `json:"hits"`    // this one included
	Pointer   string  `json:"pointer,omitempty"`
	Marked    string  `json:"-"`
}

// Snippet markers: control characters that do not occur in logs (brackets do, in regexes and paths).
const (
	MarkOpen  = '\x02'
	MarkClose = '\x03'
)

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

func Terms(text string) []string {
	seen := map[string]bool{}
	var terms []string
	for _, w := range termRx.FindAllString(strings.ToLower(text), -1) {
		if len(w) < 2 || stopwords[w] || seen[w] {
			continue
		}
		seen[w] = true
		terms = append(terms, w)
	}
	return terms
}

// MatchExpr turns free text into an FTS5 OR query of its words (quoted, stop words left out).
func MatchExpr(text string) string {
	terms := Terms(text)
	for i, t := range terms {
		terms[i] = `"` + t + `"`
	}
	return strings.Join(terms, " OR ")
}

func rawTerms(q string) []string {
	var out []string
	for _, t := range Terms(q) {
		switch t {
		case "and", "or", "not", "near":
			continue
		}
		out = append(out, t)
	}
	return out
}

func homeTilde(p string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "" && strings.HasPrefix(p, h+"/") {
		return "~" + p[len(h):]
	}
	return p
}

// Event weights: an answer or a question is worth more than a tool call that only mentions the words.
var weights = map[string]float64{"assistant_msg": 1.6, "user_msg": 1.3, "user_error_paste": 1.3, "tool_error": 1.2, "tool_call": 0.8}

// searchOnlyWeight applies to a tool call whose only match is the pattern of a search command (grep, rg, sed,
// ...): that is the agent searching, not the answer.
const searchOnlyWeight = 0.2

func Weight(eventType, body string, terms []string) float64 {
	w, ok := weights[eventType]
	if !ok {
		w = 1
	}
	if eventType == "user_msg" && strings.Contains(body, "?") {
		w += 0.2
	}
	if (eventType == "tool_call" || eventType == "tool_error") && SearchOnly(body, terms) {
		w *= searchOnlyWeight
	}
	return w
}

var searchTools = map[string]bool{"grep": true, "egrep": true, "fgrep": true, "rg": true, "ag": true, "ack": true,
	"sed": true, "awk": true, "gawk": true, "find": true, "fd": true, "ripgrep": true}

func isSearchCmd(seg string) bool {
	f := strings.Fields(seg)
	for len(f) > 0 && (strings.Contains(f[0], "=") || f[0] == "sudo" || f[0] == "command" || f[0] == "xargs" ||
		f[0] == "(" || f[0] == "{" || f[0] == "!" || f[0] == "then" || f[0] == "do") {
		f = f[1:]
	}
	if len(f) == 0 {
		return false
	}
	name := strings.TrimLeft(f[0], "({")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if searchTools[name] {
		return true
	}
	if name == "git" {
		for _, a := range f[1:] {
			if a == "grep" || strings.HasPrefix(a, "--grep") || a == "-S" || a == "-G" {
				return true
			}
		}
	}
	return false
}

// segments splits one line of shell at the unquoted separators. Quote state resets at each line, so
// an apostrophe in output text cannot swallow the next lines.
func segments(line string) []string {
	var out []string
	var q rune
	last := 0
	for i, r := range line {
		switch {
		case q != 0:
			if r == q {
				q = 0
			}
		case r == '\'' || r == '"':
			q = r
		case r == '|' || r == ';' || r == '&':
			out = append(out, line[last:i])
			last = i + 1
		}
	}
	return append(out, line[last:])
}

var grepOutRx = regexp.MustCompile(`^\S+:\d+[:-]`)

// SearchOnly tells if the terms occur in body only inside search commands (and in the path:line output of
// a search). A body with no term at all is not search-only.
func SearchOnly(body string, terms []string) bool {
	if len(terms) == 0 {
		return false
	}
	lower := strings.ToLower(body)
	if !hasTerm(lower, terms) {
		return false
	}
	var rest strings.Builder
	searching := false
	for _, line := range strings.Split(lower, "\n") {
		if searching && grepOutRx.MatchString(line) {
			continue
		}
		for _, seg := range segments(line) {
			if isSearchCmd(seg) {
				searching = true
				continue
			}
			rest.WriteString(seg)
			rest.WriteByte(' ')
		}
		rest.WriteByte('\n')
	}
	return !hasTerm(rest.String(), terms)
}

// hasTerm checks the terms as stems (FTS5 uses the Porter stemmer: "prerender" matches "prerendering").
func hasTerm(lower string, terms []string) bool {
	for _, t := range terms {
		stem := t
		if n := len(t); n > 5 {
			stem = t[:n-2]
		}
		if strings.Contains(lower, stem) {
			return true
		}
	}
	return false
}

// CurrentSession guesses the session that runs the search: the session of project whose last event is the
// newest, if it is less than 10 minutes before now. It returns "" when there is none.
func CurrentSession(db *sql.DB, project string, now time.Time) (string, error) {
	if project == "" || project == "?" {
		return "", nil
	}
	from := now.UTC().Add(-10 * time.Minute).Format("2006-01-02T15:04:05")
	var sid string
	err := db.QueryRow(`select session_id from docs where project = ? and ts >= ? group by session_id
		order by max(ts) desc, session_id limit 1`, project, from).Scan(&sid)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return sid, err
}

type hit struct {
	Result
	body string
	pos  int
}

func Search(db *sql.DB, q Query) ([]Result, error) {
	if q.Limit <= 0 {
		q.Limit = 8
	}
	if q.PerSession <= 0 {
		q.PerSession = 1
	}
	if q.SnippetTokens <= 0 {
		q.SnippetTokens = 64
	}
	if q.SnippetTokens > 64 { // FTS5 limit
		q.SnippetTokens = 64
	}
	m := q.Text
	terms := rawTerms(q.Text)
	if !q.Raw {
		m = MatchExpr(q.Text)
		terms = Terms(q.Text)
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
	for _, s := range q.Exclude {
		add("d.session_id is not ?", s)
	}
	for p := range q.ExcludeProjects {
		add("d.project is not ?", p)
	}
	// read more than Limit: the weights reorder the bm25 list, duplicates (the same prompt in a history file
	// and a transcript) drop out, and hits of one session are grouped
	sqlq := fmt.Sprintf(`select d.agent, d.session_id, d.ts, d.project, d.event_type, d.raw_file, d.raw_line,
		snippet(fts, 0, char(2), char(3), '…', %d), bm25(fts),
		case when d.event_type in ('tool_call', 'tool_error') then substr(fts.body, 1, 20000) else '' end
		from fts join docs d on d.id = fts.rowid
		where %s order by bm25(fts) limit ?`, q.SnippetTokens, strings.Join(where, " and "))
	args = append(args, max(q.Limit*40, 200))
	rs, err := db.Query(sqlq, args...)
	if err != nil {
		return nil, err
	}
	var hits []hit
	for rs.Next() {
		var h hit
		var agent, sid, ts, proj, et, rf sql.NullString
		var rl sql.NullInt64
		if err := rs.Scan(&agent, &sid, &ts, &proj, &et, &rf, &rl, &h.Marked, &h.Score, &h.body); err != nil {
			rs.Close()
			return nil, err
		}
		h.Agent, h.Session, h.Ts, h.Project, h.EventType = agent.String, sid.String, ts.String, proj.String, et.String
		if rf.String != "" {
			h.Pointer = homeTilde(rf.String) + ":" + fmt.Sprint(rl.Int64)
		}
		body := h.body
		if body == "" {
			body = h.Marked
		}
		h.Score *= Weight(h.EventType, body, terms)
		h.pos = len(hits)
		hits = append(hits, h)
	}
	rs.Close()
	if err := rs.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score < hits[j].Score })
	var out []Result
	idx := map[string][]int{}
	seen := map[string]bool{}
	for _, h := range hits {
		key := h.Session + "\x00" + strings.Join(strings.Fields(h.Marked), " ")
		if seen[key] {
			continue
		}
		seen[key] = true
		if ix := idx[h.Session]; len(ix) > 0 {
			out[ix[0]].Hits++
			if len(ix) >= q.PerSession {
				continue
			}
		}
		if len(out) >= q.Limit {
			continue
		}
		r := h.Result
		r.Snippet = strings.NewReplacer(string(MarkOpen), "[", string(MarkClose), "]").Replace(r.Marked)
		if len(idx[h.Session]) == 0 {
			r.Hits = 1
		}
		idx[h.Session] = append(idx[h.Session], len(out))
		out = append(out, r)
	}
	return out, nil
}

func Projects(db *sql.DB) ([]string, error) {
	rs, err := db.Query(`select distinct project from docs where project is not null`)
	if err != nil {
		return nil, err
	}
	defer rs.Close()
	var out []string
	for rs.Next() {
		var p string
		if err := rs.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rs.Err()
}
