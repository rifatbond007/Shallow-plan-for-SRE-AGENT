package ingest

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

var nginxAccessRe = regexp.MustCompile(`^(?P<ip>\S+)\s+\S+\s+\S+\s+\[(?P<ts>[^\]]+)\]\s+"(?P<method>\S+)\s+(?P<uri>\S+)\s+\S+"\s+(?P<status>\d{3})\s+(?P<bytes>\d+)\s+"(?P<ref>[^"]*)"\s+"(?P<ua>[^"]*)"`)

type NginxAccessParser struct{}

func (p *NginxAccessParser) Source() Source { return SourceNginxAccess }

func (p *NginxAccessParser) Parse(r io.Reader) ([]LogEntry, error) {
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

func (p *NginxAccessParser) parseLine(line string) (LogEntry, error) {
	m := nginxAccessRe.FindStringSubmatch(line)
	if m == nil {
		return LogEntry{}, fmt.Errorf("nginx access: no match")
	}

	ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[2])
	if err != nil {
		return LogEntry{}, fmt.Errorf("nginx access: bad timestamp: %w", err)
	}

	statusStr := m[5]
	status, _ := strconv.Atoi(statusStr)

	var sev Severity
	switch {
	case status >= 500:
		sev = SevError
	case status >= 400:
		sev = SevWarn
	default:
		sev = SevInfo
	}

	fields := map[string]string{
		"ip":      m[1],
		"method":  m[3],
		"uri":     m[4],
		"status":  statusStr,
		"bytes":   m[6],
		"referer": m[7],
		"ua":      m[8],
	}

	return LogEntry{
		ID:        entryID(line, SourceNginxAccess),
		Timestamp: ts.UTC(),
		Source:    SourceNginxAccess,
		Severity:  sev,
		Message:   fmt.Sprintf("%s %s → %s", m[3], m[4], statusStr),
		Fields:    fields,
		Raw:       line,
	}, nil
}
