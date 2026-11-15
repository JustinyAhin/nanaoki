package lessons

import (
	"math"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Store ports lessons.Store: retrieval over one set of lessons.
type Store struct {
	Lessons []*Lesson
	PStacks map[string]map[string]bool
	bySig   map[string][]*Lesson
	byCause map[string][]*Lesson
	prefs   []*Lesson
	IDF     map[string]float64
}

// MinScore is the minimum score for error and question hits.
const MinScore = 2.0

// NewStore indexes lessons. pstacks are the project stacks known at the store time.
func NewStore(ls []*Lesson, pstacks map[string]map[string]bool) *Store {
	s := &Store{Lessons: ls, PStacks: pstacks, bySig: map[string][]*Lesson{}, byCause: map[string][]*Lesson{}}
	for _, L := range ls {
		switch L.Kind {
		case "sig":
			s.bySig[L.Key] = append(s.bySig[L.Key], L)
		case "cause":
			s.byCause[L.Key] = append(s.byCause[L.Key], L)
		default:
			L.toks = TokSet(L.Text)
			s.prefs = append(s.prefs, L)
		}
	}
	df := map[string]int{}
	for _, L := range s.prefs {
		for t := range L.toks {
			df[t]++
		}
	}
	s.IDF = make(map[string]float64, len(df))
	for t, n := range df {
		s.IDF[t] = math.Log(float64(len(s.prefs)+1) / (float64(n) + 0.5))
	}
	return s
}

var (
	wordRx = regexp.MustCompile(`[a-z][a-z0-9_.-]{2,}`)
	stop   = map[string]bool{}
)

func init() {
	for _, w := range strings.Fields("the and for you with that this from are was were have has not but all any can will just what when then them " +
		"they your our out into also its it's use run make sure please should would could there here about") {
		stop[w] = true
	}
}

// Toks returns lowercase words of at least three characters, without stop words.
func Toks(text string) []string {
	var out []string
	for _, t := range wordRx.FindAllString(strings.ToLower(text), -1) {
		if !stop[t] {
			out = append(out, t)
		}
	}
	return out
}

// TokSet returns the unique words from Toks.
func TokSet(text string) map[string]bool {
	m := map[string]bool{}
	for _, t := range Toks(text) {
		m[t] = true
	}
	return m
}

func (s *Store) idf(t string) float64 {
	if v, ok := s.IDF[t]; ok {
		return v
	}
	return 1.0
}

func sortedToks(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Overlap ports Store.overlap: idf-weighted share of the lesson tokens found in the query.
func (s *Store) Overlap(q map[string]bool, L *Lesson) float64 {
	if L.toks == nil {
		L.toks = TokSet(L.Text)
	}
	num, den := 0.0, 0.0
	common := false
	for _, t := range sortedToks(L.toks) {
		w := s.idf(t)
		den += w
		if q[t] {
			num += w
			common = true
		}
	}
	if !common {
		return 0
	}
	if den == 0 {
		den = 1
	}
	return num / den
}

// Hit is one surfaced lesson.
type Hit struct {
	Score  float64 `json:"score"`
	Why    string  `json:"why"`
	Lesson *Lesson `json:"lesson"`
}

// Retrieve ports Store.retrieve for error and question triggers (k lessons with score >= MinScore). For
// session_start it returns the plain top k (the pilot rule); use RetrieveStartStrict for the strict rule.
func (s *Store) Retrieve(tr Trigger, k int) []Hit {
	type cand struct {
		h   Hit
		ord int
	}
	cands := map[string]*cand{}
	n := 0
	stacks := map[string]bool{}
	for _, x := range tr.SessStacks {
		stacks[x] = true
	}
	for x := range s.PStacks[tr.Project] {
		stacks[x] = true
	}
	add := func(L *Lesson, base float64, why string) {
		sc := base
		if L.Projects[tr.Project] > 0 {
			sc += 1.0
			why += "+project"
		}
		if len(stacks) > 0 && len(L.Stacks) > 0 {
			inter, uni := 0, len(stacks)
			for _, x := range L.Stacks {
				if stacks[x] {
					inter++
				} else {
					uni++
				}
			}
			sc += 0.5 * float64(inter) / float64(uni)
		}
		if tr.Head != "" {
			for _, h := range L.Heads {
				if h == tr.Head {
					sc += 0.5
					break
				}
			}
		}
		sc += 0.1 * math.Log1p(float64(L.Sessions))
		c := cands[L.ID]
		if c == nil {
			cands[L.ID] = &cand{Hit{sc, why, L}, n}
			n++
		} else if sc > c.h.Score {
			c.h = Hit{sc, why, L}
		}
	}
	if tr.Kind == "error" {
		for _, L := range s.bySig[tr.Sig] {
			add(L, 5.0, "signature")
		}
		for _, L := range s.byCause[tr.Cause] {
			add(L, 4.0, "cause")
		}
	} else {
		q := TokSet(tr.Text)
		for _, L := range s.prefs {
			ov := s.Overlap(q, L)
			if ov >= 0.25 {
				add(L, 4.0*ov, "text"+strconv.FormatFloat(ov, 'f', 2, 64))
			} else if tr.Kind == "session_start" && L.Projects[tr.Project] > 0 {
				add(L, 1.0, "brief")
			}
		}
	}
	all := make([]*cand, 0, len(cands))
	for _, c := range cands {
		all = append(all, c)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].h.Score != all[j].h.Score {
			return all[i].h.Score > all[j].h.Score
		}
		return all[i].ord < all[j].ord
	})
	var out []Hit
	for _, c := range all {
		if tr.Kind != "session_start" && c.h.Score < MinScore {
			continue
		}
		if len(out) == k {
			break
		}
		out = append(out, c.h)
	}
	return out
}

