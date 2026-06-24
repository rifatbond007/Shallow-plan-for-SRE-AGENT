package analysis

import (
	"regexp"

	"github.com/rifatbond007/sre-ai-agent/internal/ingest"
)

var DefaultPatterns = []Pattern{
	{ID: "nginx_502", Severity: ingest.SevError, Regex: `connect\(\) failed.*upstream`, Label: "Upstream connection failure"},
	{ID: "nginx_504", Severity: ingest.SevError, Regex: `upstream timed out`, Label: "Upstream timeout"},
	{ID: "app_panic", Severity: ingest.SevFatal, Regex: `panic: runtime error:`, Label: "Runtime panic"},
	{ID: "app_nil", Severity: ingest.SevFatal, Regex: `nil pointer dereference`, Label: "Nil pointer dereference"},
	{ID: "app_deadline", Severity: ingest.SevError, Regex: `context deadline exceeded`, Label: "Deadline exceeded"},
	{ID: "db_conn", Severity: ingest.SevError, Regex: `connection refused.*postgres`, Label: "Database connection refused"},
	{ID: "db_5xx", Severity: ingest.SevError, Regex: `SQLSTATE 5`, Label: "PostgreSQL server error"},
}

type PatternMatcher struct {
	patterns []Pattern
	compiled []*regexp.Regexp
}

func NewPatternMatcher() *PatternMatcher {
	pm := &PatternMatcher{patterns: DefaultPatterns}
	for _, p := range DefaultPatterns {
		pm.compiled = append(pm.compiled, regexp.MustCompile(p.Regex))
	}
	return pm
}

func (pm *PatternMatcher) Match(inc ingest.Incident) *PatternMatch {
	for i, p := range pm.patterns {
		for _, entry := range inc.Entries {
			if pm.compiled[i].MatchString(entry.Message) {
				var lines []string
				for _, e := range inc.Entries {
					if pm.compiled[i].MatchString(e.Message) {
						lines = append(lines, e.Raw)
					}
				}
				return &PatternMatch{
					PatternID: p.ID,
					Label:     p.Label,
					Severity:  p.Severity,
					Lines:     lines,
				}
			}
		}
	}
	return nil
}
