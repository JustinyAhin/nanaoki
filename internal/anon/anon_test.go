package anon

import (
	"os"
	"testing"
)

func TestAnon(t *testing.T) {
	home, _ := os.UserHomeDir()
	a := New([]string{"birthdays", "acme-shop", "web", "x"})
	if a.Project("acme-shop") != "project-a" || a.Project("birthdays") != "project-b" || a.Project("acme-shop") != "project-a" {
		t.Fatalf("aliases not stable: %v", a.alias)
	}
	if a.Project("") != "" || a.Project("?") != "?" {
		t.Error("empty project must stay")
	}
	got := a.Text("cd " + home + "/work/projects/birthdays && grep web src; mail me@example.com about Birthdays/acme-shop")
	if got != "cd ~/work/projects/project-b && grep web src; mail <email> about project-b/project-a" {
		t.Errorf("text = %q", got)
	}
	lesson := a.Lesson("Wrap hrefs/goto with resolve() from $app/paths. Fix seen: `bunx eslint src/lib/books/index.ts` and `ls src/a.ts`.")
	if lesson != "Wrap hrefs/goto with resolve() from $app/paths. Fix seen: `bunx eslint <path>` and `ls <path>`." {
		t.Errorf("lesson = %q", lesson)
	}
	w := New([]string{"can", "remote", "harborline"})
	if got := w.Text("it can't reach the remote of harborline"); got != "it can't reach the remote of project-a" {
		t.Errorf("common words = %q", got)
	}
	if a.Session("01a0a0d8-d1c3") != "01a0a0" {
		t.Error("session not shortened")
	}
	var n *Anon
	if n.Text("x /Users/me") != "x /Users/me" || n.Project("p") != "p" {
		t.Error("nil Anon must change nothing")
	}
}
