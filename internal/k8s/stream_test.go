package k8s

import "testing"

func TestSplitComplete(t *testing.T) {
	cases := []struct{ in, send, rest string }{
		{"", "", ""},
		{"abc", "", "abc"},
		{"a\nb", "a\n", "b"},
		{"a\nb\n", "a\nb\n", ""},
		{"привет\nпо", "привет\n", "по"},
	}
	for _, c := range cases {
		s, r := splitComplete([]byte(c.in))
		if string(s) != c.send || string(r) != c.rest {
			t.Errorf("%q: got (%q,%q), want (%q,%q)", c.in, s, r, c.send, c.rest)
		}
	}
}
