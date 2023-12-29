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
	limitAvg         float32
	cyclicBufferHits []int
	totalHits        int
	largestTs        int
	alertState       int
	reporter         AlertReporter
}

func NewAlertListener(periodSeconds int, limitAvg float32, reporter AlertReporter) *AlertListener {
	return &AlertListener{
		limitAvg:         limitAvg,
		cyclicBufferHits: make([]int, periodSeconds),
		largestTs:        0,
		alertState:       AllGood,
		reporter:         reporter,
	}
}

func (al *AlertListener) logEvent(ts int, _ map[string]string) {
	index := al.indexFor(ts)
	if al.largestTs == 0 {
		al.largestTs = ts
	}

	if ts <= al.largestTs-len(al.cyclicBufferHits) {
		// if ts is too far in the past, ignore it
		return

	} else if ts < al.largestTs {
		// if ts is recent, record it
		al.cyclicBufferHits[index]++
		al.totalHits++

	} else {
		// ts >= largestTs
		// TODO: what if ts is way in the future?  should only clear out the buffer once
		for _ts := al.largestTs + 1; _ts <= ts; _ts++ {
			i := al.indexFor(_ts)
			al.totalHits -= al.cyclicBufferHits[i]
			al.cyclicBufferHits[i] = 0
		}

		al.largestTs = ts
		al.cyclicBufferHits[index]++
		al.totalHits++
	}

	al.checkAlerting()
}

func (al *AlertListener) checkAlerting() {
	avgHits := float32(al.totalHits) / float32(len(al.cyclicBufferHits))
	if al.alertState == AllGood && avgHits >= al.limitAvg {
		al.alertState = Alerting
		al.reporter.AlertStarted(al.largestTs, avgHits)

	} else if al.alertState == Alerting && avgHits < al.limitAvg {
		al.alertState = AllGood
		al.reporter.AlertRecovered(al.largestTs, avgHits)
	}
}

func (al *AlertListener) indexFor(ts int) int {
	return ts % len(al.cyclicBufferHits)
}
