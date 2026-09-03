package links

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		url               string
		kind, label, meta string
	}{
		{"https://atlas.atlassian.net/browse/ATL-412", "jira", "ATL-412", ""},
		{"https://atlas.atlassian.net/jira/software/projects/ATL/boards/3", "jira", "Jira", ""},
		{"https://atlas.atlassian.net/wiki/spaces/ENG/pages/1", "confluence", "Confluence", ""},
		{"https://github.com/atlas/api/pull/128", "github", "GitHub", "PR #128"},
		{"https://github.com/atlas/api/issues/9", "github", "GitHub", "#9"},
		{"https://github.com/atlas/api", "github", "GitHub", "atlas/api"},
		{"https://gitlab.com/nimbus/dashboard/-/merge_requests/12", "gitlab", "GitLab", "MR !12"},
		{"https://git.nimbus.dev/gitlab/group/repo/-/issues/7", "gitlab", "GitLab", "#7"},
		{"https://gitlab.com/nimbus/dashboard", "gitlab", "GitLab", "nimbus/dashboard"},
		{"https://atlas.slack.com/archives/C123/p456", "slack", "Slack", ""},
		{"slack://channel?team=T1&id=C1", "slack", "Slack", ""},
		{"https://www.figma.com/design/AbCdEf1234567890AbCdEf/Dashboard-v3?node-id=1", "figma", "Figma", "Dashboard v3"},
		{"https://www.notion.so/team/Release-notes-0123456789abcdef0123456789abcdef", "notion", "Notion", "Release notes"},
		{"https://sentry.io/organizations/atlas/issues/1/", "sentry", "Sentry", ""},
		{"https://docs.example.com/x", "other", "docs.example.com", ""},
		{"example.com/path", "other", "example.com", ""},
	}
	for _, c := range cases {
		got := Classify(c.url)
		if got.Kind != c.kind || got.Label != c.label || got.Meta != c.meta {
			t.Errorf("%s: got %s/%s/%s, want %s/%s/%s", c.url, got.Kind, got.Label, got.Meta, c.kind, c.label, c.meta)
		}
	}
}

func TestParse(t *testing.T) {
	title, ls := Parse("Полагодити ретраї — https://atlas.atlassian.net/browse/ATL-412")
	if title != "Полагодити ретраї" || len(ls) != 1 || ls[0].Label != "ATL-412" {
		t.Errorf("got %q %+v", title, ls)
	}
	title, ls = Parse("https://github.com/atlas/api/pull/128")
	if title != "GitHub · PR #128" || len(ls) != 1 {
		t.Errorf("url-only: got %q %+v", title, ls)
	}
	title, ls = Parse("Два лінки https://a.example.com/1, https://b.example.com/2.")
	if title != "Два лінки" || len(ls) != 2 || ls[0].URL != "https://a.example.com/1" || ls[1].URL != "https://b.example.com/2" {
		t.Errorf("two links: got %q %+v", title, ls)
	}
	title, ls = Parse("просто текст")
	if title != "просто текст" || len(ls) != 0 {
		t.Errorf("plain: got %q %+v", title, ls)
	}
}
