// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package opslog

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/sapcc/go-api-declarations/cadf"
	"github.com/sapcc/go-bits/audittools"
)

// InitAuditor initializes the audit trail based on configuration.
// Returns a NullAuditor if audit sink is not configured or disabled.
func InitAuditor(ctx context.Context, cfg AuditSinkConfig, registry prometheus.Registerer) audittools.Auditor {
	if !cfg.Enabled {
		log.Info().Msg("Audit sink disabled, using NullAuditor")
		return audittools.NewNullAuditor()
	}

	if cfg.RabbitMQURL == "" || cfg.QueueName == "" {
		log.Warn().Msg("Audit sink enabled but RabbitMQ not configured, using NullAuditor")
		return audittools.NewNullAuditor()
	}

	queueSize := cfg.InternalQueueSize
	if queueSize == 0 {
		queueSize = 20 // Default from audittools
	}

	auditor, err := audittools.NewAuditor(ctx, audittools.AuditorOpts{
		ConnectionURL: cfg.RabbitMQURL,
		QueueName:     cfg.QueueName,
		Observer: audittools.Observer{
			TypeURI: "service/storage/object",
			Name:    "prysm-ops-log",
			ID:      audittools.GenerateUUID(),
		},
		Registry: registry,
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to initialize auditor, falling back to NullAuditor")
		return audittools.NewNullAuditor()
	}

	log.Info().
		Str("queue", cfg.QueueName).
		Int("buffer_size", queueSize).
		Bool("debug", cfg.Debug).
		Msg("Audit trail initialized successfully")

	return auditor
}

// ToAuditEvent converts an S3OperationLog to an audittools.Event.
func (opLog *S3OperationLog) ToAuditEvent() (audittools.Event, error) {
	// Parse timestamp
	eventTime, err := time.Parse("2006-01-02T15:04:05.999999Z", opLog.Time)
	if err != nil {
		// Fallback to current time if parsing fails
		eventTime = time.Now()
		log.Warn().Err(err).Str("time", opLog.Time).Msg("Failed to parse ops log timestamp, using current time")
	}

	// Build HTTP request for audit context
	req, err := buildHTTPRequest(opLog)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to build HTTP request for audit event")
	}

	// Parse HTTP status code
	reasonCode, err := strconv.Atoi(opLog.HTTPStatus)
	if err != nil {
		reasonCode = 500 // Default to internal server error if parsing fails
		log.Warn().Err(err).Str("http_status", opLog.HTTPStatus).Msg("Failed to parse HTTP status")
	}

	return audittools.Event{
		Time:       eventTime,
		Request:    req,
		User:       buildUserInfo(opLog),
		ReasonCode: reasonCode,
		Action:     mapOperationToAction(opLog.Operation),
		Target:     buildTarget(opLog),
	}, nil
}

// buildHTTPRequest constructs an http.Request from the ops log entry.
func buildHTTPRequest(opLog *S3OperationLog) (*http.Request, error) {
	// Parse the URI
	uri := opLog.URI
	// Extract path from "GET /path HTTP/1.1" format
	parts := strings.Fields(uri)
	path := "/"
	if len(parts) >= 2 {
		path = parts[1]
	}

	// Build URL
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	// Create request
	method := "GET" // Default
	if len(parts) >= 1 {
		method = parts[0]
	}

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", opLog.UserAgent)
	if opLog.Referrer != "" {
		req.Header.Set("Referer", opLog.Referrer)
	}

	// Set remote address
	req.RemoteAddr = opLog.RemoteAddr

	return req, nil
}

// mapOperationToAction converts RadosGW operation names to CADF actions.
func mapOperationToAction(operation string) cadf.Action {
	switch operation {
	case "list_buckets", "list_bucket":
		return "read/list"
	case "get_obj", "get_bucket_info", "head_obj", "head_bucket":
		return "read"
	case "put_obj", "create_bucket":
		return "create"
	case "delete_obj", "delete_bucket":
		return "delete"
	case "copy_obj":
		return "update/copy"
	case "post_obj":
		return "update"
	default:
		log.Debug().Str("operation", operation).Msg("Unknown operation, using 'unknown' action")
		return cadf.UnknownAction
	}
}

// buildTarget creates a CADF Target resource from the ops log entry.
func buildTarget(opLog *S3OperationLog) audittools.Target {
	// Determine target type and ID
	if opLog.Object != "" {
		// Object-level operation
		return &ObjectTarget{
			Bucket: opLog.Bucket,
			Object: opLog.Object,
		}
	} else if opLog.Bucket != "" {
		// Bucket-level operation
		return &BucketTarget{
			Bucket: opLog.Bucket,
		}
	} else {
		// Account-level operation (e.g., list_buckets)
		projectID := extractProjectID(opLog.User)
		return &AccountTarget{
			ProjectID: projectID,
		}
	}
}

