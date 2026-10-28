package util

import (
	"errors"
	"sync"
	"time"

	"github.com/raids-lab/crater/internal/bizerr"

	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"

	// "github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	jwt "github.com/golang-jwt/jwt/v5"
)

type (
	JWTClaims struct {
		Purpose          string           `json:"purpose"`
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
		AccountAccessMode model.AccessMode `json:"accessMode"`       // AccessMode in account
		PublicAccessMode  model.AccessMode `json:"publicaccessmode"` // Public Accessmode
		RolePlatform      model.Role       `json:"rolePlatform"`     // Role in platform (e.g. guest, user, admin)
	}
)

type TokenManager struct {
	secretKey        string
	refreshSecretKey string
	accessTokenTTL   int
	refreshTokenTTL  int
}

var (
	once     sync.Once
	tokenMgr *TokenManager
)

func GetTokenMgr() *TokenManager {
	once.Do(func() {
		tokenConfig := config.NewTokenConf()
		tokenMgr = newTokenManager(tokenConfig.AccessTokenSecret, tokenConfig.RefreshTokenSecret,
			tokenConfig.AccessTokenExpiryHour,
			tokenConfig.RefreshTokenExpiryHour,
		)
	})
	return tokenMgr
}

func newTokenManager(secretKey, refreshSecretKey string, accessTokenTTL, refreshTokenTTL int) *TokenManager {
	return &TokenManager{
		secretKey,
		refreshSecretKey,
		accessTokenTTL,
		refreshTokenTTL,
	}
}

func (tm *TokenManager) createToken(msg *JWTMessage, ttl int, purpose, secret string) (string, error) {
	expiresAt := time.Now().Add(time.Hour * time.Duration(ttl))

	claims := &JWTClaims{
		Purpose:          purpose,
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
	return token.SignedString([]byte(secret))
}

// CreateTokens creates a new access token and a new refresh token
func (tm *TokenManager) CreateTokens(msg *JWTMessage) (
	accessToken string, refreshToken string, err error) {
	accessToken, err = tm.createToken(msg, tm.accessTokenTTL, "access", tm.secretKey)
	if err != nil {
		klog.Error(err)
		return "", "", err
	}
	refreshToken, err = tm.createToken(msg, tm.refreshTokenTTL, "refresh", tm.refreshSecretKey)
	if err != nil {
		klog.Error(err)
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

func (tm *TokenManager) CheckToken(requestToken string) (JWTMessage, error) {
	return tm.checkToken(requestToken, "access", tm.secretKey)
}

func (tm *TokenManager) CheckRefreshToken(requestToken string) (JWTMessage, error) {
	return tm.checkToken(requestToken, "refresh", tm.refreshSecretKey)
}

func (tm *TokenManager) checkToken(requestToken, purpose, secret string) (JWTMessage, error) {
	claims := JWTClaims{}
	_, err := jwt.ParseWithClaims(requestToken, &claims, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if claims.Purpose != purpose || claims.UserID == 0 || claims.QueueID == 0 {
		err = jwt.ErrTokenInvalidClaims
	}
	if err != nil {
		return JWTMessage{}, err
	}
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

// Only a validly signed, otherwise valid token with an elapsed deadline is expired.
func TokenValidationError(err error) error {
	if errors.Is(err, jwt.ErrTokenExpired) &&
		!errors.Is(err, jwt.ErrTokenSignatureInvalid) &&
		!errors.Is(err, jwt.ErrTokenMalformed) &&
		!errors.Is(err, jwt.ErrTokenNotValidYet) {
		return bizerr.Auth.TokenExpired.Wrap(err, "Session expired. Refresh the session or sign in again.")
	}
	return bizerr.Auth.TokenInvalid.Wrap(err, "Invalid session. Sign in again.")
}

func ValidatePlatformUser(user *model.User) error {
	if user == nil {
		return bizerr.Auth.TokenInvalid.New("Account no longer exists. Sign in again.")
	}
	switch user.Status {
	case model.StatusActive:
		return nil
	case model.StatusPending:
		return bizerr.Auth.AccountNotActivated.New("Account is not activated. Contact the platform administrator.")
	case model.StatusInactive:
		return bizerr.Forbidden.AccountDisabled.New("Account is disabled. Contact the platform administrator.")
	default:
		return bizerr.Internal.ServiceError.New("Account status is invalid. Contact the platform administrator.")
	}
}
