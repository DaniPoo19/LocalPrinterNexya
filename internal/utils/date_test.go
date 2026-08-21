package utils

import (
	"strings"
	"testing"
	"time"
)

func TestParseAndFormatDate(t *testing.T) {
	// ISO format test
	isoInput := "2026-08-20T19:10:57.000Z"
	result := ParseAndFormatDate(isoInput)
	if !strings.Contains(result, "20/08/2026") || (!strings.Contains(result, "p. m.") && !strings.Contains(result, "a. m.")) {
		t.Errorf("Unexpected formatted date for ISO input '%s': got '%s'", isoInput, result)
	}

	// Empty format test (uses current time)
	emptyResult := ParseAndFormatDate("")
	if emptyResult == "" || (!strings.Contains(emptyResult, "p. m.") && !strings.Contains(emptyResult, "a. m.")) {
		t.Errorf("Unexpected formatted date for empty input: got '%s'", emptyResult)
	}

	// Already formatted test
	alreadyFormatted := "20/08/2026, 07:15 p. m."
	formattedResult := ParseAndFormatDate(alreadyFormatted)
	if formattedResult != alreadyFormatted {
		t.Errorf("Expected already formatted string to stay unchanged: got '%s', expected '%s'", formattedResult, alreadyFormatted)
	}
}

func TestFormatSpanishDateTime12h(t *testing.T) {
	testTime := time.Date(2026, 8, 20, 19, 15, 0, 0, bogotaLocation)
	formatted := FormatSpanishDateTime12h(testTime)
	expected := "20/08/2026, 07:15 p. m."
	if formatted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, formatted)
	}

	testMorning := time.Date(2026, 8, 20, 9, 5, 0, 0, bogotaLocation)
	formattedMorning := FormatSpanishDateTime12h(testMorning)
	expectedMorning := "20/08/2026, 09:05 a. m."
	if formattedMorning != expectedMorning {
		t.Errorf("Expected '%s', got '%s'", expectedMorning, formattedMorning)
	}
}
