package main

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type testPeriodicReporter struct {
	t       *testing.T
	reports map[int][]*SectionStats
}

func NewTestPeriodicReporter(t *testing.T) *testPeriodicReporter {
	return &testPeriodicReporter{
		t:       t,
		reports: map[int][]*SectionStats{},
	}
}

func (r *testPeriodicReporter) ReportStats(timestamp, periodSeconds int, stats []*SectionStats) {
	require.Nil(r.t, r.reports[timestamp])
	r.reports[timestamp] = stats
}

func TestPeriodicStatsLogger_logEvent(t *testing.T) {

	tests := []struct {
		name             string
		periodSeconds    int
		futureBufferSize int
		limitAvg         float32
		events           []map[string]string
		wantReports      map[int][]*SectionStats
	}{
		{
			name:             "basic",
			periodSeconds:    10,
			futureBufferSize: 2,
			limitAvg:         2.0,
			events: []map[string]string{
				{TimestampField: "0", RequestFieldName: "GET /api/login HTTP/1.0"},
				{TimestampField: "0", RequestFieldName: "GET /api/user HTTP/1.0"},
				{TimestampField: "1", RequestFieldName: "GET /report HTTP/1.0"},
				{TimestampField: "2", RequestFieldName: "GET /report HTTP/1.0"},
				{TimestampField: "2", RequestFieldName: "GET /api/account HTTP/1.0"},
				{TimestampField: "3", RequestFieldName: "GET /api/login HTTP/1.0"},
			},
			wantReports: map[int][]*SectionStats{
				10: {{"/api", 4}, {"/report", 2}},
			},
		},
		{
			name:             "multi",
			periodSeconds:    10,
			futureBufferSize: 2,
			limitAvg:         2.0,
			events: []map[string]string{
				{TimestampField: "0", RequestFieldName: "GET /api/login HTTP/1.0"},
				{TimestampField: "1", RequestFieldName: "GET /report HTTP/1.0"},
				{TimestampField: "2", RequestFieldName: "GET /report HTTP/1.0"},
				{TimestampField: "2", RequestFieldName: "GET /api/account HTTP/1.0"},
				{TimestampField: "3", RequestFieldName: "GET /api/login HTTP/1.0"},
				{TimestampField: "10", RequestFieldName: "GET /api/login HTTP/1.0"},
				{TimestampField: "11", RequestFieldName: "GET /api/user HTTP/1.0"},
				{TimestampField: "12", RequestFieldName: "GET /report/hits HTTP/1.0"},
			},
			wantReports: map[int][]*SectionStats{
				10: {{"/api", 3}, {"/report", 2}},
				20: {{"/api", 2}, {"/report", 1}},
			},
		},
		{
			name:             "late",
			periodSeconds:    10,
			futureBufferSize: 2,
			limitAvg:         2.0,
			events: []map[string]string{
				{TimestampField: "0", RequestFieldName: "GET /api/login HTTP/1.0"},
				{TimestampField: "1", RequestFieldName: "GET /report HTTP/1.0"},
				{TimestampField: "2", RequestFieldName: "GET /report HTTP/1.0"},
				{TimestampField: "2", RequestFieldName: "GET /api/account HTTP/1.0"},
				{TimestampField: "3", RequestFieldName: "GET /api/login HTTP/1.0"},
				{TimestampField: "10", RequestFieldName: "GET /api/login HTTP/1.0"},
				{TimestampField: "9", RequestFieldName: "GET /api/ponies HTTP/1.0"},
				{TimestampField: "11", RequestFieldName: "GET /api/user HTTP/1.0"},
				{TimestampField: "12", RequestFieldName: "GET /report/hits HTTP/1.0"},
			},
			wantReports: map[int][]*SectionStats{
				10: {{"/api", 3}, {"/report", 2}},
				20: {{"/api", 3}, {"/report", 1}},
			},
		},
		{
			name:             "empty",
			periodSeconds:    10,
			futureBufferSize: 2,
			limitAvg:         2.0,
			events:           []map[string]string{},
			wantReports:      map[int][]*SectionStats{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewTestPeriodicReporter(t)
			al := NewPeriodicStatsLogger(tt.periodSeconds, reporter)
			for _, event := range tt.events {
				ts, _ := strconv.Atoi(event[TimestampField])
				al.logEvent(ts, event)
			}
			al.done()
			require.Equal(t, tt.wantReports, reporter.reports)
		})
	}
}
