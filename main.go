package main

import (
	"fmt"
	"os"
)

const statsDurationSeconds = 10
const alertDurationSeconds = 120
const futureBufferSize = 5
const alertAvgHitsLimit = 20.0
const topSectionCount = 3

// TODO:
//	- alerts
//	- use a heap for storing sections
//	- listeners API
//	- use csv field names
//	- FIXED: reporting recovery could come late, if there are seconds without any traffic
//	- FIXED: reporting recovery could be wrong if I have just advanced to the next second and haven't gotten all it's hits yet
//		- FIXED: I need to only report for periods after largestTs has advanced past it!!!

// notes for writeup
//	- terse local scope variable names in golang
//		- no spaces around +, -
//	- could have used heap for periodic reporter
//	- created AlertReporter for testing - but certainly seems useful
//	- it might report recovery for a time ts1 when I receive an early hit for ts2 > ts1
//		- if so, it will subsequently report a new alert it gets more hits for ts1
//		- should I give it some confidence buffer?  Just having received one datapoint for a future time period may not be enough
//		- see test "errant recovery"
//	- it's possible an alert start could be missed when a hit comes in for an earlier ts
//		- WONT FIX: this is the larger buffer thing - big comment in alert.go
//		- see test "log arrives too late"

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

	logReader := NewSimpleStreamReader()
	logReader.AddListener(NewPeriodicStatsLogger(statsDurationSeconds))
	logReader.AddListener(NewAlertListener(alertDurationSeconds, futureBufferSize, alertAvgHitsLimit, &MainReporter{}))
	err = logReader.ProcessStructuredLog(f)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}
