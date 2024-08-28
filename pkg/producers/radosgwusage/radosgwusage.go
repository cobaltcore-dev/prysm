// Copyright 2024 Clyso GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package radosgwusage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ceph/go-ceph/rgw/admin"
	"github.com/nats-io/nats.go"
)

// collectUsageMetrics collects usage metrics from the RadosGW and processes them.
// It retrieves usage statistics, bucket data, and user data, and then processes
// these data points into a list of UsageEntry, which can be used for further processing.
func collectUsageMetrics(cfg RadosGWUsageConfig) ([]UsageEntry, error) {
	// Validate the configuration to ensure necessary fields are set
	if cfg.AdminURL == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("invalid configuration: AdminURL, AccessKey, and SecretKey must be provided")
	}

	// Create a new RadosGW admin client using the provided configuration.
	httpClient := &http.Client{Timeout: 30 * time.Second}
	co, err := admin.New(cfg.AdminURL, cfg.AccessKey, cfg.SecretKey, httpClient)
	if err != nil {
		return nil, err // Return an error if the client creation fails.
	}

	// Set up the usage request to include both entries and summaries.
	showEntries := true
	showSummary := true
	usageRequest := admin.Usage{
		ShowEntries: &showEntries,
		ShowSummary: &showSummary,
	}

	// Fetch usage statistics from RadosGW.
	// usageCtx, usageCancel := context.WithTimeout(context.Background(), timeout)
	// defer usageCancel()
	usage, err := co.GetUsage(context.Background(), usageRequest)
	if err != nil {
		return nil, fmt.Errorf("fetching usage data fails: %v", err)
	}

	// Fetch bucket data from RadosGW concurrently.
	bucketNames, err := co.ListBuckets(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %v", err)
	}

	var bucketData []admin.Bucket
	bucketDataCh := make(chan admin.Bucket, len(bucketNames))
	errCh := make(chan error, len(bucketNames))

	for _, bucketName := range bucketNames {
		go func(bucketName string) {
			bucketInfo, err := co.GetBucketInfo(context.Background(), admin.Bucket{Bucket: bucketName})
			if err != nil {
				log.Error().
					Str("bucket_name", bucketName).
					Err(err).
					Msg("error fetching info for bucket")
				errCh <- err
				return
			}
			bucketDataCh <- bucketInfo
		}(bucketName)
	}

	var bucketsProcessed, bucketsFailed int
	for i := 0; i < len(bucketNames); i++ {
		select {
		case data := <-bucketDataCh:
			bucketData = append(bucketData, data)
			bucketsProcessed++
		case err := <-errCh:
			log.Error().
				Err(err).
				Msg("error received during bucket data collection")
			bucketsFailed++
		}
	}
	close(bucketDataCh)
	close(errCh)

	log.Info().
		Int("buckets_processed", bucketsProcessed).
		Int("buckets_failed", bucketsFailed).
		Msg("bucket data collection completed")

	// Fetch user data from RadosGW concurrently.
	userIDs, err := co.GetUsers(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get user list: %v", err)
	}

	var userData []admin.User
	userDataCh := make(chan admin.User, len(*userIDs))
	errCh = make(chan error, len(*userIDs))

	for _, userName := range *userIDs {
		go func(userName string) {
			userInfo, err := co.GetUser(context.Background(), admin.User{ID: userName})
			if err != nil {
				log.Error().
					Str("user", userName).
					Err(err).
					Msg("error fetching user info")
				errCh <- err
				return
			}
			userDataCh <- userInfo
		}(userName)
	}

	var usersProcessed, usersFailed int
	for i := 0; i < len(*userIDs); i++ {
		select {
		case data := <-userDataCh:
			userData = append(userData, data)
			usersProcessed++
		case err := <-errCh:
			log.Error().
				Err(err).
				Msg("error received during user data collection")
			usersFailed++
		}
	}
	close(userDataCh)
	close(errCh)

	log.Info().
		Int("users_processed", usersProcessed).
		Int("users_failed", usersFailed).
		Msg("user data collection completed")

	// Initialize a dictionary to store usage metrics, organized by categories.
	usageDict := make(map[string]map[string]map[string]UsageMetrics)
	processUsageData(usage, usageDict) // Process the usage data into the usageDict.

	var entries []UsageEntry
	// Process the collected bucket data and add it to the entries list.
	processBucketData(cfg, bucketData, usageDict, &entries)
	// Process the collected user data and add it to the entries list.
	processUserData(cfg, &entries, userData, co)

	return entries, nil // Return the processed usage entries.
}

