package chat

import (
	"reflect"
	"testing"
)

func TestQuotedTerms(t *testing.T) {
	cases := map[string][]string{
		"之前「数据库连接超时」是怎么解决的？":                  {"数据库连接超时"},
		"「A」和「B」还有“C”以及\"D\"":                 {"A", "B", "C"}, // at most 3
		"「问题或报错关键词」没改就发了":                     {"问题或报错关键词"},
		"没有引号的问题":                             nil,
		"「…」 和 「 」 不算":                        nil,
		"「" + string(make([]rune, 0)) + "未闭合": nil,
	}
	for in, want := range cases {
		if got := quotedTerms(in); !reflect.DeepEqual(got, want) {
			t.Errorf("quotedTerms(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTitleAndSummary(t *testing.T) {
	if got := titleOf("  总结一下\n今天的工作  "); got != "总结一下 今天的工作" {
		t.Errorf("titleOf = %q", got)
	}
	if got := summarize(`{"total": 12, "conversations": []}`); got != "找到 12 个" {
		t.Errorf("summarize json = %q", got)
	}
	if got := summarize("# 日程 2026-11-21（3 项）\n- a"); got != "日程 2026-11-21（3 项）（24 字）" {
		t.Errorf("summarize text = %q", got)
	}
}
