package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

type PingStore interface {
	InsertPing(ctx context.Context, projectID int, ts time.Time) error
	GetPingsBetween(ctx context.Context, projectID int, from, to time.Time) ([]time.Time, error)
}

type Interval struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Status string    `json:"status"`
}

type Stats struct {
	AvailabilityPercent float64 `json:"availabilityPercent"`
	TotalAvailableHours float64 `json:"totalAvailableHours"`
	TotalOutageHours    float64 `json:"totalOutageHours"`
}

type AvailabilityService struct {
	store           PingStore
	outageThreshold time.Duration
	nowFn           func() time.Time
}

func NewAvailabilityService(store PingStore, outageThreshold time.Duration) *AvailabilityService {
	return &AvailabilityService{
		store:           store,
		outageThreshold: outageThreshold,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *AvailabilityService) RecordPing(ctx context.Context, projectID int) error {
	if projectID <= 0 {
		return errors.New("projectID must be positive")
	}
	return s.store.InsertPing(ctx, projectID, time.Now().UTC())
}

func (s *AvailabilityService) BuildIntervals(ctx context.Context, projectID int, from, to time.Time) ([]Interval, Stats, error) {
	if projectID <= 0 {
		return nil, Stats{}, errors.New("projectID must be positive")
	}
	if !to.After(from) {
		return nil, Stats{}, errors.New("to must be after from")
	}

	fetchFrom := from.Add(-s.outageThreshold)
	pings, err := s.store.GetPingsBetween(ctx, projectID, fetchFrom, to)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("get pings: %w", err)
	}

	sort.Slice(pings, func(i, j int) bool { return pings[i].Before(pings[j]) })

	intervals := make([]Interval, 0)
	currentStart := from
	currentStatus := statusAt(from, pings, s.outageThreshold)

	for t := from.Add(30 * time.Second); !t.After(to); t = t.Add(30 * time.Second) {
		st := statusAt(t, pings, s.outageThreshold)
		if st == currentStatus {
			continue
		}
		intervals = append(intervals, Interval{Start: currentStart, End: t, Status: currentStatus})
		currentStart = t
		currentStatus = st
	}
	intervals = append(intervals, Interval{Start: currentStart, End: to, Status: currentStatus})

	statsEnd := to
	now := s.nowFn()
	if now.Before(statsEnd) {
		statsEnd = now
	}
	stats := calcStats(intervals, from, statsEnd)
	return intervals, stats, nil
}

func statusAt(point time.Time, pings []time.Time, threshold time.Duration) string {
	minTs := point.Add(-threshold)
	for i := len(pings) - 1; i >= 0; i-- {
		if pings[i].After(point) {
			continue
		}
		if pings[i].Before(minTs) {
			break
		}
		return "available"
	}
	return "outage"
}

func calcStats(intervals []Interval, from, to time.Time) Stats {
	if !to.After(from) {
		return Stats{}
	}

	var available, outage time.Duration
	for _, iv := range intervals {
		start := iv.Start
		if start.Before(from) {
			start = from
		}
		end := iv.End
		if end.After(to) {
			end = to
		}
		if !end.After(start) {
			continue
		}

		d := end.Sub(start)
		if iv.Status == "available" {
			available += d
		} else {
			outage += d
		}
	}
	total := available + outage
	if total <= 0 {
		return Stats{}
	}
	toHours := func(d time.Duration) float64 {
		return float64(int((d.Hours()*10)+0.5)) / 10
	}
	pct := float64(int(((float64(available)/float64(total))*1000)+0.5)) / 10

	return Stats{
		AvailabilityPercent: pct,
		TotalAvailableHours: toHours(available),
		TotalOutageHours:    toHours(outage),
	}
}
