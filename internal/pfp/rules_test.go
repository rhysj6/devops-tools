package pfp

import (
	"regexp"
	"testing"
)

func HelperGetNeededLineCount(r Rule, e int, t *testing.T) {
	m := r.GetNeededLineCount()

	if r.GetNeededLineCount() != e {
		t.Fatalf("Expected %v lines got %v", e, m)
	}
}

func TestRuleGetNeededLineCountHandlesSingleCheck(t *testing.T) {
	r := Rule{Checks: []LineMatcher{{Contains: "Hello?"}}, MaxLines: 100}

	HelperGetNeededLineCount(r, 1, t)
}

func TestRuleGetNeededLineCountHandlesMoreHandlersThanLimit(t *testing.T) {
	r := Rule{Checks: []LineMatcher{{}}, MaxLines: 0}

	HelperGetNeededLineCount(r, 1, t)
}

func TestRuleGetNeededLineCountHandlesMaxLines(t *testing.T) {
	r := Rule{
		Checks:   []LineMatcher{{Contains: "Something"}, {Contains: "Something"}},
		MaxLines: 17,
	}

	HelperGetNeededLineCount(r, 17, t)
}

func HelperCheckLine(lm LineMatcher, l string, e bool, t *testing.T) {
	r := lm.CheckLine(l)

	if r != e {
		t.Fatalf("Expected %t but got %t", e, r)
	}
}

func TestCheckLineHandlesContainsWithNoRegex(t *testing.T) {
	line := "ERROR: Something failed"

	HelperCheckLine(LineMatcher{Contains: "ERROR"}, line, true, t)

	HelperCheckLine(LineMatcher{Contains: "ERROR: Something"}, line, true, t)

	HelperCheckLine(LineMatcher{Contains: "INFO"}, line, false, t)
}

func TestCheckLineHandlesContainsWithRegex(t *testing.T) {
	line := "ERROR: Something failed"

	re1, err := regexp.Compile(`ERROR:`)
	re2, err := regexp.Compile(`ERROR:Hi`)
	if err != nil {
		t.Fatal("Error compiling regex")
	}

	HelperCheckLine(LineMatcher{Contains: "ERROR", Regex: re1}, line, true, t)

	HelperCheckLine(LineMatcher{Contains: "ERROR:", Regex: re2}, line, false, t)
}

func TestCheckLineHandlesRegexWithNoContains(t *testing.T) {
	line := "ERROR: Something failed"

	re1, err := regexp.Compile(`ERROR:`)
	re2, err := regexp.Compile(`ERROR:Hi`)
	if err != nil {
		t.Fatal("Error compiling regex")
	}

	HelperCheckLine(LineMatcher{Regex: re1}, line, true, t)

	HelperCheckLine(LineMatcher{Regex: re2}, line, false, t)
}

func TestCheckLineHandlesNoValues(t *testing.T) {
	line := "ERROR: Something failed"

	HelperCheckLine(LineMatcher{}, line, false, t)
}
