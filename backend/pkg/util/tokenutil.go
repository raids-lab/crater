package util

import (
	"time"

	"github.com/raids-lab/crater/pkg/models"

	// "github.com/amitshekhariitbhu/go-backend-clean-architecture/domain"
	jwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	ID        uint   `json:"id"`
	UserName  string `json:"username"`
	Role      string `json:"role"`
	NameSpace string `json:"namespace"`
	jwt.RegisteredClaims
}

// CreateAccessToken generates a new access token for the given user with the specified secret and expiry time.
// It returns the generated access token and any error encountered during the process.
func CreateAccessToken(user *models.User, secret string, expiry int) (accessToken string, err error) {
	expirationTime := time.Now().Add(time.Hour * time.Duration(expiry))

	claims := &Claims{
		ID:        user.ID,
		UserName:  user.UserName,
		Role:      user.Role,
		NameSpace: user.NameSpace,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime), // Use jwt.NewNumericDate for type-safe expiration
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func CreateRefreshToken(user *models.User, secret string, expiry int) (refreshToken string, err error) {
	expirationTime := time.Now().Add(time.Hour * time.Duration(expiry))
	claimsRefresh := &Claims{
		ID:        user.ID,
		UserName:  user.UserName,
		Role:      user.Role,
		NameSpace: user.NameSpace,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claimsRefresh)
	rt, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return rt, err
}

func CheckAndGetUser(requestToken string, secret string) (models.User, error) {
	claims := Claims{}
	_, err := jwt.ParseWithClaims(requestToken, &claims, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	return models.User{
		UserName:  claims.UserName,
		Role:      claims.Role,
		NameSpace: claims.NameSpace,
	}, err
}
