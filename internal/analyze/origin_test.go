package analyze

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

// Synthetic data only.

const injectedBlock = "<system>\n## Plain language\nUse plain English.\n</system>\n" +
	"Error: something failed in step one\nError: cannot read config\nfatal: not allowed here\nmore text"

func paste(id int64, sid, ts, rf string, side int) *Err {
	return &Err{ID: id, Agent: "codex", Sid: sid, Ts: ts, EType: "user_error_paste", Sig: "Error: cannot read config", RawFile: rf, RawLine: 7, Side: side}
}

func TestResolveOriginsAndDropPastes(t *testing.T) {
	tmpl := strings.Repeat("Run the nightly check and report every failed step. ", 6) + "\nError: check failed\nError: x\nError: y"
	d := &Data{}
	var msgs []Msg
	add := func(id int64, sid, text string, side int) {
		rf := fmt.Sprintf("f%d", id)
		msgs = append(msgs, Msg{Agent: "codex", Sid: sid, Text: text, RawFile: rf, RawLine: 7, Side: side})
		d.Errs = append(d.Errs, paste(id, sid, "2026-10-18T10:00:00Z", rf, side))
	}
	add(1, "s1", "my build fails:\nError: cannot read config\nError: y\nError: z", 0) // typed
	add(2, "s2", injectedBlock, 0)                                                    // injected by a wrapper app
	add(3, "s3", "parent agent prompt\nError: a\nError: b\nError: c", 1)              // subagent / codex exec
	for i, sid := range []string{"t1", "t2", "t3"} {                                  // one template in 3 sessions
		add(int64(10+i), sid, tmpl, 0)
	}
	// a tool error is never dropped
	d.Errs = append(d.Errs, &Err{ID: 99, Agent: "codex", Sid: "s9", Ts: "2026-10-18T11:00:00Z", EType: "tool_error", Sig: "x"})

	ResolveOrigins(d, msgs)
	want := map[int64]string{1: rules.OriginTyped, 2: rules.OriginInjected, 3: rules.OriginScripted,
		10: rules.OriginTemplate, 11: rules.OriginTemplate, 12: rules.OriginTemplate}
	for _, e := range d.Errs {
		if w, ok := want[e.ID]; ok && e.Origin != w {
			t.Errorf("paste %d: origin %s, want %s", e.ID, e.Origin, w)
		}
	}

	// --no-filters keeps everything
	if n := dropPastes(d, true); n != nil || len(d.Errs) != 7 {
		t.Fatalf("no-filters dropped %v, %d left", n, len(d.Errs))
	}
	n := dropPastes(d, false)
	if n["injected"] != 1 || n["scripted"] != 1 || n["template"] != 3 || len(d.Errs) != 2 {
		t.Fatalf("dropped %v, %d left", n, len(d.Errs))
	}
	if d.Errs[0].ID != 1 || d.Errs[1].ID != 99 {
		t.Errorf("kept %d and %d, want the typed paste and the tool error", d.Errs[0].ID, d.Errs[1].ID)
	}
}

func TestPromptOriginsInRuleCounts(t *testing.T) {
	j := -1
	for i, r := range rules.PromptRules {
		if r.Key == "short_no_fluff" {
			j = i
		}
	}
	mk := func(sid, day, text string) *Prompt {
		return &Prompt{Agent: "claude", Sid: sid, Ts: day + "T10:00:00Z", Text: text, Rules: 1 << uint(j)}
	}
	tmpl := strings.Repeat("Write the weekly update for the team blog and keep it short. ", 4) + "Then keep it short, no fluff."
	d := &Data{Prompts: []*Prompt{
		mk("a", "2026-10-01", "keep the post short, no fluff"),
		mk("b", "2026-10-02", "no fluff please, the copy reads like a robot"),
		mk("c", "2026-10-03", "/tighten no fluff"),
		mk("d", "2026-10-04", "$tighten no fluff and copy to my clipboard"),
		mk("e", "2026-10-05", tmpl),
		mk("f", "2026-10-06", tmpl),
		mk("g", "2026-10-07", tmpl),
	}}
	var msgs []Msg
	for i, p := range d.Prompts {
		msgs = append(msgs, Msg{Agent: p.Agent, Sid: p.Sid, Text: p.Text, RawFile: "h", RawLine: int64(i)})
	}
	ResolveOrigins(d, msgs)

	it := promptRules(d.Prompts, Filter{}, true)[j]
	if it.Msgs != 2 || it.Sessions != 2 || it.AllMsgs != 7 || it.AllSessions != 7 {
		t.Errorf("typed only: %+v", it)
	}
	if it.ByOrigin["command"] != 2 || it.ByOrigin["template"] != 3 || it.ByOrigin["typed"] != 2 {
		t.Errorf("by origin = %v", it.ByOrigin)
	}
	all := promptRules(d.Prompts, Filter{}, false)[j]
	if all.Msgs != 7 || all.Sessions != 7 {
		t.Errorf("no filters: %+v", all)
	}
}

// A batch run puts one error in 40 sessions on 2 days; a real habit puts another error in 3 sessions on 3 days.
func TestSpreadRule(t *testing.T) {
	var reps []Repeat
	ep := func(sid, ts, key string) {
		e := &Err{Agent: "codex", Sid: sid, Ts: ts, Project: "p"}
		reps = append(reps, Repeat{X: &Episode{Agent: "codex", Sid: sid, Kind: "sig", Key: key, Project: "p", Ts: ts, N: 1, Errs: []*Err{e}}})
	}
	for i := 0; i < 40; i++ {
		ep(fmt.Sprintf("b%d", i), fmt.Sprintf("2026-10-%02dT10:%02d:00Z", 18+i%2, i), "batch message")
	}
	for i, day := range []string{"2026-09-01", "2026-09-15", "2026-10-02"} {
		ep(fmt.Sprintf("h%d", i), day+"T09:00:00Z", "real habit")
	}
	items, keys, events := repeatedErrors(reps, Filter{}, 3, map[*Episode]float64{})
	if keys != 1 || events != 3 || len(items) != 1 || items[0].Key != "sig:real habit" {
		t.Errorf("min 3 days: keys=%d events=%d items=%+v", keys, events, items)
	}
	items, keys, _ = repeatedErrors(reps, Filter{}, 2, map[*Episode]float64{})
	if keys != 2 || items[0].Key != "sig:batch message" || items[0].Sessions != 40 || items[0].Days != 2 {
		t.Errorf("min 2 days (no filters): keys=%d items=%+v", keys, items)
	}
}
