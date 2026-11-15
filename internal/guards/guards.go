package guards

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	Warn  = "warn"  // tell the agent and suggest the fixed command; the command still goes to the normal permission flow
	Block = "block" // deny the command (only guards marked safe: the command fails for certain)
)

type Finding struct {
	Guard    string `json:"guard"`
	Cause    string `json:"cause"` // rules.Lesson id
	Decision string `json:"decision"`
	Message  string `json:"message"`
	Fixed    string `json:"fixed,omitempty"` // the suggested command
	words    []Word
}

// Env is what the guards may look at. Tests fill it by hand.
type Env struct {
	Cwd      string
	Shell    string            // login shell path; the zsh guards run only for zsh
	SSHUsers map[string]string // host -> user, from fixed lessons
	// Replay mode (eval): do not read the file system or PATH; assume the glob matched nothing and the
	// "=word" is not a command (the error already happened).
	Replay    bool
	FormatRan bool // a format script already ran in this session
	LookPath  func(string) (string, error)
	ReadFile  func(string) ([]byte, error)
	NodeMajor func() (int, bool) // major version of the node on PATH
}

func (e *Env) lookPath(s string) (string, error) {
	if e.LookPath != nil {
		return e.LookPath(s)
	}
	return exec.LookPath(s)
}

func (e *Env) readFile(p string) ([]byte, error) {
	if e.ReadFile != nil {
		return e.ReadFile(p)
	}
	return os.ReadFile(p)
}

func (e *Env) isZsh() bool { return strings.HasSuffix(e.Shell, "zsh") }

func Check(cmd string, e *Env) []Finding {
	var out []Finding
	cmds := Parse(cmd)
	if e.isZsh() {
		f1, ok1 := zshEq(cmd, cmds, e)
		f2, ok2 := zshGlob(cmd, cmds, e)
		if ok1 && ok2 { // one suggested command that quotes both
			bad := append(append([]Word(nil), f1.words...), f2.words...)
			sort.Slice(bad, func(i, j int) bool { return bad[i].Start < bad[j].Start })
			fixed := replaceSpans(cmd, bad, func(w Word) string { return singleQuote(w.Value) })
			f1.Fixed, f2.Fixed = fixed, fixed
		}
		if ok1 {
			out = append(out, f1)
		}
		if ok2 {
			out = append(out, f2)
		}
	}
	if f, ok := formatBeforeCommit(cmd, cmds, e); ok {
		out = append(out, f)
	}
	if f, ok := nodeVersion(cmds, e); ok {
		out = append(out, f)
	}
	if f, ok := sshUser(cmd, cmds, e); ok {
		out = append(out, f)
	}
	return out
}

// zshEq: zsh expands an unquoted word that starts with "=" to the path of that command ("=ls" -> /bin/ls).
// When no such command exists, zsh stops with "=... not found". Safe to block: the command cannot run.
// Words inside [[ ... ]] are not expanded and are skipped.
func zshEq(src string, cmds []Command, e *Env) (Finding, bool) {
	var bad []Word
	for _, c := range cmds {
		inTest := false
		for i, w := range c.Words {
			if i == 0 && w.Value == "noglob" {
				break
			}
			if w.Value == "[[" && !w.Quoted[0] {
				inTest = true
			}
			if w.Value == "]]" {
				inTest = false
			}
			if inTest || w.HasSubst || len(w.Value) < 2 || w.Value[0] != '=' || w.Quoted[0] {
				continue
			}
			name := w.Value[1:]
			if !e.Replay && strings.Trim(name, "=") != "" {
				if _, err := e.lookPath(name); err == nil {
					continue // "=ls" is a valid path expansion
				}
			}
			bad = append(bad, w)
		}
	}
	if len(bad) == 0 {
		return Finding{}, false
	}
	return Finding{Guard: "zsh_eq", Cause: "zsh_eq_quote", Decision: Block,
		Message: "In zsh an unquoted word that starts with '=' expands to a command path, so `" + bad[0].Value +
			"` fails with \"not found\". Quote it.",
		Fixed: replaceSpans(src, bad, func(w Word) string { return singleQuote(w.Value) }), words: bad}, true
}

