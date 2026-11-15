package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/JustinyAhin/nanaoki/internal/rules"
)

type Options struct {
	DB      string
	Paths   Paths
	Workers int
	Rebuild bool      // drop the store and ingest everything again
	Log     io.Writer // progress lines; nil = quiet
}

type Stats struct {
	FilesSeen, FilesParsed, FilesUnchanged, FilesGone int
	RowsInserted, RowsDeduped, RowsDerived            int
	SQLiteThreads, SQLiteMissing                      int
	Errors                                            []string
	Elapsed                                           time.Duration
}

type result struct {
	idx  int
	rows []Row
	err  error
}

var derivedSources = []string{"codex_sqlite", "claude_history", "codex_history"}

// Run builds or updates the store. Files whose (path, size, mtime) did not change are skipped.
// Events from files that are gone (for example Claude's 30-day cleanup) are kept.
func Run(o Options) (Stats, error) {
	t0 := time.Now()
	var st Stats
	logf := func(format string, a ...any) {
		if o.Log != nil {
			fmt.Fprintf(o.Log, format+"\n", a...)
		}
	}
	if o.Workers <= 0 {
		o.Workers = runtime.NumCPU()
	}
	db, err := OpenStore(o.DB)
	if err != nil {
		return st, err
	}
	defer db.Close()
	ctx := context.Background()
	if o.Rebuild {
		for _, q := range []string{"delete from events", "delete from files", "delete from meta"} {
			if _, err := db.Exec(q); err != nil {
				return st, err
			}
		}
	}

	files := Discover(o.Paths)
	st.FilesSeen = len(files)
	type known struct {
		id          int64
		size, mtime int64
	}
	prev := map[string]known{}
	rs, err := db.Query(`select id, path, size, mtime_ns from files`)
	if err != nil {
		return st, err
	}
	for rs.Next() {
		var k known
		var p string
		if err := rs.Scan(&k.id, &p, &k.size, &k.mtime); err == nil {
			prev[p] = k
		}
	}
	rs.Close()
	present := map[string]bool{}
	var todo []SrcFile
	for _, f := range files {
		present[f.Path] = true
		if k, ok := prev[f.Path]; ok && k.size == f.Size && k.mtime == f.Mtime {
			st.FilesUnchanged++
			continue
		}
		todo = append(todo, f)
	}
	for p := range prev {
		if !present[p] {
			st.FilesGone++
		}
	}
	fresh := len(prev) == 0

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return st, err
	}
	defer tx.Rollback()
	if fresh {
		// bulk load without indexes, add them at the end
		for _, q := range []string{"drop index if exists ix_type", "drop index if exists ix_sess", "drop index if exists ix_sig",
			"drop index if exists ix_ts", "drop index if exists ix_file", "drop index if exists ix_src"} {
			if _, err := tx.Exec(q); err != nil {
				return st, err
			}
		}
	}
	if _, err := tx.Exec(`update files set present = 0`); err != nil {
		return st, err
	}
	for _, f := range todo {
		if k, ok := prev[f.Path]; ok {
			if _, err := tx.Exec(`delete from events where file_id = ?`, k.id); err != nil {
				return st, err
			}
		}
	}
	// derived sources are rebuilt every run (they depend on the full set of rollouts)
	for _, s := range derivedSources {
		if _, err := tx.Exec(`delete from events where source = ?`, s); err != nil {
			return st, err
		}
	}

	seen := map[[16]byte]struct{}{}
	if !fresh {
		rs, err := tx.Query(`select dkey from events where dkey is not null`)
		if err != nil {
			return st, err
		}
		for rs.Next() {
			var b []byte
			if rs.Scan(&b) == nil && len(b) == 16 {
				var k [16]byte
				copy(k[:], b)
				seen[k] = struct{}{}
			}
		}
		rs.Close()
	}

	// Parse in parallel, write in file order.
	ins := newInserter(tx)
	jobs := make(chan int)
	results := make(chan result, o.Workers*2)
	window := make(chan struct{}, o.Workers*4) // bounds parsed-but-unwritten files
	for w := 0; w < o.Workers; w++ {
		go func() {
			dec, _ := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxWindow(1<<30))
			defer dec.Close()
			for i := range jobs {
				f := todo[i]
				var rows []Row
				var err error
				if f.Kind == "claude" {
					rows, err = parseClaude(f.Path, o.Paths.Rel(f.Path))
				} else {
					rows, err = parseCodex(f.Path, o.Paths.Rel(f.Path), dec)
				}
				results <- result{i, rows, err}
			}
		}()
	}
	go func() {
		for i := range todo {
			window <- struct{}{}
			jobs <- i
		}
		close(jobs)
	}()
	pending := map[int]result{}
	next := 0
	var werr error
	upsert, err := tx.Prepare(`insert into files (path, kind, size, mtime_ns, rows, present) values (?,?,?,?,?,1)
		on conflict(path) do update set kind=excluded.kind, size=excluded.size, mtime_ns=excluded.mtime_ns, rows=excluded.rows, present=1
		returning id`)
	if err != nil {
		return st, err
	}
	defer upsert.Close()
	for next < len(todo) {
		r := <-results
		pending[r.idx] = r
		for {
			cur, ok := pending[next]
			if !ok {
				break
			}
			delete(pending, next)
			<-window
			f := todo[next]
			next++
			if werr != nil {
				continue
			}
			if cur.err != nil {
				st.Errors = append(st.Errors, fmt.Sprintf("%s: %v", o.Paths.Rel(f.Path), cur.err))
			}
			var fid int64
			if werr = upsert.QueryRow(f.Path, f.Kind, f.Size, f.Mtime, 0).Scan(&fid); werr != nil {
				continue
			}
			kept := 0
			for _, row := range cur.rows {
				k := dkey(row)
				if _, dup := seen[k]; dup {
					st.RowsDeduped++
					continue
				}
				seen[k] = struct{}{}
				if werr = ins.add(row, fid, k[:]); werr != nil {
					break
				}
				kept++
			}
			if werr == nil {
				_, werr = tx.Exec(`update files set rows = ? where id = ?`, kept, fid)
			}
			st.FilesParsed++
			st.RowsInserted += kept
			if o.Log != nil && st.FilesParsed%500 == 0 {
				logf("  %d/%d files, %d rows, %.0fs", st.FilesParsed, len(todo), st.RowsInserted, time.Since(t0).Seconds())
			}
		}
	}
	if werr != nil {
		return st, werr
	}
	if err := ins.flush(); err != nil {
		return st, err
	}
	// files that are on disk but were unchanged are present too
	for _, f := range files {
		if k, ok := prev[f.Path]; ok && k.size == f.Size && k.mtime == f.Mtime {
			if _, err := tx.Exec(`update files set present = 1 where id = ?`, k.id); err != nil {
				return st, err
			}
		}
	}

	knownT := map[string]bool{}
	sidCwd := map[string]string{}
	rs2, err := tx.Query(`select session_id, cwd from events where agent='codex' and source='codex_rollout' and session_id is not null order by id`)
	if err != nil {
		return st, err
	}
	for rs2.Next() {
		var sid string
		var cwd sql.NullString
		if rs2.Scan(&sid, &cwd) == nil {
			knownT[sid] = true
			if cwd.Valid && cwd.String != "" {
				if _, ok := sidCwd[sid]; !ok {
					sidCwd[sid] = cwd.String
				}
			}
		}
	}
	rs2.Close()
	rs3, err := tx.Query(`select path from files where kind='codex'`)
	if err != nil {
		return st, err
	}
	for rs3.Next() {
		var p string
		if rs3.Scan(&p) == nil {
			if t := ThreadIDFromName(p); t != "" {
				knownT[t] = true
			}
		}
	}
	rs3.Close()
	emit := func(r Row) error {
		st.RowsDerived++
		return ins.add(r, nil, nil)
	}
	if st.SQLiteThreads, st.SQLiteMissing, err = codexSQLiteRows(o.Paths, knownT, emit); err != nil {
		st.Errors = append(st.Errors, "codex sqlite: "+err.Error())
	}
	if err := claudeHistoryRows(o.Paths, emit); err != nil {
		st.Errors = append(st.Errors, "claude history: "+err.Error())
	}
	if err := codexHistoryRows(o.Paths, sidCwd, emit); err != nil {
		st.Errors = append(st.Errors, "codex history: "+err.Error())
	}
	if err := ins.close(); err != nil {
		return st, err
	}
	for _, q := range indexes {
		if _, err := tx.Exec(q); err != nil {
			return st, err
		}
	}
	if err := fillCauses(tx, o.Workers); err != nil {
		return st, err
	}
	st.Elapsed = time.Since(t0)
	if _, err := tx.Exec(`insert or replace into meta (k, v) values ('last_ingest', ?), ('ingest_seconds', ?), ('rules_version', ?)`,
		time.Now().UTC().Format(time.RFC3339), fmt.Sprintf("%.1f", st.Elapsed.Seconds()), rules.Version); err != nil {
		return st, err
	}
	if err := tx.Commit(); err != nil {
		return st, err
	}
	st.Elapsed = time.Since(t0)
	return st, nil
}

