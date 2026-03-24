package logparser

import (
	"regexp"
	"strings"
)

// LineCheck defines a single line condition using contains and/or regex.
type LineCheck struct {
	Contains  string         `json:"contains"`
	RegexText string         `mapstructure:"regex" json:"regex"`
	Regex     *regexp.Regexp `mapstructure:"-" json:"-"`
}

// MatchRule defines an ordered set of checks and metadata for a log match.
type MatchRule struct {
	Name     string      `json:"name"`
	Checks   []LineCheck `mapstructure:"patterns" json:"patterns"`
	MaxLines int         `mapstructure:"maxlines" json:"maxLines"`
	Solution string      `mapstructure:"solution" json:"solution"`
	Category string      `mapstructure:"category" json:"category"`
}

func (r MatchRule) getNeededLineCount() int {
	if len(r.Checks) <= 1 {
		return 1
	}
	return max(len(r.Checks), r.MaxLines)
}

func (lc *LineCheck) CheckLine(l string) bool {
	if lc.Contains != "" && !strings.Contains(l, lc.Contains) {
		return false
	}
	if lc.Regex != nil {
		return lc.Regex.MatchString(l)
	}
	return lc.Contains != ""
}
