package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type testAlertReporter struct {
	AlertTimes   []int
	RecoverTimes []int
}

func (r *testAlertReporter) AlertStarted(timestamp int, avgHits float32) {
	r.AlertTimes = append(r.AlertTimes, timestamp)
}
func (r *testAlertReporter) AlertRecovered(timestamp int, avgHits float32) {
	r.RecoverTimes = append(r.RecoverTimes, timestamp)
}
func (r *testAlertReporter) Reset() {
	r.AlertTimes = nil
	r.RecoverTimes = nil
}

func TestAlertListener_logEvent(t *testing.T) {
	reporter := testAlertReporter{}

	tests := []struct {
		name             string
		alertPeriod      int
		futureBufferSize int
		limitAvg         float32
		hits             []int
		wantAlertTimes   []int
		wantRecoverTimes []int
	}{
		{
			name:             "basic",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         2.0,
			hits:             []int{0, 1, 3, 1, 2, 2, 3, 3, 3, 4, 5, 6, 8},
			wantAlertTimes:   []int{4},
			wantRecoverTimes: []int{6},
		},
		{
			name:             "too aggressive on recovery? does not recover at 6",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         2.0,
			hits:             []int{0, 1, 3, 1, 2, 2, 3, 3, 3, 4, 5, 6, 6, 8},
			wantAlertTimes:   []int{4},
			wantRecoverTimes: nil,
		},
		{
			name:             "alert ts too late? should be ts=2, not 3",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         2.0,
			hits:             []int{0, 0, 1, 0, 1, 2, 2, 3, 2, 2, 2, 4},
			wantAlertTimes:   []int{2},
			wantRecoverTimes: nil,
		},
		{
			name:             "recovery too late? should be ts=6, not 12",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         2.0,
			hits:             []int{0, 1, 3, 1, 2, 2, 3, 3, 3, 4, 5, 12},
			wantAlertTimes:   []int{4},
			wantRecoverTimes: []int{6},
		},
		{
			name:             "no alert",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         1.0,
			hits:             []int{0, 1, 2, 3, 5, 6, 7, 8, 10, 12},
			wantAlertTimes:   nil,
			wantRecoverTimes: nil,
		},
		{
			name:             "no alert 2",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         2.0,
			hits:             []int{0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 12},
			wantAlertTimes:   nil,
			wantRecoverTimes: nil,
		},
		{
			name:             "empty",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         2.0,
			hits:             []int{},
			wantAlertTimes:   nil,
			wantRecoverTimes: nil,
		},
		{
			name:             "concentrated",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         2.0,
			hits:             []int{0, 0, 0, 0, 0, 0, 0, 0, 0, 3, 4, 3, 5, 7},
			wantAlertTimes:   []int{3},
			wantRecoverTimes: []int{5},
		},
		{
			name:             "concentrated and some late",
			alertPeriod:      5,
			futureBufferSize: 4,
			limitAvg:         2.0,
			hits:             []int{0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 4, 3, 8, 9},
			wantAlertTimes:   []int{0},
			wantRecoverTimes: []int{5},
		},
		{
			name:             "log arrives late - but within future buffer margin",
			alertPeriod:      5,
			futureBufferSize: 2,
			limitAvg:         1.0,
			hits:             []int{1, 2, 3, 4, 1, 7, 8},
			wantAlertTimes:   []int{4},
			wantRecoverTimes: []int{6},
		},
		{
			name:             "errant recovery - late ts=6 logs should not cause recovery msg at 6",
			alertPeriod:      3,
			futureBufferSize: 3,
			limitAvg:         1.0,
			hits:             []int{1, 2, 3, 4, 5, 7, 6, 6, 8, 9},
			wantAlertTimes:   []int{3},
			wantRecoverTimes: nil,
		},
	}
	dummyEvent := map[string]string{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter.Reset()
			al := NewAlertListener(tt.alertPeriod, tt.futureBufferSize, tt.limitAvg, &reporter)
			for _, ts := range tt.hits {
				al.logEvent(ts, dummyEvent)
			}
			al.done()
			require.Equal(t, tt.wantAlertTimes, reporter.AlertTimes, "alert times")
			require.Equal(t, tt.wantRecoverTimes, reporter.RecoverTimes, "recovery times")
		})
	}
}
