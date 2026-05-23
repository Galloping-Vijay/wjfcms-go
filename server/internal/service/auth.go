package service

import (
	"errors"
	"time"

	"wjfcm-go/internal/config"
	"wjfcm-go/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AdminClaims struct {
	AdminID uint64 `json:"admin_id"`
	Account string `json:"account"`
	Type    string `json:"type"`
	jwt.RegisteredClaims
}

type UserClaims struct {
	UserID uint64 `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type LoginResult struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	Admin        model.Admin `json:"admin"`
	Permissions  []string    `json:"permissions"`
}

type UserLoginResult struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

func LoginAdmin(db *gorm.DB, cfg config.Config, account string, password string) (LoginResult, error) {
	var admin model.Admin
	if err := db.Where("account = ? OR tel = ? OR email = ?", account, account, account).First(&admin).Error; err != nil {
		return LoginResult{}, errors.New("账号或密码错误")
	}
	if admin.Status != 1 {
		return LoginResult{}, errors.New("账号已被禁用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return LoginResult{}, errors.New("账号或密码错误")
	}

	token, err := MakeAdminToken(admin, cfg)
	if err != nil {
		return LoginResult{}, err
	}
	refreshToken, err := MakeAdminRefreshToken(admin, cfg)
	if err != nil {
		return LoginResult{}, err
	}

	permissions, err := AdminPermissionURLs(db, admin.ID)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{Token: token, RefreshToken: refreshToken, Admin: admin, Permissions: permissions}, nil
}

func MakeAdminToken(admin model.Admin, cfg config.Config) (string, error) {
	return makeAdminJWT(admin, cfg, "access", time.Duration(cfg.JWT.ExpiresMinutes)*time.Minute)
}

func MakeAdminRefreshToken(admin model.Admin, cfg config.Config) (string, error) {
	return makeAdminJWT(admin, cfg, "refresh", time.Duration(cfg.JWT.RefreshExpiresMinutes)*time.Minute)
}

func makeAdminJWT(admin model.Admin, cfg config.Config, tokenType string, ttl time.Duration) (string, error) {
	expiresAt := time.Now().Add(ttl)
	claims := AdminClaims{
		AdminID: admin.ID,
		Account: admin.Account,
		Type:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   admin.Account,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

func ParseAdminToken(tokenText string, cfg config.Config) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenText, &AdminClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Type != "" && claims.Type != "access" {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}

func ParseAdminRefreshToken(tokenText string, cfg config.Config) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenText, &AdminClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}

func LoginUser(db *gorm.DB, cfg config.Config, account string, password string) (UserLoginResult, error) {
	var user model.User
	if err := db.Where("email = ? OR tel = ? OR name = ?", account, account, account).First(&user).Error; err != nil {
		return UserLoginResult{}, errors.New("账号或密码错误")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return UserLoginResult{}, errors.New("账号或密码错误")
	}

	token, err := MakeUserToken(user, cfg)
	if err != nil {
		return UserLoginResult{}, err
	}
	return UserLoginResult{Token: token, User: user}, nil
}

func MakeUserToken(user model.User, cfg config.Config) (string, error) {
	expiresAt := time.Now().Add(time.Duration(cfg.JWT.ExpiresMinutes) * time.Minute)
	claims := UserClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

func ParseUserToken(tokenText string, cfg config.Config) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenText, &UserClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
