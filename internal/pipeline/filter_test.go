package pipeline

import (
	"testing"
	"time"

	"github.com/renaldid/logpilot/pkg/logentry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEntries = []logentry.LogEntry{
	{Timestamp: time.Now(), Service: "api", Level: logentry.LogLevelInfo, Message: "request received", Raw: ""},
	{Timestamp: time.Now(), Service: "api", Level: logentry.LogLevelError, Message: "internal server error", Raw: ""},
	{Timestamp: time.Now(), Service: "worker", Level: logentry.LogLevelWarn, Message: "slow job detected", Raw: ""},
	{Timestamp: time.Now(), Service: "worker", Level: logentry.LogLevelDebug, Message: "job completed", Raw: ""},
	{Timestamp: time.Now(), Service: "db", Level: logentry.LogLevelInfo, Message: "connection established", Raw: ""},
}

func TestApply_NoFilters(t *testing.T) {
	result, err := Apply(testEntries, FilterOptions{})
	require.NoError(t, err)
	assert.Equal(t, testEntries, result)
}

func TestApply_EmptyEntries(t *testing.T) {
	result, err := Apply(nil, FilterOptions{Query: "anything"})
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestApply_ServiceFilter_SingleService(t *testing.T) {
	opts := FilterOptions{
		EnabledServices: map[string]bool{"api": true},
	}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	for _, e := range result {
		assert.Equal(t, "api", e.Service)
	}
}

func TestApply_ServiceFilter_MultipleServices(t *testing.T) {
	opts := FilterOptions{
		EnabledServices: map[string]bool{"api": true, "db": true},
	}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestApply_ServiceFilter_NoneEnabled(t *testing.T) {
	opts := FilterOptions{
		EnabledServices: map[string]bool{"nonexistent": true},
	}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestApply_ServiceFilter_FalseValueExcludesService(t *testing.T) {
	opts := FilterOptions{
		EnabledServices: map[string]bool{"api": false, "worker": true},
	}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	for _, e := range result {
		assert.Equal(t, "worker", e.Service)
	}
}

func TestApply_LevelFilter_ErrorOnly(t *testing.T) {
	opts := FilterOptions{
		EnabledLevels: map[logentry.LogLevel]bool{logentry.LogLevelError: true},
	}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, logentry.LogLevelError, result[0].Level)
}

func TestApply_LevelFilter_MultipleLevel(t *testing.T) {
	opts := FilterOptions{
		EnabledLevels: map[logentry.LogLevel]bool{
			logentry.LogLevelInfo: true,
			logentry.LogLevelWarn: true,
		},
	}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Len(t, result, 3) // 2 INFO + 1 WARN
}

func TestApply_LevelFilter_NilMeansAll(t *testing.T) {
	opts := FilterOptions{EnabledLevels: nil}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Len(t, result, len(testEntries))
}

func TestApply_FuzzySearch(t *testing.T) {
	opts := FilterOptions{Query: "server"}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	// "internal server error" and "connection established" should match
	for _, e := range result {
		assert.Contains(t, e.Message, "server")
	}
}

func TestApply_FuzzySearch_NoMatch(t *testing.T) {
	opts := FilterOptions{Query: "xyzabcnotexist"}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestApply_RegexSearch(t *testing.T) {
	opts := FilterOptions{Query: "^request", RegexMode: true}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "request received", result[0].Message)
}

func TestApply_RegexSearch_CaseInsensitive(t *testing.T) {
	opts := FilterOptions{Query: "(?i)JOB", RegexMode: true}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestApply_RegexSearch_InvalidPattern(t *testing.T) {
	opts := FilterOptions{Query: "[invalid(regex", RegexMode: true}
	_, err := Apply(testEntries, opts)
	assert.Error(t, err)
}

func TestApply_CombinedFilters(t *testing.T) {
	opts := FilterOptions{
		EnabledServices: map[string]bool{"api": true},
		EnabledLevels:   map[logentry.LogLevel]bool{logentry.LogLevelError: true},
	}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "api", result[0].Service)
	assert.Equal(t, logentry.LogLevelError, result[0].Level)
}

func TestApply_RegexSearch_NoMatch(t *testing.T) {
	opts := FilterOptions{Query: "^NOMATCH$", RegexMode: true}
	result, err := Apply(testEntries, opts)
	require.NoError(t, err)
	assert.Empty(t, result)
}
