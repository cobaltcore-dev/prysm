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

type UsageEntryCategory struct {
	Category      string `json:"category"`
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	Ops           uint64 `json:"ops"`
	SuccessfulOps uint64 `json:"successful_ops"`
}

type UsageEntryBucket struct {
	Bucket     string               `json:"bucket"`
	Time       string               `json:"time"`
	Epoch      uint64               `json:"epoch"`
	Owner      string               `json:"owner"`
	Categories []UsageEntryCategory `json:"categories"`
}

type UsageEntry struct {
	User    string             `json:"user"`
	Buckets []UsageEntryBucket `json:"buckets"`
}

type UsageSummaryTotal struct {
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
	Ops           uint64 `json:"ops"`
	SuccessfulOps uint64 `json:"successful_ops"`
}

type UsageSummary struct {
	User       string               `json:"user"`
	Categories []UsageEntryCategory `json:"categories"`
	Total      UsageSummaryTotal    `json:"total"`
}

// Usage struct
type Usage struct {
	Entries     []UsageEntry   `json:"entries"`
	Summary     []UsageSummary `json:"summary"`
	UserID      string         `url:"uid"`
	Start       string         `url:"start"` //Example:	2012-09-25 16:00:00
	End         string         `url:"end"`
	ShowEntries *bool          `url:"show-entries"`
	ShowSummary *bool          `url:"show-summary"`
	RemoveAll   *bool          `url:"remove-all"` //true
}

type KVUsage struct {
	Entries []UsageEntry   `json:"entries"`
	Summary []UsageSummary `json:"summary"`
}

// GetUsage retrieves bandwidth usage information from the object store.
func (api *API) GetUsage(ctx context.Context, usage Usage) (Usage, error) {
	// Define valid query parameters
	validParams := []string{"uid", "start", "end", "show-entries", "show-summary"}

	// Build the request parameters
	params := valueToURLParams(usage, validParams)

	// Make the API call
	body, err := api.call(ctx, http.MethodGet, "/usage", params, nil)
	if err != nil {
		return Usage{}, err
	}

	// Decode the response
	var usageResponse Usage
	if err := json.Unmarshal(body, &usageResponse); err != nil {
		return Usage{}, fmt.Errorf("%s: %w. Response: %s", unmarshalError, err, string(body))
	}

	return usageResponse, nil
}