func globChars(w Word) bool {
	open := -1
	for i := 0; i < len(w.Value); i++ {
		if w.Quoted[i] {
			continue
		}
		switch w.Value[i] {
		case '*', '?':
			return true
		case '[':
			open = i
		case ']':
			if open >= 0 && i > open+1 {
				return true
			}
		}
	}
	return false
}

// globPattern turns the word into a filepath.Match pattern (quoted glob chars escaped).
func globPattern(w Word) string {
	var b strings.Builder
	for i := 0; i < len(w.Value); i++ {
		c := w.Value[i]
		if w.Quoted[i] && (c == '*' || c == '?' || c == '[' || c == ']' || c == '\\') {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

// zshGlob: zsh stops with "no matches found" when an unquoted glob matches no file (bash passes the word
// through). Typical: URLs with '?', SvelteKit [slug] paths, --include=*.md. The guard checks the file system
// in the cwd: a glob that matches files is fine. Warn only (the files can change before the command runs).
func zshGlob(src string, cmds []Command, e *Env) (Finding, bool) {
	var bad []Word
	cwd := e.Cwd
	for _, c := range cmds {
		if c.Words[0].Value == "cd" || c.Words[0].Value == "pushd" {
			cwd = cdTarget(cwd, c)
			continue
		}
		if cwd == "" && !e.Replay {
			continue // after a cd we cannot follow, the cwd is unknown: do not guess
		}
		for i, w := range c.Words {
			if i == 0 && (w.Value == "noglob" || w.Value == "[[" || w.Value == "[") {
				break
			}
			if w.HasSubst || !globChars(w) {
				continue
			}
			if i == 0 || isAssignment(w) {
				continue
			}
			if strings.Contains(w.Value, "**") && !e.Replay {
				continue // recursive glob: not checked
			}
			if !e.Replay {
				p := globPattern(w)
				if strings.HasPrefix(p, "~/") {
					home, _ := os.UserHomeDir()
					p = filepath.Join(home, p[2:])
				} else if !filepath.IsAbs(p) {
					p = filepath.Join(cwd, p)
				}
				if m, err := filepath.Glob(p); err == nil && len(m) > 0 {
					continue
				}
			}
			bad = append(bad, w)
		}
	}
	if len(bad) == 0 {
		return Finding{}, false
	}
	return Finding{Guard: "zsh_glob", Cause: "zsh_glob_nomatch", Decision: Warn,
		Message: "zsh fails with \"no matches found\" when an unquoted glob matches no file: `" + bad[0].Value +
			"`. Quote words with [ ] * ? (URLs, [slug] routes, --include=*.md).",
		Fixed: replaceSpans(src, bad, func(w Word) string { return singleQuote(w.Value) }), words: bad}, true
}

// cdTarget is the cwd after `cd DIR`, or "" when it cannot be known (cd -, a variable, a glob).
func cdTarget(cwd string, c Command) string {
	args := c.Words[1:]
	for len(args) > 0 && strings.HasPrefix(args[0].Value, "-") && args[0].Value != "-" {
		args = args[1:] // -P, -L
	}
	home, _ := os.UserHomeDir()
	if len(args) == 0 {
		return home
	}
	w := args[0]
	if w.HasSubst || w.Value == "-" || globChars(w) {
		return ""
	}
	d := w.Value
	switch {
	case d == "~":
		return home
	case strings.HasPrefix(d, "~/"):
		return filepath.Join(home, d[2:])
	case filepath.IsAbs(d):
		return d
	case cwd == "":
		return ""
	}
	return filepath.Join(cwd, d)
}

var assignRx = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

func isAssignment(w Word) bool { return assignRx.MatchString(w.Value) && !w.Quoted[0] }

// gitSub returns the git subcommand of a simple command (skipping "-C dir" and "-c k=v").
func gitSub(c Command) (string, string) {
	ws := c.Words
	if len(ws) == 0 || filepath.Base(ws[0].Value) != "git" {
		return "", ""
	}
	dir := ""
	for i := 1; i < len(ws); i++ {
		v := ws[i].Value
		if (v == "-C" || v == "-c") && i+1 < len(ws) {
			if v == "-C" {
				dir = ws[i+1].Value
			}
			i++
			continue
		}
		if strings.HasPrefix(v, "-") {
			continue
		}
		return v, dir
	}
	return "", dir
}

func RepoRoot(dir string) string {
	for d := dir; d != "" && d != "/" && d != "."; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
	}
	return ""
}

// FormatScript returns the package manager command for the repo's format script ("" when there is none).
func FormatScript(root string, e *Env) string {
	b, err := e.readFile(filepath.Join(root, "package.json"))
	if err != nil {
		return ""
	}
	var pj struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pj) != nil {
		return ""
	}
	name := ""
	for _, k := range []string{"format", "fmt", "prettier", "format:write"} {
		if pj.Scripts[k] != "" {
			name = k
			break
		}
	}
	if name == "" {
		return ""
	}
	pm := "npm"
	for _, x := range []struct{ file, pm string }{{"bun.lock", "bun"}, {"bun.lockb", "bun"}, {"pnpm-lock.yaml", "pnpm"}, {"yarn.lock", "yarn"}} {
		if _, err := e.readFile(filepath.Join(root, x.file)); err == nil {
			pm = x.pm
			break
		}
	}
	return pm + " run " + name
}

var FormatRx = regexp.MustCompile(`\b(prettier|biome (format|check)|dprint fmt|gofmt|go fmt|ruff format|black)\b|\brun (format|fmt|prettier)\b|\b(bun|pnpm|yarn|npm) (run )?format\b`)

// formatBeforeCommit: the repo has a format script, the session has not run a formatter, and the command is
// git commit. Warn and suggest "<pm> run format && <command>".
func formatBeforeCommit(full string, cmds []Command, e *Env) (Finding, bool) {
	if e.FormatRan {
		return Finding{}, false
	}
	commit := false
	dir := e.Cwd
	for _, c := range cmds {
		var vals []string
		for _, w := range c.Words {
			vals = append(vals, w.Value)
		}
		line := strings.Join(vals, " ")
		if FormatRx.MatchString(line) {
			return Finding{}, false
		}
		if sub, d := gitSub(c); sub == "commit" {
			commit = true
			if d != "" {
				if filepath.IsAbs(d) {
					dir = d
				} else {
					dir = filepath.Join(e.Cwd, d)
				}
			}
		}
	}
	if !commit {
		return Finding{}, false
	}
	root := dir
	if !e.Replay {
		root = RepoRoot(dir)
	}
	if root == "" {
		return Finding{}, false
	}
	fs := FormatScript(root, e)
	if fs == "" {
		return Finding{}, false
	}
	return Finding{Guard: "format_before_commit", Cause: "prettier_check", Decision: Warn,
		Message: "This repo has a format script and this session has not run it. Format before commit, or the format check fails.",
		Fixed:   fs + " && " + strings.TrimSpace(full)}, true
}

var nodeHeads = map[string]bool{"npm": true, "npx": true, "pnpm": true, "yarn": true, "node": true, "corepack": true}

var majorRx = regexp.MustCompile(`(\d+)`)

// PinnedNode reads the Node major version the repo pins: .nvmrc, .node-version, .tool-versions, or the first
// number of package.json engines.node. ok=false when nothing is pinned or the pin has no number (lts/*).
func PinnedNode(root string, e *Env) (int, string, bool) {
	for _, f := range []string{".nvmrc", ".node-version"} {
		if b, err := e.readFile(filepath.Join(root, f)); err == nil {
			if m := majorRx.FindString(strings.TrimSpace(string(b))); m != "" {
				n, _ := strconv.Atoi(m)
				return n, f, true
			}
		}
	}
	if b, err := e.readFile(filepath.Join(root, ".tool-versions")); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			fs := strings.Fields(l)
			if len(fs) >= 2 && fs[0] == "nodejs" {
				if m := majorRx.FindString(fs[1]); m != "" {
					n, _ := strconv.Atoi(m)
					return n, ".tool-versions", true
				}
			}
		}
	}
	if b, err := e.readFile(filepath.Join(root, "package.json")); err == nil {
		var pj struct {
			Engines map[string]string `json:"engines"`
		}
		if json.Unmarshal(b, &pj) == nil {
			if m := majorRx.FindString(pj.Engines["node"]); m != "" {
				n, _ := strconv.Atoi(m)
				return n, "package.json engines.node", true
			}
		}
	}
	return 0, "", false
}

