package main

import (
	"fmt"
	"os"
)

const statsDurationSeconds = 10
const alertDurationSeconds = 120
const alertAvgHitsLimit = 20.0
const topSectionCount = 3

// TODO:
//	- alerts
//	- use a heap for storing sections
//	- listeners API
//	- use csv field names

// notes for writeup
// - terse local scope variable names in golang
// - could have used heap for periodic reporter
// - created AlertReporter for testing - but certainly seems useful

type MainReporter struct{}

func (r *MainReporter) AlertStarted(timestamp int, avgHits float32) {
	fmt.Printf("[1;31mHigh traffic generated an alert - hits = %.3f, triggered at time %d[m\n", avgHits, timestamp)
}

func (r *MainReporter) AlertRecovered(timestamp int, avgHits float32) {
	fmt.Printf("[1;32mHigh traffic alert recovered at time %d - hits = %.3f[m\n", timestamp, avgHits)
}

func main() {
	csvfile := os.Args[1]
	f, err := os.Open(csvfile)
	if err != nil {
		fmt.Printf("failed to open %s: %v\n", csvfile, err)
	}

	logReader := NewLogReader()
	logReader.AddListener(NewPeriodicStatsLogger(statsDurationSeconds))
	logReader.AddListener(NewAlertListener(alertDurationSeconds, alertAvgHitsLimit, &MainReporter{}))
	err = logReader.ProcessStructuredLog(f)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
