//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"sync"
	"syscall/js"
	"time"
)

// PerformanceLogger aggregates success messages and logs them periodically
type PerformanceLogger struct {
	mutex           sync.Mutex
	operationCounts map[string]int64
	totalParticles  int64
	totalOperations int64
	lastLogTime     time.Time
	logInterval     time.Duration

	// Time-based aggregation buckets
	timeAggregates map[string]*TimeAggregate
	stopCh         chan struct{}
	id             string // diagnostics: unique logger id
	stopped        bool   // diagnostics: track if stopped
}

// TimeAggregate stores performance data for a specific time period
type TimeAggregate struct {
	StartTime      time.Time        `json:"start_time"`
	EndTime        time.Time        `json:"end_time"`
	Duration       time.Duration    `json:"duration"`
	Operations     int64            `json:"operations"`
	Particles      int64            `json:"particles"`
	OperationTypes map[string]int64 `json:"operation_types"`
	PeakOpsPerSec  float64          `json:"peak_ops_per_sec"`
	AvgOpsPerSec   float64          `json:"avg_ops_per_sec"`
	PeakParticles  int64            `json:"peak_particles"`
	AvgParticles   float64          `json:"avg_particles"`
	LastUpdated    time.Time        `json:"last_updated"`
}

// TimePeriod represents different aggregation periods
type TimePeriod struct {
	Name     string
	Duration time.Duration
	Format   string // Time format for bucket keys
}

var aggregationPeriods = []TimePeriod{
	{"minute", time.Minute, "2006-01-02T15:04"},
	{"hour", time.Hour, "2006-01-02T15"},
	{"day", 24 * time.Hour, "2006-01-02"},
	{"week", 7 * 24 * time.Hour, "2006-W02"},  // ISO week
	{"month", 30 * 24 * time.Hour, "2006-01"}, // Approximate month
	{"year", 365 * 24 * time.Hour, "2006"},
}

// NewPerformanceLogger creates a new performance logger
func NewPerformanceLogger(interval time.Duration) *PerformanceLogger {
	logger := &PerformanceLogger{
		operationCounts: make(map[string]int64),
		timeAggregates:  make(map[string]*TimeAggregate),
		lastLogTime:     time.Now(),
		logInterval:     interval,
		stopCh:          make(chan struct{}),
		id:              fmt.Sprintf("logger-%d", time.Now().UnixNano()),
		stopped:         false,
	}

	wasmLog("[PERF-LOGGER]", logger.id, "NewPerformanceLogger created")

	// Initialize time aggregates for all periods
	logger.initializeTimeAggregates()

	// Start background logging goroutine
	go logger.startPeriodicLogging()

	// Start time aggregate cleanup goroutine
	go logger.startAggregateCleanup()

	return logger
}

// initializeTimeAggregates sets up initial time buckets
func (pl *PerformanceLogger) initializeTimeAggregates() {
	now := time.Now()

	for _, period := range aggregationPeriods {
		bucketKey := pl.getTimeBucketKey(now, period)
		pl.timeAggregates[bucketKey] = &TimeAggregate{
			StartTime:      pl.getBucketStartTime(now, period),
			EndTime:        pl.getBucketEndTime(now, period),
			Duration:       period.Duration,
			OperationTypes: make(map[string]int64),
		}
	}
}

// getTimeBucketKey generates a bucket key for the given time and period
func (pl *PerformanceLogger) getTimeBucketKey(t time.Time, period TimePeriod) string {
	switch period.Name {
	case "week":
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	default:
		return t.Format(period.Format)
	}
}

