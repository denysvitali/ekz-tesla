package ekz

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ScheduleScheduler manages charging based on predefined tariff schedules
type ScheduleScheduler struct {
	autostartFunc   func() error
	highTariffTimes []TimeRange
	stopChan        chan struct{}
	wg              sync.WaitGroup
	mu              sync.RWMutex
	running         bool
}

// TimeRange represents a time range during the day
type TimeRange struct {
	StartHour   int
	StartMinute int
	EndHour     int
	EndMinute   int
	Weekdays    []time.Weekday // empty means all weekdays
}

// DefaultHighTariffSchedule returns the default EKZ high tariff schedule
// High tariff: Monday-Friday 07:00-20:00
func DefaultHighTariffSchedule() []TimeRange {
	return []TimeRange{
		{
			StartHour:   7,
			StartMinute: 0,
			EndHour:     20,
			EndMinute:   0,
			Weekdays:    []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		},
	}
}

// ParseTimeRangeString parses a time range string like "7:00-20:00" or "7:00-20:00:Mon,Tue,Wed,Thu,Fri"
func ParseTimeRangeString(s string) (TimeRange, error) {
	var tr TimeRange

	// Split by the last colon to separate weekdays from time range
	colonParts := strings.Split(s, ":")
	var timeRangePart string
	var weekdaysPart string

	if len(colonParts) >= 4 { // Has weekdays: "7:00-20:00:Mon,Tue,Wed,Thu,Fri"
		timeRangePart = strings.Join(colonParts[:len(colonParts)-1], ":")
		weekdaysPart = colonParts[len(colonParts)-1]
	} else if len(colonParts) == 3 { // Simple time range: "7:00-20:00"
		timeRangePart = s
	} else {
		return tr, fmt.Errorf("invalid time range format: %s", s)
	}

	// Parse time range part
	timeParts := strings.Split(timeRangePart, "-")
	if len(timeParts) != 2 {
		return tr, fmt.Errorf("invalid time range format: %s", timeRangePart)
	}

	// Parse start time
	startParts := strings.Split(timeParts[0], ":")
	if len(startParts) != 2 {
		return tr, fmt.Errorf("invalid start time format: %s", timeParts[0])
	}
	startHour, err := strconv.Atoi(startParts[0])
	if err != nil {
		return tr, fmt.Errorf("invalid start hour: %s", startParts[0])
	}
	startMinute, err := strconv.Atoi(startParts[1])
	if err != nil {
		return tr, fmt.Errorf("invalid start minute: %s", startParts[1])
	}

	// Parse end time
	endParts := strings.Split(timeParts[1], ":")
	if len(endParts) != 2 {
		return tr, fmt.Errorf("invalid end time format: %s", timeParts[1])
	}
	endHour, err := strconv.Atoi(endParts[0])
	if err != nil {
		return tr, fmt.Errorf("invalid end hour: %s", endParts[0])
	}
	endMinute, err := strconv.Atoi(endParts[1])
	if err != nil {
		return tr, fmt.Errorf("invalid end minute: %s", endParts[1])
	}

	tr.StartHour = startHour
	tr.StartMinute = startMinute
	tr.EndHour = endHour
	tr.EndMinute = endMinute

	// Parse weekdays if specified
	if weekdaysPart != "" {
		weekdayNames := strings.Split(weekdaysPart, ",")
		for _, name := range weekdayNames {
			name = strings.TrimSpace(name)
			switch strings.ToLower(name) {
			case "mon", "monday":
				tr.Weekdays = append(tr.Weekdays, time.Monday)
			case "tue", "tuesday":
				tr.Weekdays = append(tr.Weekdays, time.Tuesday)
			case "wed", "wednesday":
				tr.Weekdays = append(tr.Weekdays, time.Wednesday)
			case "thu", "thursday":
				tr.Weekdays = append(tr.Weekdays, time.Thursday)
			case "fri", "friday":
				tr.Weekdays = append(tr.Weekdays, time.Friday)
			case "sat", "saturday":
				tr.Weekdays = append(tr.Weekdays, time.Saturday)
			case "sun", "sunday":
				tr.Weekdays = append(tr.Weekdays, time.Sunday)
			default:
				return tr, fmt.Errorf("invalid weekday: %s", name)
			}
		}
	}

	return tr, nil
}

