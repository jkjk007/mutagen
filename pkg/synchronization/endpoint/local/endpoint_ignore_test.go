package local

import (
	"testing"

	mutagenignore "github.com/mutagen-io/mutagen/pkg/synchronization/core/ignore/mutagen"
)

// TestEventPathIgnored verifies that watch events occurring inside ignored
// content are recognized as irrelevant. Event paths are relative to the
// synchronization root and use fastpath-style separators.
func TestEventPathIgnored(t *testing.T) {
	// Create an ignorer with a directory ignore and a pattern ignore.
	ignorer, err := mutagenignore.NewIgnorer([]string{"/.git", "*.tmp"})
	if err != nil {
		t.Fatal("unable to create ignorer:", err)
	}

	// Create an endpoint with just the ignorer set. eventPathIgnored only
	// consults that field.
	endpoint := &endpoint{ignorer: ignorer}

	// Define test cases.
	tests := []struct {
		path   string
		expect bool
	}{
		// The ignored directory itself.
		{".git", true},
		// Content beneath an ignored directory.
		{".git/index.lock", true},
		{".git/objects/ab/cdef", true},
		// Files matching a pattern ignore.
		{"build/output.tmp", true},
		{"output.tmp", true},
		// Regular content.
		{"main.go", false},
		{"src/main.go", false},
		{"src/.github/workflows/ci.yml", false},
	}

	// Run test cases.
	for _, test := range tests {
		if actual := endpoint.eventPathIgnored(test.path); actual != test.expect {
			t.Errorf("path %q: expected %v, got %v", test.path, test.expect, actual)
		}
	}
}

// TestEventPathIgnoredNoIgnores verifies that nothing is filtered when there
// are no ignore patterns.
func TestEventPathIgnoredNoIgnores(t *testing.T) {
	// Create an ignorer without patterns.
	ignorer, err := mutagenignore.NewIgnorer(nil)
	if err != nil {
		t.Fatal("unable to create ignorer:", err)
	}

	// Create an endpoint with just the ignorer set.
	endpoint := &endpoint{ignorer: ignorer}

	// Ensure that paths, including ones that look like ignores, are not
	// filtered.
	for _, path := range []string{".git/index.lock", "output.tmp", "src/main.go"} {
		if endpoint.eventPathIgnored(path) {
			t.Errorf("path %q was unexpectedly ignored", path)
		}
	}
}
