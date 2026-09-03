package web

import "testing"

func TestMatchQuery(t *testing.T) {
	title := "Полагодити ретраї у webhook-хендлері (рев'ю)"
	yes := []string{"ретра", "РЕТРАЇ", "webhook хендлер", "хендлері ретраї", "ревʼю", "рев’ю", "webhook-хендлері", ""}
	no := []string{"ретраї слак", "хендлерами"}
	for _, q := range yes {
		if !matchQuery(title, q) {
			t.Errorf("%q should match", q)
		}
	}
	for _, q := range no {
		if matchQuery(title, q) {
			t.Errorf("%q should not match", q)
		}
	}
}
