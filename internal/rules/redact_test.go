package rules

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"
)

func TestRedact(t *testing.T) {
	SetNames([]string{"Jane Roe", "Jane", "jroe"})
	defer func() { namesOK = false }()
	cases := map[string]string{
		"mail jane.roe@example.com now":                          "mail <email> now",
		"call +1 (555) 123-4567 today":                           "call <phone> today",
		"key sk-ant-abcdefghijklmnop":                            "key <secret>",
		"export API_KEY=abc123 && run":                           "export API_KEY=<redacted> && run",
		"cd /Users/jroe/work/x":                                  "cd ~/work/x",
		"Thanks Jane, it works":                                  "Thanks Nanaoki, it works",
		"ping 10.0.0.12 now":                                     "ping <ip> now",
		"ping @someone about it":                                 "ping @<handle> about it",
		"import @scope/pkg.js and @lib.ts":                       "import @scope/pkg.js and @lib.ts",
		"psql postgres://u:p@h/db -c x":                          "psql <db-url> -c x",
		"tok abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGH end":   "tok <secret> end",
		"run gggggggggggggggggggggggggggggggggggggggggggggg end": "run gggggggggggggggggggggggggggggggggggggggggggggg end",
		"version 1.2.3-alpha+4":                                  "version 1.2.3-alpha+4",
	}
	for in, want := range cases {
		if got := Redact(in, 0); got != want {
			t.Errorf("Redact(%q) = %q, want %q", in, got, want)
		}
	}
	if got := Redact("abcdef", 3); got != "abc…" {
		t.Errorf("limit: %q", got)
	}
}

// TestRedactDiff compares with outputs of the Python redact() (REDACT_DIFF=cases.jsonl; not in the repo).
func TestRedactDiff(t *testing.T) {
	p := os.Getenv("REDACT_DIFF")
	if p == "" {
		t.Skip("set REDACT_DIFF")
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			t.Fatalf("read header: %v", err)
		}
		t.Fatal("comparison file is empty")
	}
	var hdr struct{ Names []string }
	if err := json.Unmarshal(sc.Bytes(), &hdr); err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if hdr.Names == nil {
		t.Fatal("header must contain a names array")
	}
	SetNames(hdr.Names)
	defer func() { namesOK = false }()
	n, bad := 0, 0
	for sc.Scan() {
		var c struct{ In, Out *string }
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatalf("decode line %d: %v", n+2, err)
		}
		if c.In == nil || c.Out == nil {
			t.Fatalf("line %d must contain in and out strings", n+2)
		}
		n++
		if got := Redact(*c.In, 160); got != *c.Out {
			bad++
			if bad <= 8 {
				t.Errorf("diff:\n in  %q\n py  %q\n go  %q", Trunc(*c.In, 300), *c.Out, got)
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read comparison cases: %v", err)
	}
	if n == 0 {
		t.Fatal("comparison file contains no cases")
	}
	t.Logf("%d cases, %d differ", n, bad)
}
