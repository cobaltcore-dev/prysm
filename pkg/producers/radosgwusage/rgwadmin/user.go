// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package rgwadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type User struct {
	ID                  string         `json:"user_id" url:"uid"`
	DisplayName         string         `json:"display_name" url:"display-name"`
	Email               string         `json:"email" url:"email"`
	Suspended           *int           `json:"suspended" url:"suspended"`
	MaxBuckets          *int           `json:"max_buckets" url:"max-buckets"`
	Subusers            []SubuserSpec  `json:"subusers" url:"-"`
	Keys                []UserKeySpec  `json:"keys"`
	SwiftKeys           []SwiftKeySpec `json:"swift_keys" url:"-"`
	Caps                []UserCapSpec  `json:"caps"`
	OpMask              string         `json:"op_mask"`
	DefaultPlacement    string         `json:"default_placement"`
	DefaultStorageClass string         `json:"default_storage_class"`
	PlacementTags       []interface{}  `json:"placement_tags"`
	BucketQuota         QuotaSpec      `json:"bucket_quota"`
	UserQuota           QuotaSpec      `json:"user_quota"`
	TempURLKeys         []interface{}  `json:"temp_url_keys"`
	Type                string         `json:"type"`
	MfaIds              []interface{}  `json:"mfa_ids"` //revive:disable-line:var-naming old-yet-exported public api
	KeyType             string         `url:"key-type"`
	Tenant              string         `url:"tenant"`
	GenerateKey         *bool          `url:"generate-key"`
	PurgeData           *int           `url:"purge-data"`
	GenerateStat        *bool          `url:"stats"`
	Stat                UserStat       `json:"stats"`
	UserCaps            string         `url:"user-caps"`
}

func (u *User) GetKVUser() *KVUser {
	return &KVUser{
		ID:                  u.ID,
		DisplayName:         u.DisplayName,
		Email:               u.Email,
		Suspended:           u.Suspended,
		MaxBuckets:          u.MaxBuckets,
		Caps:                u.Caps,
		OpMask:              u.OpMask,
		DefaultPlacement:    u.DefaultPlacement,
		DefaultStorageClass: u.DefaultStorageClass,
		PlacementTags:       u.PlacementTags,
		BucketQuota:         u.BucketQuota,
		UserQuota:           u.UserQuota,
		Type:                u.Type,
		Tenant:              u.Tenant,
		Stats:               u.Stat,
	}
}

type KVUser struct {
	ID                  string        `json:"user_id"`
	DisplayName         string        `json:"display_name"`
	Email               string        `json:"email"`
	Suspended           *int          `json:"suspended"`
	MaxBuckets          *int          `json:"max_buckets"`
	Caps                []UserCapSpec `json:"caps"`
	OpMask              string        `json:"op_mask"`
	DefaultPlacement    string        `json:"default_placement"`
	DefaultStorageClass string        `json:"default_storage_class"`
	PlacementTags       []interface{} `json:"placement_tags"`
	BucketQuota         QuotaSpec     `json:"bucket_quota"`
	UserQuota           QuotaSpec     `json:"user_quota"`
	TempURLKeys         []interface{} `json:"temp_url_keys"`
	Type                string        `json:"type"`
	Tenant              string        `json:"tenant"`
	Stats               UserStat      `json:"stats"`
}

func (user *KVUser) GetUserIdentification() string {
	if len(user.Tenant) > 0 {
		return fmt.Sprintf("%s$%s", user.ID, user.Tenant)
	}
	return user.ID
}

// SubuserSpec represents a subusers of a ceph-rgw user
type SubuserSpec struct {
	Name   string        `json:"id" url:"subuser"`
	Access SubuserAccess `json:"permissions" url:"access"`

	// these are always nil in answers, they are only relevant in requests
	GenerateKey *bool   `json:"-" url:"generate-key"`
	SecretKey   *string `json:"-" url:"secret-key"`
	Secret      *string `json:"-" url:"secret"`
	PurgeKeys   *bool   `json:"-" url:"purge-keys"`
	KeyType     *string `json:"-" url:"key-type"`
}

// SubuserAccess represents an access level for a subuser
type SubuserAccess string

