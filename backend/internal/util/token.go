package util

import (
	"sync"
	"time"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"

	// "github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	jwt "github.com/golang-jwt/jwt/v5"
)

type (
	JWTClaims struct {
		UserID       uint       `json:"uid"` // User ID
		ProjectID    uint       `json:"pid"` // Project ID
		ProjectRole  model.Role `json:"pro"` // User role of the project
		ClusterID    uint       `json:"cid"` // Cluster ID
		ClusterRole  model.Role `json:"cro"` // User role of the cluster
		PlatformRole model.Role `json:"plf"` // User role of the platform
		jwt.RegisteredClaims
	}
	JWTMessage struct {
		UserID       uint       `json:"uid"` // User ID
		ProjectID    uint       `json:"pid"` // Project ID
		ProjectRole  model.Role `json:"pro"` // User role of the project
		ClusterID    uint       `json:"cid"` // Cluster ID
		ClusterRole  model.Role `json:"cro"` // User role of the cluster
		PlatformRole model.Role `json:"plf"` // User role of the platform
	}
)

type TokenManager struct {
	secretKey       string
	accessTokenTTL  int
	refreshTokenTTL int
}

const (
	DefaultClusterID   uint = 0
	DefaultClusterRole      = model.RoleGuest
)

var (
	once     sync.Once
	tokenMgr *TokenManager
)

func GetTokenMgr() *TokenManager {
	once.Do(func() {
		tokenConfig := config.NewTokenConf()
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
		UserID:       msg.UserID,
		ProjectID:    msg.ProjectID,
		ClusterID:    msg.ClusterID,
		ClusterRole:  msg.ClusterRole,
		ProjectRole:  msg.ProjectRole,
		PlatformRole: msg.PlatformRole,
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
		UserID:       claims.UserID,
		ProjectID:    claims.ProjectID,
		ProjectRole:  claims.ProjectRole,
		ClusterID:    claims.ClusterID,
		ClusterRole:  claims.ClusterRole,
		PlatformRole: claims.PlatformRole,
	}, err
}
