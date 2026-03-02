// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage/rgwadmin"
	"github.com/nats-io/nats.go"
)

func TestProcessBucketMetrics_ContinuesWhenUsageKeyMissing(t *testing.T) {
	key := BuildUserTenantBucketKey("user-a", "tenant-a", "bucket-a")

	numObjects := uint64(7)
	sizeActual := uint64(1024)
	quotaEnabled := true
	quotaMaxSize := int64(2048)
	quotaMaxObjects := int64(10)

	bucket := rgwadmin.Bucket{
		Bucket:    "bucket-a",
		Owner:     "user-a$tenant-a",
		Tenant:    "tenant-a",
		Zonegroup: "zone-a",
		Mtime:     "2026-02-01T00:00:00Z",
		Usage: rgwadmin.BucketUsage{
			RgwMain: rgwadmin.BucketUsageRgwMain{
				NumObjects: &numObjects,
				SizeActual: &sizeActual,
			},
		},
		BucketQuota: rgwadmin.QuotaSpec{
			Enabled:    &quotaEnabled,
			MaxSize:    &quotaMaxSize,
			MaxObjects: &quotaMaxObjects,
		},
	}

	bucketJSON, err := json.Marshal(bucket)
	if err != nil {
		t.Fatalf("marshal bucket: %v", err)
	}

	bucketData := newTestKV("bucket_data", map[string][]byte{
		key: bucketJSON,
	})
	// Intentionally empty: simulate `nats: key not found` for usage data.
	userUsageData := newTestKV("user_usage_data", nil)
	bucketMetrics := newTestKV("bucket_metrics", nil)

	processBucketMetrics(key, bucketData, userUsageData, bucketMetrics)

	entry, err := bucketMetrics.Get(key)
	if err != nil {
		t.Fatalf("expected bucket metric to be stored, got error: %v", err)
	}

	var got UserBucketMetrics
	if err := json.Unmarshal(entry.Value(), &got); err != nil {
		t.Fatalf("unmarshal stored metric: %v", err)
	}

	if got.BucketID != "bucket-a" {
		t.Fatalf("unexpected bucket id: %q", got.BucketID)
	}
	if got.User != "user-a" {
		t.Fatalf("unexpected user: %q", got.User)
	}
	if got.Tenant != "tenant-a" {
		t.Fatalf("unexpected tenant: %q", got.Tenant)
	}
	if got.Zonegroup != "zone-a" {
		t.Fatalf("unexpected zonegroup: %q", got.Zonegroup)
	}
	if got.ObjectCount != numObjects {
		t.Fatalf("unexpected object count: %d", got.ObjectCount)
	}
	if got.BucketSize != sizeActual {
		t.Fatalf("unexpected bucket size: %d", got.BucketSize)
	}
	if !got.QuotaEnabled {
		t.Fatalf("expected quota enabled")
	}
	if got.QuotaMaxSize == nil || *got.QuotaMaxSize != quotaMaxSize {
		t.Fatalf("unexpected quota max size: %+v", got.QuotaMaxSize)
	}
	if got.QuotaMaxObjects == nil || *got.QuotaMaxObjects != quotaMaxObjects {
		t.Fatalf("unexpected quota max objects: %+v", got.QuotaMaxObjects)
	}
}

type testKV struct {
	bucket string
	data   map[string][]byte
}

func newTestKV(bucket string, seed map[string][]byte) *testKV {
	cloned := make(map[string][]byte, len(seed))
	for k, v := range seed {
		vv := make([]byte, len(v))
		copy(vv, v)
		cloned[k] = vv
	}
	return &testKV{
		bucket: bucket,
		data:   cloned,
	}
}

func (kv *testKV) Get(key string) (nats.KeyValueEntry, error) {
	v, ok := kv.data[key]
	if !ok {
		return nil, nats.ErrKeyNotFound
	}
	vv := make([]byte, len(v))
	copy(vv, v)
	return &testKVEntry{bucket: kv.bucket, key: key, value: vv}, nil
}

