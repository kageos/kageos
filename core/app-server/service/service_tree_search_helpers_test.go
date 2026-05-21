package service

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
)

func TestRankAndLimitSearchResourcesUsesHeatWithinSameMatchTier(t *testing.T) {
	trees := []*model.ServiceTree{
		{
			Name:        "工单数量趋势统计",
			Type:        model.ServiceTreeTypeFunction,
			Description: "按日统计新增工单数量",
			Tags:        "工单管理,统计分析",
			RunCount:    114,
		},
		{
			Name:        "效率统计",
			Type:        model.ServiceTreeTypeFunction,
			Description: "工单效率指标",
			RunCount:    999,
		},
		{
			Name:        "工单管理",
			Type:        model.ServiceTreeTypeFunction,
			Description: "一个简单的工单管理系统",
			RunCount:    723,
		},
		{
			Name:        "工单管理案例",
			Type:        model.ServiceTreeTypePackage,
			Description: "本目录收录工单管理案例",
		},
	}

	got := rankAndLimitSearchResources(trees, "工单", 1, 30)

	if got[0].Name != "工单管理" {
		t.Fatalf("expected hottest title match first, got %q", got[0].Name)
	}
	if got[1].Name != "工单数量趋势统计" {
		t.Fatalf("expected lower-heat title match second, got %q", got[1].Name)
	}
	if got[2].Name != "工单管理案例" {
		t.Fatalf("expected package title match before description-only hit, got %q", got[2].Name)
	}
}

func TestRankAndLimitSearchResourcesKeepsStrongerMatchTierBeforeHeat(t *testing.T) {
	trees := []*model.ServiceTree{
		{
			Name:        "工单管理",
			Type:        model.ServiceTreeTypeFunction,
			Description: "一个简单的工单管理系统",
			RunCount:    1200,
		},
		{
			Name:     "工单",
			Type:     model.ServiceTreeTypePackage,
			RunCount: 0,
		},
		{
			Name:        "数据分析",
			Type:        model.ServiceTreeTypeFunction,
			Description: "工单数据分析",
			RunCount:    5000,
		},
	}

	got := rankAndLimitSearchResources(trees, "工单", 1, 30)

	if got[0].Name != "工单" {
		t.Fatalf("expected exact name match first, got %q", got[0].Name)
	}
	if got[1].Name != "工单管理" {
		t.Fatalf("expected title contains match before description-only hot hit, got %q", got[1].Name)
	}
}

func TestBuildResourceSearchResultsDoesNotReturnFullDocContent(t *testing.T) {
	fullContentLikeDescription := strings.Repeat("这是很长的文档正文内容", 40)
	trees := []*model.ServiceTree{
		{
			Name:             "长文档",
			Type:             model.ServiceTreeTypeDocs,
			Description:      fullContentLikeDescription,
			FullCodePath:     "/demo/app/docs/guide",
			SearchDocSummary: "这是文档摘要",
		},
	}

	got := buildResourceSearchResults(trees, "文档")

	if got[0].Description != "这是文档摘要" {
		t.Fatalf("expected doc summary as description, got %q", got[0].Description)
	}
	if got[0].Snippet != "这是文档摘要" {
		t.Fatalf("expected doc summary as snippet, got %q", got[0].Snippet)
	}
}

func TestBuildResourceSearchResultsTruncatesDocDescriptionFallback(t *testing.T) {
	fullContentLikeDescription := strings.Repeat("文档正文", 80)
	trees := []*model.ServiceTree{
		{
			Name:         "无摘要文档",
			Type:         model.ServiceTreeTypeDocs,
			Description:  fullContentLikeDescription,
			FullCodePath: "/demo/app/docs/guide",
		},
	}

	got := buildResourceSearchResults(trees, "文档")

	if len([]rune(got[0].Description)) > 223 {
		t.Fatalf("expected truncated doc description, got length %d", len([]rune(got[0].Description)))
	}
	if got[0].Description == fullContentLikeDescription {
		t.Fatal("expected doc description fallback to be truncated")
	}
}