// processUsageData processes the usage data and updates the usageDict accordingly.
// The function iterates through each entry in the usage data, categorizing metrics by user, bucket, and category.
func processUsageData(usage admin.Usage, usageDict map[string]map[string]map[string]UsageMetrics) {
	for _, entry := range usage.Entries {
		bucketOwner := "unknown_user" // Use a placeholder for entries without a user

		if entry.User != "" {
			bucketOwner = entry.User
		}

		if _, ok := usageDict[bucketOwner]; !ok {
			usageDict[bucketOwner] = make(map[string]map[string]UsageMetrics)
		}

		for _, bucket := range entry.Buckets {
			bucketName := bucket.Bucket
			if bucketName == "" {
				bucketName = "bucket_root" // Use a placeholder for root or unnamed buckets
			}

			if _, ok := usageDict[bucketOwner][bucketName]; !ok {
				usageDict[bucketOwner][bucketName] = make(map[string]UsageMetrics)
			}

			for _, category := range bucket.Categories {
				categoryName := category.Category

				if _, ok := usageDict[bucketOwner][bucketName][categoryName]; !ok {
					usageDict[bucketOwner][bucketName][categoryName] = UsageMetrics{}
				}

				metrics := usageDict[bucketOwner][bucketName][categoryName]

				// Accumulate metrics
				if category.Ops > 0 {
					metrics.Ops += category.Ops
				}
				if category.SuccessfulOps > 0 {
					metrics.SuccessfulOps += category.SuccessfulOps
				}
				if category.BytesSent > 0 {
					metrics.BytesSent += category.BytesSent
				}
				if category.BytesReceived > 0 {
					metrics.BytesReceived += category.BytesReceived
				}

				usageDict[bucketOwner][bucketName][categoryName] = metrics
			}
		}
	}
	log.Info().
		Int("total_entries_processed", len(usage.Entries)).
		Msg("usage data processing completed")
}

