/*
   Copyright 2025 Google LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       https://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package apps

import "testing"

func TestWordwrap(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  []string
	}{{
		"empty",
		"",
		nil,
	}, {
		"one",
		"1",
		[]string{"1"},
	}, {
		"twenty",
		"01234567890123456789",
		[]string{"01234567890123456789"},
	}, {
		"hello world",
		"hello world",
		[]string{"hello world"},
	}, {
		"extra spaces",
		"  hello    world    ",
		[]string{"hello world"},
	}, {
		"newlines",
		"hello\nworld",
		[]string{"hello", "world"},
	}, {
		"hello twenty",
		"hello 01234567890123456789",
		[]string{"hello", "01234567890123456789"},
	}, {
		"hello thirty",
		"hello 012345678901234567890123456789",
		[]string{"hello", "01234567890123456789", "0123456789"},
	}, {
		"no weird tabs",
		"t\tv\vf\f",
		[]string{"t v f"},
	}, {
		"four score",
		"four score and seven years ago",
		[]string{"four score and seven", "years ago"},
	}, {
		"not even twenties",
		"012345678901234 012345678901234 0123",
		[]string{"012345678901234", "012345678901234 0123"},
	}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := wordwrap(c.in)
			if len(out) != len(c.out) {
				t.Errorf("len(wordwrap(%q)) = %d; want %d", c.in, len(out), len(c.out))
			}
			for i := range out {
				if i > len(c.out) {
					t.Errorf("Extra line %d: %q", i, out[i])
				}
				if out[i] != c.out[i] {
					t.Errorf("want %q; got %q", c.out[i], out[i])
				}
			}
		})
	}
}
