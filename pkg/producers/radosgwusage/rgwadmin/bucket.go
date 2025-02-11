// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0
package rgwadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ExplicitPlacement struct {
	DataPool      string `json:"data_pool"`
	DataExtraPool string `json:"data_extra_pool"`
	IndexPool     string `json:"index_pool"`
}

type BucketUsageRgwMain struct {
	Size           *uint64 `json:"size"`
	SizeActual     *uint64 `json:"size_actual"`
	SizeUtilized   *uint64 `json:"size_utilized"`
	SizeKb         *uint64 `json:"size_kb"`
	SizeKbActual   *uint64 `json:"size_kb_actual"`
	SizeKbUtilized *uint64 `json:"size_kb_utilized"`
	NumObjects     *uint64 `json:"num_objects"`
}

type BucketUsageRgwMultimeta struct {
	Size           *uint64 `json:"size"`
	SizeActual     *uint64 `json:"size_actual"`
	SizeUtilized   *uint64 `json:"size_utilized"`
	SizeKb         *uint64 `json:"size_kb"`
	SizeKbActual   *uint64 `json:"size_kb_actual"`
	SizeKbUtilized *uint64 `json:"size_kb_utilized"`
	NumObjects     *uint64 `json:"num_objects"`
}

type BucketUsage struct {
	RgwMain      BucketUsageRgwMain      `json:"rgw.main"`
	RgwMultimeta BucketUsageRgwMultimeta `json:"rgw.multimeta"`
}

type Bucket struct {
	Bucket            string            `json:"bucket" url:"bucket"`
	NumShards         *uint64           `json:"num_shards"`
	Tenant            string            `json:"tenant"`
	Zonegroup         string            `json:"zonegroup"`
	PlacementRule     string            `json:"placement_rule"`
	ExplicitPlacement ExplicitPlacement `json:"explicit_placement"`
	ID                string            `json:"id"`
	Marker            string            `json:"marker"`
	IndexType         string            `json:"index_type"`
	Owner             string            `json:"owner"`
	Ver               string            `json:"ver"`
	MasterVer         string            `json:"master_ver"`
	Mtime             string            `json:"mtime"`
	CreationTime      *time.Time        `json:"creation_time"`
	MaxMarker         string            `json:"max_marker"`
	Usage             BucketUsage       `json:"usage"`
	BucketQuota       QuotaSpec         `json:"bucket_quota"`
	Policy            *bool             `url:"policy"`
	PurgeObject       *bool             `url:"purge-objects"`
}

// ListBuckets retrieves a list of all buckets in the object store.
func (api *API) ListBuckets(ctx context.Context) ([]string, error) {
	body, err := api.call(ctx, http.MethodGet, "/bucket", nil, nil)
	if err != nil {
		return nil, err
	}

	var buckets []string
	if err := json.Unmarshal(body, &buckets); err != nil {
		return nil, fmt.Errorf("%s: %w. Response: %s", unmarshalError, err, string(body))
	}

	return buckets, nil
}

// GetBucketInfo retrieves information about a specific bucket.
func (api *API) GetBucketInfo(ctx context.Context, bucket Bucket) (Bucket, error) {
	// Define valid query parameters
	validParams := []string{"bucket", "uid", "stats"}

	// Build request parameters
	params := valueToURLParams(bucket, validParams)

	// Make API request
	body, err := api.call(ctx, http.MethodGet, "/bucket", params, nil)
	if err != nil {
		return Bucket{}, err
	}

	// Decode response
	var bucketInfo Bucket
	if err := json.Unmarshal(body, &bucketInfo); err != nil {
		return Bucket{}, fmt.Errorf("%s: %w. Response: %s", unmarshalError, err, string(body))
	}

	return bucketInfo, nil
}
