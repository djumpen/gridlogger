package service

import (
	"strings"
	"testing"
	"time"
)

func TestShouldNotifyTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		previous string
		next     string
		want     bool
	}{
		{name: "available to outage", previous: "available", next: "outage", want: true},
		{name: "outage to available", previous: "outage", next: "available", want: true},
		{name: "no change", previous: "outage", next: "outage", want: false},
		{name: "unknown to available", previous: "unknown", next: "available", want: false},
		{name: "available to unknown", previous: "available", next: "unknown", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldNotifyTransition(tc.previous, tc.next)
			if got != tc.want {
				t.Fatalf("unexpected result: got %v want %v", got, tc.want)
			}
		})
	}
}

func TestBuildProjectStatusNotificationMessage(t *testing.T) {
	t.Parallel()

	project := Project{
		ID:   7,
		Name: "Коновальця 36Б",
		City: "Київ",
		Slug: "36b",
	}

	outage := buildProjectStatusNotificationMessage(project, projectStatusOutage, 2*time.Hour+30*time.Minute)
	if outage == "" {
		t.Fatal("outage message is empty")
	}
	if !strings.HasPrefix(outage, "🔴 Світло відсутнє: ") {
		t.Fatalf("unexpected outage prefix: %q", outage)
	}
	if !strings.Contains(outage, "\nСвітло було доступне 2год 30хв\n") {
		t.Fatalf("outage message must contain previous available duration: %q", outage)
	}
	if strings.Contains(outage, "м. Київ") {
		t.Fatalf("outage message must not include city: %q", outage)
	}
	if !strings.Contains(outage, "https://svitlo.homes/36b") {
		t.Fatalf("outage message must include public link: %q", outage)
	}

	available := buildProjectStatusNotificationMessage(project, projectStatusAvailable, 75*time.Minute)
	if available == "" {
		t.Fatal("available message is empty")
	}
	if !strings.HasPrefix(available, "🟢 Світло зʼявилося: ") {
		t.Fatalf("unexpected available prefix: %q", available)
	}
	if !strings.Contains(available, "\nВідключення тривало 1год 15хв\n") {
		t.Fatalf("available message must contain previous outage duration: %q", available)
	}
}
