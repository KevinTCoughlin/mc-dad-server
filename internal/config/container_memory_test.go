package config

import "testing"

func TestExplicitContainerPlan(t *testing.T) {
	t.Parallel()

	plan, err := ExplicitContainerPlan("4G")
	if err != nil {
		t.Fatalf("ExplicitContainerPlan() error = %v", err)
	}
	if plan.Heap != "4G" || plan.Limit != "5G" || plan.Percent != 0 {
		t.Fatalf("plan = %+v, want heap 4G, limit 5G, percent 0", plan)
	}

	plan, err = ExplicitContainerPlan(" 512M ")
	if err != nil {
		t.Fatalf("ExplicitContainerPlan() error = %v", err)
	}
	if plan.Heap != "512M" || plan.Limit != "1536M" || plan.Percent != 0 {
		t.Fatalf("plan = %+v, want heap 512M, limit 1536M, percent 0", plan)
	}

	if _, err := ExplicitContainerPlan("auto"); err == nil {
		t.Fatal("expected auto to be rejected as an explicit heap")
	}
	if _, err := ExplicitContainerPlan("75%"); err == nil {
		t.Fatal("expected a percentage to be rejected as an explicit heap")
	}
}

func TestAutoContainerPlanUsesHeapFromRAM(t *testing.T) {
	t.Parallel()

	const (
		mib = 1024 * 1024
		gib = 1024 * mib
	)
	for _, total := range []uint64{512 * mib, gib, 2 * gib, 4 * gib, 8 * gib, 16 * gib, 64 * gib} {
		want, tight, err := HeapFromRAM(total)
		if err != nil {
			t.Fatalf("HeapFromRAM(%d) error = %v", total, err)
		}
		plan, gotTight, err := AutoContainerPlan(total)
		if err != nil {
			t.Fatalf("AutoContainerPlan(%d) error = %v", total, err)
		}
		if plan.Limit != want || plan.Heap != "" || plan.Percent != ContainerHeapPercent || gotTight != tight {
			t.Fatalf("AutoContainerPlan(%d) = %+v tight=%v, want limit %s percent %d tight=%v",
				total, plan, gotTight, want, ContainerHeapPercent, tight)
		}
	}
}

func TestAutoContainerPlanRejectsZero(t *testing.T) {
	t.Parallel()

	if _, _, err := AutoContainerPlan(0); err == nil {
		t.Fatal("expected an error for 0 RAM")
	}
}

func TestMemoryLimitForHeap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"2G", "3G"},
		{"4G", "5G"},
		{"8G", "9G"},
		{"512M", "1536M"},
		{"1024M", "2048M"},
		{"", "3G"},
		{"bad", "3G"},
	}
	for _, tt := range tests {
		if got := MemoryLimitForHeap(tt.input); got != tt.want {
			t.Errorf("MemoryLimitForHeap(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