// processBucketData processes the bucket data and adds relevant details to the entries list.
// It also ensures that categories from the usageDict are correctly included in the output.
func processBucketData(cfg RadosGWUsageConfig, bucketData []admin.Bucket, usageDict map[string]map[string]map[string]UsageMetrics, entries *[]UsageEntry) {
	var bucketsProcessed int

	for _, bucket := range bucketData {
		bucketName := bucket.Bucket
		bucketOwner := bucket.Owner
		bucketShards := bucket.NumShards
		bucketZonegroup := bucket.Zonegroup

		var bucketUsageBytes, bucketUtilizedBytes uint64
		var bucketUsageObjects uint64
		var bucketQuotaEnabled bool
		var bucketQuotaMaxSize int64
		var bucketQuotaMaxSizeBytes int
		var bucketQuotaMaxObjects int64
		var totalOps uint64
		var totalBytesSent uint64
		var totalBytesReceived uint64
		var maxOps uint64
		var totalThroughputBytes uint64
		var totalLatencySeconds float64
		var totalRequests uint64
		var currentOps uint64

		// Calculate usage bytes, utilized bytes, and object count
		if bucket.Usage.RgwMain.SizeActual != nil {
			bucketUsageBytes = *bucket.Usage.RgwMain.SizeActual
		} else if bucket.Usage.RgwMain.SizeKbActual != nil {
			bucketUsageBytes = uint64(*bucket.Usage.RgwMain.SizeKbActual) * 1024
		}

		if bucket.Usage.RgwMain.SizeUtilized != nil {
			bucketUtilizedBytes = *bucket.Usage.RgwMain.SizeUtilized
		}

		if bucket.Usage.RgwMain.NumObjects != nil {
			bucketUsageObjects = *bucket.Usage.RgwMain.NumObjects
		}

		if bucket.BucketQuota.Enabled != nil {
			bucketQuotaEnabled = *bucket.BucketQuota.Enabled
		}
		if bucket.BucketQuota.MaxSize != nil {
			bucketQuotaMaxSize = int64(*bucket.BucketQuota.MaxSize)
		}
		if bucket.BucketQuota.MaxSizeKb != nil {
			bucketQuotaMaxSizeBytes = int(*bucket.BucketQuota.MaxSizeKb) * 1024
		}
		if bucket.BucketQuota.MaxObjects != nil {
			bucketQuotaMaxObjects = *bucket.BucketQuota.MaxObjects
		}

		// Populate the usage dictionary or UsageEntry list
		if _, ok := usageDict[bucketOwner]; !ok {
			usageDict[bucketOwner] = make(map[string]map[string]UsageMetrics)
		}
		if _, ok := usageDict[bucketOwner][bucketName]; !ok {
			usageDict[bucketOwner][bucketName] = make(map[string]UsageMetrics)
		}

		// Prepare the category usage data from usageDict
		var categories []CategoryUsage
		if bucketCategoryUsage, ok := usageDict[bucketOwner][bucketName]; ok {
			for categoryName, metrics := range bucketCategoryUsage {
				categories = append(categories, CategoryUsage{
					Category:      categoryName,
					Ops:           metrics.Ops,
					SuccessfulOps: metrics.SuccessfulOps,
					BytesSent:     metrics.BytesSent,
					BytesReceived: metrics.BytesReceived,
				})
				// Aggregate metrics for total operations and other stats
				totalOps += metrics.Ops
				totalBytesSent += metrics.BytesSent
				totalBytesReceived += metrics.BytesReceived
				totalThroughputBytes += metrics.BytesSent + metrics.BytesReceived
				totalLatencySeconds += float64(metrics.Ops) * 0.05 //FIXME Simulated latency (e.g., 50ms per operation)
				// FIXME: currentOps and maxOps to be retrieved from a NATS subject or similar source
				if metrics.Ops > maxOps {
					maxOps = metrics.Ops
				}
				totalRequests++ // Each category operation is counted as a request
			}
		}

		// Calculate size in KB if not directly provided
		var sizeKb, sizeKbActual, sizeKbUtilized *uint64

		// Check if SizeKb is provided; if not, calculate it from bucketUsageBytes
		if bucket.Usage.RgwMain.SizeKb != nil {
			sizeKb = bucket.Usage.RgwMain.SizeKb
		} else {
			calculatedSizeKb := bucketUsageBytes / 1024
			sizeKb = &calculatedSizeKb
		}

		// Check if SizeKbActual is provided; if not, calculate it from bucketUsageBytes
		if bucket.Usage.RgwMain.SizeKbActual != nil {
			sizeKbActual = bucket.Usage.RgwMain.SizeKbActual
		} else {
			calculatedSizeKbActual := bucketUsageBytes / 1024
			sizeKbActual = &calculatedSizeKbActual
		}

		// Check if SizeKbUtilized is provided; if not, calculate it from bucketUtilizedBytes
		if bucket.Usage.RgwMain.SizeKbUtilized != nil {
			sizeKbUtilized = bucket.Usage.RgwMain.SizeKbUtilized
		} else {
			calculatedSizeKbUtilized := bucketUtilizedBytes / 1024
			sizeKbUtilized = &calculatedSizeKbUtilized
		}

		// Find or create the UsageEntry for the bucket owner
		entry := findOrCreateEntry(entries, bucketOwner)

		// Append the bucket information to the user's entry
		entry.Buckets = append(entry.Buckets, BucketUsage{
			Bucket:    bucketName,
			Owner:     bucketOwner,
			Zonegroup: bucketZonegroup,
			Store:     cfg.Store,
			Usage: UsageStats{
				RgwMain: struct {
					Size           *uint64 `json:"size"`
					SizeActual     *uint64 `json:"size_actual"`
					SizeUtilized   *uint64 `json:"size_utilized"`
					SizeKb         *uint64 `json:"size_kb"`
					SizeKbActual   *uint64 `json:"size_kb_actual"`
					SizeKbUtilized *uint64 `json:"size_kb_utilized"`
					NumObjects     *uint64 `json:"num_objects"`
				}{
					Size:           &bucketUsageBytes,
					SizeActual:     &bucketUsageBytes,
					SizeUtilized:   &bucketUtilizedBytes,
					SizeKb:         sizeKb,
					SizeKbActual:   sizeKbActual,
					SizeKbUtilized: sizeKbUtilized,
					NumObjects:     &bucketUsageObjects,
				},
			},
			BucketQuota: admin.QuotaSpec{
				UID:        bucketOwner,
				Bucket:     bucketName,
				QuotaType:  "bucket",
				Enabled:    &bucketQuotaEnabled,
				CheckOnRaw: false,
				MaxSize:    &bucketQuotaMaxSize,
				MaxSizeKb:  &bucketQuotaMaxSizeBytes,
				MaxObjects: &bucketQuotaMaxObjects,
			},
			NumShards:            *bucketShards,
			Categories:           categories,
			TotalOps:             totalOps,
			TotalBytesSent:       totalBytesSent,
			TotalBytesReceived:   totalBytesReceived,
			TotalThroughputBytes: totalThroughputBytes,
			TotalLatencySeconds:  totalLatencySeconds,
			TotalRequests:        totalRequests,
			CurrentOps:           currentOps,
			MaxOps:               maxOps,
		})

		bucketsProcessed++
	}
	log.Info().
		Int("buckets_processed", bucketsProcessed).
		Msg("bucket data processing completed")
}

