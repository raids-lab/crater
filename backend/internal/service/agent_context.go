package service

import (
	"os"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/pkg/crypto"
)

// The token must remain valid for the four-minute Agent run plus its terminal
// persistence callback, while still being short-lived if intercepted.
const agentContextLifetime = 6 * time.Minute

const (
	AgentInternalTokenEnv     = "CRATER_AGENT_INTERNAL_TOKEN" // #nosec G101 -- environment variable name only.
	AgentContextSigningKeyEnv = "CRATER_AGENT_CONTEXT_SIGNING_KEY"
	minimumAgentSecretLength  = 32
)

type AgentContextClaims struct {
	UserID        uint          `json:"userId"`
	AccountID     uint          `json:"accountId"`
	IdentityScope AgentIdentity `json:"identityScope"`
	jwt.RegisteredClaims
}

func SignAgentContext(userID, accountID uint, identity AgentIdentity, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if err := validateAgentSecret(secret, AgentContextSigningKeyEnv); err != nil {
		return "", err
	}
	now := time.Now()
	claims := AgentContextClaims{
		UserID:        userID,
		AccountID:     accountID,
		IdentityScope: identity,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "crater-backend",
			Audience:  jwt.ClaimStrings{"crater-agent"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(agentContextLifetime)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", bizerr.Internal.ServiceError.Wrap(err, "failed to sign Agent context")
	}
	return signed, nil
}

func ParseAgentContext(tokenString, secret string) (*AgentContextClaims, error) {
	secret = strings.TrimSpace(secret)
	if err := validateAgentSecret(secret, AgentContextSigningKeyEnv); err != nil {
		return nil, err
	}
	claims := &AgentContextClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, bizerr.Auth.TokenInvalid.New("invalid Agent context signing method")
			}
			return []byte(secret), nil
		},
		jwt.WithAudience("crater-agent"),
		jwt.WithIssuer("crater-backend"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, bizerr.Auth.TokenInvalid.Wrap(err, "invalid or expired Agent context")
	}
	if !token.Valid || claims.UserID == 0 || claims.AccountID == 0 {
		return nil, bizerr.Auth.TokenInvalid.New("invalid or expired Agent context")
	}
	if _, err := ParseAgentIdentity(string(claims.IdentityScope)); err != nil {
		return nil, bizerr.Auth.TokenInvalid.Wrap(err, "invalid Agent identity")
	}
	return claims, nil
}

// ValidateAgentRuntimeSecurity rejects Agent startup with missing or weak
// service authentication, context signing, or credential encryption keys.
func ValidateAgentRuntimeSecurity() error {
	if err := validateAgentSecret(strings.TrimSpace(os.Getenv(AgentInternalTokenEnv)), AgentInternalTokenEnv); err != nil {
		return err
	}
	if err := validateAgentSecret(
		strings.TrimSpace(os.Getenv(AgentContextSigningKeyEnv)), AgentContextSigningKeyEnv,
	); err != nil {
		return err
	}
	if err := crypto.ValidateKeyConfiguration(); err != nil {
		return bizerr.ServiceError.ServiceUnavailable.Wrap(err, "Agent credential encryption key is invalid")
	}
	return nil
}

func validateAgentSecret(secret, name string) error {
	if len(secret) < minimumAgentSecretLength {
		return bizerr.ServiceError.ServiceUnavailable.New(name + " must contain at least 32 characters")
	}
	return nil
}
