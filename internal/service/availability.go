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
	GetFirstPing(ctx context.Context, projectID int) (time.Time, bool, error)
	GetLastPingAt(ctx context.Context, projectID int) (time.Time, bool, error)
	// GetRunStart returns the earliest ping in [from, to] that ends the most recent gap longer than maxGap
	// (i.e. the start of the current uninterrupted run of pings). found=false when there are no pings in range.
	GetRunStart(ctx context.Context, projectID int, from, to time.Time, maxGap time.Duration) (time.Time, bool, error)
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

	firstPing, hasFirstPing, err := s.store.GetFirstPing(ctx, projectID)
	if err != nil {
		return nil, Stats{}, fmt.Errorf("get first ping: %w", err)
	}

	sort.Slice(pings, func(i, j int) bool { return pings[i].Before(pings[j]) })

	intervals := make([]Interval, 0)
	currentStart := from
	currentStatus := statusAt(from, pings, s.outageThreshold, firstPing, hasFirstPing)

	for t := from.Add(30 * time.Second); !t.After(to); t = t.Add(30 * time.Second) {
		st := statusAt(t, pings, s.outageThreshold, firstPing, hasFirstPing)
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

// statusLookback caps how far back CurrentStatus searches for the start of an uninterrupted run.
const statusLookback = 30 * 24 * time.Hour

// CurrentStatus is the live power state of a project, for integrations such as Home Assistant.
type CurrentStatus struct {
	ProjectID              int        `json:"projectId"`
	Status                 string     `json:"status"` // available | outage | unknown
	Since                  *time.Time `json:"since"`  // when the current status began (capped at 30 days for "available")
	LastPingAt             *time.Time `json:"lastPingAt"`
	OutageThresholdSeconds int        `json:"outageThresholdSeconds"`
	CheckedAt              time.Time  `json:"checkedAt"`
}

// CurrentStatus uses the same rule as BuildIntervals/statusAt: available while the latest ping is at most
// outageThreshold old. Outage starts at lastPing+threshold; availability starts at the first ping after the
// last gap longer than the threshold.
func (s *AvailabilityService) CurrentStatus(ctx context.Context, projectID int) (CurrentStatus, error) {
	if projectID <= 0 {
		return CurrentStatus{}, errors.New("projectID must be positive")
	}
	now := s.nowFn()
	out := CurrentStatus{
		ProjectID:              projectID,
		Status:                 "unknown",
		OutageThresholdSeconds: int(s.outageThreshold / time.Second),
		CheckedAt:              now,
	}

	last, found, err := s.store.GetLastPingAt(ctx, projectID)
	if err != nil {
		return CurrentStatus{}, fmt.Errorf("get last ping: %w", err)
	}
	if !found {
		return out, nil
	}
	last = last.UTC()
	out.LastPingAt = &last

	if last.Before(now.Add(-s.outageThreshold)) {
		since := last.Add(s.outageThreshold)
		out.Status, out.Since = "outage", &since
		return out, nil
	}

	start, found, err := s.store.GetRunStart(ctx, projectID, now.Add(-statusLookback), now, s.outageThreshold)
	if err != nil {
		return CurrentStatus{}, fmt.Errorf("get run start: %w", err)
	}
	out.Status = "available"
	if found {
		start = start.UTC()
		out.Since = &start
	}
	return out, nil
}

func statusAt(point time.Time, pings []time.Time, threshold time.Duration, firstPing time.Time, hasFirstPing bool) string {
	if !hasFirstPing || point.Before(firstPing) {
		return "unknown"
	}

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
		} else if iv.Status == "outage" {
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
