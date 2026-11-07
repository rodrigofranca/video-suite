package jobcontracts

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Queue conventions shared by the API (producer) and the worker (consumer).
const (
	// StreamJobs is the Redis stream that carries processing jobs.
	StreamJobs = "jobs"
	// ConsumerGroup is the Redis consumer group the workers read from.
	ConsumerGroup = "workers"

	FieldJobID   = "job_id"
	FieldType    = "type"
	FieldPayload = "payload"
)

// Message is one entry of the jobs stream. The job row in PostgreSQL is the
// source of truth; the message only tells a worker which job to run.
type Message struct {
	JobID   string
	Type    string
	Payload json.RawMessage // JSON-encoded input struct for Type
}

// NewMessage builds a message from a typed job input (e.g. TranscodeHLSInput).
func NewMessage(jobID, jobType string, input any) (Message, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return Message{}, fmt.Errorf("encode %s payload: %w", jobType, err)
	}
	return Message{JobID: jobID, Type: jobType, Payload: payload}, nil
}

// Values returns the stream fields, ready for XADD.
func (m Message) Values() map[string]any {
	return map[string]any{
		FieldJobID:   m.JobID,
		FieldType:    m.Type,
		FieldPayload: string(m.Payload),
	}
}

// ParseMessage reads a message from the fields returned by XREADGROUP.
func ParseMessage(values map[string]any) (Message, error) {
	var m Message
	var errs []error
	read := func(field string) string {
		v, ok := values[field].(string)
		if !ok || v == "" {
			errs = append(errs, fmt.Errorf("missing field %q", field))
		}
		return v
	}
	m.JobID = read(FieldJobID)
	m.Type = read(FieldType)
	payload := read(FieldPayload)
	if err := errors.Join(errs...); err != nil {
		return m, err
	}
	if !json.Valid([]byte(payload)) {
		return m, fmt.Errorf("field %q is not valid JSON", FieldPayload)
	}
	m.Payload = json.RawMessage(payload)
	return m, nil
}

// Decode unmarshals the message payload into the input struct T.
func Decode[T any](m Message) (T, error) {
	var input T
	if err := json.Unmarshal(m.Payload, &input); err != nil {
		return input, fmt.Errorf("decode %s payload: %w", m.Type, err)
	}
	return input, nil
}
