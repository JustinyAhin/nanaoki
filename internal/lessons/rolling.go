// Package lessons is the rolling lesson store (RollingStore, Store,
// retrieve_start_strict) to Go. Same lesson kinds, text, conditions and scoring. No new heuristics.
//
// Lesson kinds:
//   - cause: an env/stack error cause (rules.Lesson) that a later call in the same session fixed.
//   - sig:   an exact informative error signature with no env/stack cause, fixed the same way.
//   - pref:  an instruction-like sentence the user wrote (PREF_RX), keyed by its normalized form.
package lessons

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Config selects the history the store reads.
type Config struct {
	Since               string          // errors from this ts ("" = all); the eval uses 2026-02-01
	ExcludeProjects     map[string]bool // errors and prefs from these projects are left out (eval: client projects)
	ExcludePrefProjects map[string]bool // prefs only (eval: personal folders)
}

// Episode is one (agent, session, kind, key) error episode with its earliest fix time.
type Episode struct {
	Agent, Sid, Kind, Key, Project, Ts, FixTs string
	First                                     *analyze.Err
}

type prefOcc struct{ Ts, Sid, Proj, Text string }

// Rolling holds everything needed to build the lessons visible at any time t.
type Rolling struct {
	DB        *sql.DB
	Cfg       Config
	Data      *analyze.Data         // errors, successes and prompts of the event store
	groups    map[string][]*Episode // kind \x00 key -> fixed episodes in ts order
	pstack    map[string]map[string][]string
	repo      map[string]map[string][]string
	prefKeys  []string // first-seen order
	prefOcc   map[string][]prefOcc
	pstackMem map[string]map[string]map[string]bool
}

func gk(kind, key string) string { return kind + "\x00" + key }

// Load reads the event store. It ports RollingStore.__init__.
func Load(db *sql.DB, cfg Config) (*Rolling, error) {
	r := &Rolling{DB: db, Cfg: cfg, groups: map[string][]*Episode{}, prefOcc: map[string][]prefOcc{},
		pstackMem: map[string]map[string]map[string]bool{}}
	d, err := analyze.Load(db, cfg.Since)
	if err != nil {
		return nil, err
	}
	r.Data = d
	type epKey struct{ agent, sid, kind, key string }
	eps := map[epKey]*Episode{}
	var order []*Episode
	for _, e := range d.Errs {
		if cfg.ExcludeProjects[e.Project] {
			continue
		}
		fixTs := ""
		if e.EType != "user_error_paste" && e.Head != "" {
			lst := d.OK[e.Agent+"\x00"+e.Sid]
			i := sort.Search(len(lst), func(i int) bool { return lst[i].Ts > e.Ts })
			lim := addMin(e.Ts, 30)
			for ; i < len(lst) && cut19(lst[i].Ts) <= lim; i++ {
				if lst[i].Head == e.Head {
					fixTs = lst[i].Ts
					break
				}
			}
		}
		kind, key := "sig", e.Sig
		if e.Cause != "" && rules.Scope[e.Cause] != "code" {
			kind, key = "cause", e.Cause
		}
		if kind == "sig" && !rules.Informative(key) {
			continue
		}
		k := epKey{e.Agent, e.Sid, kind, key}
		x := eps[k]
		if x == nil {
			x = &Episode{Agent: e.Agent, Sid: e.Sid, Kind: kind, Key: key, Project: e.Project, Ts: e.Ts, First: e}
			eps[k] = x
			order = append(order, x)
		}
		if fixTs != "" && (x.FixTs == "" || fixTs < x.FixTs) {
			x.FixTs = fixTs
		}
	}
	for _, x := range order {
		if x.FixTs != "" {
			r.groups[gk(x.Kind, x.Key)] = append(r.groups[gk(x.Kind, x.Key)], x)
		}
	}
	for _, v := range r.groups {
		sort.SliceStable(v, func(i, j int) bool { return v[i].Ts < v[j].Ts })
	}
	if r.pstack, err = projectStackTable(db); err != nil {
		return nil, err
	}
	if r.repo, err = repoSignalTable(db); err != nil {
		return nil, err
	}
	if err := r.loadPrefs(); err != nil {
		return nil, err
	}
	return r, nil
}

