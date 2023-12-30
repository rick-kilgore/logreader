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
	done()
}

type SimpleStreamReader struct {
	listeners []StructuredLogListener
	fields    map[string]int
}

func NewSimpleStreamReader() *SimpleStreamReader {
	return &SimpleStreamReader{
		fields: map[string]int{},
	}
}

func (r *SimpleStreamReader) AddListener(listener StructuredLogListener) {
	if r.findListener(listener) < 0 {
		r.listeners = append(r.listeners, listener)
	}
}

func (r *SimpleStreamReader) findListener(target StructuredLogListener) int {
	for i, listener := range r.listeners {
		if listener == target {
			return i
		}
	}
	return -1
}

func (r *SimpleStreamReader) RemoveListener(target StructuredLogListener) bool {
	for i, listener := range r.listeners {
		if listener == target {
			r.listeners = append(r.listeners[:i], r.listeners[i+1:]...)
			return true
		}
	}
	return false
}

func (r *SimpleStreamReader) ProcessStructuredLog(stream io.Reader) error {
	rdr := csv.NewReader(stream)
	if err := r.readHeader(rdr); err != nil && err != io.EOF {
		return err
	}

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

func (r *SimpleStreamReader) readHeader(rdr *csv.Reader) error {
	record, err := rdr.Read()
	if err != nil {
		return err
	}
	for i, name := range record {
		r.fields[name] = i
	}
	return nil
}

func (r *SimpleStreamReader) parseRecord(record []string) (int, map[string]string, error) {
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
