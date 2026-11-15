package guards

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func noPath(string) (string, error) { return "", errors.New("not found") }

func fakeFS(files map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		if b, ok := files[p]; ok {
			return []byte(b), nil
		}
		return nil, os.ErrNotExist
	}
}

func find(fs []Finding, guard string) *Finding {
	for i := range fs {
		if fs[i].Guard == guard {
			return &fs[i]
		}
	}
	return nil
}

func TestParse(t *testing.T) {
	cs := Parse(`cd /x && echo 'a b' "c $HOME" d\ e | grep -n "x" > out.txt; ls`)
	if len(cs) != 4 {
		t.Fatalf("got %d commands", len(cs))
	}
	w := cs[1].Words
	if w[1].Value != "a b" || !w[1].Quoted[0] || !w[2].HasSubst || w[3].Value != "d e" {
		t.Errorf("words = %+v", w)
	}
	if len(cs[2].Words) != 3 { // grep -n x (the redirection target is dropped)
		t.Errorf("redirect: %+v", cs[2].Words)
	}
	if got := Parse("cat <<'EOF'\n*.md ?\nEOF"); len(got) != 1 || len(got[0].Words) != 1 {
		t.Errorf("heredoc body must be skipped: %+v", got)
	}
}

func TestZshEq(t *testing.T) {
	e := &Env{Shell: "/bin/zsh", LookPath: noPath}
	f := find(Check(`echo ===== && echo done`, e), "zsh_eq")
	if f == nil || f.Decision != Block || f.Fixed != `echo '=====' && echo done` {
		t.Fatalf("got %+v", f)
	}
	for _, ok := range []string{`echo '====='`, `[[ $a == b ]]`, `echo a=b`, `echo "=x"`} {
		if find(Check(ok, e), "zsh_eq") != nil {
			t.Errorf("%q must pass", ok)
		}
	}
	// "=ls" is a valid path expansion when ls exists
	e.LookPath = func(s string) (string, error) { return "/bin/" + s, nil }
	if find(Check(`file =ls`, e), "zsh_eq") != nil {
		t.Error("=ls must pass when ls is a command")
	}
	if find(Check(`echo =====`, &Env{Shell: "/bin/bash", LookPath: noPath}), "zsh_eq") != nil {
		t.Error("bash: no zsh guard")
	}
}

func TestZshGlob(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), nil, 0o644)
	e := &Env{Shell: "/bin/zsh", Cwd: dir, LookPath: noPath}
	cases := map[string]string{
		`curl https://x.test/a?b=1`:          `curl 'https://x.test/a?b=1'`,
		`cat src/routes/[slug]/+page.svelte`: `cat 'src/routes/[slug]/+page.svelte'`,
		`grep -r foo --include=*.ts .`:       `grep -r foo '--include=*.ts' .`,
		`rg x src/routes/[id] && ls`:         `rg x 'src/routes/[id]' && ls`,
	}
	for in, want := range cases {
		f := find(Check(in, e), "zsh_glob")
		if f == nil || f.Fixed != want || f.Decision != Warn {
			t.Errorf("%q: got %+v, want fixed %q", in, f, want)
		}
	}
	for _, ok := range []string{`ls *.md`, `curl 'https://x.test/a?b=1'`, `echo $?`, `noglob curl x?y`, `[ -f x ]`,
		`FOO=*.md env`, `echo "$(ls *.zz)"`} {
		if f := find(Check(ok, e), "zsh_glob"); f != nil {
			t.Errorf("%q must pass, got %+v", ok, f)
		}
	}
	// Both were false warnings in live use: the glob is checked in the folder after `cd`, and quotes
	// inside $(...) inside "..." must not end the string.
	sub := filepath.Join(dir, "sub")
	os.Mkdir(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "x.go"), nil, 0o644)
	for _, ok := range []string{`cd sub; grep -n x *.go`, `cd ` + sub + ` && ls *.go`, `cd "$D"; ls *.go`,
		`echo "$w: $(node -e 'console.log("a")')" ; ls *.md`} {
		if f := find(Check(ok, e), "zsh_glob"); f != nil {
			t.Errorf("%q must pass, got %+v", ok, f)
		}
	}
	if find(Check(`cd sub; ls *.md`, e), "zsh_glob") == nil {
		t.Error("cd sub; ls *.md: sub has no .md file, must warn")
	}

	r := &Env{Shell: "/bin/zsh", Replay: true}
	if find(Check(`ls *.md`, r), "zsh_glob") == nil {
		t.Error("replay assumes no match")
	}
}

