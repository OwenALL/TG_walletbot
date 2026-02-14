// Package handler 提供 Telegram Bot 的各模块处理器
package handler

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	tgpkg "github.com/TGlimmer/TG_walletbot/pkg/telegram"
)

// --- 钱包主页消息构建 ---

// 自定义 emoji ID 常量 (从原版 WalletBot 逆向抓取，每行使用不同的自定义 emoji)
// 需要 Bot 主人有 Telegram Premium 或 Bot 购买了 Fragment 用户名
const (
	emojiNickname = "6235662197276544197" // 昵称行图标
	emojiID       = "6235699516247379290" // ID行图标
	emojiUSDT     = "6213220344614882714" // USDT 币种图标
	emojiTRX      = "6213214271531126888" // TRX 币种图标
	emojiCNY      = "6215350725703111290" // CNY 币种图标
	emojiTON      = "6212998226086203185" // TON 币种图标 (闪兑页面用)
)

// customEmoji 构建自定义 emoji HTML 标签
// fallback 为不支持自定义 emoji 的客户端显示的替代 Unicode 字符
func customEmoji(docID, fallback string) string {
	return fmt.Sprintf(`<tg-emoji emoji-id="%s">%s</tg-emoji>`, docID, fallback)
}

// BuildWalletHomeText 构建钱包主页消息文本
// 每行使用不同的自定义 emoji (tg-emoji HTML 标签)，需要 Bot 主人有 Telegram Premium
func BuildWalletHomeText(user *entity.User, balances map[string]decimal.Decimal) string {
	displayName := user.DisplayName()
	usdt := formatBalance(balances[entity.CurrencyUSDT])
	trx := formatBalance(balances[entity.CurrencyTRX])
	cny := formatBalance(balances[entity.CurrencyCNY])

	return fmt.Sprintf(
		"%s 昵称: <a href=\"tg://user?id=%d\">%s</a>\n"+
			"%s ID:  <code>%d</code>\n\n"+
			"%s USDT : <code>%s</code>\n"+
			"%s TRX : <code>%s</code>\n"+
			"%s CNY : <code>%s</code>\n"+
			"────────────────────",
		customEmoji(emojiNickname, "💰"), user.TelegramID, displayName,
		customEmoji(emojiID, "💰"), user.TelegramID,
		customEmoji(emojiUSDT, "💰"), usdt,
		customEmoji(emojiTRX, "💰"), trx,
		customEmoji(emojiCNY, "💰"), cny,
	)
}

// formatBalance 格式化余额显示 (去除尾部多余零)
func formatBalance(amount decimal.Decimal) string {
	// 保留最多 4 位小数，去除尾零
	return amount.StringFixed(4)
}

// --- 键盘布局 ---

// BuildMainMenuKeyboard 构建 /start 完整版主菜单键盘
// 13 个按钮，按需求文档排列，带 emoji 图标
func BuildMainMenuKeyboard() *tgpkg.InlineKeyboardBuilder {
	return tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("💰 充值", "Wallet--tronlink_deposit"),
			tgpkg.Button("💸 提币", "Wallet--withdraw_menu"),
		).
		Row(
			tgpkg.Button("💳 转账", "Transfer--send_dashboard"),
			tgpkg.Button("📥 收钱", "Transfer--receive_dashboard"),
		).
		Row(
			tgpkg.Button("🧧 红包", "RedPacket--dashboard"),
		).
		Row(
			tgpkg.Button("👤 个人中心", "Profile--dashboard"),
			tgpkg.Button("🌐 Language", "Setting--language"),
		).
		Row(
			tgpkg.Button("📱 话费/电报星星/会员", "Service--dashboard"),
		).
		Row(
			tgpkg.Button("🤝 商户", "Shop--dashboard"),
			tgpkg.Button("⚡ 闪兑", "Exchange--dashboard"),
		).
		Row(
			tgpkg.Button("🏦 余额宝", "Finance--dashboard"),
			tgpkg.URLButton("💬 官方交易群", "https://t.me/WalletBotGroup"),
		).
		Row(
			tgpkg.URLButton("🎮 游戏中心", "https://t.me/WalletBotGame"),
		)
}

// BuildSimpleMenuKeyboard 构建 /settings 精简版菜单键盘
// 7 个按钮，带 emoji 图标
func BuildSimpleMenuKeyboard() *tgpkg.InlineKeyboardBuilder {
	return tgpkg.NewInlineKeyboard().
		Row(
			tgpkg.Button("💰 充值", "Wallet--tronlink_deposit"),
			tgpkg.Button("💸 提币", "Wallet--withdraw_menu"),
		).
		Row(
			tgpkg.Button("⚡ 闪兑", "Exchange--dashboard"),
			tgpkg.Button("🧧 红包", "RedPacket--dashboard"),
		).
		Row(
			tgpkg.Button("📱 话费/电报星星/会员", "Service--dashboard"),
			tgpkg.Button("🏦 余额宝", "Finance--dashboard"),
		).
		Row(
			tgpkg.Button("👤 个人中心", "Profile--dashboard"),
		)
}
