package version

import "testing"

func TestString_Defaults(t *testing.T) {
	result := String()
	expected := "dev (none) unknown"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestString_CustomValues(t *testing.T) {
	// Save originals
	origVersion, origCommit, origDate := Version, Commit, Date
	defer func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	}()

	Version = "1.2.3"
	Commit = "abc1234"
	Date = "2024-01-01"

	result := String()
	expected := "1.2.3 (abc1234) 2024-01-01"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
