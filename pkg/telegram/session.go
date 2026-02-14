package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Session 用户会话数据
type Session struct {
	Module string                 `json:"module"` // 当前模块 (Wallet/Transfer/Exchange 等)
	Step   string                 `json:"step"`   // 当前步骤
	Data   map[string]interface{} `json:"data"`   // 临时数据
}

// SessionService Redis 会话管理服务
type SessionService struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewSessionService 创建会话管理服务
func NewSessionService(rdb *redis.Client) *SessionService {
	return &SessionService{
		rdb: rdb,
		ttl: 30 * time.Minute,
	}
}

// sessionKey 生成会话 Redis key
func (s *SessionService) sessionKey(userID int64) string {
	return fmt.Sprintf("session:%d", userID)
}

// Get 获取用户当前会话
// 如果不存在返回空闲状态会话
func (s *SessionService) Get(ctx context.Context, userID int64) *Session {
	data, err := s.rdb.Get(ctx, s.sessionKey(userID)).Bytes()
	if err != nil {
		return &Session{Module: "", Step: "", Data: make(map[string]interface{})}
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return &Session{Module: "", Step: "", Data: make(map[string]interface{})}
	}
	if session.Data == nil {
		session.Data = make(map[string]interface{})
	}
	return &session
}

// SetStep 设置当前会话步骤
func (s *SessionService) SetStep(ctx context.Context, userID int64, module, step string) error {
	session := s.Get(ctx, userID)
	session.Module = module
	session.Step = step
	return s.save(ctx, userID, session)
}

// SetStepWithData 设置步骤同时写入数据
func (s *SessionService) SetStepWithData(ctx context.Context, userID int64, module, step string, data map[string]interface{}) error {
	session := s.Get(ctx, userID)
	session.Module = module
	session.Step = step
	for k, v := range data {
		session.Data[k] = v
	}
	return s.save(ctx, userID, session)
}

// GetStep 获取当前步骤
func (s *SessionService) GetStep(ctx context.Context, userID int64) (module, step string) {
	session := s.Get(ctx, userID)
	return session.Module, session.Step
}

// SetData 存储临时数据
func (s *SessionService) SetData(ctx context.Context, userID int64, key string, value interface{}) error {
	session := s.Get(ctx, userID)
	session.Data[key] = value
	return s.save(ctx, userID, session)
}

// GetData 获取临时数据
func (s *SessionService) GetData(ctx context.Context, userID int64, key string) (interface{}, bool) {
	session := s.Get(ctx, userID)
	v, ok := session.Data[key]
	return v, ok
}

// GetDataString 获取临时数据为字符串
func (s *SessionService) GetDataString(ctx context.Context, userID int64, key string) string {
	v, ok := s.GetData(ctx, userID, key)
	if !ok {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return str
}

// GetDataFloat 获取临时数据为浮点数
func (s *SessionService) GetDataFloat(ctx context.Context, userID int64, key string) float64 {
	v, ok := s.GetData(ctx, userID, key)
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

// Clear 清除用户会话
func (s *SessionService) Clear(ctx context.Context, userID int64) error {
	return s.rdb.Del(ctx, s.sessionKey(userID)).Err()
}

// IsIdle 判断用户是否处于空闲状态 (无进行中的会话)
func (s *SessionService) IsIdle(ctx context.Context, userID int64) bool {
	session := s.Get(ctx, userID)
	return session.Module == "" && session.Step == ""
}

// save 序列化并保存会话到 Redis
func (s *SessionService) save(ctx context.Context, userID int64, session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("序列化会话数据失败: %w", err)
	}
	return s.rdb.Set(ctx, s.sessionKey(userID), data, s.ttl).Err()
}
