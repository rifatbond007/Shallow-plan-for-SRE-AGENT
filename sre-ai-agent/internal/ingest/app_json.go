package ingest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type AppJSONParser struct{}

func (p *AppJSONParser) Source() Source { return SourceAppJSON }

func (p *AppJSONParser) Parse(r io.Reader) ([]LogEntry, error) {
	var entries []LogEntry
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		entry, err := p.parseLine(line)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func (p *AppJSONParser) parseLine(line string) (LogEntry, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return LogEntry{}, fmt.Errorf("app_json: %w", err)
	}

	ts := p.extractTime(raw)
	sev := p.extractSeverity(raw)
	msg := p.extractString(raw, "msg", "message")
	stack := p.extractString(raw, "stack", "stack_trace", "stacktrace")

	fields := make(map[string]string)
	for k, v := range raw {
		switch k {
		case "ts", "time", "timestamp", "level", "severity", "msg", "message", "error", "stack", "stack_trace", "stacktrace":
			continue
		}
		fields[k] = fmt.Sprintf("%v", v)
	}
	if errStr := p.extractString(raw, "error"); errStr != "" {
		fields["error"] = errStr
	}

	return LogEntry{
		ID:         entryID(line, SourceAppJSON),
		Timestamp:  ts,
		Source:     SourceAppJSON,
		Severity:   sev,
		Message:    msg,
		Fields:     fields,
		Raw:        line,
		StackTrace: stack,
	}, nil
}

func (p *AppJSONParser) extractTime(m map[string]any) time.Time {
	for _, key := range []string{"ts", "time", "timestamp"} {
		if v, ok := m[key]; ok {
			switch val := v.(type) {
			case string:
				if t, err := time.Parse(time.RFC3339, val); err == nil {
					return t.UTC()
				}
				if t, err := time.Parse("2006-01-02T15:04:05Z07:00", val); err == nil {
					return t.UTC()
				}
			case float64:
				return time.Unix(int64(val), 0).UTC()
			}
		}
	}
	return time.Now().UTC()
}

func (p *AppJSONParser) extractSeverity(m map[string]any) Severity {
	for _, key := range []string{"level", "severity"} {
		if v, ok := m[key]; ok {
			s := strings.ToUpper(fmt.Sprintf("%v", v))
			switch s {
			case "DEBUG":
				return SevDebug
			case "INFO":
				return SevInfo
			case "WARN", "WARNING":
				return SevWarn
			case "ERROR":
				return SevError
			case "FATAL", "CRIT", "CRITICAL":
				return SevFatal
			}
		}
	}
	return SevInfo
}

func (p *AppJSONParser) extractString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
