package lessons

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/guards"
)

// Index is the lesson store at build time, cached on disk for the hooks (one gob file, read in a few ms).
type Index struct {
	BuiltAt    string
	Lessons    []*Lesson // all error lessons, and pref lessons seen in StartMinSessions+ sessions
	IDF        map[string]float64
	PStacks    map[string][]string // project -> stacks (command signals)
	RepoStacks map[string][]string // project -> stacks (manifest / framework files)
	NPrefs     int                 // pref lessons used for the IDF
	SSHUsers   map[string]string   // host -> user: `ssh HOST` failed and `ssh USER@HOST` fixed it (local only)
}

// DefaultDir is ~/.local/share/nanaoki.
func DefaultDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "nanaoki")
}

func IndexPath() string { return filepath.Join(DefaultDir(), "lessons.gob") }

func (r *Rolling) BuildIndex() *Index {
	all := r.All("")
	st := NewStore(all, r.pstacksAt(tsMax))
	ix := &Index{BuiltAt: time.Now().UTC().Format(time.RFC3339), IDF: st.IDF, PStacks: map[string][]string{},
		RepoStacks: map[string][]string{}, NPrefs: len(st.prefs)}
	for _, L := range all {
		if L.Kind != "pref" || L.Sessions >= StartMinSessions {
			ix.Lessons = append(ix.Lessons, L)
		}
	}
	for p, m := range st.PStacks {
		if len(m) > 0 {
			ix.PStacks[p] = sortedKeys(m)
		}
	}
	ix.SSHUsers = r.SSHUsersAt(tsMax, "")
	for p := range r.repo {
		if m := r.RepoStacksAt(p, tsMax); len(m) > 0 {
			ix.RepoStacks[p] = sortedKeys(m)
		}
	}
	return ix
}

// SSHUsersAt applies the "fix seen" rule to SSH: a fixed episode (fix before t, not in session excl) whose
// failed command is `ssh HOST` and whose fix is `ssh USER@HOST`. The latest fix wins.
func (r *Rolling) SSHUsersAt(t, excl string) map[string]string {
	out := map[string]string{}
	var eps []*Episode
	for _, v := range r.groups {
		for _, x := range v {
			if x.First.Head == "ssh" && x.FixTs < t && x.Sid != excl {
				eps = append(eps, x)
			}
		}
	}
	sort.Slice(eps, func(i, j int) bool { return eps[i].Ts < eps[j].Ts })
	for _, x := range eps {
		if h, u, ok := guards.SSHPair(x.First.Cmd, r.fixCmd(x, t)); ok {
			out[h] = u
		}
	}
	return out
}

// Save writes the index atomically.
func (ix *Index) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := gob.NewEncoder(f).Encode(ix); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func LoadIndex(path string) (*Index, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var ix Index
	if err := gob.NewDecoder(f).Decode(&ix); err != nil {
		return nil, err
	}
	return &ix, nil
}

// Store returns a retrieval store over the index, with the build-time IDF.
func (ix *Index) Store() *Store {
	ps := map[string]map[string]bool{}
	for p, v := range ix.PStacks {
		ps[p] = set(v)
	}
	s := NewStore(ix.Lessons, ps)
	s.IDF = ix.IDF
	return s
}

func (ix *Index) RepoStacksFn() func(string) map[string]bool {
	return func(p string) map[string]bool { return set(ix.RepoStacks[p]) }
}

func (ix *Index) ByID() map[string]*Lesson {
	m := make(map[string]*Lesson, len(ix.Lessons))
	for _, L := range ix.Lessons {
		m[L.ID] = L
	}
	return m
}

func set(v []string) map[string]bool {
	m := make(map[string]bool, len(v))
	for _, x := range v {
		m[x] = true
	}
	return m
}

// SortLessons orders lessons by sessions (desc), then id.
func SortLessons(ls []*Lesson) {
	sort.SliceStable(ls, func(i, j int) bool {
		if ls[i].Sessions != ls[j].Sessions {
			return ls[i].Sessions > ls[j].Sessions
		}
		return ls[i].ID < ls[j].ID
	})
}