// fillCauses stores the cause id of each error event ("" = no cause). Rows are recomputed when the cause
// table changed since the last ingest.
func fillCauses(tx *sql.Tx, workers int) error {
	var ver string
	_ = tx.QueryRow(`select v from meta where k='rules_version'`).Scan(&ver)
	if ver != rules.Version {
		if _, err := tx.Exec(`update events set cause = null where cause is not null`); err != nil {
			return err
		}
	}
	type item struct {
		id               int64
		etype, text, out string
		cause            string
	}
	rs, err := tx.Query(`select id, event_type, coalesce(text,''), coalesce(output,'') from events where is_error=1 and cause is null`)
	if err != nil {
		return err
	}
	var items []item
	for rs.Next() {
		var it item
		if err := rs.Scan(&it.id, &it.etype, &it.text, &it.out); err != nil {
			rs.Close()
			return err
		}
		items = append(items, it)
	}
	rs.Close()
	next := int64(-1)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(atomic.AddInt64(&next, 1))
				if i >= len(items) {
					return
				}
				items[i].cause = rules.CauseOfEvent(items[i].etype, items[i].text, items[i].out)
			}
		}()
	}
	wg.Wait()
	up, err := tx.Prepare(`update events set cause = ? where id = ?`)
	if err != nil {
		return err
	}
	defer up.Close()
	for _, it := range items {
		if _, err := up.Exec(it.cause, it.id); err != nil {
			return err
		}
	}
	return nil
}
