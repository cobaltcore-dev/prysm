// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package rgwadmin

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Known API error reasons
const (
	ErrUserExists            errorReason = "UserAlreadyExists"
	ErrNoSuchUser            errorReason = "NoSuchUser"
	ErrInvalidAccessKey      errorReason = "InvalidAccessKey"
	ErrInvalidSecretKey      errorReason = "InvalidSecretKey"
	ErrInvalidKeyType        errorReason = "InvalidKeyType"
	ErrKeyExists             errorReason = "KeyExists"
	ErrEmailExists           errorReason = "EmailExists"
	ErrInvalidCapability     errorReason = "InvalidCapability"
	ErrSubuserExists         errorReason = "SubuserExists"
	ErrNoSuchSubUser         errorReason = "NoSuchSubUser"
	ErrInvalidAccess         errorReason = "InvalidAccess"
	ErrIndexRepairFailed     errorReason = "IndexRepairFailed"
	ErrBucketNotEmpty        errorReason = "BucketNotEmpty"
	ErrObjectRemovalFailed   errorReason = "ObjectRemovalFailed"
	ErrBucketUnlinkFailed    errorReason = "BucketUnlinkFailed"
	ErrBucketLinkFailed      errorReason = "BucketLinkFailed"
	ErrNoSuchObject          errorReason = "NoSuchObject"
	ErrIncompleteBody        errorReason = "IncompleteBody"
	ErrNoSuchCap             errorReason = "NoSuchCap"
	ErrInternalError         errorReason = "InternalError"
	ErrAccessDenied          errorReason = "AccessDenied"
	ErrNoSuchBucket          errorReason = "NoSuchBucket"
	ErrNoSuchKey             errorReason = "NoSuchKey"
	ErrInvalidArgument       errorReason = "InvalidArgument"
	ErrUnknown               errorReason = "Unknown"
	ErrSignatureDoesNotMatch errorReason = "SignatureDoesNotMatch"

	unmarshalError = "failed to unmarshal RGW response"
)

// Internal error variables
var (
	errMissingUserID          = errors.New("missing user ID")
	errMissingSubuserID       = errors.New("missing subuser ID")
	errMissingUserAccessKey   = errors.New("missing user access key")
	errMissingUserDisplayName = errors.New("missing user display name")
	errMissingUserCap         = errors.New("missing user capabilities")
	errMissingBucketID        = errors.New("missing bucket ID")
	errMissingBucket          = errors.New("missing bucket")
	errMissingUserBucket      = errors.New("missing user bucket")
	errUnsupportedKeyType     = errors.New("unsupported key type")
)

// errorReason represents an API error reason.
type errorReason string

// statusError represents an error response from RGW.
type statusError struct {
	Code      string `json:"Code,omitempty"`
	RequestID string `json:"RequestId,omitempty"`
	HostID    string `json:"HostId,omitempty"`
}

// Error implements the error interface for `errorReason`.
func (e errorReason) Error() string { return string(e) }

// Error implements the error interface for `statusError`.
func (e statusError) Error() string {
	return fmt.Sprintf("RGW Error: Code=%s, RequestID=%s, HostID=%s", e.Code, e.RequestID, e.HostID)
}

// Is allows `statusError` to be compared against known `errorReason` values.
func (e statusError) Is(target error) bool {
	if reason, ok := target.(errorReason); ok {
		return e.Code == string(reason)
	}
	return false
}

// handleStatusError parses and returns an appropriate error from the RGW response.
func handleStatusError(decodedResponse []byte) error {
	var errResp statusError
	if err := json.Unmarshal(decodedResponse, &errResp); err != nil {
		return fmt.Errorf("%s: %s (%w)", unmarshalError, string(decodedResponse), err)
	}

	if errResp.Code == "" {
		return errors.New("unknown error response from RGW")
	}

	return errResp
}
