// Package middleware 提供 HTTP 中间件
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/OwenALL/TG_walletbot/pkg/response"
)

// AdminClaims 管理员 JWT Claims
type AdminClaims struct {
	AdminID  uint64 `json:"admin_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(adminID uint64, username, role, secret string, expireHours int) (string, error) {
	claims := AdminClaims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString, secret string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}

// JWTAuth JWT 认证中间件
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "缺少认证信息")
			c.Abort()
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "认证格式错误")
			c.Abort()
			return
		}

		claims, err := ParseToken(parts[1], secret)
		if err != nil {
			response.Unauthorized(c, "认证已过期或无效")
			c.Abort()
			return
		}

		// 将管理员信息存入 context
		c.Set("admin_id", claims.AdminID)
		c.Set("admin_username", claims.Username)
		c.Set("admin_role", claims.Role)
		c.Next()
	}
}

// AdminOnly 管理员权限检查 (排除 viewer)
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("admin_role")
		if !exists {
			response.Unauthorized(c, "缺少认证信息")
			c.Abort()
			return
		}
		if role == "viewer" {
			response.Error(c, http.StatusForbidden, "只读角色无此操作权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

// SuperAdminOnly 超级管理员权限检查
func SuperAdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("admin_role")
		if !exists {
			response.Unauthorized(c, "缺少认证信息")
			c.Abort()
			return
		}
		if role != "super_admin" {
			response.Error(c, http.StatusForbidden, "需要超级管理员权限")
			c.Abort()
			return
		}
		c.Next()
	}
}
