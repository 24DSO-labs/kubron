// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import "testing"

func TestParseTermControl(t *testing.T) {
	c, r, err := parseTermControl([]byte(`{"type":"resize","cols":120,"rows":40}`))
	if err != nil || c != 120 || r != 40 {
		t.Fatalf("%d x %d, %v", c, r, err)
	}
	for _, bad := range []string{
		`не json`, `{}`, `{"type":"exec","cols":1,"rows":1}`,
		`{"type":"resize","cols":0,"rows":10}`, `{"type":"resize","cols":10,"rows":-1}`,
		`{"type":"resize","cols":100000,"rows":10}`, `{"type":"resize","cols":10}`,
	} {
		if _, _, err := parseTermControl([]byte(bad)); err == nil {
			t.Errorf("%s: ждали ошибку", bad)
		}
	}
}

type fakeCloser struct{ n int }

func (f *fakeCloser) Close() error { f.n++; return nil }

func TestTermConnsCloseSession(t *testing.T) {
	var tc termConns
	a1, a2, b := &fakeCloser{}, &fakeCloser{}, &fakeCloser{}
	tc.add("a", a1)
	tc.add("a", a2)
	tc.add("b", b)
	tc.remove("a", a2)
	tc.closeSession("a")
	if a1.n != 1 || a2.n != 0 || b.n != 0 {
		t.Errorf("a1=%d a2=%d b=%d", a1.n, a2.n, b.n)
	}
	tc.closeSession("a") // повторно — без паники
	tc.closeSession("нет")
}
