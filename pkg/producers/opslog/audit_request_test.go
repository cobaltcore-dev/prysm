// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sapcc/go-api-declarations/cadf"
	"github.com/sapcc/go-bits/audittools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildHTTPRequestNeverNil checks that request lines url.Parse or
// http.NewRequest reject still give a request, so audittools can render the
// event instead of dereferencing nil.
func TestBuildHTTPRequestNeverNil(t *testing.T) {
	testCases := []struct {
		uri, path string
	}{
		{"GET /b/k?acl HTTP/1.1", "/b/k?acl"},
		{"GET /b/%zz HTTP/1.1", "/b/%zz"},
		{"PUT /swift/v1/AUTH_p/c/discount% HTTP/1.1", "/swift/v1/AUTH_p/c/discount%"},
		{"DELETE /b/x%2 HTTP/1.1", "/b/x%2"},
		{"G@T /b HTTP/1.1", "/b"}, // not a valid method token
		{"GET", "/"},
		{"", "/"},
	}
	for _, tc := range testCases {
		opLog := &S3OperationLog{
			Operation: "put_obj", Bucket: "b", Object: "k", URI: tc.uri,
			User: "u$proj", HTTPStatus: "201", Time: "2026-11-06T10:00:00.000000Z",
			UserAgent: "curl/8.7.1", RemoteAddr: "10.0.0.1",
		}
		event, err := opLog.ToAuditEvent("")
		require.NoError(t, err)
		require.NotNil(t, event.Request, tc.uri)

		var rendered cadf.Event
		require.NotPanics(t, func() { rendered = event.ToCADF(cadf.Resource{}) }, tc.uri)
		assert.Equal(t, tc.path, rendered.RequestPath, tc.uri)
		assert.Equal(t, "curl/8.7.1", rendered.Initiator.Host.Agent, tc.uri)
		assert.Equal(t, "10.0.0.1", rendered.Initiator.Host.Address, tc.uri)
	}
}

// TestProcessLogEntries_MalformedURI drives an authenticated upload of an
// object named "discount%" (as curl sends it, unencoded) through
// processLogEntries with go-bits' MockAuditor, which renders CADF in Record
// like the real auditor. The producer must not panic and the events around
// it must be published.
func TestProcessLogEntries_MalformedURI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops.log")
	content := `{"trans_id":"a1","operation":"put_obj","bucket":"c","object":"report.txt","uri":"PUT /swift/v1/AUTH_p/c/report.txt HTTP/1.1","user":"u$proj","http_status":"201","time":"2026-11-06T10:00:00.000000Z"}` +
		`{"trans_id":"a2","operation":"put_obj","bucket":"c","object":"discount%","uri":"PUT /swift/v1/AUTH_p/c/discount% HTTP/1.1","user":"u$proj","http_status":"201","time":"2026-11-06T10:00:01.000000Z"}` +
		`{"trans_id":"a3","operation":"delete_obj","bucket":"c","object":"report.txt","uri":"DELETE /swift/v1/AUTH_p/c/report.txt HTTP/1.1","user":"u$proj","http_status":"204","time":"2026-11-06T10:00:02.000000Z"}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg := OpsLogConfig{
		LogFilePath: path,
		AuditSink:   AuditSinkConfig{Enabled: true, IncludeReads: true},
	}
	auditor := audittools.NewMockAuditor()

	require.NotPanics(t, func() {
		_, err := processLogEntries(cfg, nil, nil, NewMetrics(), auditor, 0)
		require.NoError(t, err)
	})

	events := auditor.RecordedEvents()
	require.Len(t, events, 3, "the upload and the delete around the bad line are published")
	assert.Equal(t, "/swift/v1/AUTH_p/c/discount%", events[1].RequestPath)
	assert.Equal(t, "/swift/v1/AUTH_p/c/report.txt", events[2].RequestPath)
}
