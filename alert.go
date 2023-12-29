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
		largestTs:        -1,
		alertState:       AllGood,
		reporter:         reporter,
	}
}

func (al *AlertListener) logEvent(ts int, _ map[string]string) {
	index := al.indexFor(ts)
	if al.largestTs == -1 {
		al.largestTs = ts
	}

	if ts <= al.largestTs-len(al.cyclicBufferHits) {
		// if ts is too far in the past, ignore it
		return

	} else if ts < al.largestTs {
		// if ts < al.largestTs, but recent, record the hit
		al.cyclicBufferHits[index]++
		al.totalHits++

	} else {
		// ts >= largestTs
		if ts > al.largestTs {
			// Checking immediately after ts advances could lead to a false recovery
			// detection if hits for earlier times subsequently arrive.
			al.checkAlerting()
		}

		// if timestamps between largestTs and ts are missing, clear out the data
		// TODO: what if ts is way in the future?  should only clear out the buffer once
		for _ts := al.largestTs + 1; _ts <= ts; _ts++ {
			i := al.indexFor(_ts)
			al.totalHits -= al.cyclicBufferHits[i]
			al.cyclicBufferHits[i] = 0

			// we may have recovered from an alert during periods of inactivity
			if _ts < ts && al.alertState == Alerting {
				al.largestTs = _ts
				al.checkAlerting()
			}
		}

		al.largestTs = ts
		al.cyclicBufferHits[index]++
		al.totalHits++
	}
}

func (al *AlertListener) done() {
	al.checkAlerting()
}

func (al *AlertListener) indexFor(ts int) int {
	return ts % len(al.cyclicBufferHits)
}

func (al *AlertListener) checkAlerting() {
	avgHits := float32(al.totalHits) / float32(len(al.cyclicBufferHits))
	if al.alertState == AllGood && avgHits >= al.limitAvg {
		al.alertState = Alerting
		al.reporter.AlertStarted(al.determineAlertTime(), avgHits)

	} else if al.alertState == Alerting && avgHits < al.limitAvg {
		al.alertState = AllGood
		al.reporter.AlertRecovered(al.largestTs, avgHits)
	}
}

// Since logs can come out of order, it's possible to learn that an
// alert happened at a time earlier than the current value of largestTs.
// This method checks to see if that was the case.  Most of the time, it
// should probably return largestTs.
//
// NOTE: this logic will not catch a potential alert where earlier hit
// counts that have been lost (overwritten) from the circular buffer are
// required to tip the average hit count over the limit.
//
// This miss could be fixed by making the circular buffer larger than the
// alerting period by some error of margin number of seconds.  The logic
// in this module would become a bit more complex in doing so.
//
// In addition, with the given problem statement of an alert time period
// of 120 seconds, it seems pretty unlikely that this will happen.
func (al *AlertListener) determineAlertTime() int {
	periodSeconds := len(al.cyclicBufferHits)
	avgHits := float32(al.totalHits) / float32(periodSeconds)
	if avgHits < al.limitAvg {
		return -1
	}

	ts := al.largestTs
	totalHits := al.totalHits
	for {
		tsHits := al.cyclicBufferHits[al.indexFor(ts)]
		if float32(totalHits-tsHits)/float32(periodSeconds) < al.limitAvg {
			break
		}
		totalHits -= tsHits
		ts--
	}
	return ts
}
