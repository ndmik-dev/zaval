package web

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{"Atlas": "atlas", "My Pet Project": "my-pet-project", "  zaval  ": "zaval", "Проєкт": ""}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}
