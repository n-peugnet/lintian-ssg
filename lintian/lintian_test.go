// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package lintian_test

import (
	"fmt"
	"testing"

	"github.com/n-peugnet/lintian-ssg/lintian"
)

const baseURL = "https://salsa.debian.org/lintian/lintian/-/blob//tags/"

func TestSource(t *testing.T) {
	cases := []struct {
		tag      lintian.Tag
		expected string
	}{
		{ // Basic case
			lintian.Tag{Name: "test-tag", NameSpaced: false},
			"t/test-tag.tag",
		},
		{ // namespaced case
			lintian.Tag{Name: "teams/js/test-tag", NameSpaced: true},
			"teams/js/test-tag.tag",
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("%d %s", i, c.expected), func(t *testing.T) {
			expected := baseURL + c.expected
			actual := c.tag.Source()
			if actual != expected {
				t.Fatalf("\nexpected: %q\nactual  : %q", expected, actual)
			}
		})
	}
}
