package main

// alertState
const (
	AllGood int = iota
	Alerting
)

// This interface is used by the AlertListener to report alert state back to
// the caller.  It must be implemented by the caller and passed to
// NewAlertListener().
type AlertReporter interface {
	AlertStarted(timestamp int, avgHits float32)
	AlertRecovered(timestamp int, avgHits float32)
}

// AlertListener is the entry point for using the alerting logic in this file.
// It implements the SimpleStreamReader's associated interface
// StructuredLogListener.
//
// AlertListener keeps a cyclic buffer if size periodSeconds + futureBufferSize.
// The futureBufferSize space is a staging space where hits for the most
// recently seen timestamps can accumulate until we have confidence that we all
// hits have been reported.  Once the rolling window advances far enough that a
// given timestamp is no longer in the futureBufferSize space, then it is
// analyzed for a potential alert or recovery event.
type AlertListener struct {
	limitAvg          float32
	periodSeconds     int
	cyclicBufferHits  []int
	futureBufferSize  int
	largestTs         int
	largestReportedTs int
	totalHits         int
	alertState        int
	reporter          AlertReporter
}

func NewAlertListener(periodSeconds, futureBufferSize int, limitAvg float32, reporter AlertReporter) *AlertListener {
	return &AlertListener{
		limitAvg:          limitAvg,
		periodSeconds:     periodSeconds,
		cyclicBufferHits:  make([]int, periodSeconds+futureBufferSize),
		futureBufferSize:  futureBufferSize,
		largestTs:         -1,
		largestReportedTs: -1,
		alertState:        AllGood,
		reporter:          reporter,
	}
}

func (al *AlertListener) logEvent(logTs int, _ map[string]string) {
	logTsIdx := al.indexFor(logTs)
	if al.largestTs == -1 {
		al.largestTs = logTs
		al.largestReportedTs = logTs - 1
		al.cyclicBufferHits[logTsIdx]++
		return
	}

	if logTs <= al.largestReportedTs-al.periodSeconds {
		// if logTs is too far in the past, ignore it
		return
	}

	if logTs > al.largestTs {
		// NOTE: advanceReportingTo() also clears out the cells holding
		// older data so we can reuse them for future hits.
		al.advanceReportingTo(logTs - al.futureBufferSize)
	}

	al.cyclicBufferHits[logTsIdx]++
	if logTs <= al.largestReportedTs {
		al.totalHits++
		al.checkAlertingAt(al.largestReportedTs)
	}
}

func (al *AlertListener) done() {
}

func (al *AlertListener) indexFor(ts int) int {
	if ts < 0 {
		ts = ts + len(al.cyclicBufferHits)
	}
	return ts % len(al.cyclicBufferHits)
}

// This method advances the rolling window one or more seconds until
// advanceToTs is moved from the futureBufferSize space into the portion of the
// cyclic buffer can be analyzed for alert events.  As it advances, it runs the
// alert analysis for each newly included timestamp.
func (al *AlertListener) advanceReportingTo(advanceToTs int) {
	for nextTs := al.largestReportedTs + 1; nextTs <= advanceToTs; nextTs++ {
		oldTs := nextTs - al.periodSeconds
		oldIdx, nextIdx := al.indexFor(oldTs), al.indexFor(nextTs)
		al.totalHits -= al.cyclicBufferHits[oldIdx]
		al.totalHits += al.cyclicBufferHits[nextIdx]
		al.checkAlertingAt(nextTs)
		al.cyclicBufferHits[oldIdx] = 0
		al.largestReportedTs = nextTs
	}
}

func (al *AlertListener) checkAlertingAt(reportTs int) {
	avgHits := float32(al.totalHits) / float32(al.periodSeconds)
	if al.alertState == AllGood && avgHits >= al.limitAvg {
		al.alertState = Alerting
		al.reporter.AlertStarted(reportTs, avgHits)

	} else if al.alertState == Alerting && avgHits < al.limitAvg {
		al.alertState = AllGood
		al.reporter.AlertRecovered(reportTs, avgHits)
	}
}
