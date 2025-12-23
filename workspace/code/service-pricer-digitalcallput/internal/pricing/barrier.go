package pricing

import (
	"fmt"
	"regexp"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BarrierType classifies the barrier specification
type BarrierType string

const (
	BarrierTypeRelative BarrierType = "RELATIVE"
	BarrierTypeAbsolute BarrierType = "ABSOLUTE"
	BarrierTypeNone     BarrierType = "NONE"
)

// Barrier represents a parsed barrier specification
type Barrier struct {
	Type            BarrierType
	RawValue        string  // Original input value
	CalculatedValue float64 // Resolved barrier price
}

var (
	relativeBarrierRegex = regexp.MustCompile(`^([+-])(\d+\.?\d*)$`)
	absoluteBarrierRegex = regexp.MustCompile(`^(\d+\.?\d*)$`)
)

// ParseBarrier parses a barrier string into a Barrier struct
// Valid formats: "+50", "-100" (relative), "1.2345" (absolute), or empty (none)
func ParseBarrier(barrierStr string) (*Barrier, error) {
	if barrierStr == "" {
		return &Barrier{
			Type:     BarrierTypeNone,
			RawValue: "",
		}, nil
	}

	// Check for relative barrier
	if matches := relativeBarrierRegex.FindStringSubmatch(barrierStr); matches != nil {
		value, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid barrier format: %s", barrierStr)
		}

		if matches[1] == "-" {
			value = -value
		}

		return &Barrier{
			Type:            BarrierTypeRelative,
			RawValue:        barrierStr,
			CalculatedValue: value,
		}, nil
	}

	// Check for absolute barrier
	if matches := absoluteBarrierRegex.FindStringSubmatch(barrierStr); matches != nil {
		value, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid barrier format: %s", barrierStr)
		}

		if value <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "Absolute barrier must be positive: %s", barrierStr)
		}

		return &Barrier{
			Type:            BarrierTypeAbsolute,
			RawValue:        barrierStr,
			CalculatedValue: value,
		}, nil
	}

	return nil, status.Errorf(codes.InvalidArgument, "Invalid barrier format: %s", barrierStr)
}

// ResolveBarrier calculates the final barrier value based on the entry spot
// For relative barriers: entry_spot + relative_value (in pips)
// For absolute barriers: the value itself
// For none: entry_spot
func (b *Barrier) ResolveBarrier(entrySpot float64) (float64, error) {
	switch b.Type {
	case BarrierTypeRelative:
		// Relative barrier is in pips (0.00001 for most FX pairs)
		// For simplicity, we'll use 0.00001 as pip value
		pipValue := 0.00001
		return entrySpot + (b.CalculatedValue * pipValue), nil

	case BarrierTypeAbsolute:
		return b.CalculatedValue, nil

	case BarrierTypeNone:
		return entrySpot, nil

	default:
		return 0, fmt.Errorf("unknown barrier type: %s", b.Type)
	}
}

// IsRelative returns true if the barrier is relative to entry spot
func (b *Barrier) IsRelative() bool {
	return b.Type == BarrierTypeRelative
}

// IsAbsolute returns true if the barrier is an absolute value
func (b *Barrier) IsAbsolute() bool {
	return b.Type == BarrierTypeAbsolute
}

// IsNone returns true if no barrier was specified
func (b *Barrier) IsNone() bool {
	return b.Type == BarrierTypeNone
}
