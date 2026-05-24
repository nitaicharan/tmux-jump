package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteResultFourFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "res")

	// Override the global outPath flag pointer for the duration of the test.
	saved := *outPath
	*outPath = path
	t.Cleanup(func() { *outPath = saved })

	writeResult(Match{Row: 3, Col: 5, Len: 2, WordEnd: 13})

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	got := string(data)
	want := "3,5,13,2\n"
	if got != want {
		t.Errorf("writeResult wrote %q, want %q", got, want)
	}
}
