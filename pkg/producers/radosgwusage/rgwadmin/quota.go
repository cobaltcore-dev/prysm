// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package rgwadmin

type QuotaSpec struct {
	UID        string `json:"user_id" url:"uid"`
	Bucket     string `json:"bucket" url:"bucket"`
	QuotaType  string `url:"quota-type"`
	Enabled    *bool  `json:"enabled" url:"enabled"`
	CheckOnRaw bool   `json:"check_on_raw"`
	MaxSize    *int64 `json:"max_size" url:"max-size"`
	MaxSizeKb  *int   `json:"max_size_kb" url:"max-size-kb"`
	MaxObjects *int64 `json:"max_objects" url:"max-objects"`
}
