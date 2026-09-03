package web

import "strings"

// matchQuery is the board and palette search: every word of the query must
// appear somewhere in the title, any order, case-insensitive, apostrophes
// and dashes folded so «ревʼю» finds «рев'ю».
func matchQuery(title, query string) bool {
	title = fold(title)
	for _, w := range strings.Fields(fold(query)) {
		if !strings.Contains(title, w) {
			return false
		}
	}
	return true
}

var folder = strings.NewReplacer("ʼ", "", "'", "", "’", "", "`", "", "-", " ", "–", " ", "—", " ", "ё", "е")

func fold(s string) string {
	return folder.Replace(strings.ToLower(s))
}