// ---------- stacks ----------

var stackRx = []struct {
	name string
	rx   *regexp.Regexp
}{
	{"cloudflare", rules.Py(`\bwrangler\b|cloudflare`, "(?i)")},
	{"sveltekit", rules.Py(`svelte`, "(?i)")},
	{"statamic", rules.Py(`statamic|php artisan`, "(?i)")},
	{"supabase", rules.Py(`\bsupabase\b`, "(?i)")},
	{"bun", rules.Py(`\bbunx?\b`, "")},
	{"node", rules.Py(`\b(npm|npx|pnpm|yarn|node)\b`, "")},
	{"python", rules.Py(`\b(python3?|uv|pip|pytest)\b`, "")},
	{"go", rules.Py(`\bgo (build|test|run|mod)\b`, "")},
	{"next", rules.Py(`\bnext (build|dev)\b|next\.config`, "")},
	{"laravel", rules.Py(`\b(artisan|composer)\b`, "")},
	{"postgres", rules.Py(`\bpsql\b|postgres`, "(?i)")},
}

// StacksOfText finds tools and frameworks named in command text.
func StacksOfText(text string) []string {
	var out []string
	for _, s := range stackRx {
		if s.rx.MatchString(text) {
			out = append(out, s.name)
		}
	}
	return out
}

var fileStackRx = []struct {
	name string
	rx   *regexp.Regexp
}{
	{"sveltekit", rules.Py(`svelte\.config\.(js|ts)|"@sveltejs/kit"|"svelte"\s*:`, "")},
	{"cloudflare", rules.Py(`wrangler\.(toml|jsonc?)|"wrangler"\s*:|"@cloudflare/`, "")},
	{"next", rules.Py(`next\.config\.(js|mjs|ts)|"next"\s*:`, "")},
	{"supabase", rules.Py(`supabase/config\.toml|"@supabase/`, "")},
	{"bun", rules.Py(`\bbun\.lockb?\b|bunfig\.toml`, "")},
	{"node", rules.Py(`\bpackage\.json\b|pnpm-lock\.yaml|package-lock\.json`, "")},
	{"python", rules.Py(`pyproject\.toml|requirements\.txt|\buv\.lock\b`, "")},
	{"go", rules.Py(`\bgo\.mod\b`, "")},
	{"laravel", rules.Py(`composer\.json|\bartisan\b|"laravel/framework"`, "")},
	{"statamic", rules.Py(`"statamic/cms"|config/statamic`, "")},
}

// FileStacks ports the FILE_STACK_RX match over a command + output blob.
func FileStacks(blob string) []string {
	var out []string
	for _, s := range fileStackRx {
		if s.rx.MatchString(blob) {
			out = append(out, s.name)
		}
	}
	return out
}

func addSignal(tab map[string]map[string][]string, proj, stack, ts string) {
	m := tab[proj]
	if m == nil {
		m = map[string][]string{}
		tab[proj] = m
	}
	m[stack] = append(m[stack], ts)
}

func sortTab(tab map[string]map[string][]string) {
	for _, p := range tab {
		for _, v := range p {
			sort.Strings(v)
		}
	}
}

const nonMsg = `event_type not in ('user_msg','assistant_msg','compaction')`

// projectStackTable ports project_stack_table: stack signals from command text, with timestamps.
func projectStackTable(db *sql.DB) (map[string]map[string][]string, error) {
	rs, err := db.Query(`select project, ts, text from events where ` + nonMsg + ` and text is not null`)
	if err != nil {
		return nil, err
	}
	type row struct{ proj, ts, text string }
	var rows []row
	for rs.Next() {
		var p, ts sql.NullString
		var text string
		if err := rs.Scan(&p, &ts, &text); err != nil {
			rs.Close()
			return nil, err
		}
		rows = append(rows, row{p.String, ts.String, rules.Trunc(text, 400)})
	}
	rs.Close()
	res := make([][]string, len(rows))
	parFor(len(rows), func(i int) { res[i] = StacksOfText(rows[i].text) })
	tab := map[string]map[string][]string{}
	for i, r := range rows {
		for _, s := range res[i] {
			addSignal(tab, r.proj, s, r.ts)
		}
	}
	sortTab(tab)
	return tab, nil
}

