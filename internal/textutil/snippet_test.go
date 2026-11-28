package textutil

import "testing"

func TestSnippet(t *testing.T) {
	for _, tc := range []struct {
		in    string
		limit int
		want  string
	}{
		{" \t你好\n 世界  ", 5, "你好 世界"},
		{"你好 世界", 4, "你好 世…"},
		{" \n\t ", 0, ""},
	} {
		if got := Snippet(tc.in, tc.limit); got != tc.want {
			t.Errorf("Snippet(%q, %d) = %q, want %q", tc.in, tc.limit, got, tc.want)
		}
	}
}
