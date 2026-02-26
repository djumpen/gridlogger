package service

import (
	"context"
	"testing"
	"time"
)

type memoryStore struct {
	pings []time.Time
}

func (m *memoryStore) InsertPing(_ context.Context, _ int, ts time.Time) error {
	m.pings = append(m.pings, ts)
	return nil
}

func (m *memoryStore) GetPingsBetween(_ context.Context, _ int, from, to time.Time) ([]time.Time, error) {
	out := make([]time.Time, 0)
	for _, p := range m.pings {
		if p.Before(from) || p.After(to) {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (m *memoryStore) GetFirstPing(_ context.Context, _ int) (time.Time, bool, error) {
	if len(m.pings) == 0 {
		return time.Time{}, false, nil
	}
	first := m.pings[0]
	for _, p := range m.pings[1:] {
		if p.Before(first) {
			first = p
		}
	}
	return first, true, nil
}

func TestBuildIntervals(t *testing.T) {
	base := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	store := &memoryStore{
		pings: []time.Time{
			base.Add(30 * time.Second),
			base.Add(90 * time.Second),
			base.Add(6 * time.Minute),
		},
	}
	svc := NewAvailabilityService(store, 2*time.Minute)

	intervals, stats, err := svc.BuildIntervals(context.Background(), 1, base, base.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("BuildIntervals error: %v", err)
	}
	if len(intervals) == 0 {
		t.Fatalf("expected intervals")
	}
	if stats.AvailabilityPercent <= 0 || stats.AvailabilityPercent >= 100 {
		t.Fatalf("expected mixed availability, got %+v", stats)
	}
}

func TestBuildIntervals_StatsIgnoreFutureTime(t *testing.T) {
	base := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	store := &memoryStore{}
	svc := NewAvailabilityService(store, 2*time.Minute)
	svc.nowFn = func() time.Time {
		return base.Add(2 * time.Hour)
	}

	_, stats, err := svc.BuildIntervals(context.Background(), 1, base, base.Add(24*time.Hour))
	if err != nil {
		t.Fatalf("BuildIntervals error: %v", err)
	}

	if stats.TotalOutageHours != 0 {
		t.Fatalf("expected zero outage when no first ping exists, got %.1f", stats.TotalOutageHours)
	}
	if stats.TotalAvailableHours != 0 {
		t.Fatalf("expected no available time, got %.1f", stats.TotalAvailableHours)
	}
	if stats.AvailabilityPercent != 0 {
		t.Fatalf("expected 0%% availability, got %.1f", stats.AvailabilityPercent)
	}
}

func TestBuildIntervals_PreFirstPingIsUnknown(t *testing.T) {
	base := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	firstPing := base.Add(3 * time.Minute)
	store := &memoryStore{
		pings: []time.Time{firstPing},
	}
	svc := NewAvailabilityService(store, 2*time.Minute)

	intervals, stats, err := svc.BuildIntervals(context.Background(), 1, base, base.Add(10*time.Minute))
	if err != nil {
		t.Fatalf("BuildIntervals error: %v", err)
	}

	if len(intervals) < 3 {
		t.Fatalf("expected at least 3 intervals, got %d", len(intervals))
	}

	if intervals[0].Status != "unknown" {
		t.Fatalf("expected first interval to be unknown, got %q", intervals[0].Status)
	}
	if !intervals[0].Start.Equal(base) {
		t.Fatalf("expected unknown interval start at window start, got %s", intervals[0].Start)
	}
	if !intervals[0].End.Equal(firstPing) {
		t.Fatalf("expected unknown interval to end at first ping, got %s", intervals[0].End)
	}

	if stats.TotalOutageHours <= 0.0 {
		t.Fatalf("expected outage time after threshold passes, got %.1f", stats.TotalOutageHours)
	}
	if stats.AvailabilityPercent <= 0.0 || stats.AvailabilityPercent >= 100.0 {
		t.Fatalf("expected mixed availability after first ping, got %.1f", stats.AvailabilityPercent)
	}
}