// repoSignalTable ports repo_signal_table: manifest and framework files seen in commands and outputs.
func repoSignalTable(db *sql.DB) (map[string]map[string][]string, error) {
	var likes []string
	for _, w := range []string{"package.json", "svelte.config", "wrangler.", "next.config", "pyproject.toml", "go.mod",
		"composer.json", "bun.lock", "requirements.txt", "supabase/config", "artisan", "uv.lock"} {
		likes = append(likes, "text like '%"+w+"%'")
	}
	rs, err := db.Query(`select project, ts, text, output from events where ` + nonMsg + ` and (` + strings.Join(likes, " or ") + `)`)
	if err != nil {
		return nil, err
	}
	tab := map[string]map[string][]string{}
	for rs.Next() {
		var p, ts, text, out sql.NullString
		if err := rs.Scan(&p, &ts, &text, &out); err != nil {
			rs.Close()
			return nil, err
		}
		blob := rules.Trunc(text.String, 600) + "\n" + rules.Trunc(out.String, 4000)
		for _, s := range FileStacks(blob) {
			addSignal(tab, p.String, s, ts.String)
		}
	}
	rs.Close()
	sortTab(tab)
	return tab, nil
}

// countBefore counts timestamps strictly before t. v must be sorted.
func countBefore(v []string, t string) int { return sort.SearchStrings(v, t) }

// PstackOf ports pstack_of: stacks with 3+ command signals before t.
func (r *Rolling) PstackOf(p, t string) map[string]bool {
	out := map[string]bool{}
	for s, v := range r.pstack[p] {
		if countBefore(v, t) >= 3 {
			out[s] = true
		}
	}
	return out
}

func (r *Rolling) pstacksAt(t string) map[string]map[string]bool {
	if m, ok := r.pstackMem[t]; ok {
		return m
	}
	m := map[string]map[string]bool{}
	for p := range r.pstack {
		m[p] = r.PstackOf(p, t)
	}
	r.pstackMem[t] = m
	return m
}

// RepoStacksAt returns stacks with at least RepoSignalMin file signals strictly before t.
func (r *Rolling) RepoStacksAt(p, t string) map[string]bool {
	out := map[string]bool{}
	for s, v := range r.repo[p] {
		if countBefore(v, t) >= RepoSignalMin {
			out[s] = true
		}
	}
	return out
}

// RepoSignalMin: a stack counts for a repo after this many file signals.
const RepoSignalMin = 2

// ---------- prefs ----------

var (
	prefRx = rules.Py(`\b(always|never|don't|do not|dont|make sure|must|prefer|only when|without|instead of|`+
		`before you|remember|no need to|stop|avoid|use \w+ (for|instead|not))\b`, "(?i)")
	notPrefRx   = rules.Py(`\b(i|we) (don't|do not) (know|understand|think)|what do you think`, "(?i)")
	bareSkillRx = rules.Py(`^\s*(/[\w:.-]+|\[\$[\w:.-]+\]\([^)]*\))(\s*(,|and)?\s*(copy|shorten|copy to my clipboard|`+
		`and copy to my clipboard|shorten and copy to my clipboard)?)*\s*$`, "(?i)")
	nsURL   = rules.Py(`https?://\S+`, "")
	nsPath  = rules.Py(`(/[\w.@-]+)+`, "")
	nsDigit = rules.Py(`\d+`, "")
	nsOther = rules.Py(`[^a-z N]+`, "")
	nsWS    = rules.Py(`\s+`, "")
)

// IsBareSkillCall ports is_bare_skill_call.
func IsBareSkillCall(t string) bool { return bareSkillRx.MatchString(t) }

// NormSentence removes URLs, paths and punctuation, lowercases text, and replaces numbers with N.
func NormSentence(s string) string {
	s = strings.ToLower(s)
	s = nsURL.ReplaceAllLiteralString(s, "URL")
	s = nsPath.ReplaceAllLiteralString(s, "PATH")
	s = nsDigit.ReplaceAllLiteralString(s, "N")
	s = nsOther.ReplaceAllLiteralString(s, " ")
	return rules.Strip(nsWS.ReplaceAllLiteralString(s, " "))
}

