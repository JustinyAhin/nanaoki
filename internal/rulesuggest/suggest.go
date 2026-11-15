// Package rulesuggest proposes CLAUDE.md and AGENTS.md lines from rules the user keeps typing. It never writes a file.
package rulesuggest

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// MinSessions: a rule or decision restated in this many sessions is stable.
const MinSessions = 2

type Rule struct {
	Kind     string         `json:"kind"` // restated | decision
	Key      string         `json:"key"`
	Text     string         `json:"text"`
	Scope    string         `json:"scope"` // global | project
	Project  string         `json:"project,omitempty"`
	Sessions int            `json:"sessions"`
	Projects map[string]int `json:"projects"`
	First    string         `json:"first"`
	Last     string         `json:"last"`
}

type Options struct {
	Before          string          // only prompts before this ts ("" = all)
	ExcludeSession  string          // eval: the trigger's own session
	ExcludeProjects map[string]bool // prompts of these projects are left out
}

// Suggest returns the rules, most sessions first.
func Suggest(d *analyze.Data, o Options) []Rule {
	var out []Rule
	matched := map[int]bool{}
	for j, rl := range rules.PromptRules {
		if rl.Kind != "pref" {
			continue
		}
		x := Rule{Kind: "restated", Key: rl.Key, Text: rl.Label + ".", Projects: map[string]int{}}
		sess := map[string]bool{}
		for _, p := range d.Prompts {
			if !p.Typed() || !p.Has(j) || (o.Before != "" && p.Ts >= o.Before) || p.Sid == o.ExcludeSession ||
				o.ExcludeProjects[p.Project] {
				continue
			}
			sess[p.Agent+"\x00"+p.Sid] = true
			x.Projects[p.Project]++
			if x.First == "" || p.Ts < x.First {
				x.First = p.Ts
			}
			if p.Ts > x.Last {
				x.Last = p.Ts
			}
		}
		x.Sessions = len(sess)
		if x.Sessions >= MinSessions {
			matched[j] = true
			out = append(out, scope(x))
		}
	}
	// decisions: the C2 instruction sentences (lessons.PrefSentences), from typed prompts only
	type occ struct {
		text, first, last string
		sess              map[string]bool
		projects          map[string]int
	}
	byKey := map[string]*occ{}
	var order []string
	for _, p := range d.Prompts {
		if !p.Typed() || (o.Before != "" && p.Ts >= o.Before) || p.Sid == o.ExcludeSession || o.ExcludeProjects[p.Project] {
			continue
		}
		for _, ps := range lessons.PrefSentences(rules.PromptBody(p.Text)) {
			x := byKey[ps[0]]
			if x == nil {
				x = &occ{text: ps[1], first: p.Ts, sess: map[string]bool{}, projects: map[string]int{}}
				byKey[ps[0]] = x
				order = append(order, ps[0])
			}
			x.sess[p.Agent+"\x00"+p.Sid] = true
			x.projects[p.Project]++
			if p.Ts < x.first {
				x.first, x.text = p.Ts, ps[1]
			}
			if p.Ts > x.last {
				x.last = p.Ts
			}
		}
	}
	for _, k := range order {
		x := byKey[k]
		if len(x.sess) < MinSessions {
			continue
		}
		dup := false
		for j := range matched {
			if rules.PromptRules[j].Rx.MatchString(x.text) {
				dup = true // the restated rule already says it
			}
		}
		if dup {
			continue
		}
		out = append(out, scope(Rule{Kind: "decision", Key: k, Text: rules.Redact(x.text, 240), Sessions: len(x.sess),
			Projects: x.projects, First: x.first, Last: x.last}))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sessions > out[j].Sessions })
	return out
}

// scope: a rule seen in one project only belongs to that project's AGENTS.md; else it is global.
func scope(x Rule) Rule {
	x.Scope = "global"
	if len(x.Projects) == 1 {
		for p := range x.Projects {
			x.Scope, x.Project = "project", p
		}
	}
	return x
}

// Present reports a rule that the file already states: the rule's own regex (restated rules) or the
// normalized sentence (decisions) is in the file text.
func Present(x Rule, file string) bool {
	if x.Kind == "restated" {
		for _, rl := range rules.PromptRules {
			if rl.Key == x.Key {
				return rl.Rx.MatchString(file)
			}
		}
		return false
	}
	return strings.Contains(lessons.NormSentence(file), x.Key)
}

// Diff prints a unified diff that appends the missing rules to path. It reads path (if it exists) and
// writes nothing.
func Diff(path, display string, rs []Rule) (string, int) {
	b, _ := os.ReadFile(path)
	old := string(b)
	var add []Rule
	for _, x := range rs {
		if !Present(x, old) {
			add = append(add, x)
		}
	}
	if len(add) == 0 {
		return "", 0
	}
	lines := strings.Split(strings.TrimRight(old, "\n"), "\n")
	n := len(lines)
	if old == "" {
		n = 0
	}
	var plus []string
	if n > 0 {
		plus = append(plus, "")
	}
	plus = append(plus, "## Rules from your sessions (proposed by nanaoki)", "")
	for _, x := range add {
		plus = append(plus, "- "+strings.Join(strings.Fields(x.Text), " "))
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- a/%s\n+++ b/%s\n@@ -%d,0 +%d,%d @@\n", display, display, n, n+1, len(plus))
	for _, l := range plus {
		sb.WriteString("+" + l + "\n")
	}
	return sb.String(), len(add)
}
