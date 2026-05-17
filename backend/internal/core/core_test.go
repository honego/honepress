package core

import (
	"runtime"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version() != "0.1.0" {
		t.Fatalf("version mismatch: %s", Version())
	}
}

func TestVersionStatement(t *testing.T) {
	statements := VersionStatement()
	if len(statements) != 2 {
		t.Fatalf("statement count mismatch: %#v", statements)
	}
	firstStatement := statements[0]
	for _, expectedPart := range []string{"HonePress 0.1.0", Build, runtime.Version(), runtime.GOOS + "/" + runtime.GOARCH} {
		if !strings.Contains(firstStatement, expectedPart) {
			t.Fatalf("version statement missing %q in %q", expectedPart, firstStatement)
		}
	}
	if statements[1] == "" {
		t.Fatalf("intro statement must not be empty")
	}
}