func isPySpace(r rune) bool {
	return unicode.IsSpace(r) || r == '\x1c' || r == '\x1d' || r == '\x1e' || r == '\x1f'
}

// SplitSentences ports re.split(r'\n+|(?<=[.!?])\s+', t).
func SplitSentences(t string) []string {
	rs := []rune(t)
	var out []string
	start := 0
	for i := 0; i < len(rs); {
		if rs[i] == '\n' {
			j := i
			for j < len(rs) && rs[j] == '\n' {
				j++
			}
			out = append(out, string(rs[start:i]))
			start, i = j, j
			continue
		}
		if i > 0 && isPySpace(rs[i]) && (rs[i-1] == '.' || rs[i-1] == '!' || rs[i-1] == '?') {
			j := i
			for j < len(rs) && isPySpace(rs[j]) {
				j++
			}
			out = append(out, string(rs[start:i]))
			start, i = j, j
			continue
		}
		i++
	}
	return append(out, string(rs[start:]))
}

// PrefSentences returns the (normalized, original) instruction-like sentences of one user message, with the
// filters of build_prefs / _pref_occurrences.
func PrefSentences(text string) [][2]string {
	lt := strings.TrimLeftFunc(text, unicode.IsSpace)
	if strings.HasPrefix(lt, "<") || strings.HasPrefix(lt, "#") || strings.HasPrefix(lt, "{") || strings.HasPrefix(lt, "[") ||
		strings.Contains(text, "pasted_content") || IsBareSkillCall(text) {
		return nil
	}
	var out [][2]string
	for _, part := range SplitSentences(text) {
		part = rules.Strip(part)
		n := rules.RuneLen(part)
		if n < 25 || n > 260 || !prefRx.MatchString(part) || strings.HasSuffix(part, "?") || notPrefRx.MatchString(part) {
			continue
		}
		ns := NormSentence(part)
		if len(strings.Fields(ns)) < 6 {
			continue
		}
		out = append(out, [2]string{ns, part})
	}
	return out
}

func (r *Rolling) loadPrefs() error {
	rs, err := r.DB.Query(`select session_id, project, ts, text from events where event_type='user_msg'
		and length(text) < 1500 order by ts, id`)
	if err != nil {
		return err
	}
	defer rs.Close()
	for rs.Next() {
		var sid, proj, ts, text sql.NullString
		if err := rs.Scan(&sid, &proj, &ts, &text); err != nil {
			return err
		}
		if r.Cfg.ExcludeProjects[proj.String] || r.Cfg.ExcludePrefProjects[proj.String] {
			continue
		}
		for _, p := range PrefSentences(text.String) {
			if _, ok := r.prefOcc[p[0]]; !ok {
				r.prefKeys = append(r.prefKeys, p[0])
			}
			r.prefOcc[p[0]] = append(r.prefOcc[p[0]], prefOcc{ts.String, sid.String, proj.String, p[1]})
		}
	}
	return rs.Err()
}

// ---------- lessons ----------

// Example is one evidence row of a lesson.
type Example struct {
	Project string `json:"project"`
	Agent   string `json:"agent,omitempty"`
	Session string `json:"session,omitempty"`
	Ts      string `json:"ts,omitempty"`
	Failed  string `json:"failed,omitempty"`
	Error   string `json:"error,omitempty"`
	Fix     string `json:"fix,omitempty"`
	Pointer string `json:"pointer,omitempty"`
}

// Lesson mirrors the Python lesson dict.
type Lesson struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Key        string         `json:"key"`
	Scope      string         `json:"scope"`
	Text       string         `json:"text"`
	Projects   map[string]int `json:"projects"`
	Stacks     []string       `json:"stacks"`
	Heads      []string       `json:"heads"`
	Agents     []string       `json:"agents"`
	Episodes   int            `json:"n_episodes"`
	Sessions   int            `json:"n_sessions"`
	FirstTs    string         `json:"first_ts"`
	LastTs     string         `json:"last_ts"`
	Examples   []Example      `json:"examples"`
	Client     bool           `json:"client"`
	EpisodeTs  []string       `json:"-"` // start of each episode (error) or occurrence (pref), for the score
	toks       map[string]bool
	sessionIDs []string
}

