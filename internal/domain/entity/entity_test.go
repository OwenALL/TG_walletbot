package entity

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// =============================================================================
// User 实体测试
// =============================================================================

func TestUser_TableName(t *testing.T) {
	u := User{}
	if got := u.TableName(); got != "users" {
		t.Errorf("User.TableName() = %q, 期望 %q", got, "users")
	}
}

func TestUser_FullName(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		want      string
	}{
		{
			name:      "有名有姓",
			firstName: "John",
			lastName:  "Doe",
			want:      "John Doe",
		},
		{
			name:      "只有名",
			firstName: "John",
			lastName:  "",
			want:      "John",
		},
		{
			name:      "名姓都为空",
			firstName: "",
			lastName:  "",
			want:      "",
		},
		{
			name:      "中文名",
			firstName: "张",
			lastName:  "三",
			want:      "张 三",
		},
		{
			name:      "只有姓没有名",
			firstName: "",
			lastName:  "Doe",
			want:      " Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{FirstName: tt.firstName, LastName: tt.lastName}
			if got := u.FullName(); got != tt.want {
				t.Errorf("User.FullName() = %q, 期望 %q", got, tt.want)
			}
		})
	}
}

func TestUser_DisplayName(t *testing.T) {
	tests := []struct {
		name      string
		user      User
		want      string
	}{
		{
			name: "优先使用 FullName",
			user: User{FirstName: "John", LastName: "Doe", Username: "johndoe"},
			want: "John Doe",
		},
		{
			name: "FullName 为空时使用 Username",
			user: User{FirstName: "", LastName: "", Username: "johndoe"},
			want: "@johndoe",
		},
		{
			name: "都为空时返回'未知用户'",
			user: User{FirstName: "", LastName: "", Username: ""},
			want: "未知用户",
		},
		{
			name: "有 FirstName 就不使用 Username",
			user: User{FirstName: "John", Username: "johndoe"},
			want: "John",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &tt.user
			if got := u.DisplayName(); got != tt.want {
				t.Errorf("User.DisplayName() = %q, 期望 %q", got, tt.want)
			}
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"正常状态", UserStatusActive, true},
		{"冻结状态", UserStatusFrozen, false},
		{"封禁状态", UserStatusBanned, false},
		{"未知状态0", 0, false},
		{"未知状态99", 99, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Status: tt.status}
			if got := u.IsActive(); got != tt.want {
				t.Errorf("User.IsActive() = %v, 期望 %v (status=%d)", got, tt.want, tt.status)
			}
		})
	}
}

