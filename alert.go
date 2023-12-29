package main

// alertState
const (
	AllGood int = iota
	Alerting
)

type AlertReporter interface {
	AlertStarted(timestamp int, avgHits float32)
	AlertRecovered(timestamp int, avgHits float32)
}

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
		al.advanceReportingTo(logTs - al.futureBufferSize)
	}

	// if timestamps between largestTs and logTs are missing, clear out the data
	// TODO: what if logTs is way in the future?  should only clear out the buffer once
	for ts := al.largestTs + 1; ts <= logTs; ts++ {
		i := al.indexFor(ts)
		al.cyclicBufferHits[i] = 0
		al.largestTs = logTs
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

func (al *AlertListener) advanceReportingTo(advancedToTs int) {
	for nextTs := al.largestReportedTs + 1; nextTs <= advancedToTs; nextTs++ {
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
