package main_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	main "github.com/n-peugnet/lintian-ssg"
)

func expectPanic(t *testing.T, substr string, fn func()) {
	defer func() {
		if err := recover(); err != nil {
			if !strings.Contains(fmt.Sprint(err), substr) {
				t.Fatalf("panic does not contain %q: %q", substr, err)
			}
		} else {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func TestEmptyPATH(t *testing.T) {
	os.Setenv("PATH", "")
	expectPanic(t, `ERROR: list tags: exec: "lintian-explain-tags"`, main.Run)
}