func TestUser_IsFrozen(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"冻结状态", UserStatusFrozen, true},
		{"正常状态", UserStatusActive, false},
		{"封禁状态", UserStatusBanned, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{Status: tt.status}
			if got := u.IsFrozen(); got != tt.want {
				t.Errorf("User.IsFrozen() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestUser_HasPIN(t *testing.T) {
	tests := []struct {
		name    string
		pinHash string
		want    bool
	}{
		{"已设置 PIN", "abc123hash", true},
		{"未设置 PIN (空字符串)", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{PINHash: tt.pinHash}
			if got := u.HasPIN(); got != tt.want {
				t.Errorf("User.HasPIN() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestUser_IsPINLocked(t *testing.T) {
	futureTime := time.Now().Add(30 * time.Minute)
	pastTime := time.Now().Add(-30 * time.Minute)

	tests := []struct {
		name           string
		pinLockedUntil *time.Time
		want           bool
	}{
		{"未锁定 (nil)", nil, false},
		{"锁定中 (未来时间)", &futureTime, true},
		{"锁定已过期 (过去时间)", &pastTime, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{PINLockedUntil: tt.pinLockedUntil}
			if got := u.IsPINLocked(); got != tt.want {
				t.Errorf("User.IsPINLocked() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestUserStatusConstants(t *testing.T) {
	// 验证状态常量值不重复且符合预期
	statuses := map[int8]string{
		UserStatusActive: "Active",
		UserStatusFrozen: "Frozen",
		UserStatusBanned: "Banned",
	}

	if len(statuses) != 3 {
		t.Error("用户状态常量存在重复值")
	}

	// 验证具体值
	if UserStatusActive != 1 {
		t.Errorf("UserStatusActive = %d, 期望 1", UserStatusActive)
	}
	if UserStatusFrozen != 2 {
		t.Errorf("UserStatusFrozen = %d, 期望 2", UserStatusFrozen)
	}
	if UserStatusBanned != 3 {
		t.Errorf("UserStatusBanned = %d, 期望 3", UserStatusBanned)
	}
}

// =============================================================================
// Wallet 实体测试
// =============================================================================

func TestWallet_TableName(t *testing.T) {
	w := Wallet{}
	if got := w.TableName(); got != "wallets" {
		t.Errorf("Wallet.TableName() = %q, 期望 %q", got, "wallets")
	}
}

func TestWallet_AvailableBalance(t *testing.T) {
	tests := []struct {
		name          string
		balance       decimal.Decimal
		frozenBalance decimal.Decimal
		want          decimal.Decimal
	}{
		{
			name:          "无冻结金额",
			balance:       decimal.NewFromFloat(100.5),
			frozenBalance: decimal.Zero,
			want:          decimal.NewFromFloat(100.5),
		},
		{
			name:          "有冻结金额",
			balance:       decimal.NewFromFloat(100),
			frozenBalance: decimal.NewFromFloat(30),
			want:          decimal.NewFromFloat(70),
		},
		{
			name:          "全部冻结",
			balance:       decimal.NewFromFloat(50),
			frozenBalance: decimal.NewFromFloat(50),
			want:          decimal.Zero,
		},
		{
			name:          "零余额",
			balance:       decimal.Zero,
			frozenBalance: decimal.Zero,
			want:          decimal.Zero,
		},
		{
			name:          "高精度金额",
			balance:       decimal.RequireFromString("100.12345678"),
			frozenBalance: decimal.RequireFromString("0.00000001"),
			want:          decimal.RequireFromString("100.12345677"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wallet{Balance: tt.balance, FrozenBalance: tt.frozenBalance}
			got := w.AvailableBalance()
			if !got.Equal(tt.want) {
				t.Errorf("Wallet.AvailableBalance() = %s, 期望 %s", got, tt.want)
			}
		})
	}
}

func TestWallet_HasSufficientBalance(t *testing.T) {
	tests := []struct {
		name          string
		balance       decimal.Decimal
		frozenBalance decimal.Decimal
		amount        decimal.Decimal
		want          bool
	}{
		{
			name:          "余额充足",
			balance:       decimal.NewFromFloat(100),
			frozenBalance: decimal.NewFromFloat(20),
			amount:        decimal.NewFromFloat(50),
			want:          true,
		},
		{
			name:          "余额刚好够",
			balance:       decimal.NewFromFloat(100),
			frozenBalance: decimal.NewFromFloat(20),
			amount:        decimal.NewFromFloat(80),
			want:          true,
		},
		{
			name:          "余额不足",
			balance:       decimal.NewFromFloat(100),
			frozenBalance: decimal.NewFromFloat(20),
			amount:        decimal.NewFromFloat(81),
			want:          false,
		},
		{
			name:          "全部冻结后余额为零",
			balance:       decimal.NewFromFloat(100),
			frozenBalance: decimal.NewFromFloat(100),
			amount:        decimal.RequireFromString("0.01"),
			want:          false,
		},
		{
			name:          "转账金额为零",
			balance:       decimal.NewFromFloat(100),
			frozenBalance: decimal.Zero,
			amount:        decimal.Zero,
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wallet{Balance: tt.balance, FrozenBalance: tt.frozenBalance}
			if got := w.HasSufficientBalance(tt.amount); got != tt.want {
				t.Errorf("Wallet.HasSufficientBalance(%s) = %v, 期望 %v (available=%s)",
					tt.amount, got, tt.want, w.AvailableBalance())
			}
		})
	}
}

func TestCurrencyConstants(t *testing.T) {
	if CurrencyUSDT != "USDT" {
		t.Errorf("CurrencyUSDT = %q, 期望 %q", CurrencyUSDT, "USDT")
	}
	if CurrencyTRX != "TRX" {
		t.Errorf("CurrencyTRX = %q, 期望 %q", CurrencyTRX, "TRX")
	}
	if CurrencyCNY != "CNY" {
		t.Errorf("CurrencyCNY = %q, 期望 %q", CurrencyCNY, "CNY")
	}

	// 验证不重复
	currencies := map[string]bool{
		CurrencyUSDT: true,
		CurrencyTRX:  true,
		CurrencyCNY:  true,
	}
	if len(currencies) != 3 {
		t.Error("货币类型常量存在重复值")
	}
}

// =============================================================================
// Transaction 实体测试
// =============================================================================

func TestTransaction_TableName(t *testing.T) {
	tx := Transaction{}
	if got := tx.TableName(); got != "transactions" {
		t.Errorf("Transaction.TableName() = %q, 期望 %q", got, "transactions")
	}
}

func TestTransaction_IsIncome(t *testing.T) {
	incomeTypes := []string{
		TxTypeDeposit,
		TxTypeTransferIn,
		TxTypeExchangeIn,
		TxTypeRedpacketRecv,
		TxTypeRedpacketRefund,
		TxTypeFinanceOut,
		TxTypeFinanceProfit,
	}

	expenseTypes := []string{
		TxTypeWithdraw,
		TxTypeTransferOut,
		TxTypeExchangeOut,
		TxTypeRedpacketSend,
		TxTypeFinanceIn,
	}

	for _, txType := range incomeTypes {
		t.Run("收入_"+txType, func(t *testing.T) {
			tx := &Transaction{Type: txType}
			if !tx.IsIncome() {
				t.Errorf("Transaction.IsIncome() = false, 期望 true (type=%s)", txType)
			}
		})
	}

	for _, txType := range expenseTypes {
		t.Run("支出_"+txType, func(t *testing.T) {
			tx := &Transaction{Type: txType}
			if tx.IsIncome() {
				t.Errorf("Transaction.IsIncome() = true, 期望 false (type=%s)", txType)
			}
		})
	}

	// 测试未知类型
	t.Run("未知类型", func(t *testing.T) {
		tx := &Transaction{Type: "unknown_type"}
		if tx.IsIncome() {
			t.Error("Transaction.IsIncome() 对未知类型应返回 false")
		}
	})
}

func TestTransactionTypeConstants(t *testing.T) {
	// 验证所有交易类型常量值不重复
	allTypes := []string{
		TxTypeDeposit,
		TxTypeWithdraw,
		TxTypeTransferIn,
		TxTypeTransferOut,
		TxTypeExchangeIn,
		TxTypeExchangeOut,
		TxTypeRedpacketSend,
		TxTypeRedpacketRecv,
		TxTypeRedpacketRefund,
		TxTypeFinanceIn,
		TxTypeFinanceOut,
		TxTypeFinanceProfit,
	}

	seen := make(map[string]bool)
	for _, txType := range allTypes {
		if txType == "" {
			t.Error("交易类型常量不应为空字符串")
		}
		if seen[txType] {
			t.Errorf("交易类型常量存在重复: %s", txType)
		}
		seen[txType] = true
	}

	if len(seen) != 12 {
		t.Errorf("交易类型常量数量 = %d, 期望 12", len(seen))
	}
}

// =============================================================================
// Deposit 实体测试
// =============================================================================

func TestDeposit_TableName(t *testing.T) {
	d := Deposit{}
	if got := d.TableName(); got != "deposits" {
		t.Errorf("Deposit.TableName() = %q, 期望 %q", got, "deposits")
	}
}

func TestDeposit_IsCredited(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"已到账", DepositStatusCredited, true},
		{"待确认", DepositStatusPending, false},
		{"已确认", DepositStatusConfirmed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Deposit{Status: tt.status}
			if got := d.IsCredited(); got != tt.want {
				t.Errorf("Deposit.IsCredited() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestDepositStatusConstants(t *testing.T) {
	if DepositStatusPending != 0 {
		t.Errorf("DepositStatusPending = %d, 期望 0", DepositStatusPending)
	}
	if DepositStatusConfirmed != 1 {
		t.Errorf("DepositStatusConfirmed = %d, 期望 1", DepositStatusConfirmed)
	}
	if DepositStatusCredited != 2 {
		t.Errorf("DepositStatusCredited = %d, 期望 2", DepositStatusCredited)
	}
}

// =============================================================================
// Withdrawal 实体测试
// =============================================================================

func TestWithdrawal_TableName(t *testing.T) {
	w := Withdrawal{}
	if got := w.TableName(); got != "withdrawals" {
		t.Errorf("Withdrawal.TableName() = %q, 期望 %q", got, "withdrawals")
	}
}

func TestWithdrawal_IsPending(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"待审核", WithdrawalStatusPending, true},
		{"处理中", WithdrawalStatusProcessing, false},
		{"已完成", WithdrawalStatusCompleted, false},
		{"已拒绝", WithdrawalStatusRejected, false},
		{"失败", WithdrawalStatusFailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Withdrawal{Status: tt.status}
			if got := w.IsPending(); got != tt.want {
				t.Errorf("Withdrawal.IsPending() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestWithdrawalStatusConstants(t *testing.T) {
	statuses := map[int8]string{
		WithdrawalStatusPending:    "Pending",
		WithdrawalStatusProcessing: "Processing",
		WithdrawalStatusCompleted:  "Completed",
		WithdrawalStatusRejected:   "Rejected",
		WithdrawalStatusFailed:     "Failed",
	}
	if len(statuses) != 5 {
		t.Error("提币状态常量存在重复值")
	}

	if WithdrawalStatusPending != 0 {
		t.Errorf("WithdrawalStatusPending = %d, 期望 0", WithdrawalStatusPending)
	}
	if WithdrawalStatusProcessing != 1 {
		t.Errorf("WithdrawalStatusProcessing = %d, 期望 1", WithdrawalStatusProcessing)
	}
	if WithdrawalStatusCompleted != 2 {
		t.Errorf("WithdrawalStatusCompleted = %d, 期望 2", WithdrawalStatusCompleted)
	}
	if WithdrawalStatusRejected != 3 {
		t.Errorf("WithdrawalStatusRejected = %d, 期望 3", WithdrawalStatusRejected)
	}
	if WithdrawalStatusFailed != 4 {
		t.Errorf("WithdrawalStatusFailed = %d, 期望 4", WithdrawalStatusFailed)
	}
}

func TestCNYWithdrawal_TableName(t *testing.T) {
	cw := CNYWithdrawal{}
	if got := cw.TableName(); got != "cny_withdrawals" {
		t.Errorf("CNYWithdrawal.TableName() = %q, 期望 %q", got, "cny_withdrawals")
	}
}

// =============================================================================
// Transfer 实体测试
// =============================================================================

func TestTransfer_TableName(t *testing.T) {
	tr := Transfer{}
	if got := tr.TableName(); got != "transfers" {
		t.Errorf("Transfer.TableName() = %q, 期望 %q", got, "transfers")
	}
}

func TestTransfer_IsCompleted(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"已完成", TransferStatusCompleted, true},
		{"待确认", TransferStatusPending, false},
		{"已拒绝", TransferStatusRejected, false},
		{"已取消", TransferStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Transfer{Status: tt.status}
			if got := tr.IsCompleted(); got != tt.want {
				t.Errorf("Transfer.IsCompleted() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestTransferTypeConstants(t *testing.T) {
	if TransferTypeDirect != 1 {
		t.Errorf("TransferTypeDirect = %d, 期望 1", TransferTypeDirect)
	}
	if TransferTypeRequest != 2 {
		t.Errorf("TransferTypeRequest = %d, 期望 2", TransferTypeRequest)
	}
	if TransferTypeDirect == TransferTypeRequest {
		t.Error("转账类型常量不应重复")
	}
}

func TestTransferStatusConstants(t *testing.T) {
	statuses := map[int8]string{
		TransferStatusPending:   "Pending",
		TransferStatusCompleted: "Completed",
		TransferStatusRejected:  "Rejected",
		TransferStatusCancelled: "Cancelled",
	}
	if len(statuses) != 4 {
		t.Error("转账状态常量存在重复值")
	}
}

// =============================================================================
// Exchange 实体测试
// =============================================================================

func TestExchange_TableName(t *testing.T) {
	e := Exchange{}
	if got := e.TableName(); got != "exchanges" {
		t.Errorf("Exchange.TableName() = %q, 期望 %q", got, "exchanges")
	}
}

// =============================================================================
// ExchangeRate 实体测试
// =============================================================================

func TestExchangeRate_TableName(t *testing.T) {
	e := ExchangeRate{}
	if got := e.TableName(); got != "exchange_rates" {
		t.Errorf("ExchangeRate.TableName() = %q, 期望 %q", got, "exchange_rates")
	}
}

func TestExchangeRate_EffectiveRate(t *testing.T) {
	tests := []struct {
		name   string
		rate   decimal.Decimal
		spread decimal.Decimal
		want   decimal.Decimal
	}{
		{
			name:   "无加点 (spread=0)",
			rate:   decimal.NewFromFloat(7.2),
			spread: decimal.Zero,
			want:   decimal.NewFromFloat(7.2),
		},
		{
			name:   "加点1% (spread=1)",
			rate:   decimal.NewFromFloat(100),
			spread: decimal.NewFromFloat(1),
			want:   decimal.NewFromFloat(99), // 100 * (1 - 1/100) = 99
		},
		{
			name:   "加点0.5%",
			rate:   decimal.NewFromFloat(200),
			spread: decimal.NewFromFloat(0.5),
			want:   decimal.NewFromFloat(199), // 200 * (1 - 0.5/100) = 199
		},
		{
			name:   "加点5%",
			rate:   decimal.NewFromFloat(7.2),
			spread: decimal.NewFromFloat(5),
			want:   decimal.NewFromFloat(6.84), // 7.2 * (1 - 5/100) = 6.84
		},
		{
			name:   "高精度汇率",
			rate:   decimal.RequireFromString("7.24561234"),
			spread: decimal.Zero,
			want:   decimal.RequireFromString("7.24561234"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ExchangeRate{Rate: tt.rate, Spread: tt.spread}
			got := e.EffectiveRate()
			if !got.Equal(tt.want) {
				t.Errorf("ExchangeRate.EffectiveRate() = %s, 期望 %s", got, tt.want)
			}
		})
	}
}

// =============================================================================
// RedPacket 实体测试
// =============================================================================

func TestRedPacket_TableName(t *testing.T) {
	r := RedPacket{}
	if got := r.TableName(); got != "red_packets" {
		t.Errorf("RedPacket.TableName() = %q, 期望 %q", got, "red_packets")
	}
}

func TestRedPacketClaim_TableName(t *testing.T) {
	c := RedPacketClaim{}
	if got := c.TableName(); got != "red_packet_claims" {
		t.Errorf("RedPacketClaim.TableName() = %q, 期望 %q", got, "red_packet_claims")
	}
}

func TestRedPacketCover_TableName(t *testing.T) {
	c := RedPacketCover{}
	if got := c.TableName(); got != "red_packet_covers" {
		t.Errorf("RedPacketCover.TableName() = %q, 期望 %q", got, "red_packet_covers")
	}
}

func TestRedPacket_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		expireAt time.Time
		want     bool
	}{
		{
			name:     "已过期 (过去时间)",
			expireAt: time.Now().Add(-1 * time.Hour),
			want:     true,
		},
		{
			name:     "未过期 (未来时间)",
			expireAt: time.Now().Add(1 * time.Hour),
			want:     false,
		},
		{
			name:     "远过期 (很久以前)",
			expireAt: time.Now().Add(-24 * 365 * time.Hour),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedPacket{ExpireAt: tt.expireAt}
			if got := r.IsExpired(); got != tt.want {
				t.Errorf("RedPacket.IsExpired() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestRedPacket_IsFullyClaimed(t *testing.T) {
	tests := []struct {
		name         string
		totalCount   int
		claimedCount int
		want         bool
	}{
		{"全部领取", 10, 10, true},
		{"部分领取", 10, 5, false},
		{"无人领取", 10, 0, false},
		{"领取超过总数 (边界)", 10, 11, true},
		{"单个红包已领取", 1, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedPacket{TotalCount: tt.totalCount, ClaimedCount: tt.claimedCount}
			if got := r.IsFullyClaimed(); got != tt.want {
				t.Errorf("RedPacket.IsFullyClaimed() = %v, 期望 %v (total=%d, claimed=%d)",
					got, tt.want, tt.totalCount, tt.claimedCount)
			}
		})
	}
}

func TestRedPacket_RemainingAmount(t *testing.T) {
	tests := []struct {
		name          string
		totalAmount   decimal.Decimal
		claimedAmount decimal.Decimal
		want          decimal.Decimal
	}{
		{
			name:          "无人领取",
			totalAmount:   decimal.NewFromFloat(100),
			claimedAmount: decimal.Zero,
			want:          decimal.NewFromFloat(100),
		},
		{
			name:          "部分领取",
			totalAmount:   decimal.NewFromFloat(100),
			claimedAmount: decimal.NewFromFloat(30.5),
			want:          decimal.NewFromFloat(69.5),
		},
		{
			name:          "全部领取",
			totalAmount:   decimal.NewFromFloat(100),
			claimedAmount: decimal.NewFromFloat(100),
			want:          decimal.Zero,
		},
		{
			name:          "高精度",
			totalAmount:   decimal.RequireFromString("100.12345678"),
			claimedAmount: decimal.RequireFromString("50.12345677"),
			want:          decimal.RequireFromString("50.00000001"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RedPacket{TotalAmount: tt.totalAmount, ClaimedAmount: tt.claimedAmount}
			got := r.RemainingAmount()
			if !got.Equal(tt.want) {
				t.Errorf("RedPacket.RemainingAmount() = %s, 期望 %s", got, tt.want)
			}
		})
	}
}

func TestRedPacketTypeConstants(t *testing.T) {
	if RedPacketTypeEqual != 1 {
		t.Errorf("RedPacketTypeEqual = %d, 期望 1", RedPacketTypeEqual)
	}
	if RedPacketTypeRandom != 2 {
		t.Errorf("RedPacketTypeRandom = %d, 期望 2", RedPacketTypeRandom)
	}
	if RedPacketTypeEqual == RedPacketTypeRandom {
		t.Error("红包类型常量不应重复")
	}
}

func TestRedPacketStatusConstants(t *testing.T) {
	statuses := map[int8]string{
		RedPacketStatusPending: "Pending",
		RedPacketStatusSent:    "Sent",
		RedPacketStatusClaimed: "Claimed",
		RedPacketStatusExpired: "Expired",
	}
	if len(statuses) != 4 {
		t.Error("红包状态常量存在重复值")
	}

	if RedPacketStatusPending != 0 {
		t.Errorf("RedPacketStatusPending = %d, 期望 0", RedPacketStatusPending)
	}
	if RedPacketStatusSent != 1 {
		t.Errorf("RedPacketStatusSent = %d, 期望 1", RedPacketStatusSent)
	}
	if RedPacketStatusClaimed != 2 {
		t.Errorf("RedPacketStatusClaimed = %d, 期望 2", RedPacketStatusClaimed)
	}
	if RedPacketStatusExpired != 3 {
		t.Errorf("RedPacketStatusExpired = %d, 期望 3", RedPacketStatusExpired)
	}
}

// =============================================================================
// UserSettings 实体测试
// =============================================================================

func TestUserSettings_TableName(t *testing.T) {
	s := UserSettings{}
	if got := s.TableName(); got != "user_settings" {
		t.Errorf("UserSettings.TableName() = %q, 期望 %q", got, "user_settings")
	}
}

func TestUserSettings_NeedPINForAmount(t *testing.T) {
	today := time.Now()
	yesterday := time.Now().AddDate(0, 0, -1)

	tests := []struct {
		name     string
		settings UserSettings
		currency string
		amount   decimal.Decimal
		want     bool
	}{
		{
			name: "USDT 在免密额度内",
			settings: UserSettings{
				SmallFreeUSDT:  decimal.NewFromFloat(100),
				DailySpentUSDT: decimal.NewFromFloat(20),
				DailySpentDate: &today,
			},
			currency: "USDT",
			amount:   decimal.NewFromFloat(50),
			want:     false, // 20 + 50 = 70 <= 100
		},
		{
			name: "USDT 超出免密额度",
			settings: UserSettings{
				SmallFreeUSDT:  decimal.NewFromFloat(100),
				DailySpentUSDT: decimal.NewFromFloat(60),
				DailySpentDate: &today,
			},
			currency: "USDT",
			amount:   decimal.NewFromFloat(50),
			want:     true, // 60 + 50 = 110 > 100
		},
		{
			name: "USDT 刚好等于额度",
			settings: UserSettings{
				SmallFreeUSDT:  decimal.NewFromFloat(100),
				DailySpentUSDT: decimal.NewFromFloat(50),
				DailySpentDate: &today,
			},
			currency: "USDT",
			amount:   decimal.NewFromFloat(50),
			want:     false, // 50 + 50 = 100, 不大于 100
		},
		{
			name: "USDT 免密额度为0 (关闭免密)",
			settings: UserSettings{
				SmallFreeUSDT: decimal.Zero,
			},
			currency: "USDT",
			amount:   decimal.NewFromFloat(1),
			want:     true,
		},
		{
			name: "TRX 在免密额度内",
			settings: UserSettings{
				SmallFreeTRX:  decimal.NewFromFloat(200),
				DailySpentTRX: decimal.NewFromFloat(50),
				DailySpentDate: &today,
			},
			currency: "TRX",
			amount:   decimal.NewFromFloat(100),
			want:     false, // 50 + 100 = 150 <= 200
		},
		{
			name: "TRX 超出免密额度",
			settings: UserSettings{
				SmallFreeTRX:  decimal.NewFromFloat(200),
				DailySpentTRX: decimal.NewFromFloat(150),
				DailySpentDate: &today,
			},
			currency: "TRX",
			amount:   decimal.NewFromFloat(60),
			want:     true, // 150 + 60 = 210 > 200
		},
		{
			name: "CNY 总是需要 PIN",
			settings: UserSettings{
				SmallFreeUSDT: decimal.NewFromFloat(100),
			},
			currency: "CNY",
			amount:   decimal.NewFromFloat(1),
			want:     true,
		},
		{
			name: "未知币种总是需要 PIN",
			settings: UserSettings{
				SmallFreeUSDT: decimal.NewFromFloat(100),
			},
			currency: "BTC",
			amount:   decimal.NewFromFloat(1),
			want:     true,
		},
		{
			name: "消费日期为昨天 (重置为0)",
			settings: UserSettings{
				SmallFreeUSDT:  decimal.NewFromFloat(100),
				DailySpentUSDT: decimal.NewFromFloat(90),
				DailySpentDate: &yesterday,
			},
			currency: "USDT",
			amount:   decimal.NewFromFloat(50),
			want:     false, // 昨天的消费不计入，0 + 50 = 50 <= 100
		},
		{
			name: "消费日期为 nil (首次消费)",
			settings: UserSettings{
				SmallFreeUSDT:  decimal.NewFromFloat(100),
				DailySpentUSDT: decimal.NewFromFloat(90),
				DailySpentDate: nil,
			},
			currency: "USDT",
			amount:   decimal.NewFromFloat(50),
			want:     false, // nil 不匹配今天，dailySpent=0, 0+50=50 <= 100
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &tt.settings
			if got := s.NeedPINForAmount(tt.currency, tt.amount); got != tt.want {
				t.Errorf("UserSettings.NeedPINForAmount(%s, %s) = %v, 期望 %v",
					tt.currency, tt.amount, got, tt.want)
			}
		})
	}
}

// =============================================================================
// SubAccount 实体测试
// =============================================================================

func TestSubAccount_TableName(t *testing.T) {
	s := SubAccount{}
	if got := s.TableName(); got != "sub_accounts" {
		t.Errorf("SubAccount.TableName() = %q, 期望 %q", got, "sub_accounts")
	}
}

func TestSubAccountStatusConstants(t *testing.T) {
	statuses := map[int8]string{
		SubAccountStatusPending:   "Pending",
		SubAccountStatusConfirmed: "Confirmed",
		SubAccountStatusRemoved:   "Removed",
	}
	if len(statuses) != 3 {
		t.Error("备用账号状态常量存在重复值")
	}

	if SubAccountStatusPending != 0 {
		t.Errorf("SubAccountStatusPending = %d, 期望 0", SubAccountStatusPending)
	}
	if SubAccountStatusConfirmed != 1 {
		t.Errorf("SubAccountStatusConfirmed = %d, 期望 1", SubAccountStatusConfirmed)
	}
	if SubAccountStatusRemoved != 2 {
		t.Errorf("SubAccountStatusRemoved = %d, 期望 2", SubAccountStatusRemoved)
	}
}

// =============================================================================
// FinanceInvestment 实体测试
// =============================================================================

func TestFinanceInvestment_TableName(t *testing.T) {
	f := FinanceInvestment{}
	if got := f.TableName(); got != "finance_investments" {
		t.Errorf("FinanceInvestment.TableName() = %q, 期望 %q", got, "finance_investments")
	}
}

func TestFinanceInvestment_DailyProfit(t *testing.T) {
	tests := []struct {
		name       string
		amount     decimal.Decimal
		annualRate decimal.Decimal
		want       decimal.Decimal
	}{
		{
			name:       "年化5%，本金10000",
			amount:     decimal.NewFromFloat(10000),
			annualRate: decimal.NewFromFloat(5),
			want:       decimal.NewFromFloat(10000).Mul(decimal.NewFromFloat(0.05)).Div(decimal.NewFromInt(365)),
		},
		{
			name:       "年化10%，本金1000",
			amount:     decimal.NewFromFloat(1000),
			annualRate: decimal.NewFromFloat(10),
			want:       decimal.NewFromFloat(1000).Mul(decimal.NewFromFloat(0.10)).Div(decimal.NewFromInt(365)),
		},
		{
			name:       "年化0%，本金10000",
			amount:     decimal.NewFromFloat(10000),
			annualRate: decimal.Zero,
			want:       decimal.Zero,
		},
		{
			name:       "本金为0",
			amount:     decimal.Zero,
			annualRate: decimal.NewFromFloat(5),
			want:       decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FinanceInvestment{Amount: tt.amount, AnnualRate: tt.annualRate}
			got := f.DailyProfit()
			if !got.Equal(tt.want) {
				t.Errorf("FinanceInvestment.DailyProfit() = %s, 期望 %s", got, tt.want)
			}
		})
	}
}

func TestFinanceInvestment_IsHolding(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"持有中", FinanceStatusHolding, true},
		{"已取出", FinanceStatusWithdrawn, false},
		{"未知状态", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FinanceInvestment{Status: tt.status}
			if got := f.IsHolding(); got != tt.want {
				t.Errorf("FinanceInvestment.IsHolding() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestFinanceStatusConstants(t *testing.T) {
	if FinanceStatusHolding != 1 {
		t.Errorf("FinanceStatusHolding = %d, 期望 1", FinanceStatusHolding)
	}
	if FinanceStatusWithdrawn != 2 {
		t.Errorf("FinanceStatusWithdrawn = %d, 期望 2", FinanceStatusWithdrawn)
	}
	if FinanceStatusHolding == FinanceStatusWithdrawn {
		t.Error("余额宝状态常量不应重复")
	}
}

// =============================================================================
// AdminUser 实体测试
// =============================================================================

func TestAdminUser_TableName(t *testing.T) {
	a := AdminUser{}
	if got := a.TableName(); got != "admin_users" {
		t.Errorf("AdminUser.TableName() = %q, 期望 %q", got, "admin_users")
	}
}

func TestAdminLog_TableName(t *testing.T) {
	l := AdminLog{}
	if got := l.TableName(); got != "admin_logs" {
		t.Errorf("AdminLog.TableName() = %q, 期望 %q", got, "admin_logs")
	}
}

func TestAdminUser_IsSuperAdmin(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"超级管理员", AdminRoleSuperAdmin, true},
		{"普通管理员", AdminRoleAdmin, false},
		{"只读角色", AdminRoleViewer, false},
		{"空角色", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AdminUser{Role: tt.role}
			if got := a.IsSuperAdmin(); got != tt.want {
				t.Errorf("AdminUser.IsSuperAdmin() = %v, 期望 %v (role=%s)", got, tt.want, tt.role)
			}
		})
	}
}

func TestAdminUser_IsViewer(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"只读角色", AdminRoleViewer, true},
		{"超级管理员", AdminRoleSuperAdmin, false},
		{"普通管理员", AdminRoleAdmin, false},
		{"空角色", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AdminUser{Role: tt.role}
			if got := a.IsViewer(); got != tt.want {
				t.Errorf("AdminUser.IsViewer() = %v, 期望 %v (role=%s)", got, tt.want, tt.role)
			}
		})
	}
}

func TestAdminRoleConstants(t *testing.T) {
	if AdminRoleSuperAdmin != "super_admin" {
		t.Errorf("AdminRoleSuperAdmin = %q, 期望 %q", AdminRoleSuperAdmin, "super_admin")
	}
	if AdminRoleAdmin != "admin" {
		t.Errorf("AdminRoleAdmin = %q, 期望 %q", AdminRoleAdmin, "admin")
	}
	if AdminRoleViewer != "viewer" {
		t.Errorf("AdminRoleViewer = %q, 期望 %q", AdminRoleViewer, "viewer")
	}

	// 验证不重复
	roles := map[string]bool{
		AdminRoleSuperAdmin: true,
		AdminRoleAdmin:      true,
		AdminRoleViewer:     true,
	}
	if len(roles) != 3 {
		t.Error("管理员角色常量存在重复值")
	}
}

// =============================================================================
// WalletKey 实体测试
// =============================================================================

func TestWalletKey_TableName(t *testing.T) {
	wk := WalletKey{}
	if got := wk.TableName(); got != "wallet_keys" {
		t.Errorf("WalletKey.TableName() = %q, 期望 %q", got, "wallet_keys")
	}
}

// =============================================================================
// SystemConfig 实体测试
// =============================================================================

func TestSystemConfig_TableName(t *testing.T) {
	sc := SystemConfig{}
	if got := sc.TableName(); got != "system_configs" {
		t.Errorf("SystemConfig.TableName() = %q, 期望 %q", got, "system_configs")
	}
}

func TestSystemConfigKeyConstants(t *testing.T) {
	// 收集所有系统配置 key 常量
	allKeys := []string{
		ConfigWithdrawUSDTMin,
		ConfigWithdrawUSDTMax,
		ConfigWithdrawUSDTDailyMax,
		ConfigWithdrawUSDTFee,
		ConfigWithdrawUSDTAutoThreshold,
		ConfigWithdrawTRXMin,
		ConfigWithdrawTRXMax,
		ConfigWithdrawTRXFee,
		ConfigWithdrawCNYMin,
		ConfigWithdrawCNYDailyMax,
		ConfigTransferUSDTMin,
		ConfigExchangeMinUSDT,
		ConfigExchangeMinCNY,
		ConfigExchangeMinTRX,
		ConfigFinanceMin,
		ConfigFinanceMax,
		ConfigFinanceAnnualRate,
		ConfigPINMaxFail,
		ConfigPINLockMinutes,
		ConfigRedpacketExpireHours,
		ConfigDepositConfirmations,
		ConfigAdminTelegramID,
		ConfigSupportURL,
		ConfigTradingGroupURL,
		ConfigGameCenterURL,
		ConfigMaintenanceMode,
		ConfigTronFeeLimit,
		ConfigTronUSDTContract,
		ConfigTronGRPCTimeout,
		ConfigTronGRPCBatchTimeout,
	}

	// 验证每个 key 非空
	for _, key := range allKeys {
		if key == "" {
			t.Errorf("系统配置 key 不应为空字符串")
		}
	}

	// 验证不重复
	seen := make(map[string]bool)
	for _, key := range allKeys {
		if seen[key] {
			t.Errorf("系统配置 key 重复: %s", key)
		}
		seen[key] = true
	}

	expectedCount := 30
	if len(seen) != expectedCount {
		t.Errorf("系统配置 key 数量 = %d, 期望 %d", len(seen), expectedCount)
	}

	// 验证具体常量值
	configExpected := map[string]string{
		"ConfigWithdrawUSDTMin":           "withdraw_usdt_min",
		"ConfigWithdrawUSDTMax":           "withdraw_usdt_max",
		"ConfigWithdrawUSDTDailyMax":      "withdraw_usdt_daily_max",
		"ConfigWithdrawUSDTFee":           "withdraw_usdt_fee",
		"ConfigWithdrawUSDTAutoThreshold": "withdraw_usdt_auto_threshold",
		"ConfigWithdrawTRXMin":            "withdraw_trx_min",
		"ConfigWithdrawTRXMax":            "withdraw_trx_max",
		"ConfigWithdrawTRXFee":            "withdraw_trx_fee",
		"ConfigWithdrawCNYMin":            "withdraw_cny_min",
		"ConfigWithdrawCNYDailyMax":       "withdraw_cny_daily_max",
		"ConfigTransferUSDTMin":           "transfer_usdt_min",
		"ConfigExchangeMinUSDT":           "exchange_min_usdt",
		"ConfigExchangeMinCNY":            "exchange_min_cny",
		"ConfigExchangeMinTRX":            "exchange_min_trx",
		"ConfigFinanceMin":                "finance_min",
		"ConfigFinanceMax":                "finance_max",
		"ConfigFinanceAnnualRate":         "finance_annual_rate",
		"ConfigPINMaxFail":                "pin_max_fail",
		"ConfigPINLockMinutes":            "pin_lock_minutes",
		"ConfigRedpacketExpireHours":      "redpacket_expire_hours",
		"ConfigDepositConfirmations":      "deposit_confirmations",
		"ConfigAdminTelegramID":           "admin_telegram_id",
		"ConfigSupportURL":                "support_url",
		"ConfigTradingGroupURL":           "trading_group_url",
		"ConfigGameCenterURL":             "game_center_url",
		"ConfigMaintenanceMode":           "maintenance_mode",
		"ConfigTronFeeLimit":              "tron_fee_limit",
		"ConfigTronUSDTContract":          "tron_usdt_contract",
		"ConfigTronGRPCTimeout":           "tron_grpc_timeout",
		"ConfigTronGRPCBatchTimeout":      "tron_grpc_batch_timeout",
	}

	// 用实际常量值做映射验证
	actualValues := map[string]string{
		"ConfigWithdrawUSDTMin":           ConfigWithdrawUSDTMin,
		"ConfigWithdrawUSDTMax":           ConfigWithdrawUSDTMax,
		"ConfigWithdrawUSDTDailyMax":      ConfigWithdrawUSDTDailyMax,
		"ConfigWithdrawUSDTFee":           ConfigWithdrawUSDTFee,
		"ConfigWithdrawUSDTAutoThreshold": ConfigWithdrawUSDTAutoThreshold,
		"ConfigWithdrawTRXMin":            ConfigWithdrawTRXMin,
		"ConfigWithdrawTRXMax":            ConfigWithdrawTRXMax,
		"ConfigWithdrawTRXFee":            ConfigWithdrawTRXFee,
		"ConfigWithdrawCNYMin":            ConfigWithdrawCNYMin,
		"ConfigWithdrawCNYDailyMax":       ConfigWithdrawCNYDailyMax,
		"ConfigTransferUSDTMin":           ConfigTransferUSDTMin,
		"ConfigExchangeMinUSDT":           ConfigExchangeMinUSDT,
		"ConfigExchangeMinCNY":            ConfigExchangeMinCNY,
		"ConfigExchangeMinTRX":            ConfigExchangeMinTRX,
		"ConfigFinanceMin":                ConfigFinanceMin,
		"ConfigFinanceMax":                ConfigFinanceMax,
		"ConfigFinanceAnnualRate":         ConfigFinanceAnnualRate,
		"ConfigPINMaxFail":                ConfigPINMaxFail,
		"ConfigPINLockMinutes":            ConfigPINLockMinutes,
		"ConfigRedpacketExpireHours":      ConfigRedpacketExpireHours,
		"ConfigDepositConfirmations":      ConfigDepositConfirmations,
		"ConfigAdminTelegramID":           ConfigAdminTelegramID,
		"ConfigSupportURL":                ConfigSupportURL,
		"ConfigTradingGroupURL":           ConfigTradingGroupURL,
		"ConfigGameCenterURL":             ConfigGameCenterURL,
		"ConfigMaintenanceMode":           ConfigMaintenanceMode,
		"ConfigTronFeeLimit":              ConfigTronFeeLimit,
		"ConfigTronUSDTContract":          ConfigTronUSDTContract,
		"ConfigTronGRPCTimeout":           ConfigTronGRPCTimeout,
		"ConfigTronGRPCBatchTimeout":      ConfigTronGRPCBatchTimeout,
	}

	for constName, expectedValue := range configExpected {
		actualValue, ok := actualValues[constName]
		if !ok {
			t.Errorf("缺少常量 %s", constName)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("%s = %q, 期望 %q", constName, actualValue, expectedValue)
		}
	}
}

// =============================================================================
// Merchant 实体测试
// =============================================================================

func TestMerchant_TableName(t *testing.T) {
	m := Merchant{}
	if got := m.TableName(); got != "merchants" {
		t.Errorf("Merchant.TableName() = %q, 期望 %q", got, "merchants")
	}
}

func TestMerchant_IsApproved(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"已通过", MerchantStatusApproved, true},
		{"待审核", MerchantStatusPending, false},
		{"已拒绝", MerchantStatusRejected, false},
		{"已禁用", MerchantStatusDisabled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Merchant{Status: tt.status}
			if got := m.IsApproved(); got != tt.want {
				t.Errorf("Merchant.IsApproved() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestMerchant_IsPending(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"待审核", MerchantStatusPending, true},
		{"已通过", MerchantStatusApproved, false},
		{"已拒绝", MerchantStatusRejected, false},
		{"已禁用", MerchantStatusDisabled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Merchant{Status: tt.status}
			if got := m.IsPending(); got != tt.want {
				t.Errorf("Merchant.IsPending() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestMerchant_IsRejected(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"已拒绝", MerchantStatusRejected, true},
		{"待审核", MerchantStatusPending, false},
		{"已通过", MerchantStatusApproved, false},
		{"已禁用", MerchantStatusDisabled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Merchant{Status: tt.status}
			if got := m.IsRejected(); got != tt.want {
				t.Errorf("Merchant.IsRejected() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestMerchant_IsDisabled(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   bool
	}{
		{"已禁用", MerchantStatusDisabled, true},
		{"待审核", MerchantStatusPending, false},
		{"已通过", MerchantStatusApproved, false},
		{"已拒绝", MerchantStatusRejected, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Merchant{Status: tt.status}
			if got := m.IsDisabled(); got != tt.want {
				t.Errorf("Merchant.IsDisabled() = %v, 期望 %v", got, tt.want)
			}
		})
	}
}

func TestMerchant_StatusText(t *testing.T) {
	tests := []struct {
		name   string
		status int8
		want   string
	}{
		{"待审核", MerchantStatusPending, "待审核"},
		{"正常", MerchantStatusActive, "正常"},
		{"已拒绝", MerchantStatusRejected, "已拒绝"},
		{"已关闭", MerchantStatusDisabled, "已关闭"},
		{"未知状态", 99, "未知"},
		{"负数状态", -1, "未知"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Merchant{Status: tt.status}
			if got := m.StatusText(); got != tt.want {
				t.Errorf("Merchant.StatusText() = %q, 期望 %q (status=%d)", got, tt.want, tt.status)
			}
		})
	}
}

func TestMerchantStatusConstants(t *testing.T) {
	// MerchantStatusApproved 是 MerchantStatusActive 的别名，它们共享同一个值
	// 因此有效的不同状态值为 4 个: Disabled(0), Active(1), Pending(2), Rejected(3)
	statuses := map[int8]string{
		MerchantStatusDisabled: "Disabled",
		MerchantStatusActive:   "Active",
		MerchantStatusPending:  "Pending",
		MerchantStatusRejected: "Rejected",
	}
	if len(statuses) != 4 {
		t.Error("商户状态常量存在重复值")
	}

	if MerchantStatusDisabled != 0 {
		t.Errorf("MerchantStatusDisabled = %d, 期望 0", MerchantStatusDisabled)
	}
	if MerchantStatusActive != 1 {
		t.Errorf("MerchantStatusActive = %d, 期望 1", MerchantStatusActive)
	}
	if MerchantStatusApproved != MerchantStatusActive {
		t.Errorf("MerchantStatusApproved = %d, 期望与 MerchantStatusActive(%d) 相同", MerchantStatusApproved, MerchantStatusActive)
	}
	if MerchantStatusPending != 2 {
		t.Errorf("MerchantStatusPending = %d, 期望 2", MerchantStatusPending)
	}
	if MerchantStatusRejected != 3 {
		t.Errorf("MerchantStatusRejected = %d, 期望 3", MerchantStatusRejected)
	}
}
