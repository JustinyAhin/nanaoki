package stats

import (
	"testing"

	"github.com/JustinyAhin/nanaoki/internal/activity"
	"github.com/JustinyAhin/nanaoki/internal/analyze"
	"github.com/JustinyAhin/nanaoki/internal/lessons"
)

const globErr = "Exit code 1\n(eval):1: no matches found: src/[slug]"

func TestLessonOutcome(t *testing.T) {
	L := &lessons.Lesson{ID: "cause:x", Kind: "cause", Key: "zsh_glob_nomatch"}
	before := &analyze.Err{ID: 1, Ts: "2026-11-15T10:00:00.400Z", Cause: "zsh_glob_nomatch"} // the error the lesson answered
	other := &analyze.Err{ID: 2, Ts: "2026-11-15T10:01:00.000Z", Cause: "node_version"}
	again := &analyze.Err{ID: 3, Ts: "2026-11-15T10:02:00.000Z", Cause: "zsh_glob_nomatch"}
	shown := "2026-11-15T10:00:00Z"

	if out, _ := LessonOutcome(L, shown, []*analyze.Err{before, other}, true); out != NotAgain {
		t.Fatalf("same-second trigger error or another cause must not count: %s", out)
	}
	out, e := LessonOutcome(L, shown, []*analyze.Err{before, other, again}, true)
	if out != Again || e.ID != 3 {
		t.Fatalf("want again on event 3, got %s %v", out, e)
	}
	if out, _ := LessonOutcome(L, shown, []*analyze.Err{before}, false); out != NoData {
		t.Fatalf("no later events: want no_data, got %s", out)
	}
	sig := &lessons.Lesson{ID: "sig:y", Kind: "sig", Key: "Error: boom"}
	if out, _ := LessonOutcome(sig, shown, []*analyze.Err{{ID: 4, Ts: "2026-11-15T10:03:00Z", Sig: "Error: boom"}}, true); out != Again {
		t.Fatalf("signature lesson: want again, got %s", out)
	}
}

func ev(id int64, ts, cmd, out string, isErr bool) Event {
	return Event{ID: id, Ts: ts, Tool: "Bash", Cmd: cmd, Out: out, Err: isErr}
}

func TestGuardOutcome(t *testing.T) {
	warn := activity.GuardEntry{Ts: "2026-11-15T10:00:05Z", Guard: "zsh_glob", Cause: "zsh_glob_nomatch", Decision: "warn",
		Cmd: activity.Cmd("ls src/[slug]"), Fixed: activity.Cmd("ls 'src/[slug]'")}
	cases := []struct {
		name string
		g    activity.GuardEntry
		evs  []Event
		want string
		id   int64
	}{
		{"warned command failed, then the fix", warn, []Event{
			ev(1, "2026-11-15T10:00:00Z", "git status", "", false), // earlier command: not the anchor
			ev(2, "2026-11-15T10:00:03Z", "ls src/[slug]", globErr, true),
			ev(3, "2026-11-15T10:00:20Z", "ls 'src/[slug]'", "page.svelte", false),
		}, Followed, 3},
		{"warned command failed, fix written another way", warn, []Event{
			ev(2, "2026-11-15T10:00:03Z", "ls src/[slug]", globErr, true),
			ev(3, "2026-11-15T10:00:20Z", `ls "src/[slug]" -la`, "page.svelte", false),
		}, Followed, 3},
		{"warned command failed twice", warn, []Event{
			ev(2, "2026-11-15T10:00:03Z", "ls src/[slug]", globErr, true),
			ev(3, "2026-11-15T10:00:30Z", "ls src/[slug]/x", globErr, true),
		}, Repeated, 2},
		{"warned command ran fine", warn, []Event{
			ev(2, "2026-11-15T10:00:06Z", "ls src/[slug]", "page.svelte", false),
		}, Ignored, 2},
		{"no events", warn, nil, NoData, 0},
		{"later than the window", warn, []Event{
			ev(9, "2026-11-15T11:00:00Z", "ls 'src/[slug]'", "", false),
		}, NoData, 0},
		{"blocked command: its deny event is not a failure", activity.GuardEntry{Ts: "2026-11-15T10:00:05Z", Guard: "zsh_eq",
			Cause: "zsh_eq_quote", Decision: "block", Cmd: "echo =====", Fixed: "echo '====='"}, []Event{
			ev(2, "2026-11-15T10:00:04Z", "echo =====", "nanaoki guard: ... fails with \"not found\"", true),
			ev(3, "2026-11-15T10:00:15Z", "echo '====='", "=====", false),
		}, Followed, 3},
	}
	for _, c := range cases {
		out, e := GuardOutcome(c.g, c.evs)
		if out != c.want {
			t.Errorf("%s: got %s, want %s", c.name, out, c.want)
			continue
		}
		if c.id != 0 && (e == nil || e.ID != c.id) {
			t.Errorf("%s: evidence %v, want event %d", c.name, e, c.id)
		}
	}
}

func rep(ts string, matched bool) analyze.Repeat {
	return analyze.Repeat{X: &analyze.Episode{Ts: ts}, MatchCause: matched}
}

func TestBaselineOf(t *testing.T) {
	reps := []analyze.Repeat{
		rep("2026-10-31T09:00:00Z", true), // before the before-window
		rep("2026-11-01T09:00:00Z", true),
		rep("2026-11-07T09:00:00Z", false),
		rep("2026-11-07T10:00:00Z", true),
		rep("2026-11-08T09:00:00Z", false), // first use day: after
		rep("2026-11-10T09:00:00Z", true),
		rep("2026-11-14T23:59:59Z", false),
		rep("2026-11-15T00:00:00Z", true), // after until
	}
	b := BaselineOf(reps, "2026-11-08T08:00:00Z", "2026-11-01", "2026-11-14")
	// after = 11-08..11-14 (since is before first use), 7 days; before = 11-01..11-07
	if b.After.From != "2026-11-08" || b.After.To != "2026-11-14" || b.Before.From != "2026-11-01" || b.Before.To != "2026-11-07" {
		t.Fatalf("windows: %+v", b)
	}
	if b.Before.Episodes != 3 || b.Before.Repeats != 2 || b.After.Episodes != 3 || b.After.Repeats != 1 {
		t.Fatalf("counts: before %+v after %+v", b.Before, b.After)
	}
	if b.After.Rate < 0.333 || b.After.Rate > 0.334 {
		t.Fatalf("rate %v", b.After.Rate)
	}
	if b := BaselineOf(reps, "", "", "2026-11-14"); b.After.From != "" {
		t.Fatalf("no first use: want empty, got %+v", b)
	}
}
