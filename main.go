package main

import (
	"fmt"
	"os"
)

const statsDurationSeconds = 10
const alertDurationSeconds = 120
const futureBufferSize = 5
const alertAvgHitsLimit = 10.0
const topSectionCount = 3

// MainReporter implements the AlertListener's AlertReporter interface and the
// PeriodicStatsLogger's PeriodicReporter interface.
type MainReporter struct{}

func (r *MainReporter) AlertStarted(timestamp int, avgHits float32) {
	fmt.Printf("[1;31mHigh traffic generated an alert - hits = %.3f, triggered at time %d[m\n", avgHits, timestamp)
}

func (r *MainReporter) AlertRecovered(timestamp int, avgHits float32) {
	fmt.Printf("[1;32mHigh traffic alert recovered at time %d - hits = %.3f[m\n", timestamp, avgHits)
}

func (r *MainReporter) ReportStats(timestamp, periodSeconds int, sectionStats []*SectionStats) {
	fmt.Printf("%d: top sections for last %d seconds:", timestamp, periodSeconds)
	for i := 0; i < topSectionCount && i < len(sectionStats); i++ {
		stats := sectionStats[i]
		fmt.Printf(" %s=%d", stats.name, stats.hits)
	}
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, "usage: go run . <input-csv-file>\n")
		os.Exit(1)
	}
	csvfile := os.Args[1]
	f, err := os.Open(csvfile)
	if err != nil {
		fmt.Printf("failed to open %s: %v\n", csvfile, err)
	}

	logReader := NewSimpleStreamReader()
	reporter := &MainReporter{}
	logReader.AddListener(NewPeriodicStatsLogger(statsDurationSeconds, reporter))
	logReader.AddListener(NewAlertListener(alertDurationSeconds, futureBufferSize, alertAvgHitsLimit, reporter))
	err = logReader.ProcessStructuredLog(f)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
