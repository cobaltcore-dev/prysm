// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"errors"
	"strings"
	"testing"
	"time"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

// TestRateLimitedLogger_Suppression verifies the storm-prevention mechanism:
// warnings within the interval are counted (suppressed) rather than emitted, and
// the first warning after the interval elapses emits and resets the counter.
func TestRateLimitedLogger_Suppression(t *testing.T) {
	l := &rateLimitedLogger{interval: 40 * time.Millisecond}

	// First call emits (last is zero) and leaves suppressed at 0.
	l.warn(errors.New("boom"), []byte("sample"))
	assert.Equal(t, 0, l.suppressed, "first warning emits, nothing suppressed")

	// Rapid follow-ups within the interval are suppressed, not emitted.
	l.warn(errors.New("boom"), nil)
	l.warn(errors.New("boom"), nil)
	assert.Equal(t, 2, l.suppressed, "warnings within the interval are counted")

	// After the interval elapses, the next warning emits and resets the counter.
	time.Sleep(80 * time.Millisecond)
	l.warn(errors.New("boom"), nil)
	assert.Equal(t, 0, l.suppressed, "warning after the interval emits and resets")
}

// collect runs decodeOpsLogEntries over input and returns the operations parsed
// (in order) plus the number of bytes reported as consumed.
func collect(input string) ([]string, int64) {
	var ops []string
	consumed := decodeOpsLogEntries(strings.NewReader(input), func(_ json.RawMessage, e *S3OperationLog) {
		ops = append(ops, e.Operation)
	})
	return ops, consumed
}

func TestDecodeOpsLogEntries(t *testing.T) {
	objA := `{"operation":"a"}`
	objB := `{"operation":"b"}`
	objC := `{"operation":"c"}`

	tests := []struct {
		name         string
		input        string
		wantOps      []string
		wantConsumed int64
	}{
		{
			name:         "single newline-terminated",
			input:        objA + "\n",
			wantOps:      []string{"a"},
			wantConsumed: int64(len(objA)),
		},
		{
			// The actual production bug: RGW concatenates entries with no
			// separator. The old line parser dropped both and firehosed stdout.
			name:         "concatenated without separator",
			input:        objA + objB + objC,
			wantOps:      []string{"a", "b", "c"},
			wantConsumed: int64(len(objA + objB + objC)),
		},
		{
			name:         "newline separated",
			input:        objA + "\n" + objB + "\n",
			wantOps:      []string{"a", "b"},
			wantConsumed: int64(len(objA + "\n" + objB)),
		},
		{
			name:         "whitespace between",
			input:        " " + objA + "  \n " + objB + " ",
			wantOps:      []string{"a", "b"},
			wantConsumed: int64(len(" " + objA + "  \n " + objB)),
		},
		{
			// Partial object mid-write at the tail must NOT be consumed, so it is
			// re-read whole on the next pass (not lost, not duplicated).
			name:         "partial trailing object",
			input:        objA + objB + `{"operation":"c`,
			wantOps:      []string{"a", "b"},
			wantConsumed: int64(len(objA + objB)),
		},
		{
			name:         "empty input",
			input:        "",
			wantOps:      nil,
			wantConsumed: 0,
		},
		{
			// A valid entry followed by garbage: the good entry is processed, the
			// stream stops at the garbage (rate-limited log, no storm), and the
			// offset stays at the last good boundary.
			name:         "valid then garbage",
			input:        objA + "garbage",
			wantOps:      []string{"a"},
			wantConsumed: int64(len(objA)),
		},
		{
			name:         "leading garbage yields nothing",
			input:        "garbage",
			wantOps:      nil,
			wantConsumed: 0,
		},
		{
			// A torn record in the MIDDLE must not wedge the stream: skip past the
			// next newline and recover the following entries (no permanent stall).
			name:         "garbage between valid entries recovers",
			input:        objA + "not-json\n" + objB,
			wantOps:      []string{"a", "b"},
			wantConsumed: int64(len(objA + "not-json\n" + objB)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, consumed := collect(tt.input)
			assert.Equal(t, tt.wantOps, ops, "parsed operations")
			assert.Equal(t, tt.wantConsumed, consumed, "consumed offset")
		})
	}
}

// errReader yields data, then a genuine (non-EOF) read error — an unreadable
// file surfaced through the decoder.
type errReader struct {
	data []byte
	pos  int
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos < len(r.data) {
		n := copy(p, r.data[r.pos:])
		r.pos += n
		return n, nil
	}
	return 0, r.err
}

// TestDecodeOpsLogEntries_GenuineReaderError verifies that a genuine reader error
// after a valid entry does not stall or panic: the good entry is processed and
// the function returns the last complete boundary (the error is logged, not
// silently swallowed — see the resync path).
func TestDecodeOpsLogEntries_GenuineReaderError(t *testing.T) {
	objA := `{"operation":"a"}`
	r := &errReader{data: []byte(objA), err: errors.New("disk gone")}

	var ops []string
	consumed := decodeOpsLogEntries(r, func(_ json.RawMessage, e *S3OperationLog) {
		ops = append(ops, e.Operation)
	})
	assert.Equal(t, []string{"a"}, ops, "entry before the error is processed")
	assert.Equal(t, int64(len(objA)), consumed, "offset stays at the last complete object")
}

// TestCapSampler verifies the diagnostic sampler is bounded regardless of input
// size, so a huge skipped region can never be buffered whole.
func TestCapSampler(t *testing.T) {
	var s capSampler
	n, err := s.Write(make([]byte, 1000))
	assert.NoError(t, err)
	assert.Equal(t, 1000, n, "reports a full write so TeeReader never errors")
	assert.Len(t, s.bytes(), opsLogSampleCap, "sample is capped")
}

// TestDecodeOpsLogEntries_ResumeAcrossReads simulates the real caller: a partial
// tail on the first read is completed on the second, and the offset advances
// correctly with no duplicate or lost entries.
func TestDecodeOpsLogEntries_ResumeAcrossReads(t *testing.T) {
	objA := `{"operation":"a"}`
	objB := `{"operation":"b"}`

	// First pass sees A plus a partial B.
	full := objA + objB
	firstChunk := objA + objB[:len(objB)-3] // B truncated mid-write

	ops1, consumed1 := collect(firstChunk)
	assert.Equal(t, []string{"a"}, ops1)
	assert.Equal(t, int64(len(objA)), consumed1, "partial B not consumed")

	// Second pass resumes from the reported offset and now sees the full B.
	ops2, consumed2 := collect(full[consumed1:])
	assert.Equal(t, []string{"b"}, ops2)
	assert.Equal(t, int64(len(objB)), consumed2)
}
