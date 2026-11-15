package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

type causeDef struct {
	ID, Scope, Lesson, Rx, CmdRx string
}

type cause struct {
	causeDef
	rx, crx *regexp.Regexp
}

var causes []cause

// Scope and Lesson by cause id.
var (
	Scope  = map[string]string{}
	Lesson = map[string]string{}
)

// Version changes when the cause table changes; stored causes are then recomputed.
var Version string

func init() {
	h := sha256.New()
	for _, d := range causeTable {
		h.Write([]byte(d.ID + "\x00" + d.Scope + "\x00" + d.Rx + "\x00" + d.CmdRx + "\x01"))
	}
	Version = hex.EncodeToString(h.Sum(nil))[:16]
	for _, d := range causeTable {
		c := cause{causeDef: d, rx: py(d.Rx, "(?im)")}
		if d.CmdRx != "" {
			c.crx = py(d.CmdRx, "(?i)")
		}
		causes = append(causes, c)
		Scope[d.ID] = d.Scope
		Lesson[d.ID] = d.Lesson
	}
}

// CauseOfEvent applies CauseOf the way repeat_after_fix.load does: pasted error logs have no command,
// and the output falls back to the text.
func CauseOfEvent(eventType, text, output string) string {
	cmd := text
	if eventType == "user_error_paste" {
		cmd = ""
	}
	if output == "" {
		output = text
	}
	return CauseOf(cmd, output)
}

// CauseOf returns the first rule that matches the output (or the command, for rules with a
// command regex) wins.
func CauseOf(cmd, out string) string {
	for _, c := range causes {
		if c.crx != nil && !c.crx.MatchString(cmd+"\n"+out) {
			continue
		}
		if c.rx.MatchString(out) || (c.crx != nil && c.rx.MatchString(cmd)) {
			return c.ID
		}
	}
	return ""
}

func CauseIDs() []string {
	ids := make([]string, len(causes))
	for i, c := range causes {
		ids[i] = c.ID
	}
	return ids
}
