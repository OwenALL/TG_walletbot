package telegram

import (
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestButton 测试创建普通回调按钮
func TestButton(t *testing.T) {
	btn := Button("测试按钮", "test_callback")
	assert.Equal(t, "测试按钮", btn.Text)
	assert.Equal(t, "test_callback", btn.CallbackData)
	assert.Empty(t, btn.URL, "普通按钮不应有 URL")
}

// TestURLButton 测试创建 URL 链接按钮
func TestURLButton(t *testing.T) {
	btn := URLButton("打开链接", "https://example.com")
	assert.Equal(t, "打开链接", btn.Text)
	assert.Equal(t, "https://example.com", btn.URL)
	assert.Empty(t, btn.CallbackData, "URL 按钮不应有 CallbackData")
}

// TestSwitchInlineCurrentChatButton 测试创建当前聊天内联查询按钮
func TestSwitchInlineCurrentChatButton(t *testing.T) {
	t.Run("带查询内容", func(t *testing.T) {
		btn := SwitchInlineCurrentChatButton("搜索", "query_text")
		assert.Equal(t, "搜索", btn.Text)
		assert.Equal(t, "query_text", btn.SwitchInlineQueryCurrentChat)
	})

	t.Run("空查询使用零宽空格", func(t *testing.T) {
		btn := SwitchInlineCurrentChatButton("搜索", "")
		assert.Equal(t, "搜索", btn.Text)
		assert.Equal(t, "\u200b", btn.SwitchInlineQueryCurrentChat, "空查询应替换为零宽空格")
	})
}

// TestSwitchInlineButton 测试创建选择聊天内联查询按钮
func TestSwitchInlineButton(t *testing.T) {
	t.Run("带查询内容", func(t *testing.T) {
		btn := SwitchInlineButton("分享", "share_data")
		assert.Equal(t, "分享", btn.Text)
		assert.Equal(t, "share_data", btn.SwitchInlineQuery)
	})

	t.Run("空查询使用零宽空格", func(t *testing.T) {
		btn := SwitchInlineButton("分享", "")
		assert.Equal(t, "分享", btn.Text)
		assert.Equal(t, "\u200b", btn.SwitchInlineQuery, "空查询应替换为零宽空格")
	})
}

// TestBackButton 测试返回按钮
func TestBackButton(t *testing.T) {
	btn := BackButton("go_back")
	assert.Equal(t, "返回", btn.Text)
	assert.Equal(t, "go_back", btn.CallbackData)
}

// TestMainMenuButton 测试主菜单按钮
func TestMainMenuButton(t *testing.T) {
	btn := MainMenuButton()
	assert.Equal(t, "主菜单", btn.Text)
	assert.Equal(t, "Menu--main", btn.CallbackData)
}

// TestCancelButton 测试取消按钮
func TestCancelButton(t *testing.T) {
	btn := CancelButton()
	assert.Equal(t, "取消", btn.Text)
	assert.Equal(t, "Menu--cancel", btn.CallbackData)
}

// TestInlineKeyboardBuilder_Empty 测试空键盘构建
func TestInlineKeyboardBuilder_Empty(t *testing.T) {
	kb := NewInlineKeyboard().Build()
	require.NotNil(t, kb)
	assert.Nil(t, kb.InlineKeyboard, "空键盘应无行")
}

// TestInlineKeyboardBuilder_SingleRow 测试单行键盘
func TestInlineKeyboardBuilder_SingleRow(t *testing.T) {
	kb := NewInlineKeyboard().
		Row(Button("按钮A", "a"), Button("按钮B", "b")).
		Build()

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 1, "应只有一行")
	assert.Len(t, kb.InlineKeyboard[0], 2, "第一行应有两个按钮")
	assert.Equal(t, "按钮A", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "按钮B", kb.InlineKeyboard[0][1].Text)
}

// TestInlineKeyboardBuilder_MultipleRows 测试多行键盘
func TestInlineKeyboardBuilder_MultipleRows(t *testing.T) {
	kb := NewInlineKeyboard().
		Row(Button("功能1", "func_1")).
		Row(Button("功能2", "func_2")).
		Row(Button("功能3", "func_3")).
		Build()

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 3, "应有三行")

	for i := 0; i < 3; i++ {
		assert.Len(t, kb.InlineKeyboard[i], 1, "每行应有一个按钮")
	}
}

// TestInlineKeyboardBuilder_MixedLayout 测试混合布局键盘
func TestInlineKeyboardBuilder_MixedLayout(t *testing.T) {
	kb := NewInlineKeyboard().
		Row(Button("确认", "confirm"), Button("取消", "cancel")).           // 2列
		Row(Button("上一页", "prev"), Button("1/3", "page"), Button("下一页", "next")). // 3列
		Row(Button("返回主菜单", "main_menu")).                              // 1列
		Build()

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 3)
	assert.Len(t, kb.InlineKeyboard[0], 2, "第一行 2 个按钮")
	assert.Len(t, kb.InlineKeyboard[1], 3, "第二行 3 个按钮")
	assert.Len(t, kb.InlineKeyboard[2], 1, "第三行 1 个按钮")
}

