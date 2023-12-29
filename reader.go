package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

const TimestampField = "date"

type StructuredLogListener interface {
	logEvent(timestamp int, log map[string]string)
}

type LogReader struct {
	listeners []StructuredLogListener
	fields    map[string]int
}

func NewLogReader() *LogReader {
	return &LogReader{
		fields: map[string]int{},
	}
}

func (r *LogReader) AddListener(listener StructuredLogListener) {
	if r.findListener(listener) < 0 {
		r.listeners = append(r.listeners, listener)
	}
}

func (r *LogReader) findListener(target StructuredLogListener) int {
	for i, listener := range r.listeners {
		if listener == target {
			return i
		}
	}
	return -1
}

func (r *LogReader) RemoveListener(target StructuredLogListener) bool {
	for i, listener := range r.listeners {
		if listener == target {
			r.listeners = append(r.listeners[:i], r.listeners[i+1:]...)
			return true
		}
	}
	return false
}

func (r *LogReader) ProcessStructuredLog(stream io.Reader) error {
	rdr := csv.NewReader(stream)
	r.readHeader(rdr)

	for {
		record, err := rdr.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		ts, mapped, err := r.parseRecord(record)
		if err != nil {
			return err
		}

		for _, listener := range r.listeners {
			listener.logEvent(ts, mapped)
		}
	}
	return nil
}

func (r *LogReader) readHeader(rdr *csv.Reader) error {
	record, err := rdr.Read()
	if err != nil {
		return err
	}
	for i, name := range record {
		r.fields[name] = i
	}
	return nil
}

func (r *LogReader) parseRecord(record []string) (int, map[string]string, error) {
	if len(record) != len(r.fields) {
		return -1, nil, fmt.Errorf("expected %d fields, got %d: %v", len(r.fields), len(record), record)
	}

	mapped := map[string]string{}
	for fieldName, i := range r.fields {
		mapped[fieldName] = record[i]
	}

	strTs := mapped[TimestampField]
	ts, err := strconv.Atoi(strTs)
	if err != nil {
		return -1, nil, fmt.Errorf("could not parse timestamp field %s: '%s'", TimestampField, strTs)
	}

	return ts, mapped, nil
}
