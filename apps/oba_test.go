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

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarquee(t *testing.T) {
	tc := []struct {
		in  string
		off int
		out string
	}{
		{"abcdefg", 0, "abcdefg    "},
		{"abcdefg", 1, "bcdefg    a"},
		{"abcdefg", 2, "cdefg    ab"},
		{"abcdefg", 3, "defg    abc"},
		{"abcdefg", 4, "efg    abcd"},
		{"abcdefg", 5, "fg    abcde"},
		{"abcdefg", 6, "g    abcdef"},
		{"abcdefg", 7, "    abcdefg"},
		{"abcdefg", 8, "   abcdefg "},
		{"abcdefg", 9, "  abcdefg  "},
		{"abcdefg", 10, " abcdefg   "},
		{"abcdefg", 11, "abcdefg    "},
	}
	for i, c := range tc {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			out := marquee(c.in, c.off)
			assert.Equal(t, c.out, out)
		})
	}
}