// TestInlineKeyboardBuilder_ChainCalls 测试链式调用返回同一实例
func TestInlineKeyboardBuilder_ChainCalls(t *testing.T) {
	builder := NewInlineKeyboard()
	returned := builder.Row(Button("test", "test"))
	assert.Same(t, builder, returned, "链式调用应返回同一实例")
}

// TestBuildGrid_EvenDivision 测试网格键盘（项目数能整除列数）
func TestBuildGrid_EvenDivision(t *testing.T) {
	items := []string{"USDT", "TRX", "BTC", "ETH"}
	kb := BuildGrid(items, 2, "coin")

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 2, "4 项 2 列 = 2 行")

	// 第一行
	assert.Equal(t, "USDT", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "coin_USDT", kb.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "TRX", kb.InlineKeyboard[0][1].Text)
	assert.Equal(t, "coin_TRX", kb.InlineKeyboard[0][1].CallbackData)

	// 第二行
	assert.Equal(t, "BTC", kb.InlineKeyboard[1][0].Text)
	assert.Equal(t, "coin_BTC", kb.InlineKeyboard[1][0].CallbackData)
	assert.Equal(t, "ETH", kb.InlineKeyboard[1][1].Text)
	assert.Equal(t, "coin_ETH", kb.InlineKeyboard[1][1].CallbackData)
}

// TestBuildGrid_UnevenDivision 测试网格键盘（项目数不能整除列数，有余项）
func TestBuildGrid_UnevenDivision(t *testing.T) {
	items := []string{"A", "B", "C", "D", "E"}
	kb := BuildGrid(items, 3, "item")

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 2, "5 项 3 列 = 1 完整行 + 1 余行")

	// 第一行 3 个
	assert.Len(t, kb.InlineKeyboard[0], 3)
	// 第二行 2 个（剩余项）
	assert.Len(t, kb.InlineKeyboard[1], 2)
}

// TestBuildGrid_SingleColumn 测试单列网格
func TestBuildGrid_SingleColumn(t *testing.T) {
	items := []string{"功能1", "功能2", "功能3"}
	kb := BuildGrid(items, 1, "func")

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 3, "3 项 1 列 = 3 行")

	for i, item := range items {
		assert.Len(t, kb.InlineKeyboard[i], 1)
		assert.Equal(t, item, kb.InlineKeyboard[i][0].Text)
	}
}

// TestBuildGrid_EmptyItems 测试空列表
func TestBuildGrid_EmptyItems(t *testing.T) {
	kb := BuildGrid([]string{}, 2, "empty")
	require.NotNil(t, kb)
	assert.Empty(t, kb.InlineKeyboard, "空列表应生成空键盘")
}

// TestBuildGrid_SingleItem 测试单项目
func TestBuildGrid_SingleItem(t *testing.T) {
	kb := BuildGrid([]string{"唯一"}, 3, "single")
	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 1, "单项目应生成一行")
	assert.Len(t, kb.InlineKeyboard[0], 1, "该行应有一个按钮")
	assert.Equal(t, "唯一", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "single_唯一", kb.InlineKeyboard[0][0].CallbackData)
}

// TestBuildGrid_CallbackDataFormat 测试回调数据格式
func TestBuildGrid_CallbackDataFormat(t *testing.T) {
	items := []string{"test"}
	kb := BuildGrid(items, 1, "prefix")

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 1)
	// 回调数据格式: prefix_item
	assert.Equal(t, "prefix_test", kb.InlineKeyboard[0][0].CallbackData)
}

// TestBuildGrid_LargeGrid 测试大型网格 (压力测试)
func TestBuildGrid_LargeGrid(t *testing.T) {
	items := make([]string, 20)
	for i := range items {
		items[i] = string(rune('A' + i%26))
	}
	kb := BuildGrid(items, 4, "big")

	require.NotNil(t, kb)
	// 20 项 4 列 = 5 行
	assert.Len(t, kb.InlineKeyboard, 5)
	for _, row := range kb.InlineKeyboard {
		assert.Len(t, row, 4, "每行应有 4 个按钮")
	}
}

// TestMixedButtonTypes 测试键盘中混合不同类型按钮
func TestMixedButtonTypes(t *testing.T) {
	kb := NewInlineKeyboard().
		Row(Button("回调按钮", "callback_data")).
		Row(URLButton("链接按钮", "https://example.com")).
		Row(SwitchInlineCurrentChatButton("内联搜索", "query")).
		Build()

	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 3)

	// 验证每种按钮类型
	assert.NotEmpty(t, kb.InlineKeyboard[0][0].CallbackData)
	assert.NotEmpty(t, kb.InlineKeyboard[1][0].URL)
	assert.NotEmpty(t, kb.InlineKeyboard[2][0].SwitchInlineQueryCurrentChat)
}

// TestInlineKeyboardMarkupType 测试构建结果类型
func TestInlineKeyboardMarkupType(t *testing.T) {
	kb := NewInlineKeyboard().
		Row(Button("test", "test")).
		Build()

	// 确保返回的是 *models.InlineKeyboardMarkup 类型
	var _ *models.InlineKeyboardMarkup = kb
	assert.NotNil(t, kb)
}