// ShortHash is hashlib_short: sha1 hex, 10 chars.
func ShortHash(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])[:10]
}

const tsMax = "9999-12-31T23:59:59.999Z"

func (r *Rolling) fixCmd(x *Episode, t string) string {
	rs, err := r.DB.Query(`select tool, text from events where agent=? and session_id=? and ts>? and ts<? and is_error=0
		and `+nonMsg+` order by ts, id limit 60`, x.Agent, x.Sid, x.Ts, t)
	if err != nil {
		return ""
	}
	defer rs.Close()
	for rs.Next() {
		var tool, cmd sql.NullString
		if rs.Scan(&tool, &cmd) != nil {
			return ""
		}
		if rules.Head(tool.String, cmd.String) == x.First.Head {
			return cmd.String
		}
	}
	return ""
}

// mostCommon is Counter.most_common(n): count desc, then first-seen order.
func mostCommon(items []string, n int) []string {
	cnt := map[string]int{}
	var order []string
	for _, s := range items {
		if cnt[s] == 0 {
			order = append(order, s)
		}
		cnt[s]++
	}
	sort.SliceStable(order, func(i, j int) bool { return cnt[order[i]] > cnt[order[j]] })
	if len(order) > n {
		order = order[:n]
	}
	return order
}

// errorLesson ports RollingStore._lesson for fixed episodes L (ts order), seen at time t.
func (r *Rolling) errorLesson(kind, key string, L []*Episode, t string) *Lesson {
	proj := map[string]int{}
	var projOrder []string
	var examples []Example
	seenP := map[string]bool{}
	var heads, epTs []string
	agents := map[string]bool{}
	sess := map[string]bool{}
	var sessOrder []string
	for _, x := range L {
		if proj[x.Project] == 0 {
			projOrder = append(projOrder, x.Project)
		}
		proj[x.Project]++
		heads = append(heads, x.First.Head)
		epTs = append(epTs, x.Ts)
		agents[x.Agent] = true
		if !sess[x.Sid] {
			sess[x.Sid] = true
			sessOrder = append(sessOrder, x.Sid)
		}
		if seenP[x.Project] || len(examples) >= 5 {
			continue
		}
		fx := r.fixCmd(x, t)
		seenP[x.Project] = true
		ex := Example{Project: x.Project, Agent: x.Agent, Session: x.Sid, Ts: x.Ts, Failed: rules.Redact(x.First.Cmd, 160),
			Error: rules.Redact(x.First.Sig, 160), Pointer: HomeTilde(x.First.RawFile)}
		if x.First.RawFile != "" {
			ex.Pointer += ":" + itoa(x.First.RawLine)
		}
		if fx != "" {
			ex.Fix = rules.Redact(fx, 160)
		}
		examples = append(examples, ex)
	}
	var exFix *Example
	for i := range examples {
		if examples[i].Fix != "" {
			exFix = &examples[i]
			break
		}
	}
	var text string
	if kind == "cause" {
		text = rules.Lesson[key]
		if exFix != nil {
			text += " Fix seen (" + exFix.Project + "): `" + exFix.Fix + "`."
		}
	} else {
		failed := ""
		if len(examples) > 0 {
			failed = examples[0].Failed
		}
		text = "Error `" + rules.Redact(key, 140) + "` after `" + failed + "`"
		if exFix != nil {
			text += " was fixed by: `" + exFix.Fix + "`."
		} else {
			text += " was fixed in the same session."
		}
	}
	stacks := map[string]bool{}
	for _, p := range projOrder {
		for s := range r.PstackOf(p, t) {
			stacks[s] = true
		}
	}
	scope := "sig"
	if kind == "cause" {
		scope = rules.Scope[key]
	}
	return &Lesson{ID: kind + ":" + ShortHash(kind+key), Kind: kind, Key: key, Scope: scope, Text: text, Projects: proj,
		Stacks: sortedKeys(stacks), Heads: mostCommon(heads, 5), Agents: sortedKeys(agents), Episodes: len(L),
		Sessions: len(sess), FirstTs: L[0].Ts, LastTs: L[len(L)-1].Ts, Examples: examples, EpisodeTs: epTs, sessionIDs: sessOrder}
}

