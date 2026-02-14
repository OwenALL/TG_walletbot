package port

import "context"

// NotificationService 通知服务端口
// 异步发送通知，不阻塞主流程
type NotificationService interface {
	// SendDepositNotification 发送充值到账通知
	SendDepositNotification(ctx context.Context, telegramID int64, currency, amount, txHash, newBalance string)
	// SendWithdrawalResult 发送提币处理结果通知
	SendWithdrawalResult(ctx context.Context, telegramID int64, approved bool, currency, amount, txHash, reason string)
	// SendTransferReceived 发送收到转账通知
	SendTransferReceived(ctx context.Context, telegramID int64, senderName, currency, amount, newBalance string)
	// SendPaymentRequest 发送收款请求通知
	SendPaymentRequest(ctx context.Context, telegramID int64, requesterName, currency, amount string, requestID uint64)
	// SendSecurityAlert 发送安全提醒
	SendSecurityAlert(ctx context.Context, telegramID int64, alertType, message string)
}
