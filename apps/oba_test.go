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