// The possible values of SubuserAccess
//
// There are two sets of constants as the API parameters and the
// values returned by the API do not match.  The SubuserAccess* values
// must be used when setting access level, the SubuserAccessReply*
// values are the ones that may be returned. This is a design problem
// of the upstream API. We do not feel confident to do the mapping in
// the library.
const (
	SubuserAccessNone      SubuserAccess = ""
	SubuserAccessRead      SubuserAccess = "read"
	SubuserAccessWrite     SubuserAccess = "write"
	SubuserAccessReadWrite SubuserAccess = "readwrite"
	SubuserAccessFull      SubuserAccess = "full"

	SubuserAccessReplyNone      SubuserAccess = "<none>"
	SubuserAccessReplyRead      SubuserAccess = "read"
	SubuserAccessReplyWrite     SubuserAccess = "write"
	SubuserAccessReplyReadWrite SubuserAccess = "read-write"
	SubuserAccessReplyFull      SubuserAccess = "full-control"
)

// SwiftKeySpec represents the secret key associated to a subuser
type SwiftKeySpec struct {
	User      string `json:"user"`
	SecretKey string `json:"secret_key"`
}

// UserCapSpec represents a user capability which gives access to certain ressources
type UserCapSpec struct {
	Type string `json:"type"`
	Perm string `json:"perm"`
}

// UserKeySpec is the user credential configuration
type UserKeySpec struct {
	User      string `json:"user"`
	AccessKey string `json:"access_key" url:"access-key"`
	SecretKey string `json:"secret_key" url:"secret-key"`
	// Request fields
	UID         string `url:"uid"`     // The user ID to receive the new key
	SubUser     string `url:"subuser"` // The subuser ID to receive the new key
	KeyType     string `url:"key-type"`
	GenerateKey *bool  `url:"generate-key"` // Generate a new key pair and add to the existing keyring
}

// UserStat contains information about storage consumption by the ceph user
type UserStat struct {
	Size        *uint64 `json:"size"`
	SizeRounded *uint64 `json:"size_rounded"`
	NumObjects  *uint64 `json:"num_objects"`
}

// GetUsers retrieves a list of all user IDs in the object store.
func (api *API) GetUsers(ctx context.Context) ([]string, error) {
	body, err := api.call(ctx, http.MethodGet, "/metadata/user", nil, nil)
	if err != nil {
		return nil, err
	}

	var users []string
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("%s: %w. Response: %s", unmarshalError, err, string(body))
	}

	return users, nil
}

// GetUser retrieves detailed information about a specific user.
func (api *API) GetUser(ctx context.Context, user User) (User, error) {
	// Validate user input
	if err := validateUserRequest(user); err != nil {
		return User{}, err
	}

	// Define valid query parameters
	validParams := []string{"uid", "access-key", "stats"}

	// Build request parameters
	params := valueToURLParams(user, validParams)

	// Make API request
	body, err := api.call(ctx, http.MethodGet, "/user", params, nil)
	if err != nil {
		return User{}, err
	}

	// Decode response
	var userInfo User
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return User{}, fmt.Errorf("%s: %w. Response: %s", unmarshalError, err, string(body))
	}

	return userInfo, nil
}

// Get Reduced data
func (api *API) GetKVUser(ctx context.Context, user User) (KVUser, error) {
	// Validate user input
	if err := validateUserRequest(user); err != nil {
		return KVUser{}, err
	}

	// Define valid query parameters
	validParams := []string{"uid", "access-key", "stats"}

	// Build request parameters
	params := valueToURLParams(user, validParams)

	// Make API request
	body, err := api.call(ctx, http.MethodGet, "/user", params, nil)
	if err != nil {
		return KVUser{}, err
	}

	// Decode response
	var userInfo KVUser
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return KVUser{}, fmt.Errorf("%s: %w. Response: %s", unmarshalError, err, string(body))
	}

	return userInfo, nil
}

// validateUserRequest ensures that the User struct has the required fields.
func validateUserRequest(user User) error {
	if user.ID == "" && len(user.Keys) == 0 {
		return errMissingUserID
	}

	for _, key := range user.Keys {
		if key.AccessKey == "" {
			return errMissingUserAccessKey
		}
	}

	return nil
}