// nodeVersion: the command runs Node tooling, the repo pins a Node major, and the node on PATH has another
// major. Warn and suggest switching first.
func nodeVersion(cmds []Command, e *Env) (Finding, bool) {
	run := false
	for _, c := range cmds {
		if len(c.Words) > 0 && nodeHeads[filepath.Base(c.Words[0].Value)] {
			run = true
		}
	}
	if !run || e.NodeMajor == nil {
		return Finding{}, false
	}
	root := RepoRoot(e.Cwd)
	if root == "" {
		root = e.Cwd
	}
	want, from, ok := PinnedNode(root, e)
	if !ok {
		return Finding{}, false
	}
	have, ok := e.NodeMajor()
	if !ok || have == want {
		return Finding{}, false
	}
	sw := "nvm use " + strconv.Itoa(want)
	if _, err := e.lookPath("fnm"); err == nil {
		sw = "fnm use " + strconv.Itoa(want)
	}
	return Finding{Guard: "node_version", Cause: "node_version", Decision: Warn,
		Message: "The repo pins Node " + strconv.Itoa(want) + " (" + from + ") but node on PATH is " + strconv.Itoa(have) +
			". Switch first.",
		Fixed: sw + " && <command>"}, true
}

// NodeMajorFromPath finds the node major version without running node when the install path holds it
// (fnm, nvm, volta, asdf, Homebrew node@N); else it runs `node --version`.
func NodeMajorFromPath(lookPath func(string) (string, error)) (int, bool) {
	p, err := lookPath("node")
	if err != nil {
		return 0, false
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	if m := regexp.MustCompile(`(?:/v|node@|/node/|installs/nodejs/)(\d+)(?:[./]|$)`).FindStringSubmatch(p); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, true
	}
	out, err := exec.Command(p, "--version").Output()
	if err != nil {
		return 0, false
	}
	if m := majorRx.FindString(string(out)); m != "" {
		n, _ := strconv.Atoi(m)
		return n, true
	}
	return 0, false
}

