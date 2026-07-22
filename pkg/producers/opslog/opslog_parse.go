// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	json "github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
)

// decodeOpsLogEntries decodes consecutive JSON ops-log objects from r, invoking
// handle(raw, entry) for each complete object in order. It returns the number of
// bytes consumed up to the end of the last complete object.
//
// RGW writes the ops log as a stream of JSON objects that are NOT reliably
// newline-separated — consecutive entries are frequently concatenated as
// `{...}{...}` with no delimiter. A line-based reader (ReadString('\n') +
// Unmarshal) mis-reads such a run as one malformed line, which both drops those
// entries (silent metric/audit loss) and, historically, logged the entire raw
// line per failure — a stdout firehose. json.Decoder instead yields one complete
// value at a time regardless of the whitespace between them.
//
// The returned byte count is the offset just past the last COMPLETE object, so a
// partial object still being written at the tail of the file is not consumed and
// is re-read (not lost, not duplicated) on the next pass.
func decodeOpsLogEntries(r io.Reader, handle func(raw json.RawMessage, entry *S3OperationLog)) int64 {
	// base accumulates bytes consumed by decoders discarded during resync; the
	// live decoder's InputOffset() is relative to base. lastGood is the absolute
	// offset just past the last COMPLETE object — the safe point to resume from.
	var base, lastGood int64
	dec := json.NewDecoder(r)
	var entry S3OperationLog

	for {
		var raw json.RawMessage
		err := dec.Decode(&raw)
		if err == nil {
			lastGood = base + dec.InputOffset()
			entry = S3OperationLog{}
			if uerr := json.Unmarshal(raw, &entry); uerr != nil {
				// Valid JSON but not our shape — skip this one entry (the decoder
				// has already advanced past it) rather than dropping the stream.
				opsLogParseErrLogger.warn(uerr, raw)
				continue
			}
			handle(raw, &entry)
			continue
		}

		// Clean end: only trailing whitespace remained. Return the last complete
		// boundary (not the whitespace-inclusive InputOffset).
		if err == io.EOF {
			return lastGood
		}

		// Any other error means the bytes after lastGood don't (yet) form a
		// complete value. Try to skip past the next newline, capturing a bounded
		// sample of the offending bytes for diagnostics:
		//   - newline ahead        → a torn record mid-stream; log (rate-limited,
		//     length-capped, with sample) and resume so it can't wedge the stream.
		//   - clean EOF, no newline → a partial object still being written at the
		//     tail; wait silently, don't advance past the last complete object.
		//   - genuine reader error  → surface it (never stall silently), then wait.
		errOffset := dec.InputOffset()
		mr := io.MultiReader(dec.Buffered(), r)
		var sample capSampler
		skipped, skipErr := skipPastNewline(io.TeeReader(mr, &sample))
		if skipped < 0 {
			if skipErr != nil && !errors.Is(skipErr, io.EOF) {
				log.Warn().Err(skipErr).Msg("ops-log read error during resync; will retry")
			}
			return lastGood
		}
		opsLogParseErrLogger.warn(err, sample.bytes())
		base += errOffset + skipped
		lastGood = base
		dec = json.NewDecoder(mr)
	}
}

// skipPastNewline consumes bytes from r up to and including the next '\n'. It
// returns the number of bytes consumed and nil on success, or -1 and the
// terminating error otherwise — io.EOF for a clean partial tail, any other error
// for a genuine reader failure the caller must not swallow.
func skipPastNewline(r io.Reader) (int64, error) {
	var n int64
	buf := make([]byte, 1)
	for {
		m, err := r.Read(buf)
		if m > 0 {
			n++
			if buf[0] == '\n' {
				return n, nil
			}
		}
		if err != nil {
			return -1, err
		}
	}
}

// capSampler collects up to opsLogSampleCap bytes for a diagnostic sample and
// discards the rest. It always reports a full write so io.TeeReader never errors,
// keeping the skipped-region peek bounded regardless of how large that region is.
type capSampler struct{ buf []byte }

const opsLogSampleCap = 256

func (s *capSampler) Write(p []byte) (int, error) {
	if room := opsLogSampleCap - len(s.buf); room > 0 {
		if room > len(p) {
			room = len(p)
		}
		s.buf = append(s.buf, p[:room]...)
	}
	return len(p), nil
}

func (s *capSampler) bytes() []byte { return s.buf }

// printOpsLogLine writes a single ops-log entry to stdout. Each object is
// emitted on its own line (compact), or indented when pretty-printing is on.
func printOpsLogLine(raw json.RawMessage, pretty bool) {
	if pretty {
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err == nil {
			fmt.Println(buf.String())
			return
		}
	}
	fmt.Println(string(raw))
}

// opsLogParseErrLogger is the shared rate limiter for ops-log parse warnings.
// A malformed ops-log stream must never be able to reproduce the stdout log
// storm, so parse errors are emitted at most once per interval with a suppressed
// count and a length-capped sample.
var opsLogParseErrLogger = &rateLimitedLogger{interval: 10 * time.Second}

type rateLimitedLogger struct {
	mu         sync.Mutex
	interval   time.Duration
	last       time.Time
	suppressed int
}

func (l *rateLimitedLogger) warn(err error, raw []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if !l.last.IsZero() && now.Sub(l.last) < l.interval {
		l.suppressed++
		return
	}

	ev := log.Warn().Err(err)
	if l.suppressed > 0 {
		ev = ev.Int("suppressed", l.suppressed)
	}
	ev.Str("sample", sampleBytes(raw, 256)).Msg("Skipping malformed ops-log entry")

	l.last = now
	l.suppressed = 0
}

// sampleBytes returns at most maxLen bytes of b as a string, marking truncation.
func sampleBytes(b []byte, maxLen int) string {
	if len(b) > maxLen {
		return string(b[:maxLen]) + "…(truncated)"
	}
	return string(b)
}
