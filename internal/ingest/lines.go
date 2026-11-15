package ingest

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// eachLine streams the lines of a .jsonl or .jsonl.zst file. fn gets the 1-based line number and the line
// without its newline. The slice is only valid during the call.
func eachLine(path string, dec *zstd.Decoder, fn func(ln int, line []byte)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if strings.HasSuffix(path, ".zst") {
		if dec == nil {
			d, err := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxWindow(1<<30))
			if err != nil {
				return err
			}
			defer d.Close()
			dec = d
		}
		if err := dec.Reset(f); err != nil {
			return err
		}
		r = dec
	}
	br := bufio.NewReaderSize(r, 1<<20)
	var buf []byte
	ln := 0
	for {
		chunk, err := br.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			buf = append(buf, chunk...)
			continue
		}
		var line []byte
		if len(buf) > 0 {
			buf = append(buf, chunk...)
			line = buf
		} else {
			line = chunk
		}
		if len(line) > 0 {
			ln++
			if line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			fn(ln, line)
		}
		buf = buf[:0]
		if err != nil {
			if err == io.EOF {
				return nil
			}
			// A truncated zstd stream (file still being written) ends the file like the zstd CLI does.
			return nil
		}
	}
}
