package pipeline

import (
	"regexp"

	"github.com/sahilm/fuzzy"

	"github.com/renaldid/logpilot/pkg/logentry"
)

// FilterOptions configures how Apply filters a slice of log entries.
// A nil map means "all accepted" for that dimension.
type FilterOptions struct {
	// EnabledServices maps service name → true when that service should be shown.
	// nil = show all services.
	EnabledServices map[string]bool

	// EnabledLevels maps LogLevel → true when entries of that level should be shown.
	// nil = show all levels.
	EnabledLevels map[logentry.LogLevel]bool

	// Query is the search string. Empty = no text filter.
	Query string

	// RegexMode switches text search from fuzzy to regex.
	RegexMode bool
}

// Apply returns only the entries that satisfy all active filters.
// Returns an error only when RegexMode is true and Query is an invalid regex.
func Apply(entries []logentry.LogEntry, opts FilterOptions) ([]logentry.LogEntry, error) {
	result := filterByService(entries, opts.EnabledServices)
	result = filterByLevel(result, opts.EnabledLevels)

	if opts.Query == "" {
		return result, nil
	}

	if opts.RegexMode {
		return filterByRegex(result, opts.Query)
	}
	return filterByFuzzy(result, opts.Query), nil
}

func filterByService(entries []logentry.LogEntry, enabled map[string]bool) []logentry.LogEntry {
	if len(enabled) == 0 {
		return entries
	}
	out := make([]logentry.LogEntry, 0, len(entries))
	for _, e := range entries {
		if enabled[e.Service] {
			out = append(out, e)
		}
	}
	return out
}

func filterByLevel(entries []logentry.LogEntry, enabled map[logentry.LogLevel]bool) []logentry.LogEntry {
	if len(enabled) == 0 {
		return entries
	}
	out := make([]logentry.LogEntry, 0, len(entries))
	for _, e := range entries {
		if enabled[e.Level] {
			out = append(out, e)
		}
	}
	return out
}

func filterByFuzzy(entries []logentry.LogEntry, query string) []logentry.LogEntry {
	targets := make([]string, len(entries))
	for i, e := range entries {
		targets[i] = e.Message
	}

	matches := fuzzy.Find(query, targets)
	out := make([]logentry.LogEntry, 0, len(matches))
	for _, m := range matches {
		out = append(out, entries[m.Index])
	}
	return out
}

func filterByRegex(entries []logentry.LogEntry, pattern string) ([]logentry.LogEntry, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	out := make([]logentry.LogEntry, 0, len(entries))
	for _, e := range entries {
		if re.MatchString(e.Message) {
			out = append(out, e)
		}
	}
	return out, nil
}