// Strict session-start constants (START_* in lessons.py).
const (
	StartMinOverlap  = 0.25
	StartMinSessions = 2
	StartK           = 2
)

// RetrieveStartStrict ports retrieve_start_strict: a pref lesson surfaces at session start only when its text
// overlaps the first message (>= 0.25), it comes from this repo or a repo with shared stack signals (node alone
// does not count), and it was seen in 2+ sessions. Top k.
func (s *Store) RetrieveStartStrict(tr Trigger, repoStacks func(p string) map[string]bool, k int) []Hit {
	here := repoStacks(tr.Project)
	q := TokSet(tr.Text)
	type cand struct {
		h   Hit
		ord int
	}
	var out []cand
	for i, L := range s.Lessons {
		if L.Kind != "pref" || L.Sessions < StartMinSessions {
			continue
		}
		ov := s.Overlap(q, L)
		if ov == 0 || ov < StartMinOverlap {
			continue
		}
		same := L.Projects[tr.Project] > 0
		lst := map[string]bool{}
		for _, x := range L.Stacks {
			lst[x] = true
		}
		for p := range L.Projects {
			for x := range repoStacks(p) {
				lst[x] = true
			}
		}
		shared, uni := 0, len(here)
		for x := range lst {
			if here[x] {
				if x != "node" {
					shared++
				}
			} else {
				uni++
			}
		}
		if !(same || shared > 0) {
			continue
		}
		if uni == 0 {
			uni = 1
		}
		sc := 4.0*ov + 0.5*float64(shared)/float64(uni) + 0.1*math.Log1p(float64(L.Sessions))
		why := "start" + strconv.FormatFloat(ov, 'f', 2, 64)
		if same {
			sc += 1.0
			why += "+project"
		} else {
			why += "+stack"
		}
		if sc >= MinScore {
			out = append(out, cand{Hit{sc, why, L}, i})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].h.Score > out[j].h.Score })
	var hits []Hit
	for _, c := range out {
		if len(hits) == k {
			break
		}
		hits = append(hits, c.h)
	}
	return hits
}

// Render ports render(): one "[Ln] text" line per hit.
func Render(hits []Hit) string {
	var b strings.Builder
	for i, h := range hits {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("[L" + strconv.Itoa(i+1) + "] " + h.Lesson.Text)
	}
	return b.String()
}

// ---------- helpers ----------

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func addMin(ts string, m int) string {
	if len(ts) < 19 {
		return ts
	}
	t, err := time.Parse("2006-01-02T15:04:05", ts[:19])
	if err != nil {
		return ts
	}
	return t.Add(time.Duration(m) * time.Minute).Format("2006-01-02T15:04:05")
}

func cut19(s string) string {
	if len(s) > 19 {
		return s[:19]
	}
	return s
}

func parFor(n int, fn func(i int)) {
	w := runtime.NumCPU()
	var wg sync.WaitGroup
	next := int64(-1)
	for k := 0; k < w; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(atomic.AddInt64(&next, 1))
				if i >= n {
					return
				}
				fn(i)
			}
		}()
	}
	wg.Wait()
}