func TestFormatBeforeCommit(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"format":"prettier --write ."}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "bun.lock"), nil, 0o644)
	e := &Env{Cwd: dir, LookPath: noPath}
	f := find(Check(`git add -A && git commit -m "fix: x"`, e), "format_before_commit")
	if f == nil || f.Fixed != `bun run format && git add -A && git commit -m "fix: x"` {
		t.Fatalf("got %+v", f)
	}
	if find(Check(`bun run format && git commit -m x`, e), "format_before_commit") != nil {
		t.Error("format in the same command must pass")
	}
	e.FormatRan = true
	if find(Check(`git commit -m x`, e), "format_before_commit") != nil {
		t.Error("format already ran in the session")
	}
	if find(Check(`git commit -m x`, &Env{Cwd: t.TempDir(), LookPath: noPath}), "format_before_commit") != nil {
		t.Error("no repo, no format script: must pass")
	}
}

func TestNodeVersion(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("v20.11.0\n"), 0o644)
	e := &Env{Cwd: dir, LookPath: noPath, NodeMajor: func() (int, bool) { return 22, true }}
	f := find(Check(`npm run build`, e), "node_version")
	if f == nil || f.Fixed != "nvm use 20 && <command>" {
		t.Fatalf("got %+v", f)
	}
	if find(Check(`ls`, e), "node_version") != nil {
		t.Error("no node command: must pass")
	}
	e.NodeMajor = func() (int, bool) { return 20, true }
	if find(Check(`npm run build`, e), "node_version") != nil {
		t.Error("same major: must pass")
	}
	n, from, ok := PinnedNode(dir, &Env{ReadFile: fakeFS(map[string]string{filepath.Join(dir, "package.json"): `{"engines":{"node":">=18 <19"}}`})})
	if !ok || n != 18 || from != "package.json engines.node" {
		t.Errorf("engines: %d %q %v", n, from, ok)
	}
	if _, _, ok := PinnedNode(dir, &Env{ReadFile: fakeFS(map[string]string{filepath.Join(dir, ".nvmrc"): "lts/*"})}); ok {
		t.Error("lts/* has no major")
	}
}

func TestSSHUser(t *testing.T) {
	h, u, ok := SSHPair(`ssh myvps 'uptime'`, `ssh -o ConnectTimeout=5 admin@myvps 'uptime'`)
	if !ok || h != "myvps" || u != "admin" {
		t.Fatalf("pair = %q %q %v", h, u, ok)
	}
	e := &Env{SSHUsers: map[string]string{h: u}, LookPath: noPath}
	f := find(Check(`ssh myvps "docker ps"`, e), "ssh_user")
	if f == nil || f.Fixed != `ssh admin@myvps "docker ps"` {
		t.Fatalf("got %+v", f)
	}
	if f := find(Check(`scp a.txt myvps:/tmp/`, e), "ssh_user"); f == nil || f.Fixed != `scp a.txt admin@myvps:/tmp/` {
		t.Errorf("scp: %+v", f)
	}
	for _, ok := range []string{`ssh admin@myvps ls`, `ssh other ls`} {
		if find(Check(ok, e), "ssh_user") != nil {
			t.Errorf("%q must pass", ok)
		}
	}
}

func TestCombinedFix(t *testing.T) {
	e := &Env{Shell: "/bin/zsh", Cwd: t.TempDir(), LookPath: noPath}
	fs := Check(`curl https://x.test/a?b=1 && echo ===`, e)
	if len(fs) != 2 || fs[0].Fixed != `curl 'https://x.test/a?b=1' && echo '==='` || fs[1].Fixed != fs[0].Fixed {
		t.Errorf("got %+v", fs)
	}
}
