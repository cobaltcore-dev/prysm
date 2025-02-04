// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package radosgwusage

import (
	"sync"
)

type PrysmStatus struct {
	mu           sync.Mutex
	TargetUp     float64
	ScrapeErrors int
}

func (s *PrysmStatus) UpdateTargetUp(up bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if up {
		s.TargetUp = 1
	} else {
		s.TargetUp = 0
	}
}

func (s *PrysmStatus) IncrementScrapeErrors() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ScrapeErrors++
}

func (s *PrysmStatus) GetSnapshot() (float64, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TargetUp, s.ScrapeErrors
}
