package logparser

import (
	"regexp"
	"strings"
)

// LineMatcher defines a single line condition using contains and/or regex.
type LineMatcher struct {
	Contains  string
	RegexText string         `mapstructure:"regex"`
	Regex     *regexp.Regexp `mapstructure:"-"`
}

// Rule defines an ordered set of checks and metadata for a log match.
type Rule struct {
	Name     string
	Checks   []LineMatcher `mapstructure:"patterns"`
	MaxLines int           `mapstructure:"maxlines"`
	Solution string        `mapstructure:"solution"`
	Category string        `mapstructure:"category"`
}

func (r Rule) getNeededLineCount() int {
	if len(r.Checks) <= 1 {
		return 1
	}
	return max(len(r.Checks), r.MaxLines)
}

func (m *LineMatcher) CheckLine(l string) bool {
	if m.Contains != "" && !strings.Contains(l, m.Contains) {
		return false
	}
	if m.Regex != nil {
		return m.Regex.MatchString(l)
	}
	return m.Contains != ""
}