// sshUser: an earlier lesson shows that `ssh HOST` failed and `ssh USER@HOST` fixed it. Warn on `ssh HOST`
// with no user and suggest the user from the lesson.
func sshUser(src string, cmds []Command, e *Env) (Finding, bool) {
	if len(e.SSHUsers) == 0 {
		return Finding{}, false
	}
	for _, c := range cmds {
		if len(c.Words) < 2 {
			continue
		}
		h := filepath.Base(c.Words[0].Value)
		if h != "ssh" && h != "scp" && h != "rsync" {
			continue
		}
		for _, w := range c.Words[1:] {
			v := w.Value
			if strings.HasPrefix(v, "-") || strings.Contains(v, "@") {
				continue
			}
			host := v
			if h != "ssh" {
				i := strings.Index(v, ":")
				if i <= 0 {
					continue
				}
				host = v[:i]
			}
			if u, ok := e.SSHUsers[host]; ok {
				fixed := src[:w.Start] + u + "@" + src[w.Start:]
				return Finding{Guard: "ssh_user", Cause: "ssh_vps_user", Decision: Warn,
					Message: "An earlier session fixed `" + h + " " + host + "` by using the user " + u + ". Use " + u + "@" + host + ".",
					Fixed:   fixed}, true
			}
			if h == "ssh" {
				break // first non-option word is the host
			}
		}
	}
	return Finding{}, false
}

// SSHPair extracts (host, user) when failed is `ssh HOST ...` with no user and fix is `ssh USER@HOST ...`.
func SSHPair(failed, fix string) (string, string, bool) {
	fh := sshHost(failed)
	if fh == "" || strings.Contains(fh, "@") {
		return "", "", false
	}
	xh := sshHost(fix)
	u, h, ok := strings.Cut(xh, "@")
	if !ok || h != fh || u == "" {
		return "", "", false
	}
	return h, u, true
}

func sshHost(cmd string) string {
	for _, c := range Parse(cmd) {
		if len(c.Words) < 2 || filepath.Base(c.Words[0].Value) != "ssh" {
			continue
		}
		skipNext := false
		for _, w := range c.Words[1:] {
			if skipNext {
				skipNext = false
				continue
			}
			if strings.HasPrefix(w.Value, "-") {
				if len(w.Value) == 2 && strings.ContainsAny(w.Value[1:], "bcDEeFIiJLlmOopQRSWw") {
					skipNext = true
				}
				continue
			}
			return w.Value
		}
	}
	return ""
}