// processUserData processes user data and updates the corresponding entries with user-specific information.
func processUserData(cfg RadosGWUsageConfig, entries *[]UsageEntry, users []admin.User, co *admin.API) error {
	for _, user := range users {
		// Fetch detailed user info with statistics using the GenerateStat flag
		userInfo, err := co.GetUser(context.Background(), admin.User{ID: user.ID, GenerateStat: BoolPtr(true)})
		if err != nil {
			log.Error().
				Str("user_id", user.ID).
				Err(err).
				Msg("error getting user info")
			continue // Skip to the next iteration if an error occurs
		}

		// Find the corresponding entry for the user, or create a new one if not found
		entry := findOrCreateEntry(entries, user.ID)

		// Populate user-specific data into the entry
		entry.User = user.ID
		entry.DisplayName = userInfo.DisplayName
		entry.Email = userInfo.Email
		entry.DefaultStorageClass = userInfo.DefaultStorageClass
		entry.Store = cfg.Store

		// Populate quota information
		populateQuotaInfo(entry, userInfo)

		// Populate stats information
		if userInfo.Stat != (admin.UserStat{}) { // Check if stats are present
			entry.Stats = userInfo.Stat
		}

		// Calculate the total number of buckets, objects, and data size for the user
		entry.TotalBuckets = len(entry.Buckets)
		for _, bucket := range entry.Buckets {
			entry.TotalObjects += *bucket.Usage.RgwMain.NumObjects
			entry.TotalDataSize += *bucket.Usage.RgwMain.SizeUtilized
			entry.TotalOps += bucket.TotalOps // Accumulate ops, which are the total requests
			entry.TotalBytesSent += bucket.TotalBytesSent
			entry.TotalBytesReceived += bucket.TotalBytesReceived
			entry.TotalThroughputBytes += bucket.TotalThroughputBytes
			entry.TotalLatencySeconds += bucket.TotalLatencySeconds

			// Track current and max ops for the account
			entry.CurrentOps += bucket.CurrentOps
			if bucket.MaxOps > entry.MaxOps {
				entry.MaxOps = bucket.MaxOps
			}
		}
	}

	return nil
}

