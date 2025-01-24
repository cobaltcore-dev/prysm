// Copyright 2025 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package radosgwusage

import (
	"net/http"
	"time"

	"github.com/ceph/go-ceph/rgw/admin"
)

func createRadosGWClient(cfg RadosGWUsageConfig, status *PrysmStatus) (*admin.API, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	co, err := admin.New(cfg.AdminURL, cfg.AccessKey, cfg.SecretKey, httpClient)
	if err != nil {
		// Explicitly set TargetUp to false on failure
		status.UpdateTargetUp(false)
		status.IncrementScrapeErrors()
		return nil, err
	}
	// If client creation succeeds, set TargetUp to true
	status.UpdateTargetUp(true)
	return co, nil
}
