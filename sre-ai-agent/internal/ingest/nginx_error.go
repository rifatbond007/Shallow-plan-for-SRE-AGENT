package ingest

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var nginxErrorRe = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) \[(\w+)\] (\S+): \*(\d+) (.+)$`)

var nginxErrorLevels = map[string]Severity{
	"debug":   SevDebug,
	"info":    SevInfo,
	"notice":  SevInfo,
	"warn":    SevWarn,
	"error":   SevError,
	"crit":    SevError,
	"alert":   SevFatal,
	"emerg":   SevFatal,
}

type NginxErrorParser struct{}

func (p *NginxErrorParser) Source() Source { return SourceNginxError }

func (p *NginxErrorParser) Parse(r io.Reader) ([]LogEntry, error) {
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

func (p *NginxErrorParser) parseLine(line string) (LogEntry, error) {
	m := nginxErrorRe.FindStringSubmatch(line)
	if m == nil {
		return LogEntry{}, fmt.Errorf("nginx error: no match")
	}

	ts, err := time.Parse("2006/01/02 15:04:05", m[1])
	if err != nil {
		return LogEntry{}, fmt.Errorf("nginx error: bad timestamp: %w", err)
	}

	level := strings.ToLower(m[2])
	sev, ok := nginxErrorLevels[level]
	if !ok {
		sev = SevInfo
	}

	fields := map[string]string{
		"pid_tid": m[3],
		"cid":     m[4],
	}

	return LogEntry{
		ID:        entryID(line, SourceNginxError),
		Timestamp: ts.UTC(),
		Source:    SourceNginxError,
		Severity:  sev,
		Message:   m[5],
		Fields:    fields,
		Raw:       line,
	}, nil
}
