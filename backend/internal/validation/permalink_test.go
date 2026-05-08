package validation

import "testing"

func TestNormalizePermalink(t *testing.T) {
	testCases := []struct {
		name              string
		rawPermalink      string
		expectedPermalink string
		wantError         bool
	}{
		{name: "append extension", rawPermalink: "1", expectedPermalink: "1.html"},
		{name: "trim leading slash", rawPermalink: "/docker-install", expectedPermalink: "docker-install.html"},
		{name: "reject path traversal", rawPermalink: "../1.html", wantError: true},
		{name: "reject reserved file", rawPermalink: "rss.xml", wantError: true},
		{name: "reject Next post shell", rawPermalink: "posts.html", wantError: true},
		{name: "reject non-ASCII", rawPermalink: "中文.html", wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			normalizedPermalink, err := NormalizePermalink(testCase.rawPermalink)
			if testCase.wantError {
				if err == nil {
					t.Fatalf("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if normalizedPermalink != testCase.expectedPermalink {
				t.Fatalf("permalink mismatch: got %s, want %s", normalizedPermalink, testCase.expectedPermalink)
			}
		})
	}
}

func TestMarkdownFileNameFromTitleAllowsChineseTitle(t *testing.T) {
	markdownFileName, err := MarkdownFileNameFromTitle("记录生活")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if markdownFileName != "记录生活.md" {
		t.Fatalf("markdown file name mismatch: got %s", markdownFileName)
	}
}