func (kv *testKV) GetRevision(key string, revision uint64) (nats.KeyValueEntry, error) {
	_ = revision
	return kv.Get(key)
}

func (kv *testKV) Put(key string, value []byte) (uint64, error) {
	v := make([]byte, len(value))
	copy(v, value)
	kv.data[key] = v
	return uint64(len(kv.data)), nil
}

func (kv *testKV) PutString(key string, value string) (uint64, error) {
	return kv.Put(key, []byte(value))
}

func (kv *testKV) Create(key string, value []byte) (uint64, error) {
	if _, exists := kv.data[key]; exists {
		return 0, fmt.Errorf("key exists")
	}
	return kv.Put(key, value)
}

func (kv *testKV) Update(key string, value []byte, last uint64) (uint64, error) {
	_ = last
	return kv.Put(key, value)
}

func (kv *testKV) Delete(key string, opts ...nats.DeleteOpt) error {
	_ = opts
	delete(kv.data, key)
	return nil
}

func (kv *testKV) Purge(key string, opts ...nats.DeleteOpt) error {
	_ = opts
	delete(kv.data, key)
	return nil
}

func (kv *testKV) Watch(keys string, opts ...nats.WatchOpt) (nats.KeyWatcher, error) {
	_ = keys
	_ = opts
	return nil, fmt.Errorf("not implemented")
}

func (kv *testKV) WatchAll(opts ...nats.WatchOpt) (nats.KeyWatcher, error) {
	_ = opts
	return nil, fmt.Errorf("not implemented")
}

func (kv *testKV) WatchFiltered(keys []string, opts ...nats.WatchOpt) (nats.KeyWatcher, error) {
	_ = keys
	_ = opts
	return nil, fmt.Errorf("not implemented")
}

func (kv *testKV) Keys(opts ...nats.WatchOpt) ([]string, error) {
	_ = opts
	keys := make([]string, 0, len(kv.data))
	for k := range kv.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (kv *testKV) ListKeys(opts ...nats.WatchOpt) (nats.KeyLister, error) {
	_ = opts
	return nil, fmt.Errorf("not implemented")
}

func (kv *testKV) History(key string, opts ...nats.WatchOpt) ([]nats.KeyValueEntry, error) {
	_ = opts
	entry, err := kv.Get(key)
	if err != nil {
		return nil, err
	}
	return []nats.KeyValueEntry{entry}, nil
}

func (kv *testKV) Bucket() string {
	return kv.bucket
}

func (kv *testKV) PurgeDeletes(opts ...nats.PurgeOpt) error {
	_ = opts
	return nil
}

func (kv *testKV) Status() (nats.KeyValueStatus, error) {
	return &testKVStatus{bucket: kv.bucket, values: uint64(len(kv.data))}, nil
}

type testKVEntry struct {
	bucket string
	key    string
	value  []byte
}

func (e *testKVEntry) Bucket() string {
	return e.bucket
}

func (e *testKVEntry) Key() string {
	return e.key
}

func (e *testKVEntry) Value() []byte {
	return e.value
}

func (e *testKVEntry) Revision() uint64 {
	return 1
}

func (e *testKVEntry) Created() time.Time {
	return time.Unix(0, 0)
}

func (e *testKVEntry) Delta() uint64 {
	return 0
}

func (e *testKVEntry) Operation() nats.KeyValueOp {
	return nats.KeyValuePut
}

type testKVStatus struct {
	bucket string
	values uint64
}

func (s *testKVStatus) Bucket() string {
	return s.bucket
}

func (s *testKVStatus) Values() uint64 {
	return s.values
}

func (s *testKVStatus) History() int64 {
	return 1
}

func (s *testKVStatus) TTL() time.Duration {
	return 0
}

func (s *testKVStatus) BackingStore() string {
	return "memory"
}

func (s *testKVStatus) Bytes() uint64 {
	return 0
}

func (s *testKVStatus) IsCompressed() bool {
	return false
}

func (s *testKVStatus) Config() nats.KeyValueConfig {
	return nats.KeyValueConfig{Bucket: s.bucket}
}