// extractProjectID extracts the project ID from the user field.
// User format is typically: "projectID$projectID" or just "projectID"
func extractProjectID(user string) string {
	parts := strings.Split(user, "$")
	if len(parts) > 0 {
		return parts[0]
	}
	return user
}

// buildUserInfo creates a UserInfo from the Keystone scope.
func buildUserInfo(opLog *S3OperationLog) audittools.UserInfo {
	if opLog.KeystoneScope == nil {
		// Fallback if no Keystone scope available
		return &SimpleUserInfo{
			UserID:    opLog.User,
			ProjectID: extractProjectID(opLog.User),
		}
	}

	return &KeystoneUserInfo{
		ProjectID:   opLog.KeystoneScope.Project.ID,
		ProjectName: opLog.KeystoneScope.Project.Name,
		DomainID:    opLog.KeystoneScope.Project.Domain.ID,
		DomainName:  opLog.KeystoneScope.Project.Domain.Name,
		UserID:      opLog.KeystoneScope.User.ID,
		UserName:    opLog.KeystoneScope.User.Name,
		UserDomain:  opLog.KeystoneScope.User.Domain.Name,
		Roles:       opLog.KeystoneScope.Roles,
		AppCredID:   getAppCredID(opLog.KeystoneScope.ApplicationCredential),
		AppCredName: getAppCredName(opLog.KeystoneScope.ApplicationCredential),
	}
}

func getAppCredID(appCred *KeystoneApplicationCredential) string {
	if appCred != nil {
		return appCred.ID
	}
	return ""
}

func getAppCredName(appCred *KeystoneApplicationCredential) string {
	if appCred != nil {
		return appCred.Name
	}
	return ""
}

// === Target Implementations ===

// ObjectTarget represents an object-level operation target.
type ObjectTarget struct {
	Bucket string
	Object string
}

func (t *ObjectTarget) Render() cadf.Resource {
	return cadf.Resource{
		TypeURI: "service/storage/object",
		ID:      t.Bucket + "/" + t.Object,
		Name:    t.Object,
		Attachments: []cadf.Attachment{
			{
				Name:    "bucket",
				TypeURI: "service/storage/bucket",
				Content: t.Bucket,
			},
		},
	}
}

// BucketTarget represents a bucket-level operation target.
type BucketTarget struct {
	Bucket string
}

func (t *BucketTarget) Render() cadf.Resource {
	return cadf.Resource{
		TypeURI: "service/storage/bucket",
		ID:      t.Bucket,
		Name:    t.Bucket,
	}
}

// AccountTarget represents an account-level operation target.
type AccountTarget struct {
	ProjectID string
}

func (t *AccountTarget) Render() cadf.Resource {
	return cadf.Resource{
		TypeURI: "service/storage/account",
		ID:      t.ProjectID,
		Name:    t.ProjectID,
	}
}

// === UserInfo Implementations ===

// KeystoneUserInfo represents a Keystone-authenticated user.
type KeystoneUserInfo struct {
	ProjectID   string
	ProjectName string
	DomainID    string
	DomainName  string
	UserID      string
	UserName    string
	UserDomain  string
	Roles       []string
	AppCredID   string
	AppCredName string
}

func (u *KeystoneUserInfo) AsInitiator(host cadf.Host) cadf.Resource {
	initiator := cadf.Resource{
		TypeURI:           "service/security/account/user",
		ID:                u.UserID,
		Name:              u.UserName,
		Host:              &host,
		ProjectID:         u.ProjectID,
		ProjectName:       u.ProjectName,
		DomainID:          u.DomainID,
		DomainName:        u.DomainName,
		ProjectDomainName: u.DomainName,
		Domain:            u.UserDomain,
	}

	// Add application credential if present
	if u.AppCredID != "" {
		initiator.AppCredentialID = u.AppCredID
		initiator.Attachments = []cadf.Attachment{
			{
				Name:    "application_credential_name",
				TypeURI: "xs:string",
				Content: u.AppCredName,
			},
		}
	}

	return initiator
}

// SimpleUserInfo represents a basic user without Keystone scope.
type SimpleUserInfo struct {
	UserID    string
	ProjectID string
}

func (u *SimpleUserInfo) AsInitiator(host cadf.Host) cadf.Resource {
	return cadf.Resource{
		TypeURI:   "service/security/account/user",
		ID:        u.UserID,
		Name:      u.UserID,
		Host:      &host,
		ProjectID: u.ProjectID,
	}
}
