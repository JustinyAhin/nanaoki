package search

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/JustinyAhin/nanaoki/internal/ingest"
	"github.com/JustinyAhin/nanaoki/internal/testfixture"
)

func TestSyncAndSearch(t *testing.T) {
	evs := []testfixture.Ev{
		{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:00:00.000Z", Type: "user_msg", Text: "Set up the Stripe billing portal for the pro plan"},
		{Agent: "codex", Sid: "s1", Project: "alpha", Ts: "2026-09-01T10:05:00.000Z", Type: "assistant_msg", Text: "I created two billing portal configurations: one for pro, one for teams."},
		{Agent: "claude", Sid: "s2", Project: "beta", Ts: "2026-10-01T10:00:00.000Z", Type: "tool_call", Text: "rg -n portal src", Out: "src/billing.ts:12: const portal = stripe.billingPortal"},
		{Agent: "claude", Sid: "s3", Project: "beta", Ts: "2026-11-01T10:00:00.000Z", Type: "user_msg", Text: "Are the two billing portal configurations merged?"},
	}
	p := testfixture.EventsDB(t, ingest.OpenStore, evs)
	db, err := Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st, err := Sync(db, p)
	if err != nil || st.Added != 4 {
		t.Fatalf("sync: %+v %v", st, err)
	}
	if st, err := Sync(db, p); err != nil || st.Added != 0 || st.Removed != 0 {
		t.Errorf("second sync must be a no-op: %+v %v", st, err)
	}
	res, err := Search(db, Query{Text: "billing portal configurations", Before: "2026-11-01T10:00:00.000Z"})
	if err != nil || len(res) == 0 || res[0].Session != "s1" || res[0].EventType != "assistant_msg" {
		t.Fatalf("res = %+v %v", res, err)
	}
	for _, tc := range []struct {
		name string
		q    Query
		want []string
	}{
		{"before", Query{Before: "2026-11-01T10:00:00.000Z"}, []string{"s1/assistant_msg", "s1/user_msg", "s2/tool_call"}},
		{"until", Query{Until: "2026-10-01"}, []string{"s1/assistant_msg", "s1/user_msg", "s2/tool_call"}},
		{"exclude project", Query{ExcludeProjects: map[string]bool{"beta": true}}, []string{"s1/assistant_msg", "s1/user_msg"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.q.Text = "portal"
			tc.q.PerSession = 2
			res, err := Search(db, tc.q)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, r := range res {
				got = append(got, r.Session+"/"+r.EventType)
			}
			slices.Sort(got)
			if !slices.Equal(got, tc.want) {
				t.Errorf("results = %v, want %v", got, tc.want)
			}
		})
	}
	if res, err := Search(db, Query{Text: `"billing portal" NOT teams`, Raw: true}); err != nil || len(res) != 2 {
		t.Errorf("raw: %+v %v", res, err)
	}
	ev, err := ingest.OpenStore(p)
	if err != nil {
		t.Fatal(err)
	}
	defer ev.Close()
	if _, err := ev.Exec(`delete from events where session_id = 's2'`); err != nil {
		t.Fatal(err)
	}
	if _, err := ev.Exec(`insert into events (agent, session_id, project, ts, event_type, text, dkey) values ('codex','s4','alpha','2026-11-02T00:00:00Z','user_msg','merge the portal configs', x'01')`); err != nil {
		t.Fatal(err)
	}
	st, err = Sync(db, p)
	if err != nil || st.Removed != 1 || st.Added != 1 {
		t.Errorf("resync: %+v %v", st, err)
	}
	if MatchExpr("What did we do about the Stripe portal?") != `"stripe" OR "portal"` {
		t.Errorf("match = %q", MatchExpr("What did we do about the Stripe portal?"))
	}
}

func TestRanking(t *testing.T) {
	evs := []testfixture.Ev{
		// the agent searching: the words are only in the grep and sed patterns and in grep's path:line output
		{Agent: "claude", Sid: "grep", Project: "p", Ts: "2026-09-01T10:00:00Z", Type: "tool_call",
			Text: `grep -rn "502\|prerender" web/src | head; sed -n '/prerender 502/p' x.log`, Out: "web/src/a.ts:12: // 502 prerender"},
		// the answer
		{Agent: "claude", Sid: "answer", Project: "p", Ts: "2026-09-02T10:00:00Z", Type: "assistant_msg",
			Text: "The CMS returned 502 while pages were prerendering. The fix retries each prerender request three times on 502, with a long wait between tries so the CMS has time to restart."},
		// a tool error whose output matches
		{Agent: "codex", Sid: "error", Project: "p", Ts: "2026-09-03T10:00:00Z", Type: "tool_error",
			Text: "npm run build", Out: "Error: 502 Bad Gateway during prerender of /de/"},
		// three hits in one session
		{Agent: "codex", Sid: "many", Project: "q", Ts: "2026-09-04T10:00:00Z", Type: "user_msg", Text: "why does prerender fail with 502?"},
		{Agent: "codex", Sid: "many", Project: "q", Ts: "2026-09-04T10:01:00Z", Type: "tool_call", Text: "curl -I localhost", Out: "HTTP/1.1 502 prerender"},
		{Agent: "codex", Sid: "many", Project: "q", Ts: "2026-09-04T10:02:00Z", Type: "assistant_msg", Text: "prerender 502 again"},
		// the current session: same project, last event 2 minutes ago
		{Agent: "claude", Sid: "current", Project: "q", Ts: "2026-09-05T09:58:00Z", Type: "tool_call", Text: "nanaoki search 502 prerender"},
	}
	p := testfixture.EventsDB(t, ingest.OpenStore, evs)
	db, err := Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := Sync(db, p); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	cur, err := CurrentSession(db, "q", now)
	if err != nil || cur != "current" {
		t.Fatalf("current session = %q %v", cur, err)
	}
	if cur, _ := CurrentSession(db, "q", now.Add(time.Hour)); cur != "" {
		t.Errorf("a session idle for over 10 minutes is not current: %q", cur)
	}
	if cur, _ := CurrentSession(db, "p", now); cur != "" {
		t.Errorf("no recent session in p: %q", cur)
	}
	res, err := Search(db, Query{Text: "502 prerender", Exclude: []string{cur}})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range res {
		got = append(got, r.Session)
		if r.Session == "many" && r.Hits != 3 {
			t.Errorf("session hits = %d, want 3", r.Hits)
		}
		if r.Session == "many" && r.EventType == "tool_call" {
			t.Error("the group must show its best hit, not the tool call")
		}
	}
	if len(got) != 4 || got[len(got)-1] != "grep" || slices.Contains(got, "current") {
		t.Errorf("order = %v: search commands must rank last, the current session must be left out", got)
	}
	if i, j := slices.Index(got, "answer"), slices.Index(got, "error"); i < 0 || j < 0 || i > j {
		t.Errorf("order = %v: the answer must rank above the tool error", got)
	}
}
