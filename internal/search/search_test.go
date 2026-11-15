package search

import (
	"path/filepath"
	"slices"
	"testing"

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
		{"exclude session", Query{ExcludeSession: "s3"}, []string{"s1/assistant_msg", "s1/user_msg", "s2/tool_call"}},
		{"project", Query{Project: "beta"}, []string{"s2/tool_call", "s3/user_msg"}},
		{"agent", Query{Agent: "claude"}, []string{"s2/tool_call", "s3/user_msg"}},
		{"since", Query{Since: "2026-10-01"}, []string{"s2/tool_call", "s3/user_msg"}},
		{"until", Query{Until: "2026-10-01"}, []string{"s1/assistant_msg", "s1/user_msg", "s2/tool_call"}},
		{"exclude project", Query{ExcludeProjects: map[string]bool{"beta": true}}, []string{"s1/assistant_msg", "s1/user_msg"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.q.Text = "portal"
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
