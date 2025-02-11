// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"net/http"
	"time"

	"github.com/cobaltcore-dev/prysm/pkg/producers/radosgwusage/rgwadmin"
)

func createRadosGWClient(cfg RadosGWUsageConfig, status *PrysmStatus) (*rgwadmin.API, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	co, err := rgwadmin.New(cfg.AdminURL, cfg.AccessKey, cfg.SecretKey, httpClient)
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
