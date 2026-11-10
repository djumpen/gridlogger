package service

import (
	"context"
	"testing"
	"time"
)

func newStatusSvc(pings []time.Time, threshold time.Duration, now time.Time) *AvailabilityService {
	svc := NewAvailabilityService(&memoryStore{pings: pings}, threshold)
	svc.nowFn = func() time.Time { return now }
	return svc
}

// every30s returns pings at start, start+30s, ... up to and including end.
func every30s(start, end time.Time) []time.Time {
	var out []time.Time
	for t := start; !t.After(end); t = t.Add(30 * time.Second) {
		out = append(out, t)
	}
	return out
}

func TestCurrentStatusUnknownWithoutPings(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	st, err := newStatusSvc(nil, 4*time.Minute, now).CurrentStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("CurrentStatus error: %v", err)
	}
	if st.Status != "unknown" || st.Since != nil || st.LastPingAt != nil {
		t.Fatalf("expected unknown with no timestamps, got %+v", st)
	}
	if st.OutageThresholdSeconds != 240 {
		t.Fatalf("expected threshold 240s, got %d", st.OutageThresholdSeconds)
	}
}

func TestCurrentStatusAvailableSinceEndOfLastGap(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	// power on 08:00-09:00, outage (no pings) 09:00-10:30, power back at 10:30 until now
	pings := every30s(now.Add(-4*time.Hour), now.Add(-3*time.Hour))
	restored := now.Add(-90 * time.Minute)
	pings = append(pings, every30s(restored, now.Add(-10*time.Second))...)

	st, err := newStatusSvc(pings, 4*time.Minute, now).CurrentStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("CurrentStatus error: %v", err)
	}
	if st.Status != "available" {
		t.Fatalf("expected available, got %q", st.Status)
	}
	if st.Since == nil || !st.Since.Equal(restored) {
		t.Fatalf("expected since %v, got %v", restored, st.Since)
	}
	if st.LastPingAt == nil || !st.LastPingAt.Equal(pings[len(pings)-1]) {
		t.Fatalf("unexpected lastPingAt %v", st.LastPingAt)
	}
}

func TestCurrentStatusShortGapsDoNotResetSince(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	// pings with one 3-minute gap (below the 4-minute threshold)
	pings := every30s(start, now.Add(-30*time.Minute))
	pings = append(pings, every30s(now.Add(-27*time.Minute), now)...)

	st, err := newStatusSvc(pings, 4*time.Minute, now).CurrentStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("CurrentStatus error: %v", err)
	}
	if st.Status != "available" || st.Since == nil || !st.Since.Equal(start) {
		t.Fatalf("expected available since %v, got %+v", start, st)
	}
}

func TestCurrentStatusOutageSinceLastPingPlusThreshold(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	last := now.Add(-20 * time.Minute)
	pings := every30s(now.Add(-2*time.Hour), last)

	st, err := newStatusSvc(pings, 4*time.Minute, now).CurrentStatus(context.Background(), 1)
	if err != nil {
		t.Fatalf("CurrentStatus error: %v", err)
	}
	want := last.Add(4 * time.Minute)
	if st.Status != "outage" || st.Since == nil || !st.Since.Equal(want) {
		t.Fatalf("expected outage since %v, got %+v", want, st)
	}
}

func TestCurrentStatusMatchesStatusAtBoundary(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	threshold := 4 * time.Minute
	for _, tc := range []struct {
		lastAgo time.Duration
		want    string
	}{
		{threshold, "available"}, // exactly at the threshold still counts as available (same as statusAt)
		{threshold + time.Second, "outage"},
	} {
		pings := every30s(now.Add(-time.Hour), now.Add(-tc.lastAgo))
		pings = append(pings, now.Add(-tc.lastAgo))
		st, err := newStatusSvc(pings, threshold, now).CurrentStatus(context.Background(), 1)
		if err != nil {
			t.Fatalf("CurrentStatus error: %v", err)
		}
		if got := statusAt(now, pings, threshold, pings[0], true); got != tc.want || st.Status != tc.want {
			t.Fatalf("lastAgo=%v: statusAt=%q CurrentStatus=%q want %q", tc.lastAgo, got, st.Status, tc.want)
		}
	}
}

func TestCurrentStatusRejectsInvalidProject(t *testing.T) {
	if _, err := newStatusSvc(nil, time.Minute, time.Now()).CurrentStatus(context.Background(), 0); err == nil {
		t.Fatal("expected error for projectID 0")
	}
}