// getBucketStartTime calculates the start time for a time bucket
func (pl *PerformanceLogger) getBucketStartTime(t time.Time, period TimePeriod) time.Time {
	switch period.Name {
	case "minute":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	case "hour":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case "day":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "week":
		// Get Monday of the current week
		year, week := t.ISOWeek()
		jan1 := time.Date(year, 1, 1, 0, 0, 0, 0, t.Location())
		mondayOfFirstWeek := jan1.AddDate(0, 0, -int(jan1.Weekday())+1)
		if jan1.Weekday() == time.Sunday {
			mondayOfFirstWeek = mondayOfFirstWeek.AddDate(0, 0, 7)
		}
		return mondayOfFirstWeek.AddDate(0, 0, (week-1)*7)
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "year":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// getBucketEndTime calculates the end time for a time bucket
func (pl *PerformanceLogger) getBucketEndTime(t time.Time, period TimePeriod) time.Time {
	startTime := pl.getBucketStartTime(t, period)
	switch period.Name {
	case "minute":
		return startTime.Add(time.Minute).Add(-time.Nanosecond)
	case "hour":
		return startTime.Add(time.Hour).Add(-time.Nanosecond)
	case "day":
		return startTime.AddDate(0, 0, 1).Add(-time.Nanosecond)
	case "week":
		return startTime.AddDate(0, 0, 7).Add(-time.Nanosecond)
	case "month":
		return startTime.AddDate(0, 1, 0).Add(-time.Nanosecond)
	case "year":
		return startTime.AddDate(1, 0, 0).Add(-time.Nanosecond)
	default:
		return startTime.Add(period.Duration).Add(-time.Nanosecond)
	}
}

// startAggregateCleanup removes old aggregates to prevent memory leaks
func (pl *PerformanceLogger) startAggregateCleanup() {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for range ticker.C {
		pl.cleanupOldAggregates()
	}
}

// cleanupOldAggregates removes aggregates older than retention periods
func (pl *PerformanceLogger) cleanupOldAggregates() {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	now := time.Now()
	retentionPeriods := map[string]time.Duration{
		"minute": 24 * time.Hour,            // Keep 1 day of minute data
		"hour":   7 * 24 * time.Hour,        // Keep 1 week of hour data
		"day":    30 * 24 * time.Hour,       // Keep 1 month of day data
		"week":   52 * 7 * 24 * time.Hour,   // Keep 1 year of week data
		"month":  24 * 30 * 24 * time.Hour,  // Keep 2 years of month data
		"year":   10 * 365 * 24 * time.Hour, // Keep 10 years of year data
	}

	for key, aggregate := range pl.timeAggregates {
		periodName := pl.extractPeriodFromKey(key)
		if retentionPeriod, exists := retentionPeriods[periodName]; exists {
			if now.Sub(aggregate.EndTime) > retentionPeriod {
				delete(pl.timeAggregates, key)
			}
		}
	}
}

// extractPeriodFromKey extracts the period name from a bucket key
func (pl *PerformanceLogger) extractPeriodFromKey(key string) string {
	for _, period := range aggregationPeriods {
		testKey := pl.getTimeBucketKey(time.Now(), period)
		if len(key) == len(testKey) {
			// Match format length to determine period
			switch len(key) {
			case 16: // 2006-01-02T15:04
				return "minute"
			case 13: // 2006-01-02T15
				return "hour"
			case 10: // 2006-01-02
				return "day"
			case 8: // 2006-W02
				return "week"
			case 7: // 2006-01
				return "month"
			case 4: // 2006
				return "year"
			}
		}
	}
	return "unknown"
}

// LogSuccess records a successful operation (non-blocking)
func (pl *PerformanceLogger) LogSuccess(operation string, particleCount int64) {
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	now := time.Now()

	// Update immediate counters
	pl.operationCounts[operation]++
	pl.totalParticles += particleCount
	pl.totalOperations++

	// Update time-based aggregates
	pl.updateTimeAggregates(now, operation, particleCount)
}

// updateTimeAggregates updates all time-based aggregation buckets
func (pl *PerformanceLogger) updateTimeAggregates(t time.Time, operation string, particleCount int64) {
	for _, period := range aggregationPeriods {
		bucketKey := pl.getTimeBucketKey(t, period)

		// Get or create aggregate for this time bucket
		aggregate, exists := pl.timeAggregates[bucketKey]
		if !exists {
			aggregate = &TimeAggregate{
				StartTime:      pl.getBucketStartTime(t, period),
				EndTime:        pl.getBucketEndTime(t, period),
				Duration:       period.Duration,
				OperationTypes: make(map[string]int64),
				LastUpdated:    t,
			}
			pl.timeAggregates[bucketKey] = aggregate
		}

		// Update aggregate statistics
		aggregate.Operations++
		aggregate.Particles += particleCount
		aggregate.OperationTypes[operation]++
		aggregate.LastUpdated = t

		// Calculate current rates
		elapsed := t.Sub(aggregate.StartTime).Seconds()
		if elapsed > 0 {
			currentOpsPerSec := float64(aggregate.Operations) / elapsed

			// Update peaks
			if currentOpsPerSec > aggregate.PeakOpsPerSec {
				aggregate.PeakOpsPerSec = currentOpsPerSec
			}

			if particleCount > aggregate.PeakParticles {
				aggregate.PeakParticles = particleCount
			}

			// Update averages
			aggregate.AvgOpsPerSec = currentOpsPerSec
			aggregate.AvgParticles = float64(aggregate.Particles) / float64(aggregate.Operations)
		}
	}
}

func (pl *PerformanceLogger) startPeriodicLogging() {
	wasmLog("[PERF-LOGGER]", pl.id, "startPeriodicLogging called")
	go func() {
		for {
			select {
			case <-pl.stopCh:
				wasmLog("[PERF-LOGGER]", pl.id, "Periodic logging stopped and goroutine exiting.")
				pl.stopped = true
				return
			case <-time.After(pl.logInterval):
				if pl.stopped {
					wasmLog("[PERF-LOGGER]", pl.id, "Periodic logging tick ignored, logger stopped.")
					return
				}
				wasmLog("[PERF-LOGGER]", pl.id, "Periodic logging tick.")
				pl.logAggregatedStats()
			}
		}
	}()
}

// logAggregatedStats logs the accumulated performance statistics
func (pl *PerformanceLogger) logAggregatedStats() {
	defer func() {
		if r := recover(); r != nil {
			wasmLog("[PERF-LOGGER]", pl.id, "Panic recovered in logAggregatedStats:", r)
		}
	}()
	pl.mutex.Lock()
	defer pl.mutex.Unlock()

	if pl.stopped {
		wasmLog("[PERF-LOGGER]", pl.id, "logAggregatedStats called after stop, ignoring.")
		return
	}

	if pl.totalOperations == 0 {
		wasmLog("[PERF-LOGGER]", pl.id, "No operations to log.")
		wasmLog("[PERF-LOGGER]", pl.id, "logAggregatedStats complete.")
		return
	}

	currentTime := time.Now()
	duration := currentTime.Sub(pl.lastLogTime)
	var avgParticlesPerOp int64 = 0
	var opsPerSec float64 = 0
	// Defensive: avoid division by zero
	if pl.totalOperations > 0 {
		avgParticlesPerOp = pl.totalParticles / pl.totalOperations
	}
	if duration.Seconds() > 0 {
		opsPerSec = float64(pl.totalOperations) / duration.Seconds()
	}

	wasmLog("[PERF-SUMMARY]",
		"Duration:", duration.Round(time.Second),
		"Total Operations:", pl.totalOperations,
		"Total Particles:", pl.totalParticles,
		"Avg Particles/Op:", avgParticlesPerOp,
		"Ops/Sec:", opsPerSec)

	// Log operation breakdown
	for operation, count := range pl.operationCounts {
		wasmLog("[PERF-BREAKDOWN]", operation+":", count, "operations")
	}

	// Log time-based aggregates summary
	pl.logTimeAggregatesSummary(currentTime)

	// Reset immediate counters
	pl.operationCounts = make(map[string]int64)
	pl.totalParticles = 0
	pl.totalOperations = 0
	pl.lastLogTime = currentTime

	// Always log completion
	wasmLog("[PERF-LOGGER] logAggregatedStats complete.")
}

// logTimeAggregatesSummary logs summaries of time-based aggregations
func (pl *PerformanceLogger) logTimeAggregatesSummary(currentTime time.Time) {
	aggregateSummary := make(map[string]*TimeAggregate)

	// Get the most recent aggregate for each period
	for _, period := range aggregationPeriods {
		bucketKey := pl.getTimeBucketKey(currentTime, period)
		if aggregate, exists := pl.timeAggregates[bucketKey]; exists {
			aggregateSummary[period.Name] = aggregate
		}
	}

	// Log summary for each period that has data
	for periodName, aggregate := range aggregateSummary {
		if aggregate.Operations > 0 {
			wasmLog("[TIME-AGGREGATE]",
				"Period:", periodName,
				"Operations:", aggregate.Operations,
				"Particles:", aggregate.Particles,
				"Peak Ops/Sec:", fmt.Sprintf("%.2f", aggregate.PeakOpsPerSec),
				"Avg Ops/Sec:", fmt.Sprintf("%.2f", aggregate.AvgOpsPerSec),
				"Avg Particles/Op:", fmt.Sprintf("%.0f", aggregate.AvgParticles))
		}
	}

	// Log historical trends (compare with previous periods)
	pl.logHistoricalTrends(currentTime)
}

// logHistoricalTrends compares current performance with historical data
func (pl *PerformanceLogger) logHistoricalTrends(currentTime time.Time) {
	// Compare current hour with previous hour
	currentHourKey := pl.getTimeBucketKey(currentTime, TimePeriod{"hour", time.Hour, "2006-01-02T15"})
	previousHourKey := pl.getTimeBucketKey(currentTime.Add(-time.Hour), TimePeriod{"hour", time.Hour, "2006-01-02T15"})

	if currentHour, exists1 := pl.timeAggregates[currentHourKey]; exists1 {
		if previousHour, exists2 := pl.timeAggregates[previousHourKey]; exists2 && previousHour.Operations > 0 {
			// Calculate hour-over-hour changes
			opsChange := float64(currentHour.Operations-previousHour.Operations) / float64(previousHour.Operations) * 100
			particlesChange := float64(currentHour.Particles-previousHour.Particles) / float64(previousHour.Particles) * 100

			wasmLog("[TREND-ANALYSIS]",
				"Hour-over-hour:",
				fmt.Sprintf("Ops: %+.1f%%", opsChange),
				fmt.Sprintf("Particles: %+.1f%%", particlesChange))
		}
	}

	// Compare current day with previous day
	currentDayKey := pl.getTimeBucketKey(currentTime, TimePeriod{"day", 24 * time.Hour, "2006-01-02"})
	previousDayKey := pl.getTimeBucketKey(currentTime.AddDate(0, 0, -1), TimePeriod{"day", 24 * time.Hour, "2006-01-02"})

	if currentDay, exists1 := pl.timeAggregates[currentDayKey]; exists1 {
		if previousDay, exists2 := pl.timeAggregates[previousDayKey]; exists2 && previousDay.Operations > 0 {
			// Calculate day-over-day changes
			opsChange := float64(currentDay.Operations-previousDay.Operations) / float64(previousDay.Operations) * 100
			particlesChange := float64(currentDay.Particles-previousDay.Particles) / float64(previousDay.Particles) * 100

			wasmLog("[TREND-ANALYSIS]",
				"Day-over-day:",
				fmt.Sprintf("Ops: %+.1f%%", opsChange),
				fmt.Sprintf("Particles: %+.1f%%", particlesChange))
		}
	}
}

// Only log from main thread (browser JS context)
func isMainThread() bool {
	return js.Global().Get("document").Truthy()
}

func wasmLog(args ...interface{}) {
	var strArgs []interface{}
	for _, arg := range args {
		strArgs = append(strArgs, fmt.Sprintf("%v", arg))
	}
	js.Global().Get("console").Call("log", strArgs...)
}

func flattenArgs(args []interface{}) []interface{} {
	var out []interface{}
	for _, arg := range args {
		switch v := arg.(type) {
		case []interface{}:
			out = append(out, flattenArgs(v)...)
		default:
			out = append(out, v)
		}
	}
	return out
}

func wasmWarn(args ...interface{}) {
	fmt.Println(append([]interface{}{"[WARN]"}, flattenArgs(args)...)...)
}

func wasmError(args ...interface{}) {
	fmt.Println(append([]interface{}{"[ERROR]"}, flattenArgs(args)...)...)
}

func (pl *PerformanceLogger) Stop() {
	defer func() {
		if r := recover(); r != nil {
			wasmLog("[PERF-LOGGER]", pl.id, "Panic recovered in Stop:", r)
		}
	}()
	if pl.stopCh != nil && !pl.stopped {
		wasmLog("[PERF-LOGGER]", pl.id, "Stop called, closing stopCh.")
		close(pl.stopCh)
		pl.stopCh = nil
		pl.stopped = true
	} else {
		wasmLog("[PERF-LOGGER]", pl.id, "Stop called, but already stopped or stopCh nil.")
	}
}
