package metrics

import "testing"

func TestIoregStat(t *testing.T) {
	line := `"PerformanceStatistics" = {"Device Utilization %"=42,"Renderer Utilization %"=17,"Tiler Utilization %"=5,"In use system memory"=536870912}`

	tests := []struct {
		key  string
		want int64
	}{
		{"Device Utilization %", 42},
		{"Renderer Utilization %", 17},
		{"Tiler Utilization %", 5},
		{"In use system memory", 536870912},
		{"Missing Key", 0},
	}

	for _, tc := range tests {
		got := ioregStat(line, tc.key)
		if got != tc.want {
			t.Errorf("ioregStat(%q) = %d, want %d", tc.key, got, tc.want)
		}
	}
}

func TestIoregStatNonNumericValue(t *testing.T) {
	line := `"Some Key"=notanumber`
	if got := ioregStat(line, "Some Key"); got != 0 {
		t.Errorf("expected 0 for non-numeric value, got %d", got)
	}
}

func TestIoregStatEmptyLine(t *testing.T) {
	if got := ioregStat("", "Device Utilization %"); got != 0 {
		t.Errorf("expected 0 for empty line, got %d", got)
	}
}

func TestGpuFriendlyName(t *testing.T) {
	tests := []struct {
		ioClass   string
		isAppleSi bool
		cpuModel  string
		want      string
	}{
		{"AGXFamilyA", true, "Apple M2", "Apple M2 GPU"},
		{"AGXFamilyA", true, "", "Apple GPU"},
		{"", true, "Apple M1 Pro", "Apple M1 Pro GPU"},
		{"ATIRadeon", false, "", "AMD GPU"},
		{"AMDFoo", false, "", "AMD GPU"},
		{"IntelHD630", false, "", "Intel GPU"},
		{"SomeUnknownGPU", false, "", "SomeUnknownGPU"},
		{"", false, "", "GPU"},
	}

	for _, tc := range tests {
		got := gpuFriendlyName(tc.ioClass, tc.isAppleSi, tc.cpuModel)
		if got != tc.want {
			t.Errorf("gpuFriendlyName(%q, %v, %q) = %q, want %q",
				tc.ioClass, tc.isAppleSi, tc.cpuModel, got, tc.want)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct{ a, b, want int }{
		{3, 5, 3},
		{5, 3, 3},
		{4, 4, 4},
		{0, 1, 0},
		{-1, 0, -1},
	}
	for _, tc := range tests {
		if got := min(tc.a, tc.b); got != tc.want {
			t.Errorf("min(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
