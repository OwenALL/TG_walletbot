package telegram

import "github.com/go-telegram/bot/models"

// InlineKeyboardBuilder 内联键盘构建器，支持链式调用
type InlineKeyboardBuilder struct {
	rows [][]models.InlineKeyboardButton
}

// NewInlineKeyboard 创建内联键盘构建器
func NewInlineKeyboard() *InlineKeyboardBuilder {
	return &InlineKeyboardBuilder{}
}

// Row 添加一行按钮
func (b *InlineKeyboardBuilder) Row(buttons ...models.InlineKeyboardButton) *InlineKeyboardBuilder {
	b.rows = append(b.rows, buttons)
	return b
}

// Build 构建内联键盘
func (b *InlineKeyboardBuilder) Build() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: b.rows,
	}
}

// Button 创建普通回调按钮
func Button(text, callbackData string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: callbackData,
	}
}

// URLButton 创建 URL 链接按钮
func URLButton(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

// SwitchInlineCurrentChatButton 创建当前聊天内联查询按钮
// 注意: go-telegram/bot 的 SwitchInlineQueryCurrentChat 字段使用 omitempty，
// 空字符串会被 JSON 序列化丢弃导致按钮无效。使用零宽空格作为 workaround。
func SwitchInlineCurrentChatButton(text, query string) models.InlineKeyboardButton {
	if query == "" {
		query = "\u200b" // 零宽空格，防止 omitempty 丢弃字段
	}
	return models.InlineKeyboardButton{
		Text:                         text,
		SwitchInlineQueryCurrentChat: query,
	}
}

// SwitchInlineButton 创建选择聊天内联查询按钮
// 注意: 同 SwitchInlineCurrentChatButton，空字符串需要零宽空格 workaround
func SwitchInlineButton(text, query string) models.InlineKeyboardButton {
	if query == "" {
		query = "\u200b" // 零宽空格，防止 omitempty 丢弃字段
	}
	return models.InlineKeyboardButton{
		Text:              text,
		SwitchInlineQuery: query,
	}
}

// BuildGrid 基于列表数据动态生成网格键盘
// items: 按钮文本列表
// columns: 每行列数
// prefix: callback_data 前缀，最终格式为 prefix_index
func BuildGrid(items []string, columns int, prefix string) *models.InlineKeyboardMarkup {
	var keyboard [][]models.InlineKeyboardButton
	var row []models.InlineKeyboardButton
	for i, item := range items {
		row = append(row, models.InlineKeyboardButton{
			Text:         item,
			CallbackData: prefix + "_" + item,
		})
		if (i+1)%columns == 0 {
			keyboard = append(keyboard, row)
			row = nil
		}
	}
	if len(row) > 0 {
		keyboard = append(keyboard, row)
	}
	return &models.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

// BackButton 返回按钮 (通用)
func BackButton(callbackData string) models.InlineKeyboardButton {
	return Button("返回", callbackData)
}

// MainMenuButton 主菜单按钮 (通用)
func MainMenuButton() models.InlineKeyboardButton {
	return Button("主菜单", "Menu--main")
}

// CancelButton 取消按钮 (通用)
func CancelButton() models.InlineKeyboardButton {
	return Button("取消", "Menu--cancel")
}