// NewScheduleScheduler creates a new scheduler based on time schedules
func NewScheduleScheduler(autostartFunc func() error, highTariffTimes []TimeRange) *ScheduleScheduler {
	if highTariffTimes == nil {
		highTariffTimes = DefaultHighTariffSchedule()
	}

	return &ScheduleScheduler{
		autostartFunc:   autostartFunc,
		highTariffTimes: highTariffTimes,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the schedule-based scheduling
func (ss *ScheduleScheduler) Start(ctx context.Context) error {
	ss.mu.Lock()
	if ss.running {
		ss.mu.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	ss.running = true
	ss.mu.Unlock()

	logrus.Info("Starting schedule-based scheduler")

	ss.wg.Add(1)
	go func() {
		defer ss.wg.Done()
		ss.run(ctx)
	}()

	return nil
}

// Stop gracefully stops the scheduler
func (ss *ScheduleScheduler) Stop() {
	ss.mu.Lock()
	if !ss.running {
		ss.mu.Unlock()
		return
	}
	ss.running = false
	ss.mu.Unlock()

	logrus.Info("Stopping schedule-based scheduler")
	close(ss.stopChan)
	ss.wg.Wait()
}

// IsRunning returns whether the scheduler is currently running
func (ss *ScheduleScheduler) IsRunning() bool {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.running
}

// run is the main scheduler loop
func (ss *ScheduleScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	// Initial check
	ss.checkAndCharge()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Context cancelled, stopping scheduler")
			return
		case <-ss.stopChan:
			logrus.Info("Stop signal received, stopping scheduler")
			return
		case <-ticker.C:
			ss.checkAndCharge()
		}
	}
}

// checkAndCharge checks if we should charge based on current time
func (ss *ScheduleScheduler) checkAndCharge() {
	now := time.Now()
	isHighTariff := ss.isHighTariffTime(now)

	logrus.Debugf("Current time: %s, High tariff: %v", now.Format("2006-01-02 15:04:05 Mon"), isHighTariff)

	// Only start charging during low tariff periods
	if isHighTariff {
		logrus.Debugf("Currently in high tariff period, skipping charge attempt")
		return
	}

	logrus.Info("Low tariff period detected, checking if we should start charging")

	err := ss.autostartFunc()
	if err != nil {
		logrus.Errorf("Failed to attempt autostart: %v", err)
	}
}

// isHighTariffTime checks if the given time falls within any high tariff period
func (ss *ScheduleScheduler) isHighTariffTime(t time.Time) bool {
	for _, tr := range ss.highTariffTimes {
		if ss.timeInRange(t, tr) {
			return true
		}
	}
	return false
}

// timeInRange checks if a given time falls within a TimeRange
func (ss *ScheduleScheduler) timeInRange(t time.Time, tr TimeRange) bool {
	// Check weekdays if specified
	if len(tr.Weekdays) > 0 {
		weekdayMatch := false
		for _, wd := range tr.Weekdays {
			if t.Weekday() == wd {
				weekdayMatch = true
				break
			}
		}
		if !weekdayMatch {
			return false
		}
	}

	// Convert time to minutes since midnight for easier comparison
	currentMinutes := t.Hour()*60 + t.Minute()
	startMinutes := tr.StartHour*60 + tr.StartMinute
	endMinutes := tr.EndHour*60 + tr.EndMinute

	// Handle ranges that cross midnight
	if endMinutes < startMinutes {
		return currentMinutes >= startMinutes || currentMinutes < endMinutes
	}

	return currentMinutes >= startMinutes && currentMinutes < endMinutes
}

// GetNextLowTariffPeriod returns the next time when low tariff begins
func (ss *ScheduleScheduler) GetNextLowTariffPeriod() time.Time {
	now := time.Now()

	// If we're already in a low tariff period, return current time
	if !ss.isHighTariffTime(now) {
		return now
	}

	// Find the end of the current high tariff period
	for i := 0; i < 24*60; i++ { // Check next 24 hours in 1-minute increments
		future := now.Add(time.Duration(i) * time.Minute)
		if !ss.isHighTariffTime(future) {
			return future
		}
	}

	// Fallback: return tomorrow at the same time
	return now.Add(24 * time.Hour)
}