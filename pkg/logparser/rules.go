package logparser

import (
	"regexp"
	"strings"
)

// LineMatcher defines a single line condition using contains and/or regex.
type LineMatcher struct {
	Contains  string         `json:"contains"`
	RegexText string         `mapstructure:"regex" json:"regex"`
	Regex     *regexp.Regexp `mapstructure:"-" json:"-"`
}

// Rule defines an ordered set of checks and metadata for a log match.
type Rule struct {
	Name     string        `json:"name"`
	Checks   []LineMatcher `mapstructure:"patterns" json:"patterns"`
	MaxLines int           `mapstructure:"maxlines" json:"maxLines"`
	Solution string        `mapstructure:"solution" json:"solution"`
	Category string        `mapstructure:"category" json:"category"`
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
