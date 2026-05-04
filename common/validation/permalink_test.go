package validation

import "testing"

func TestNormalizePermalink(t *testing.T) {
	testCases := []struct {
		name              string
		rawPermalink      string
		expectedPermalink string
		wantError         bool
	}{
		{name: "追加扩展名", rawPermalink: "1", expectedPermalink: "1.html"},
		{name: "去掉开头斜杠", rawPermalink: "/docker-install", expectedPermalink: "docker-install.html"},
		{name: "拒绝路径穿越", rawPermalink: "../1.html", wantError: true},
		{name: "拒绝保留文件", rawPermalink: "rss.xml", wantError: true},
		{name: "拒绝中文", rawPermalink: "中文.html", wantError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			normalizedPermalink, err := NormalizePermalink(testCase.rawPermalink)
			if testCase.wantError {
				if err == nil {
					t.Fatalf("期望返回错误")
				}
				return
			}
			if err != nil {
				t.Fatalf("不期望返回错误：%v", err)
			}
			if normalizedPermalink != testCase.expectedPermalink {
				t.Fatalf("固定链接不一致：got %s want %s", normalizedPermalink, testCase.expectedPermalink)
			}
		})
	}
}
