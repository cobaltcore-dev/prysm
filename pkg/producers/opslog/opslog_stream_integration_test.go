// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// entryJSON builds a compact ops-log entry carrying a unique trans_id.
func entryJSON(tid string) string {
	return fmt.Sprintf(
		`{"trans_id":%q,"operation":"get_obj","bucket":"b","user":"proj$proj","http_status":"200","total_time":1}`,
		tid,
	)
}

// drainCount reads the file the way the watcher loop does — open, reset offset on
// truncation, seek, stream-decode, advance — until a full pass makes no progress.
// It records every processed trans_id and counts any processed more than once.
// It is goroutine-safe (no testing.T / FailNow) so it can run concurrently with
// writers; I/O errors just yield no progress.
func drainCount(path string, startOffset int64, seen *sync.Map, dupes *int64) int64 {
	offset := startOffset
	for {
		next := readOnce(path, offset, seen, dupes)
		if next == offset {
			return offset
		}
		offset = next
	}
}

func readOnce(path string, lastOffset int64, seen *sync.Map, dupes *int64) int64 {
	f, err := os.Open(path)
	if err != nil {
		return lastOffset
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return lastOffset
	}
	if lastOffset > fi.Size() { // truncation reset — mirrors processLogEntries
		lastOffset = 0
	}
	if _, err := f.Seek(lastOffset, io.SeekStart); err != nil {
		return lastOffset
	}

	consumed := decodeOpsLogEntries(f, func(_ json.RawMessage, e *S3OperationLog) {
		if _, loaded := seen.LoadOrStore(e.TransID, struct{}{}); loaded {
			atomic.AddInt64(dupes, 1)
		}
	})
	return lastOffset + consumed
}

// TestStream_ConcurrentConcatenatedNoLoss reproduces the production conditions:
// several writers appending ops-log entries, ~half with NO newline separator
// (RGW concatenation) and some written in two chunks (partial tail windows).
// After all writes, a full drain must have processed every entry exactly once.
func TestStream_ConcurrentConcatenatedNoLoss(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	require.NoError(t, err)

	const writers = 4
	const perWriter = 750

	var writeMu sync.Mutex // O_APPEND is atomic per write; this also serializes chunked writes
	var written sync.Map
	var writtenCount int64

	// Reader runs concurrently with writers to exercise read-while-write.
	var seen sync.Map
	var dupes int64
	stop := make(chan struct{})
	var readerWG sync.WaitGroup
	readerWG.Add(1)
	go func() {
		defer readerWG.Done()
		var off int64
		for {
			select {
			case <-stop:
				return
			default:
				off = drainCount(path, off, &seen, &dupes)
				time.Sleep(time.Millisecond)
			}
		}
	}()

	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				tid := fmt.Sprintf("tx-%d-%d", id, i)
				obj := entryJSON(tid)
				sep := "\n"
				if i%2 == 0 {
					sep = "" // concatenation: no separator
				}
				writeMu.Lock()
				if i%5 == 0 { // chunked write → partial-tail window
					mid := len(obj) / 2
					_, _ = f.WriteString(obj[:mid])
					_, _ = f.WriteString(obj[mid:] + sep)
				} else {
					_, _ = f.WriteString(obj + sep)
				}
				writeMu.Unlock()
				written.Store(tid, struct{}{})
				atomic.AddInt64(&writtenCount, 1)
			}
		}(w)
	}
	wg.Wait()
	require.NoError(t, f.Close())

	// Stop the concurrent reader, then do an authoritative final drain from 0.
	close(stop)
	readerWG.Wait()

	var finalSeen sync.Map
	var finalDupes int64
	drainCount(path, 0, &finalSeen, &finalDupes)

	seenCount := 0
	finalSeen.Range(func(_, _ any) bool { seenCount++; return true })

	assert.Equal(t, int64(0), finalDupes, "no entry decoded twice in a single drain")
	assert.Equal(t, int(writtenCount), seenCount, "every written entry decoded exactly once")

	// Cross-check: every written trans_id was seen.
	missing := 0
	written.Range(func(k, _ any) bool {
		if _, ok := finalSeen.Load(k); !ok {
			missing++
		}
		return true
	})
	assert.Equal(t, 0, missing, "no written trans_id missing from the decoded set")
}

// TestStream_TruncationRecovery verifies copytruncate handling: after the file is
// truncated in place and new entries appended, the reader resets its offset and
// processes the post-truncation entries without stalling.
func TestStream_TruncationRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops.log")

	writeAll := func(flag int, tids ...string) {
		f, err := os.OpenFile(path, flag|os.O_WRONLY, 0644)
		require.NoError(t, err)
		for _, tid := range tids {
			_, _ = f.WriteString(entryJSON(tid)) // concatenated, no newline
		}
		require.NoError(t, f.Close())
	}

	var seen sync.Map
	var dupes int64

	// Phase 1: write + fully drain.
	writeAll(os.O_CREATE|os.O_APPEND, "a1", "a2", "a3")
	off := drainCount(path, 0, &seen, &dupes)

	// Phase 2: copytruncate (shrink in place) + write new entries.
	writeAll(os.O_TRUNC, "b1", "b2") // O_TRUNC empties then writes
	off = drainCount(path, off, &seen, &dupes)
	_ = off

	for _, tid := range []string{"a1", "a2", "a3", "b1", "b2"} {
		_, ok := seen.Load(tid)
		assert.Truef(t, ok, "entry %s should have been processed across truncation", tid)
	}
	assert.Equal(t, int64(0), dupes, "no duplicates across truncation")
}

// TestProcessLogEntries_RealWrapper drives the actual processLogEntries against a
// concatenated file and asserts it consumes the whole file (offset == size) with
// metrics enabled and no auditor — the integration wiring, end to end.
func TestProcessLogEntries_RealWrapper(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops.log")

	content := entryJSON("w1") + entryJSON("w2") + entryJSON("w3") // concatenated
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg := OpsLogConfig{
		LogFilePath:    path,
		MaxLogFileSize: 0, // disable size rotation so rotateLogIfNeeded is a no-op
		MetricsConfig:  MetricsConfig{TrackRequestsPerBucket: true},
	}

	newOffset, err := processLogEntries(cfg, nil, nil, NewMetrics(), nil, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), newOffset, "whole concatenated file consumed")
}
