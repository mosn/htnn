package translation

import "testing"

func TestHostMatch(t *testing.T) {
	matched := []string{"*", "*.com", "*.test.com", "v.test.com"}
	mismatched := []string{"a.test.com", "*.t.com"}

	for _, m := range matched {
		if !hostMatch(m, "v.test.com") {
			t.Errorf("hostMatch(%s, v.test.com) should be true", m)
		}
	}
	for _, m := range mismatched {
		if hostMatch(m, "v.test.com") {
			t.Errorf("hostMatch(%s, v.test.com) should be false", m)
		}
	}
}
