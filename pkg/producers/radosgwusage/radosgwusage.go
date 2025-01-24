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
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ceph/go-ceph/rgw/admin"
	"github.com/nats-io/nats.go"
)

var outputToFile = false // debug only

// collectUsageMetrics collects usage metrics from the RadosGW and processes them.
// It retrieves usage statistics, bucket data, and user data, and then processes
// these data points into a list of UsageEntry, which can be used for further processing.
func collectUsageMetrics(cfg RadosGWUsageConfig, currentTime time.Time) ([]UsageEntry, error) {
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
		// Start:       "2024-09-16 15:00:00",
		// End:         "2024-09-16 17:00:00",
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
				if outputToFile {
					logErrorToFile(err, bucketName) // Log error to file if the flag is set
				}

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
			log.Trace().
				Str("user", userName).
				Msg("starting GetUser for user")

			userInfo, err := co.GetUser(context.Background(), admin.User{ID: userName})
			if err != nil {
				log.Error().
					Str("user", userName).
					Err(err).
					Msg("error fetching user info")
				errCh <- err
				return
			}

			log.Trace().
				Str("user", userName).
				Msg("successfully fetched user info")

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
	processBucketData(cfg, bucketData, usageDict, &entries, currentTime)
	// Process the collected user data and add it to the entries list.
	processUserData(cfg, &entries, userData, co, currentTime)

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
func processBucketData(cfg RadosGWUsageConfig, bucketData []admin.Bucket, usageDict map[string]map[string]map[string]UsageMetrics, entries *[]UsageEntry, currentTime time.Time) {
	var bucketsProcessed int

	readCategories := []string{
		"get_obj",
		"list_bucket",
		"get_bucket_policy",
		"get_lifecycle",
		"get_obj_tags",
		"list_buckets",
		"get_bucket_location",
		"stat_bucket",
		"stat_account",
		"get_bucket_cors",
		"get_bucket_versioning",
		"get_obj_acl",
		"get_bucket_logging",
		"get_bucket_notification",
		"get_acls",
		"list_bucket_multiparts",
		"list_multipart",
		"get_request_payment",
		"get_bucket_public_access_block",
	}
	writeCategories := []string{
		"put_obj",
		"delete_obj",
		"put_bucket_policy",
		"put_lifecycle",
		"create_bucket",
		"delete_bucket",
		"put_obj_acl",
		"put_obj_metadata",
		"put_bucket_metadata",
		"delete_bucket_policy",
		"put_bucket_cors",
		"delete_bucket_cors",
		"put_bucket_logging",
		"delete_bucket_logging",
		"put_bucket_notification",
		"delete_bucket_notification",
		"put_bucket_versioning",
		"init_multipart",
		"complete_multipart",
		"multi_object_delete",
		"copy_obj",
		"put_acls",
		"abort_multipart",
		"post_obj",
	}

	for _, bucket := range bucketData {
		bucketName := bucket.Bucket
		bucketOwner := bucket.Owner
		bucketZonegroup := bucket.Zonegroup
		bucketShards := bucket.NumShards

		var bucketUsageBytes, bucketUtilizedBytes uint64
		var bucketUsageObjects uint64
		var bucketQuotaEnabled bool
		var bucketQuotaMaxSize int64
		var bucketQuotaMaxObjects int64
		var totalOps, readOps, writeOps, successOps uint64
		var totalBytesSent uint64
		var totalBytesReceived uint64
		var totalThroughputBytes uint64
		var totalRequests uint64
		var errorOps uint64
		var errorRate float64

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
		apiUsagePerBucket := make(map[string]uint64)
		if bucketCategoryUsage, ok := usageDict[bucketOwner][bucketName]; ok {
			for categoryName, metrics := range bucketCategoryUsage {
				categories = append(categories, CategoryUsage{
					Category:      categoryName,
					Ops:           metrics.Ops,
					SuccessfulOps: metrics.SuccessfulOps,
					BytesSent:     metrics.BytesSent,
					BytesReceived: metrics.BytesReceived,
				})

				apiUsagePerBucket[categoryName] += uint64(metrics.Ops)
				// Aggregate metrics for total operations and other stats
				totalOps += metrics.Ops
				successOps += metrics.SuccessfulOps
				totalBytesSent += metrics.BytesSent
				totalBytesReceived += metrics.BytesReceived
				totalThroughputBytes += metrics.BytesSent + metrics.BytesReceived

				totalRequests++ // Each category operation is counted as a request

				// Check if the category is a read or write operation
				if contains(readCategories, categoryName) {
					readOps += metrics.Ops
				} else if contains(writeCategories, categoryName) {
					writeOps += metrics.Ops
				} else {
					log.Info().Str("Category", categoryName).Msg("Unknown category")
				}
			}
		}

		// Calculate error rate and error ops
		errorOps = totalOps - successOps
		if totalOps > 0 {
			errorRate = (float64(errorOps) / float64(totalOps)) * 100
		}

		// Calculate error rate and error ops
		errorOps = totalOps - successOps
		if totalOps > 0 {
			errorRate = (float64(errorOps) / float64(totalOps)) * 100
		}

		// Find or create the UsageEntry for the bucket owner
		entry := findOrCreateEntryByBucketOwner(entries, bucketOwner)
		entry.ClusterID = cfg.ClusterID

		// Calculate the current metrics for the bucket
		bucketMetrics := NewRadosGWBucketMetrics()
		bucketMetrics.Meta.Name = bucketName
		bucketMetrics.Meta.Owner = bucketOwner
		bucketMetrics.Meta.Zonegroup = bucketZonegroup
		bucketMetrics.Meta.Shards = bucketShards
		bucketMetrics.Meta.CreatedAt = bucket.CreationTime

		// Set the quota for the bucket
		bucketMetrics.Quota.Enabled = bucketQuotaEnabled
		bucketMetrics.Quota.MaxSize = Uint64Ptr(uint64(bucketQuotaMaxSize))
		bucketMetrics.Quota.MaxObjects = Uint64Ptr(uint64(bucketQuotaMaxObjects))

		// Set the totals for the bucket
		bucketMetrics.Totals.DataSize = bucketUsageBytes
		bucketMetrics.Totals.UtilizedSize = bucketUtilizedBytes
		bucketMetrics.Totals.Objects = bucketUsageObjects
		bucketMetrics.Totals.ReadOps = readOps
		bucketMetrics.Totals.WriteOps = writeOps
		bucketMetrics.Totals.BytesSent = totalBytesSent
		bucketMetrics.Totals.BytesReceived = totalBytesReceived
		bucketMetrics.Totals.SuccessOps = successOps
		bucketMetrics.Totals.OpsTotal = totalOps
		bucketMetrics.Totals.ErrorRate = errorRate
		bucketMetrics.Totals.Capacity = bucketUsageBytes //FIXME We may want to refine this if capacity is calculated differently

		bucketMetrics.APIUsage = apiUsagePerBucket

		CalculateCurrentBucketMetrics(&bucketMetrics, totalBytesSent, totalBytesReceived, readOps, writeOps, currentTime)
		CalculateCurrentBucketAPIUsage(&bucketMetrics, bucketMetrics.APIUsage, currentTime)

		// Append the bucket metrics to BucketLevels
		entry.Buckets = append(entry.Buckets, bucketMetrics)

		bucketsProcessed++
	}
	log.Info().
		Int("buckets_processed", bucketsProcessed).
		Msg("bucket data processing completed")
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// processUserData processes user data and updates the corresponding entries with user-specific information.
func processUserData(cfg RadosGWUsageConfig, entries *[]UsageEntry, users []admin.User, co *admin.API, currentTime time.Time) error {
	for _, userInfo := range users {

		// Find the corresponding entry for the user, or create a new one if not found
		entry := findOrCreateEntryByBucketOwner(entries, userInfo.ID)

		// Populate static metadata only if it's not already set
		if entry.UserLevel.Meta.ID == "" {
			entry.UserLevel.Meta.ID = userInfo.ID
			entry.UserLevel.Meta.DisplayName = userInfo.DisplayName
			entry.UserLevel.Meta.Email = userInfo.Email
			entry.UserLevel.Meta.DefaultStorageClass = userInfo.DefaultStorageClass
		}

		// Populate quota information
		populateQuotaInfo(entry, userInfo)

		// Populate stats information
		if userInfo.Stat != (admin.UserStat{}) { // Check if stats are present
			entry.Stats = userInfo.Stat
		}

		// Calculate the total number of buckets, objects, and data size for the user
		entry.UserLevel.Totals.BucketsTotal = len(entry.Buckets)
		entry.UserLevel.APIUsagePerUser = make(map[string]uint64)
		for _, bucket := range entry.Buckets {
			entry.UserLevel.Totals.ObjectsTotal += bucket.Totals.Objects
			entry.UserLevel.Totals.DataSizeTotal += bucket.Totals.DataSize
			entry.UserLevel.Totals.OpsTotal += bucket.Totals.OpsTotal // Accumulate ops, which are the total requests
			entry.UserLevel.Totals.ReadOpsTotal += bucket.Totals.ReadOps
			entry.UserLevel.Totals.WriteOpsTotal += bucket.Totals.WriteOps
			entry.UserLevel.Totals.BytesSentTotal += bucket.Totals.BytesSent
			entry.UserLevel.Totals.BytesReceivedTotal += bucket.Totals.BytesReceived
			entry.UserLevel.Totals.SuccessOpsTotal += bucket.Totals.SuccessOps
			entry.UserLevel.Totals.TotalCapacity += bucket.Totals.Capacity
			entry.UserLevel.Totals.ThroughputBytesTotal += bucket.Current.ThroughputBytesPerSec

			// Calculate error rate and error ops
			errorOps := entry.UserLevel.Totals.OpsTotal - entry.UserLevel.Totals.SuccessOpsTotal
			if entry.UserLevel.Totals.OpsTotal > 0 {
				entry.UserLevel.Totals.ErrorRateTotal = (float64(errorOps) / float64(entry.UserLevel.Totals.OpsTotal)) * 100
			}

			for category, ops := range bucket.APIUsage {
				entry.UserLevel.APIUsagePerUser[category] += ops // Sum API usage from all buckets
			}
		}

		// Calculate current metrics
		CalculateCurrentUserMetrics(&entry.UserLevel, currentTime)
		CalculateCurrentAPIUsagePerUser(&entry.UserLevel, currentTime)
	}

	return nil
}

func BoolPtr(b bool) *bool {
	return &b
}

func Uint64Ptr(value uint64) *uint64 {
	return &value
}

func findOrCreateEntry(entries *[]UsageEntry, userID string) *UsageEntry {
	for i, entry := range *entries {
		if entry.UserLevel.Meta.ID == userID {
			return &(*entries)[i]
		}
	}

	*entries = append(*entries, UsageEntry{UserLevel: *NewRadosGWUserMetrics()})
	return &(*entries)[len(*entries)-1]
}

func findOrCreateEntryByBucketOwner(entries *[]UsageEntry, userID string) *UsageEntry {
	for i, entry := range *entries {
		if len(entry.Buckets) > 0 && entry.Buckets[0].Meta.Owner == userID {
			return &(*entries)[i]
		}
	}

	*entries = append(*entries, UsageEntry{UserLevel: *NewRadosGWUserMetrics()})
	return &(*entries)[len(*entries)-1]
}

// populateUserQuota populates the user quota information for a user in the entry.
func populateQuotaInfo(entry *UsageEntry, userInfo admin.User) {
	// Initialize user quota in UserLevel
	entry.UserLevel.Quota.Enabled = false // Default to quota being disabled

	// Check and populate user quota if enabled
	if userInfo.UserQuota.Enabled != nil && *userInfo.UserQuota.Enabled {
		entry.UserLevel.Quota.Enabled = true
		entry.UserLevel.Quota.MaxSize = nil
		entry.UserLevel.Quota.MaxObjects = nil

		if userInfo.UserQuota.MaxSize != nil {
			size := uint64(*userInfo.UserQuota.MaxSize)
			entry.UserLevel.Quota.MaxSize = &size
		}

		if userInfo.UserQuota.MaxObjects != nil {
			objects := uint64(*userInfo.UserQuota.MaxObjects)
			entry.UserLevel.Quota.MaxObjects = &objects
		}
	}

	if userInfo.BucketQuota.Enabled != nil && *userInfo.BucketQuota.Enabled {
		// Find the correct bucket in entry.Buckets and set the quota
		for i := range entry.Buckets {
			if entry.Buckets[i].Meta.Name == userInfo.BucketQuota.Bucket {
				entry.Buckets[i].Quota.Enabled = false

				if userInfo.BucketQuota.Enabled != nil && *userInfo.BucketQuota.Enabled {
					entry.Buckets[i].Quota.Enabled = true
					entry.Buckets[i].Quota.MaxSize = nil
					entry.Buckets[i].Quota.MaxObjects = nil

					if userInfo.BucketQuota.MaxSize != nil {
						size := uint64(*userInfo.BucketQuota.MaxSize)
						entry.Buckets[i].Quota.MaxSize = &size
					}

					if userInfo.BucketQuota.MaxObjects != nil {
						objects := uint64(*userInfo.BucketQuota.MaxObjects)
						entry.Buckets[i].Quota.MaxObjects = &objects
					}
				}

				break // Found the bucket
			}
		}
	}
}

// StartRadosGWUsageExporter starts the process of exporting RadosGW usage metrics.
// The function supports exporting metrics to Prometheus, NATS, or printing them to stdout.
// It runs indefinitely, collecting metrics at regular intervals as defined by the configuration.
func StartRadosGWUsageExporterXXX(cfg RadosGWUsageConfig) {
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
			entries, err := collectUsageMetrics(cfg, startTime)
			if err != nil {
				log.Error().
					Err(err).
					Msg("error collecting usage metrics")
				return // Skip to the next iteration if an error occurs
			}

			// Calculate duration and set it in the scrapeDurationSeconds metric
			// duration := time.Since(startTime).Seconds()

			// If Prometheus is enabled, publish the collected metrics to Prometheus
			if cfg.Prometheus {
				// publishToPrometheus(entries, duration, cfg)
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

func logErrorToFile(err error, bucketName string) {
	// Open the file in append mode
	f, fileErr := os.OpenFile("/tmp/output.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if fileErr != nil {
		log.Error().Err(fileErr).Msg("error opening log file")
		return
	}
	defer f.Close()

	// Write the error and response context to the log file
	logger := log.Output(f)
	logger.Error().
		Str("bucket_name", bucketName).
		Err(err).
		Msg("error fetching info for bucket")
}
