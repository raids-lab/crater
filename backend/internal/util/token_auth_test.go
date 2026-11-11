package util

import (
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/raids-lab/crater/internal/bizerr"
)

func TestAuthenticationTokenPurposesAndValidation(t *testing.T) {
	tm := newTokenManager("synthetic-access", "synthetic-refresh", 1, 24)
	msg := &JWTMessage{UserID: 1, AccountID: 1}
	access, refresh, err := tm.CreateTokens(msg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tm.CheckToken(access); err != nil {
		t.Fatal(err)
	}
	if _, err := tm.CheckRefreshToken(refresh); err != nil {
		t.Fatal(err)
	}
	if _, err := tm.CheckToken(refresh); err == nil {
		t.Fatal("refresh token authorized an API request")
	}
	if _, err := tm.CheckRefreshToken(access); err == nil {
		t.Fatal("access token renewed a session")
	}
	// Purpose must still be checked even if an operator configured identical keys.
	same := newTokenManager("same", "same", 1, 24)
	a, r, _ := same.CreateTokens(msg)
	if _, err := same.CheckToken(r); err == nil {
		t.Fatal("same-key refresh accepted as access")
	}
	if _, err := same.CheckRefreshToken(a); err == nil {
		t.Fatal("same-key access accepted as refresh")
	}
	for _, test := range []struct {
		name   string
		claims JWTClaims
		method jwt.SigningMethod
		key    string
		code   bizerr.BizCode
	}{
		{"expired", JWTClaims{Purpose: "access", UserID: 1, QueueID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))}}, jwt.SigningMethodHS256, "synthetic-access", bizerr.Auth.TokenExpired},
		{"legacy", JWTClaims{UserID: 1, QueueID: 1}, jwt.SigningMethodHS256, "synthetic-access", bizerr.Auth.TokenInvalid},
		{"no expiry", JWTClaims{Purpose: "access", UserID: 1, QueueID: 1}, jwt.SigningMethodHS256, "synthetic-access", bizerr.Auth.TokenInvalid},
		{"wrong algorithm", JWTClaims{Purpose: "access", UserID: 1, QueueID: 1}, jwt.SigningMethodHS384, "synthetic-access", bizerr.Auth.TokenInvalid},
		{"wrong key", JWTClaims{Purpose: "access", UserID: 1, QueueID: 1}, jwt.SigningMethodHS256, "other", bizerr.Auth.TokenInvalid},
	} {
		t.Run(test.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(test.method, test.claims).SignedString([]byte(test.key))
			if err != nil {
				t.Fatal(err)
			}
			_, err = tm.CheckToken(token)
			if err == nil || bizerrCode(TokenValidationError(err)) != test.code {
				t.Fatalf("classification = %v", err)
			}
		})
	}
}

func bizerrCode(err error) bizerr.BizCode { b, _ := bizerr.FromError(err); return b.Code }
