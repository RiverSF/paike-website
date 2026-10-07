package middleware

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"tutoring_server/internal/config"
	"tutoring_server/internal/model"
)

const (
	CtxUserID   = "userID"
	CtxUsername = "username"
	userIDClaim = "user_id"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 签发 JWT。
func GenerateToken(u *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(tokenExpireHours()) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   strconv.FormatUint(uint64(u.ID), 10),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtSecret()))
}

// ParseToken 解析 JWT。
func ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(*jwt.Token) (interface{}, error) {
		return []byte(jwtSecret()), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return claims, nil
}

// AuthRequired 校验登录态，并把用户写入上下文。
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			abortJSON(c, 401, "请先登录")
			return
		}
		claims, err := ParseToken(tokenStr)
		if err != nil {
			abortJSON(c, 401, "登录已失效，请重新登录")
			return
		}

		user, err := model.NewUserModel().GetByID(claims.UserID)
		if err != nil {
			abortJSON(c, 401, "用户不存在或已被删除")
			return
		}
		if user.Status == model.UserStatusFrozen {
			abortJSON(c, 403, "账号已被冻结，请联系管理员")
			return
		}

		c.Set(CtxUserID, user.ID)
		c.Set(CtxUsername, user.Username)
		c.Set("user", user)
		c.Next()
	}
}

// MemberRequired 校验会员是否在有效期内（订单/课表功能仅对会员开放）。
func MemberRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user")
		if !ok {
			abortJSON(c, 401, "请先登录")
			return
		}
		user, ok := v.(*model.User)
		// 站点拥有者 / 管理员（永久会员）不受会员有效期限制
		if !ok || (!user.MemberActive() && !user.IsStaff()) {
			abortJSON(c, 403, "会员已过期，请续费后使用订单与课表功能")
			return
		}
		c.Next()
	}
}

// StaffRequired 校验管理员或站点拥有者身份。
func StaffRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("user")
		if !ok {
			abortJSON(c, 401, "请先登录")
			return
		}
		user, ok := v.(*model.User)
		if !ok || !user.IsStaff() {
			abortJSON(c, 403, "无权限访问")
			return
		}
		c.Next()
	}
}

// CurrentUser 从上下文取当前登录用户。
func CurrentUser(c *gin.Context) *model.User {
	v, ok := c.Get("user")
	if !ok {
		return nil
	}
	u, _ := v.(*model.User)
	return u
}

func extractToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if len(header) > 7 && (header[:7] == "Bearer " || header[:7] == "bearer ") {
		return header[7:]
	}
	if t := c.Query("token"); t != "" {
		return t
	}
	return ""
}

func jwtSecret() string {
	// 优先级：环境变量 > app.ini [jwt] JWT_SECRET > 内置兜底。
	// 此前只读环境变量，app.ini 里配置的密钥完全不生效，本地与生产实际都在用
	// 同一个硬编码默认密钥（该默认值随源码公开，等同于无密钥）。
	if v := os.Getenv("JWT_SECRET"); v != "" {
		return v
	}
	if config.JwtConfig != nil {
		if v := strings.TrimSpace(config.JwtConfig.Secret); v != "" {
			return v
		}
	}
	return "tutoring-website-secret"
}

func tokenExpireHours() int {
	if v := os.Getenv("JWT_EXPIRE_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	if config.JwtConfig != nil && config.JwtConfig.ExpireHours > 0 {
		return config.JwtConfig.ExpireHours
	}
	return 168
}