// prefLesson ports RollingStore._prefs for one normalized sentence.
func prefLesson(n string, o []prefOcc) *Lesson {
	proj := map[string]int{}
	sess := map[string]bool{}
	var sessOrder []string
	var examples []Example
	seenP := map[string]bool{}
	var epTs []string
	for _, x := range o {
		proj[x.Proj]++
		epTs = append(epTs, x.Ts)
		if !sess[x.Sid] {
			sess[x.Sid] = true
			sessOrder = append(sessOrder, x.Sid)
		}
		if !seenP[x.Proj] && len(examples) < 5 {
			seenP[x.Proj] = true
			examples = append(examples, Example{Project: x.Proj, Session: x.Sid, Ts: x.Ts})
		}
	}
	return &Lesson{ID: "pref:" + ShortHash(n), Kind: "pref", Key: n, Scope: "pref", Text: rules.Redact(o[0].Text, 240),
		Projects: proj, Stacks: []string{}, Heads: []string{}, Agents: []string{}, Episodes: len(o), Sessions: len(sess),
		FirstTs: o[0].Ts, LastTs: o[len(o)-1].Ts, Examples: examples, EpisodeTs: epTs, sessionIDs: sessOrder}
}

// Trigger is the moment a lesson may surface.
type Trigger struct {
	Kind       string   `json:"kind"` // error | question | session_start
	Project    string   `json:"project"`
	Cause      string   `json:"cause,omitempty"`
	Sig        string   `json:"sig,omitempty"`
	Head       string   `json:"head,omitempty"`
	Text       string   `json:"text,omitempty"`
	SessStacks []string `json:"sess_stacks,omitempty"` // stacks seen in the session so far
}

// Snapshot builds a snapshot: a Store with the lessons this trigger can reach, from history before t,
// without the session excl.
func (r *Rolling) Snapshot(tr Trigger, t, excl string) *Store {
	var ls []*Lesson
	if tr.Kind == "error" {
		for _, kk := range [][2]string{{"sig", tr.Sig}, {"cause", tr.Cause}} {
			if kk[1] == "" {
				continue
			}
			var L []*Episode
			for _, x := range r.groups[gk(kk[0], kk[1])] {
				if x.FixTs < t && x.Sid != excl {
					L = append(L, x)
				}
			}
			if len(L) > 0 {
				ls = append(ls, r.errorLesson(kk[0], kk[1], L, t))
			}
		}
	} else {
		ls = r.PrefsAt(t, excl)
	}
	return NewStore(ls, r.pstacksAt(t))
}

// PrefsAt returns the pref lessons from occurrences before t, without the session excl.
func (r *Rolling) PrefsAt(t, excl string) []*Lesson {
	var out []*Lesson
	for _, n := range r.prefKeys {
		var o []prefOcc
		for _, x := range r.prefOcc[n] {
			if x.Ts < t && x.Sid != excl {
				o = append(o, x)
			}
		}
		if len(o) > 0 {
			out = append(out, prefLesson(n, o))
		}
	}
	return out
}

// All returns every lesson known at time t (no session left out): the product store.
func (r *Rolling) All(t string) []*Lesson {
	if t == "" {
		t = tsMax
	}
	var keys []string
	for k := range r.groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []*Lesson
	for _, k := range keys {
		var L []*Episode
		for _, x := range r.groups[k] {
			if x.FixTs < t {
				L = append(L, x)
			}
		}
		if len(L) > 0 {
			kind, key, _ := strings.Cut(k, "\x00")
			out = append(out, r.errorLesson(kind, key, L, t))
		}
	}
	return append(out, r.PrefsAt(t, "")...)
}

// GroupEpisodes returns the fixed episodes of one lesson key, in ts order.
func (r *Rolling) GroupEpisodes(kind, key string) []*Episode { return r.groups[gk(kind, key)] }
