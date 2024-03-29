package util

import (
	"time"

	"github.com/raids-lab/crater/pkg/logutils"

	// "github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	jwt "github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UID          uint   `json:"uid"`          // User ID
	PID          uint   `json:"pid"`          // Project ID
	PlatformRole string `json:"platformRole"` // User role of the platform
	ProjectRole  string `json:"projectRole"`  // User role of the project
	jwt.RegisteredClaims
}

type JWTMessage struct {
	UID          uint   `json:"uid"`          // User ID
	PID          uint   `json:"pid"`          // Project ID
	PlatformRole string `json:"platformRole"` // User role of the platform
	ProjectRole  string `json:"projectRole"`  // User role of the project
}

type TokenManager struct {
	secretKey       string
	accessTokenTTL  int
	refreshTokenTTL int
}

func NewTokenManager(secretKey string, accessTokenTTL, refreshTokenTTL int) *TokenManager {
	return &TokenManager{
		secretKey,
		accessTokenTTL,
		refreshTokenTTL,
	}
}

func (tm *TokenManager) createToken(msg *JWTMessage, ttl int) (string, error) {
	expiresAt := time.Now().Add(time.Hour * time.Duration(ttl))

	claims := &JWTClaims{
		UID:          msg.UID,
		PID:          msg.PID,
		PlatformRole: msg.PlatformRole,
		ProjectRole:  msg.ProjectRole,
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
		UID:          claims.UID,
		PID:          claims.PID,
		PlatformRole: claims.PlatformRole,
		ProjectRole:  claims.ProjectRole,
	}, err
}
