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

import "strings"

func wordwrap(in string) []string {
	var out []string
	splits := strings.Split(in, "\n")
	for _, s := range splits {
		out = append(out, wrapOne(s)...)
	}
	return out
}

func wrapOne(in string) []string {
	if len(in) == 0 {
		return nil
	}
	words := strings.Fields(in)
	line := ""
	var out []string
	const maxLen = 20
	for _, word := range words {
		if len(line) == maxLen {
			out = append(out, line)
			line = ""
		}
		space := " "
		if len(line) == 0 {
			space = ""
		}
		if len(line)+len(space)+len(word) > maxLen {
			out = append(out, line)
			line = ""
			space = ""
		}
		if len(word) > maxLen {
			w := word
			for len(w) > maxLen {
				out = append(out, w[:maxLen])
				w = w[maxLen:]
			}
			word = w
		}
		line += space + word
	}
	out = append(out, line)
	return out
}