func BoolPtr(b bool) *bool {
	return &b
}

func findOrCreateEntry(entries *[]UsageEntry, userID string) *UsageEntry {
	for i, entry := range *entries {
		if entry.User == userID {
			return &(*entries)[i]
		}
	}

	*entries = append(*entries, UsageEntry{User: userID})
	return &(*entries)[len(*entries)-1]
}

// populateUserQuota populates the user quota information for a user in the entry.
func populateQuotaInfo(entry *UsageEntry, userInfo admin.User) {
	falsePtr := BoolPtr(false)
	entry.UserQuota = admin.QuotaSpec{Enabled: falsePtr}

	if userInfo.UserQuota.Enabled != nil && *userInfo.UserQuota.Enabled {
		entry.UserQuota = userInfo.UserQuota
	}

	if userInfo.BucketQuota.Enabled != nil && *userInfo.BucketQuota.Enabled {
		fmt.Print("XXXXXX")
		//FIXME entry.Buckets[?].BucketQuota = userInfo.BucketQuota
	}
}

// StartRadosGWUsageExporter starts the process of exporting RadosGW usage metrics.
// The function supports exporting metrics to Prometheus, NATS, or printing them to stdout.
// It runs indefinitely, collecting metrics at regular intervals as defined by the configuration.
func StartRadosGWUsageExporter(cfg RadosGWUsageConfig) {
	// If Prometheus is enabled in the configuration, start the Prometheus metrics server
	if cfg.Prometheus {
		go startPrometheusMetricsServer(cfg.PrometheusPort)
	}

	var nc *nats.Conn
	var err error

	// If NATS is enabled in the configuration, establish a connection to the NATS server
	if cfg.UseNats {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("error connecting to NATS") // Log a fatal error if the connection fails and exit
		}
		defer nc.Close() // Ensure that the NATS connection is closed when the function exits
	}

	// Create a ticker that triggers at the specified interval to collect metrics
	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	isRunning := false

	// Run the loop indefinitely, collecting metrics on each tick
	for range ticker.C {
		// Skip this tick if the previous run hasn't finished
		if isRunning {
			log.Trace().Msg("previous metrics collection is still running; skipping this tick")
			continue
		}
		isRunning = true
		go func() {
			defer func() {
				isRunning = false // Reset the flag after the function completes
			}()

			// Start timing
			startTime := time.Now()

			// Collect usage metrics based on the configuration
			entries, err := collectUsageMetrics(cfg)
			if err != nil {
				log.Error().
					Err(err).
					Msg("error collecting usage metrics")
				return // Skip to the next iteration if an error occurs
			}

			// Calculate duration and set it in the scrapeDurationSeconds metric
			duration := time.Since(startTime).Seconds()

			// If Prometheus is enabled, publish the collected metrics to Prometheus
			if cfg.Prometheus {
				publishToPrometheus(entries, duration, cfg)
			}

			// If NATS is enabled, publish the collected metrics to the specified NATS subject
			if cfg.UseNats {
				publishToNATS(nc, cfg.NatsSubject, entries)
			} else {
				// If NATS is not enabled, output the collected metrics as JSON to stdout
				entriesJSON, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					log.Error().
						Err(err).
						Msg("error marshalling entries to JSON")
					return // Skip to the next iteration if an error occurs
				}
				if !cfg.Prometheus && !cfg.UseNats {
					fmt.Println(string(entriesJSON)) // Print the JSON-formatted metrics to stdout
				}
				log.Trace().Msg(string(entriesJSON))
			}
		}()
	}
}
