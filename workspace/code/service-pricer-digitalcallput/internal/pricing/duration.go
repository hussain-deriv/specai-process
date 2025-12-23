package pricing

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DurationType classifies the duration as time-based or tick-based
type DurationType string

const (
	DurationTypeTimeBased DurationType = "TIME_BASED"
	DurationTypeTickBased DurationType = "TICK_BASED"
)

// DurationUnit represents the unit of a duration
type DurationUnit string

const (
	DurationUnitSeconds DurationUnit = "s"
	DurationUnitMinutes DurationUnit = "m"
	DurationUnitHours   DurationUnit = "h"
	DurationUnitDays    DurationUnit = "d"
	DurationUnitTicks   DurationUnit = "t"
)

// Duration represents a parsed contract duration
type Duration struct {
	Value        int
	Unit         DurationUnit
	DurationType DurationType
}

var durationRegex = regexp.MustCompile(`^(\d+)([smhdt])$`)

// ParseDuration parses a duration string into a Duration struct
// Valid formats: \d+[smhdt] where s=seconds, m=minutes, h=hours, d=days, t=ticks
func ParseDuration(durationStr string) (*Duration, error) {
	matches := durationRegex.FindStringSubmatch(durationStr)
	if matches == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid duration format: %s. Expected format: \\d+[smhdt]", durationStr)
	}

	value, err := strconv.Atoi(matches[1])
	if err != nil || value <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid duration value: %s. Value must be a positive integer", matches[1])
	}

	unit := DurationUnit(matches[2])
	durationType := DurationTypeTimeBased

	switch unit {
	case DurationUnitSeconds, DurationUnitMinutes, DurationUnitHours, DurationUnitDays:
		durationType = DurationTypeTimeBased
	case DurationUnitTicks:
		durationType = DurationTypeTickBased
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Invalid duration unit: %s. Expected one of: s, m, h, d, t", matches[2])
	}

	return &Duration{
		Value:        value,
		Unit:         unit,
		DurationType: durationType,
	}, nil
}

// ToSeconds converts a time-based duration to seconds
// Returns an error if the duration is tick-based
func (d *Duration) ToSeconds() (int64, error) {
	if d.DurationType == DurationTypeTickBased {
		return 0, fmt.Errorf("cannot convert tick-based duration to seconds")
	}

	var seconds int64
	switch d.Unit {
	case DurationUnitSeconds:
		seconds = int64(d.Value)
	case DurationUnitMinutes:
		seconds = int64(d.Value) * 60
	case DurationUnitHours:
		seconds = int64(d.Value) * 3600
	case DurationUnitDays:
		seconds = int64(d.Value) * 86400
	default:
		return 0, fmt.Errorf("invalid time-based duration unit: %s", d.Unit)
	}

	return seconds, nil
}

// CalculateExpiryTime calculates the expiry time for a time-based duration
// For tick-based durations, this returns an error as expiry is tick-count based
func (d *Duration) CalculateExpiryTime(startTime time.Time) (time.Time, error) {
	seconds, err := d.ToSeconds()
	if err != nil {
		return time.Time{}, err
	}

	return startTime.Add(time.Duration(seconds) * time.Second), nil
}

// IsTimeBased returns true if the duration is time-based
func (d *Duration) IsTimeBased() bool {
	return d.DurationType == DurationTypeTimeBased
}

// IsTickBased returns true if the duration is tick-based
func (d *Duration) IsTickBased() bool {
	return d.DurationType == DurationTypeTickBased
}
