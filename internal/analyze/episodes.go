package analyze

import (
	"sort"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Episode is one (session, key) pair. The key is an env/stack cause, else the exact informative signature.
type Episode struct {
	Agent, Sid, Kind, Key, Sig, Cause, Project, Ts, Split, Outcome, Source string
	N, Side                                                                int
	FirstID                                                                int64
	Sigs                                                                   map[string]bool
	Errs                                                                   []*Err
}

// KeyID is "cause:<id>" or "sig:<signature>".
func (x *Episode) KeyID() string { return x.Kind + ":" + x.Key }

// Episodes ports repeat_after_fix.episodes. errs must be in ts order.
func Episodes(errs []*Err) []*Episode {
	idx := map[string]*Episode{}
	var out []*Episode
	for _, e := range errs {
		kind, key := "sig", e.Sig
		if e.Cause != "" && rules.Scope[e.Cause] != "code" {
			kind, key = "cause", e.Cause
		}
		if kind == "sig" && !rules.Informative(key) {
			continue
		}
		k := e.Agent + "\x00" + e.Sid + "\x00" + kind + "\x00" + key
		x := idx[k]
		if x == nil {
			x = &Episode{Agent: e.Agent, Sid: e.Sid, Kind: kind, Key: key, Sig: e.Sig, Cause: e.Cause, Project: e.Project,
				Ts: e.Ts, Split: e.Split, Outcome: e.Outcome, Source: e.Source, Side: e.Side, FirstID: e.ID, Sigs: map[string]bool{}}
			idx[k] = x
			out = append(out, x)
		}
		x.N++
		x.Sigs[e.Sig] = true
		x.Errs = append(x.Errs, e)
		if e.Outcome == "fixed" {
			x.Outcome = "fixed"
		}
	}
	return out
}

// FixedErr returns the first error of the episode that was fixed (nil if none).
func (x *Episode) FixedErr() *Err {
	for _, e := range x.Errs {
		if e.Outcome == "fixed" {
			return e
		}
	}
	return nil
}

// Repeat is one episode matched against earlier fixed episodes with the same key.
type Repeat struct {
	X                                      *Episode
	MatchSig, MatchCause, ExactSig         bool
	SameProject, SameAgent, OtherAgentOnly bool
	ReportedOnly, EarlierOnly              bool
	Best, Earliest                         *Episode // best ref (Python order) and earliest fix
	PriorProjects, PriorSessions           int
}

// Matched reports a repeat-after-fix.
func (r *Repeat) Matched() bool { return r.MatchSig || r.MatchCause }

func matchRow(x *Episode, mAny []*Episode) Repeat {
	r := Repeat{X: x}
	if len(mAny) == 0 {
		return r
	}
	if x.Kind == "sig" {
		r.MatchSig = true
	} else {
		r.MatchCause = true
	}
	for _, t := range mAny {
		for s := range x.Sigs {
			if t.Sigs[s] {
				r.ExactSig = true
			}
		}
		if t.Project == x.Project {
			r.SameProject = true
		}
		if t.Agent == x.Agent {
			r.SameAgent = true
		}
	}
	r.OtherAgentOnly = !r.SameAgent
	best := append([]*Episode(nil), mAny...)
	sort.SliceStable(best, func(i, j int) bool {
		a, b := best[i], best[j]
		ap, bp := a.Project != x.Project, b.Project != x.Project
		if ap != bp {
			return !ap
		}
		aa, ba := a.Agent != x.Agent, b.Agent != x.Agent
		if aa != ba {
			return !aa
		}
		return a.Ts < b.Ts
	})
	r.Best = best[0]
	r.Earliest = mAny[0]
	for _, t := range mAny {
		if t.Ts < r.Earliest.Ts {
			r.Earliest = t
		}
	}
	projs, sess := map[string]bool{}, map[string]bool{}
	for _, t := range mAny {
		projs[t.Project] = true
		sess[t.Sid] = true
	}
	r.PriorProjects, r.PriorSessions = len(projs), len(sess)
	return r
}

// Chrono is repeat-after-fix over the whole history: an episode repeats when an earlier episode in another
// session with the same key was fixed. This is the "earlier fix" rule of repeat_after_fix.py applied to all
// dates instead of a fixed TRAIN/TEST split.
func Chrono(eps []*Episode) []Repeat {
	sorted := append([]*Episode(nil), eps...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Ts < sorted[j].Ts })
	fixed := map[string][]*Episode{}
	reported := map[string][]*Episode{}
	out := make([]Repeat, 0, len(sorted))
	for _, x := range sorted {
		k := x.KeyID()
		r := matchRow(x, fixed[k])
		if !r.Matched() && len(reported[k]) > 0 {
			r.ReportedOnly = true
		}
		out = append(out, r)
		switch x.Outcome {
		case "fixed":
			fixed[k] = append(fixed[k], x)
		case "reported":
			reported[k] = append(reported[k], x)
		}
	}
	return out
}

// Split is the TRAIN/TEST split of the Python reference.
type Split struct {
	TrainStart string `json:"train_start"`
	TrainEnd   string `json:"train_end"`
	TestStart  string `json:"test_start"`
	TestEnd    string `json:"test_end"`
}

// Of ports common.split_of.
func (s Split) Of(ts string) string {
	d := ts
	if len(d) > 10 {
		d = d[:10]
	}
	if s.TrainStart <= d && d <= s.TrainEnd {
		return "train"
	}
	if s.TestStart <= d && d <= s.TestEnd {
		return "test"
	}
	return "out"
}

// TrainTest ports the Q2 loop of repeat_after_fix.py exactly (TEST episodes vs FIXED TRAIN episodes).
func TrainTest(eps []*Episode) []Repeat {
	trainFixed := map[string][]*Episode{}
	trainAny := map[string][]*Episode{}
	var test []*Episode
	for _, x := range eps {
		switch x.Split {
		case "train":
			trainAny[x.KeyID()] = append(trainAny[x.KeyID()], x)
			if x.Outcome == "fixed" {
				trainFixed[x.KeyID()] = append(trainFixed[x.KeyID()], x)
			}
		case "test":
			test = append(test, x)
		}
	}
	sort.SliceStable(test, func(i, j int) bool { return test[i].Ts < test[j].Ts })
	earlier := map[string][]*Episode{}
	var out []Repeat
	for _, x := range test {
		k := x.KeyID()
		r := matchRow(x, trainFixed[k])
		if !r.Matched() {
			for _, t := range trainAny[k] {
				if t.Outcome == "reported" {
					r.ReportedOnly = true
				}
			}
			if !r.ReportedOnly {
				for _, t := range earlier[k] {
					if t.Sid != x.Sid {
						r.EarlierOnly = true
					}
				}
			}
		}
		out = append(out, r)
		if x.Outcome == "fixed" {
			earlier[k] = append(earlier[k], x)
		}
	}
	return out
}
