// Package links pulls URLs out of task text and turns them into chips.
package links

import (
	"net/url"
	"regexp"
	"strings"
)

type Link struct {
	URL   string
	Kind  string // jira, github, gitlab, slack, figma, notion, confluence, sentry, other
	Label string
	Meta  string
}

var (
	urlRe      = regexp.MustCompile(`(?i)\b(?:https?://|slack://)\S+`)
	jiraKeyRe  = regexp.MustCompile(`/browse/([A-Z][A-Z0-9]+-\d+)`)
	ghPathRe   = regexp.MustCompile(`^/([^/]+)/([^/]+)(?:/(pull|issues|commit|tree|blob)/([^/]+))?`)
	glPathRe   = regexp.MustCompile(`^/(.+?)/-/(merge_requests|issues|commit)/([^/]+)`)
	trailingRe = regexp.MustCompile(`[\s\-–—:|,]+$`)
)

// Parse splits raw input into a title and the links found in it.
// When the text is nothing but a link, the chip label becomes the title.
func Parse(raw string) (string, []Link) {
	var out []Link
	title := urlRe.ReplaceAllStringFunc(raw, func(m string) string {
		m = strings.TrimRight(m, ".,;)")
		out = append(out, Classify(m))
		return ""
	})
	title = strings.Join(strings.Fields(title), " ")
	title = trailingRe.ReplaceAllString(title, "")
	if title == "" && len(out) > 0 {
		title = out[0].Label
		if out[0].Meta != "" {
			title += " · " + out[0].Meta
		}
	}
	return title, out
}

func Classify(raw string) Link {
	l := Link{URL: normalize(raw), Kind: "other"}
	u, err := url.Parse(l.URL)
	if err != nil {
		l.Label = "лінк"
		return l
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	path := u.Path

	switch {
	case u.Scheme == "slack" || strings.HasSuffix(host, "slack.com"):
		l.Kind, l.Label = "slack", "Slack"
	case strings.HasSuffix(host, "atlassian.net") && strings.HasPrefix(path, "/wiki"):
		l.Kind, l.Label = "confluence", "Confluence"
	case strings.HasSuffix(host, "atlassian.net") || strings.Contains(host, "jira"):
		l.Kind, l.Label = "jira", "Jira"
		if m := jiraKeyRe.FindStringSubmatch(path); m != nil {
			l.Label = m[1]
		}
	case host == "github.com":
		l.Kind, l.Label = "github", "GitHub"
		if m := ghPathRe.FindStringSubmatch(path); m != nil {
			l.Meta = m[1] + "/" + m[2]
			switch m[3] {
			case "pull":
				l.Meta = "PR #" + m[4]
			case "issues":
				l.Meta = "#" + m[4]
			case "commit":
				l.Meta = short(m[4])
			}
		}
	// Self-hosted GitLab has any host name; the "/-/" path segment gives it away.
	case strings.Contains(host, "gitlab") || glPathRe.MatchString(path):
		l.Kind, l.Label = "gitlab", "GitLab"
		if m := glPathRe.FindStringSubmatch(path); m != nil {
			switch m[2] {
			case "merge_requests":
				l.Meta = "MR !" + m[3]
			case "issues":
				l.Meta = "#" + m[3]
			case "commit":
				l.Meta = short(m[3])
			}
		} else if p := strings.Trim(path, "/"); p != "" {
			l.Meta = p
		}
	case strings.HasSuffix(host, "figma.com"):
		l.Kind, l.Label, l.Meta = "figma", "Figma", slugTitle(path)
	case strings.HasSuffix(host, "notion.so") || strings.HasSuffix(host, "notion.site"):
		l.Kind, l.Label, l.Meta = "notion", "Notion", slugTitle(path)
	case strings.HasSuffix(host, "sentry.io"):
		l.Kind, l.Label = "sentry", "Sentry"
	default:
		l.Label = host
	}
	return l
}

func normalize(u string) string {
	u = strings.TrimSpace(u)
	if strings.HasPrefix(strings.ToLower(u), "http://") || strings.HasPrefix(strings.ToLower(u), "https://") || strings.HasPrefix(u, "slack://") {
		return u
	}
	return "https://" + strings.TrimLeft(u, "/")
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// slugTitle turns the last human-readable path segment ("Dashboard-v3") into "Dashboard v3".
// Figma and Notion put an opaque id before or after it; ids are skipped.
func slugTitle(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		if p == "" || p == "file" || p == "design" || p == "board" || len(p) >= 22 && !strings.Contains(p, "-") {
			continue
		}
		// Notion appends the 32-hex id to the slug: "Title-abcdef...".
		if j := strings.LastIndex(p, "-"); j > 0 && len(p)-j-1 == 32 {
			p = p[:j]
		}
		return strings.ReplaceAll(p, "-", " ")
	}
	return ""
}
