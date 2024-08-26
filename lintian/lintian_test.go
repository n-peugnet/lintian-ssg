// SPDX-FileCopyrightText: 2024 Nicolas Peugnet <nicolas@club1.fr>
// SPDX-License-Identifier: GPL-3.0-or-later

package lintian_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func TestUnmarshal(t *testing.T) {
	cases := []struct {
		tag      string
		expected lintian.Tag
	}{
		{ // lintian v2.118.0
			"lintian_2.118.0_executable-in-usr-lib",
			lintian.Tag{
				Experimental:   true,
				Explanation:    "The package ships an executable file in /usr/lib.\n\nPlease move the file to <code>/usr/libexec</code>.\n\nWith policy revision 4.1.5, Debian adopted the Filesystem\nHierarchy Specification (FHS) version 3.0.\n\nThe FHS 3.0 describes <code>/usr/libexec</code>. Please use that\nlocation for executables.",
				LintianVersion: "2.118.0",
				Name:           "executable-in-usr-lib",
				NameSpaced:     false,
				Screens: []lintian.Screen{
					{
						Advocates: []string{"\"David Bremner\" <bremner@debian.org>"},
						Name:      "emacs/elpa/scripts",
						Reason:    "The <code>emacsen-common</code> package places installation\nand removal scripts, which for ELPA packages are executable,\nin the folder <code>/usr/lib/emacsen-common/packages</code>.\n\nAbout four hundred installation packages are affected. All of\nthem declare <code>emacsen-common</code> as an installation\nprerequisite.",
						SeeAlso: []string{
							"[Bug#974175](https://bugs.debian.org/974175)",
							"[Bug#954149](https://bugs.debian.org/954149)",
						},
					},
					{
						Advocates: []string{"\"Andrius Merkys\" <merkys@debian.org>"},
						Name:      "web/cgi/scripts",
						Reason:    "The folder <code>/usr/lib/cgi-bin/</code> is designated for\nscripts in the Common Gateway Interface (CGI). They require the\nexecutable bit so the server can run them.",
						SeeAlso: []string{
							"<https://en.wikipedia.org/wiki/Common_Gateway_Interface>",
							"<https://datatracker.ietf.org/doc/html/rfc3875.html>",
							"[Bug#1003941](https://bugs.debian.org/1003941)",
						},
					},
				},
				SeeAlso: []string{
					"[File System Structure](https://www.debian.org/doc/debian-policy/ch-opersys.html#file-system-structure) (Section 9.1.1) in the Debian Policy Manual",
					"filesystem-hierarchy",
					"<https://refspecs.linuxfoundation.org/FHS_3.0/fhs/ch04s07.html>",
					"[Bug#954149](https://bugs.debian.org/954149)",
				},
				Visibility: "pedantic",
			},
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("%d %s", i, c.tag), func(t *testing.T) {
			file, err := os.Open(filepath.Join("testdata", c.tag+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var actual lintian.Tag
			decoder := json.NewDecoder(file)
			if err := decoder.Decode(&actual); err != nil {
				t.Fatal(err)
			}
			if reflect.DeepEqual(c.expected, actual) {
				t.Fatalf("\nexpected: %v\nactual  : %v", c.expected, actual)
			}
		})
	}
}
