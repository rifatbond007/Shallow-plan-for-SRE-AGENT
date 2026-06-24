package ingest

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type WindowGrouper struct {
	Window  time.Duration
	MinSize int
}

func NewWindowGrouper(window time.Duration, minSize int) *WindowGrouper {
	return &WindowGrouper{
		Window:  window,
		MinSize: minSize,
	}
}

func DefaultGrouper() *WindowGrouper {
	return NewWindowGrouper(5*time.Minute, 3)
}

func (g *WindowGrouper) Group(entries []LogEntry) []Incident {
	if len(entries) == 0 {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	clusters := g.clusterByWindow(entries)

	var incidents []Incident
	for _, cluster := range clusters {
		if len(cluster) < g.MinSize {
			continue
		}
		incidents = append(incidents, g.buildIncident(cluster))
	}

	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].StartedAt.Before(incidents[j].StartedAt)
	})
	return incidents
}

func (g *WindowGrouper) clusterByWindow(entries []LogEntry) [][]LogEntry {
	var clusters [][]LogEntry
	current := []LogEntry{entries[0]}

	for i := 1; i < len(entries); i++ {
		timeDiff := entries[i].Timestamp.Sub(current[0].Timestamp)
		if timeDiff <= g.Window {
			current = append(current, entries[i])
		} else {
			clusters = append(clusters, current)
			current = []LogEntry{entries[i]}
		}
	}
	clusters = append(clusters, current)
	return clusters
}

func (g *WindowGrouper) buildIncident(entries []LogEntry) Incident {
	msgCount := make(map[string]int)
	var topMsg string
	var topCount int

	var errCount, warnCount int
	for _, e := range entries {
		msgCount[e.Message]++
		if msgCount[e.Message] > topCount {
			topCount = msgCount[e.Message]
			topMsg = e.Message
		}
		switch e.Severity {
		case SevError, SevFatal:
			errCount++
		case SevWarn:
			warnCount++
		}
	}

	started := entries[0].Timestamp
	ended := entries[len(entries)-1].Timestamp

	var sigs []string
	for _, e := range entries {
		if sig := extractSignature(e.Message); sig != "" {
			sigs = append(sigs, sig)
		}
	}
	sigs = uniqueStrings(sigs)

	id := fmt.Sprintf("inc_%x", started.UnixNano())[:16]

	return Incident{
		ID:         id,
		StartedAt:  started,
		EndedAt:    ended,
		Entries:    entries,
		Signatures: sigs,
		TopError:   topMsg,
		ErrorCount: errCount,
		WarnCount:  warnCount,
	}
}

func extractSignature(msg string) string {
	msg = strings.ToLower(msg)
	sigs := []struct {
		keyword string
		id      string
	}{
		{"nil pointer", "app_nil"},
		{"panic:", "app_panic"},
		{"context deadline exceeded", "app_deadline"},
		{"connect() failed", "nginx_502"},
		{"upstream timed out", "nginx_504"},
		{"connection refused", "db_conn"},
		{"sqlstate", "db_5xx"},
	}
	for _, s := range sigs {
		if strings.Contains(msg, s.keyword) {
			return s.id
		}
	}
	return ""
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
