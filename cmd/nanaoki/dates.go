package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const day = "2006-01-02"

var relRe = regexp.MustCompile(`^(\d+)([dwmy])$`)

func shift(t time.Time, n int, unit string) time.Time {
	switch unit {
	case "d":
		return t.AddDate(0, 0, -n)
	case "w":
		return t.AddDate(0, 0, -7*n)
	case "m":
		return t.AddDate(0, -n, 0)
	default:
		return t.AddDate(-n, 0, 0)
	}
}

// A span means "that long ago": with --since 30d the window covers the last 30 days, today included.
func resolveDate(s string, now time.Time) (string, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch {
	case s == "":
		return "", nil
	case s == "today":
		return now.Format(day), nil
	case dateRe.MatchString(s):
		return s, nil
	}
	m := relRe.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("bad date %q (want YYYY-MM-DD, today, or a span like 30d, 4w, 3m, 1y)", s)
	}
	n, _ := strconv.Atoi(m[1])
	if n == 0 {
		return "", fmt.Errorf("bad span %q: must be at least 1", s)
	}
	return shift(now, n, m[2]).AddDate(0, 0, 1).Format(day), nil
}

func historyRange(db *sql.DB) (string, string, error) {
	var lo, hi sql.NullString
	if err := db.QueryRow(`select min(substr(ts,1,10)), max(substr(ts,1,10)) from events where ts is not null and ts != ''`).Scan(&lo, &hi); err != nil {
		return "", "", err
	}
	if !lo.Valid || !hi.Valid {
		return "", "", fmt.Errorf("the store has no events; run nanaoki ingest first")
	}
	return lo.String, hi.String, nil
}

// resolveSplit turns a --split value into TRAIN_START, TRAIN_END, TEST_START, TEST_END.
// Forms:
//
//	T0:T1:S0:S1   explicit days (each may also be a relative span or "today")
//	last:30d      TEST = the last 30 days (also 4w, 3m, 1y); TRAIN = all history before
//	auto          TEST = the last third of your history; TRAIN = the first two thirds
func resolveSplit(s string, now time.Time, db *sql.DB) ([4]string, error) {
	var out [4]string
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "auto" || strings.HasPrefix(s, "last:") {
		first, last, err := historyRange(db)
		if err != nil {
			return out, err
		}
		lastT, _ := time.Parse(day, last)
		var testStart time.Time
		if s == "auto" {
			firstT, _ := time.Parse(day, first)
			span := int(lastT.Sub(firstT).Hours() / 24)
			if span < 3 {
				return out, fmt.Errorf("--split auto needs at least 3 days of history (have %d)", span+1)
			}
			testStart = lastT.AddDate(0, 0, -span/3)
		} else {
			ts, err := resolveDate(strings.TrimPrefix(s, "last:"), lastT)
			if err != nil || !relRe.MatchString(strings.TrimPrefix(s, "last:")) {
				return out, fmt.Errorf("--split last:SPAN wants a span like 30d, 4w, 3m (got %q)", s)
			}
			testStart, _ = time.Parse(day, ts)
		}
		trainEnd := testStart.AddDate(0, 0, -1).Format(day)
		if trainEnd < first {
			return out, fmt.Errorf("--split %s leaves no TRAIN period: history starts on %s", s, first)
		}
		return [4]string{first, trainEnd, testStart.Format(day), last}, nil
	}
	p := strings.Split(s, ":")
	if len(p) != 4 {
		return out, fmt.Errorf("--split wants auto, last:SPAN (e.g. last:30d), or TRAIN_START:TRAIN_END:TEST_START:TEST_END")
	}
	for i, d := range p {
		v, err := resolveDate(d, now)
		if err != nil || v == "" {
			return out, fmt.Errorf("bad --split date %q", d)
		}
		out[i] = v
	}
	if !(out[0] <= out[1] && out[1] < out[2] && out[2] <= out[3]) {
		return out, fmt.Errorf("--split dates must be in order: TRAIN_START <= TRAIN_END < TEST_START <= TEST_END (got %s)", strings.Join(out[:], ":"))
	}
	return out, nil
}
