package radosgwusage

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// encodeComponent encodes a string using Base64 URL encoding.
func EncodeComponent(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

// decodeComponent decodes a Base64 URL encoded string.
func DecodeComponent(s string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

const MissingUserPlaceholder = "none"
const MissingTenantPlaceholder = "none"

// splitUserTenant checks if the input string contains a '$'.
// If yes, it splits the string into the left part (user) and the right part (tenant).
// If no, it returns the full string as the user and "none" as the tenant.
func SplitUserTenant(s string) (user, tenant string) {
	if strings.Contains(s, "$") {
		parts := strings.SplitN(s, "$", 2)
		return parts[0], parts[1]
	}
	return s, ""
}

// BuildUserTenantKey builds a KV key in the format "<user>.<tenant>".
// If tenant is an empty string, it substitutes the MissingTenantPlaceholder.
func BuildUserTenantKey(user, tenant string) string {
	var encodedUser string
	if user == "" {
		encodedUser = MissingUserPlaceholder
	} else {
		encodedUser = EncodeComponent(user)
	}
	var encodedTenant string
	if tenant == "" {
		encodedTenant = MissingTenantPlaceholder
	} else {
		encodedTenant = EncodeComponent(tenant)
	}
	return fmt.Sprintf("%s.%s", encodedUser, encodedTenant)
}

// BuildUserTenantBucketKey builds a KV key in the format "<user>.<tenant>.<bucket>".
// If tenant is empty, MissingTenantPlaceholder is used.
func BuildUserTenantBucketKey(user, tenant, bucket string) string {
	var encodedUser string
	if user == "" {
		encodedUser = MissingUserPlaceholder
	} else {
		encodedUser = EncodeComponent(user)
	}
	var encodedTenant string
	if tenant == "" {
		encodedTenant = MissingTenantPlaceholder
	} else {
		encodedTenant = EncodeComponent(tenant)
	}
	encodedBucket := EncodeComponent(bucket)
	return fmt.Sprintf("%s.%s.%s", encodedUser, encodedTenant, encodedBucket)
}

// ParseKVKey attempts to split a key into its components. It supports keys separated by
// either a period (".") or a dollar sign ("$"). It returns user, tenant, and bucket (if available).
func ParseKVKey(key string) (user, tenant, bucket string, err error) {
	var parts []string
	if strings.Contains(key, "$") {
		parts = strings.Split(key, "$")
	} else {
		parts = strings.Split(key, ".")
	}

	switch len(parts) {
	case 1:
		// Only user provided.
		user, err = DecodeComponent(parts[0])
		return
	case 2:
		// Format: <user>.<tenant>
		user, err = DecodeComponent(parts[0])
		if err != nil {
			return
		}
		// If tenant equals our placeholder, we return an empty tenant.
		decodedTenant := ""
		decodedTenant, err = DecodeComponent(parts[1])
		if err != nil {
			// Alternatively, if the second part is exactly the placeholder string,
			// we can simply return empty tenant without error.
			if parts[1] == MissingTenantPlaceholder {
				tenant = ""
			} else {
				return "", "", "", err
			}
		} else {
			tenant = decodedTenant
		}
		return
	case 3:
		// Format: <user>.<tenant>.<bucket>
		user, err = DecodeComponent(parts[0])
		if err != nil {
			return
		}
		decodedTenant := ""
		decodedTenant, err = DecodeComponent(parts[1])
		if err != nil {
			if parts[1] == MissingTenantPlaceholder {
				tenant = ""
			} else {
				return "", "", "", err
			}
		} else {
			tenant = decodedTenant
		}
		bucket, err = DecodeComponent(parts[2])
		return
	default:
		err = fmt.Errorf("invalid key format: expected 1 to 3 parts, got %d", len(parts))
		return
	}
}
