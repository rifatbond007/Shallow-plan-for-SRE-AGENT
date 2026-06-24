package ingest

import (
	"crypto/sha1"
	"fmt"
	"io"
	"time"
)

type Source string

const (
	SourceNginxAccess Source = "nginx_access"
	SourceNginxError  Source = "nginx_error"
	SourceAppJSON     Source = "app_json"
)

type Severity string

const (
	SevDebug Severity = "DEBUG"
	SevInfo  Severity = "INFO"
	SevWarn  Severity = "WARN"
	SevError Severity = "ERROR"
	SevFatal Severity = "FATAL"
)

type LogEntry struct {
	ID         string
	Timestamp  time.Time
	Source     Source
	Severity   Severity
	Message    string
	Fields     map[string]string
	Raw        string
	StackTrace string
}

type Incident struct {
	ID         string
	StartedAt  time.Time
	EndedAt    time.Time
	Entries    []LogEntry
	Signatures []string
	TopError   string
	ErrorCount int
	WarnCount  int
}

type Parser interface {
	Source() Source
	Parse(reader io.Reader) ([]LogEntry, error)
}

type Normalizer interface {
	Normalize(raw string, src Source) (LogEntry, error)
}

type Grouper interface {
	Group(entries []LogEntry) []Incident
}

func entryID(raw string, src Source) string {
	h := sha1.Sum([]byte(string(src) + "|" + raw))
	return fmt.Sprintf("%x", h[:8])
}
