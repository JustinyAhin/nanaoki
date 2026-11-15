package search

import (
	"path/filepath"
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
	if st, _ := Sync(db, p); st.Added != 0 || st.Removed != 0 {
		t.Errorf("second sync must be a no-op: %+v", st)
	}
	res, err := Search(db, Query{Text: "billing portal configurations", Before: "2026-11-01T10:00:00.000Z", ExcludeSession: "s3"})
	if err != nil || len(res) == 0 || res[0].Session != "s1" || res[0].EventType != "assistant_msg" {
		t.Fatalf("res = %+v %v", res, err)
	}
	for _, r := range res {
		if r.Session == "s3" {
			t.Error("excluded session in results")
		}
	}
	if res, _ := Search(db, Query{Text: "portal", Project: "beta", Agent: "claude", Until: "2026-10-31"}); len(res) != 1 || res[0].Session != "s2" {
		t.Errorf("filters: %+v", res)
	}
	if res, _ := Search(db, Query{Text: `"billing portal" NOT teams`, Raw: true}); len(res) != 2 {
		t.Errorf("raw: %+v", res)
	}
	// the event store changes: a row is deleted and another added
	ev, _ := ingest.OpenStore(p)
	ev.Exec(`delete from events where session_id = 's2'`)
	ev.Exec(`insert into events (agent, session_id, project, ts, event_type, text, dkey) values ('codex','s4','alpha','2026-11-02T00:00:00Z','user_msg','merge the portal configs', x'01')`)
	ev.Close()
	st, _ = Sync(db, p)
	if st.Removed != 1 || st.Added != 1 {
		t.Errorf("resync: %+v", st)
	}
	if MatchExpr("What did we do about the Stripe portal?") != `"stripe" OR "portal"` {
		t.Errorf("match = %q", MatchExpr("What did we do about the Stripe portal?"))
	}
}
