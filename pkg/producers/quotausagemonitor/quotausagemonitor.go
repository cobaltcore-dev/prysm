// Copyright (C) 2024 Clyso GmbH
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package quotausagemonitor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ceph/go-ceph/rgw/admin"
	"github.com/nats-io/nats.go"
)

type QuotaUsage struct {
	UserID         string `json:"user_id"`
	TotalQuota     uint64 `json:"total_quota"`
	UsedQuota      uint64 `json:"used_quota"`
	RemainingQuota uint64 `json:"remaining_quota"`
	NodeName       string `json:"node_name"`
	InstanceID     string `json:"instance_id"`
	PhysicalSize   string `json:"physical_size"`
}

func collectQuotaUsage(cfg QuotaUsageMonitorConfig) ([]QuotaUsage, error) {
	co, err := admin.New(cfg.AdminURL, cfg.AccessKey, cfg.SecretKey, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating RGW admin connection: %v", err)
	}

	users, err := co.GetUsers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting users: %v", err)
	}

	var quotas []QuotaUsage
	for _, user := range *users {
		userInfo, err := co.GetUser(context.Background(), admin.User{ID: user, GenerateStat: boolPtr(true)})
		if err != nil {
			log.Error().Err(err).Str("user", user).Msg("Error getting user info")
			continue
		}

		if userInfo.UserQuota.Enabled != nil && *userInfo.UserQuota.Enabled {
			usedQuota := uint64(*userInfo.Stat.Size)
			totalQuota := uint64(*userInfo.UserQuota.MaxSize)

			// Handle case where usedQuota exceeds totalQuota
			if usedQuota > totalQuota {
				log.Warn().Str("user", user).Uint64("usedQuota", usedQuota).Uint64("totalQuota", totalQuota).Msg("User has used more than their total quota")
				totalQuota = usedQuota
			}

			usagePercent := float64(*userInfo.Stat.Size) / float64(*userInfo.UserQuota.MaxSize) * 100
			if usagePercent >= cfg.QuotaUsagePercent {
				quota := QuotaUsage{
					UserID:         user,
					TotalQuota:     totalQuota,
					UsedQuota:      usedQuota,
					RemainingQuota: totalQuota - usedQuota,
					NodeName:       cfg.NodeName,
					InstanceID:     cfg.InstanceID,
				}
				quotas = append(quotas, quota)
			}
		}

		// // Use ListBuckets to get the buckets for the user
		// buckets, err := co.ListUsersBuckets(context.Background(), user)
		// if err != nil {
		// 	log.Printf("Error listing buckets for user %s: %v", user, err)
		// 	continue
		// }

		// for _, bucket := range buckets {
		// 	bucketInfo, err := co.GetBucketInfo(context.Background(), admin.Bucket{Bucket: bucket})
		// 	if err != nil {
		// 		log.Printf("Error getting bucket info for %s: %v", bucket, err)
		// 		continue
		// 	}

		// 	if bucketInfo.BucketQuota.Enabled != nil && *bucketInfo.BucketQuota.Enabled {
		// 		usagePercent := float64(*userInfo.Stat.Size) / float64(*userInfo.UserQuota.MaxSize) * 100
		// 		if usagePercent >= cfg.QuotaUsagePercent {
		// 			quota := QuotaUsage{
		// 				UserID:         user,
		// 				TotalQuota:     uint64(*userInfo.UserQuota.MaxSize),
		// 				UsedQuota:      uint64(*userInfo.Stat.Size),
		// 				RemainingQuota: uint64(*userInfo.UserQuota.MaxSize) - uint64(*userInfo.Stat.Size),
		// 				NodeName:       cfg.NodeName,
		// 				InstanceID:     cfg.InstanceID,
		// 			}
		// 			quotas = append(quotas, quota)
		// 		}
		// 	}
		// }
	}

	return quotas, nil
}

func StartMonitoring(cfg QuotaUsageMonitorConfig) {
	var nc *nats.Conn
	var err error
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Fatal().Err(err).Msg("Error connecting to NATS")
		}
		defer nc.Close()
	}

	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		quotas, err := collectQuotaUsage(cfg)
		if err != nil {
			log.Error().Err(err).Msg("Error collecting quota usage")
			continue
		}

		if cfg.UseNats {
			if err := PublishToNATS(nc, quotas, cfg); err != nil {
				log.Error().Err(err).Msg("Error publishing to NATS")
			}
		} else {
			if len(quotas) > 0 {
				quotasJSON, err := json.MarshalIndent(quotas, "", "  ")
				if err != nil {
					log.Error().Err(err).Msg("Error marshalling quotas to JSON")
					continue
				}
				fmt.Println(string(quotasJSON))
			} else {
				log.Info().Msg("No quota usage found.")
			}
		}
	}
}

func boolPtr(b bool) *bool {
	return &b
}
