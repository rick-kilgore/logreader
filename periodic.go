package main

import (
	"fmt"
	"regexp"
	"sort"
)

type SectionStats struct {
	name string
	hits int
}

const RequestFieldName = "request"

type PeriodicReporter interface {
	ReportStats(timestamp, periodSeconds int, stats []*SectionStats)
}

type PeriodicStatsLogger struct {
	periodSeconds int
	periodStart   int
	sectionRegex  *regexp.Regexp
	hitsBySection map[string]*SectionStats
	reporter      PeriodicReporter
}

func NewPeriodicStatsLogger(periodSeconds int, reporter PeriodicReporter) *PeriodicStatsLogger {
	re := regexp.MustCompile("(/[^/\\s]+)")
	return &PeriodicStatsLogger{
		periodSeconds: periodSeconds,
		periodStart:   0,
		sectionRegex:  re,
		hitsBySection: map[string]*SectionStats{},
		reporter:      reporter,
	}
}

func (psl *PeriodicStatsLogger) logEvent(ts int, log map[string]string) {
	if ts-psl.periodStart >= psl.periodSeconds {
		psl.logStats()
		psl.periodStart = ts - (ts % psl.periodSeconds)
		psl.hitsBySection = map[string]*SectionStats{}
	}
	section := psl.sectionRegex.FindString(log[RequestFieldName])
	if section == "" {
		fmt.Printf("could not determine section from '%s'\n", section)
	}
	if stats, ok := psl.hitsBySection[section]; ok {
		stats.hits++
	} else {
		psl.hitsBySection[section] = &SectionStats{section, 1}
	}
}

func (psl *PeriodicStatsLogger) done() {
	psl.logStats()
}

func (psl *PeriodicStatsLogger) logStats() {
	if len(psl.hitsBySection) > 0 {
		var sections []*SectionStats
		for _, secstats := range psl.hitsBySection {
			sections = append(sections, secstats)
		}
		sort.Slice(sections, func(i, j int) bool {
			return sections[i].hits > sections[j].hits ||
				(sections[i].hits == sections[j].hits && sections[i].name < sections[j].name)
		})
		psl.reporter.ReportStats(psl.periodStart+psl.periodSeconds, psl.periodSeconds, sections)
	}
}
