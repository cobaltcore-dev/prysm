// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

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
