package util

import (
	"sync"
	"time"
	"webdav/config"
	"webdav/dao/model"
	"webdav/logutils"

	// "github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	jwt "github.com/golang-jwt/jwt/v5"
)

type TokenConf struct {
	ContextTimeout         int    `mapstructure:"CONTEXT_TIMEOUT"`
	AccessTokenExpiryHour  int    `mapstructure:"ACCESS_TOKEN_EXPIRY_HOUR"`
	RefreshTokenExpiryHour int    `mapstructure:"REFRESH_TOKEN_EXPIRY_HOUR"`
	AccessTokenSecret      string `mapstructure:"ACCESS_TOKEN_SECRET"`
	RefreshTokenSecret     string `mapstructure:"REFRESH_TOKEN_SECRET"`
}

func NewTokenConf() *TokenConf {
	return &TokenConf{
		ContextTimeout:         2,
		AccessTokenExpiryHour:  1,
		RefreshTokenExpiryHour: 168,
		AccessTokenSecret:      config.GetConfig().Auth.AccessTokenSecret,
		RefreshTokenSecret:     config.GetConfig().Auth.RefreshTokenSecret,
	}
}

type (
	JWTClaims struct {
		UserID           uint             `json:"ui"`
		QueueID          uint             `json:"qi"`
		Username         string           `json:"un"`
		QueueName        string           `json:"qn"`
		RoleQueue        model.Role       `json:"rq"`
		RolePlatform     model.Role       `json:"rp"`
		AccessMode       model.AccessMode `json:"am"`
		PublicAccessMode model.AccessMode `json:"pa"`
		jwt.RegisteredClaims
	}
	JWTMessage struct {
		UserID            uint             `json:"userID"`           // User ID
		AccountID         uint             `json:"queueID"`          // Queue ID
		Username          string           `json:"username"`         // Username
		AccountName       string           `json:"queueName"`        // Queue name
		RoleAccount       model.Role       `json:"roleQueue"`        // Role in queue (e.g. user, admin)
		AccountAccessMode model.AccessMode `json:"accessMode"`       // AccessMode in queue
		PublicAccessMode  model.AccessMode `json:"publicaccessmode"` // Public Accessmode
		RolePlatform      model.Role       `json:"rolePlatform"`     // Role in platform (e.g. guest, user, admin)
	}
)

const (
	QueueNameNull = ""
	QueueIDNull   = 0
	QueueDefault  = 1
)

type TokenManager struct {
	secretKey       string
	accessTokenTTL  int
	refreshTokenTTL int
}

var (
	once     sync.Once
	tokenMgr *TokenManager
)

func GetTokenMgr() *TokenManager {
	once.Do(func() {
		tokenConfig := NewTokenConf()
		tokenMgr = newTokenManager(tokenConfig.AccessTokenSecret,
			tokenConfig.AccessTokenExpiryHour,
			tokenConfig.RefreshTokenExpiryHour,
		)
	})
	return tokenMgr
}

func newTokenManager(secretKey string, accessTokenTTL, refreshTokenTTL int) *TokenManager {
	return &TokenManager{
		secretKey,
		accessTokenTTL,
		refreshTokenTTL,
	}
}
func (tm *TokenManager) createToken(msg *JWTMessage, ttl int) (string, error) {
	expiresAt := time.Now().Add(time.Hour * time.Duration(ttl))

	claims := &JWTClaims{
		UserID:           msg.UserID,
		QueueID:          msg.AccountID,
		Username:         msg.Username,
		QueueName:        msg.AccountName,
		RoleQueue:        msg.RoleAccount,
		RolePlatform:     msg.RolePlatform,
		AccessMode:       msg.AccountAccessMode,
		PublicAccessMode: msg.PublicAccessMode,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tm.secretKey))
}

// CreateTokens creates a new access token and a new refresh token
func (tm *TokenManager) CreateTokens(msg *JWTMessage) (
	accessToken string, refreshToken string, err error) {
	accessToken, err = tm.createToken(msg, tm.accessTokenTTL)
	if err != nil {
		logutils.Log.Error(err)
		return "", "", err
	}
	refreshToken, err = tm.createToken(msg, tm.refreshTokenTTL)
	if err != nil {
		logutils.Log.Error(err)
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (tm *TokenManager) CheckToken(requestToken string) (JWTMessage, error) {
	claims := JWTClaims{}
	_, err := jwt.ParseWithClaims(requestToken, &claims, func(_ *jwt.Token) (any, error) {
		return []byte(tm.secretKey), nil
	})
	return JWTMessage{
		UserID:            claims.UserID,
		AccountID:         claims.QueueID,
		Username:          claims.Username,
		AccountName:       claims.QueueName,
		RoleAccount:       claims.RoleQueue,
		RolePlatform:      claims.RolePlatform,
		AccountAccessMode: claims.AccessMode,
		PublicAccessMode:  claims.PublicAccessMode,
	}, err
}
