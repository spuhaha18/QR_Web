package logging

import "testing"

func TestLevelOf(t *testing.T) {
	cases := []struct {
		line   string
		want   Level
		wantOK bool
	}{
		{"2026-06-19 12:00:00,000 INFO app MainThread : label generated", INFO, true},
		{"2026-06-19 12:00:00,000 WARNING app MainThread : backup failed", WARNING, true},
		{"2026-06-19 12:00:00,000 ERROR app MainThread : boom", ERROR, true},
		{"2026-06-19 12:00:00,000 DEBUG app MainThread : trace", DEBUG, true},
		// A message merely containing "INFO" must not be read as INFO level.
		{"2026-06-19 12:00:00,000 ERROR app MainThread : INFO string in body", ERROR, true},
		{"garbage line", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := LevelOf(c.line)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("LevelOf(%q) = (%v, %v), want (%v, %v)", c.line, got, ok, c.want, c.wantOK)
		}
	}
}
