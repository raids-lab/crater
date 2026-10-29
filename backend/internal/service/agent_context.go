package service

import (
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/raids-lab/crater/internal/bizerr"
)

// The token must remain valid for the four-minute Agent run plus its terminal
// persistence callback, while still being short-lived if intercepted.
const agentContextLifetime = 6 * time.Minute

const (
	AgentInternalTokenEnv             = "CRATER_AGENT_INTERNAL_TOKEN" // #nosec G101 -- environment variable name only.
	AgentContextSigningKeyEnv         = "CRATER_AGENT_CONTEXT_SIGNING_KEY"
	AgentEncryptionKeyEnv             = "CRATER_AGENT_ENCRYPTION_KEY"
	AgentPreviousEncryptionKeysEnv    = "CRATER_AGENT_PREVIOUS_ENCRYPTION_KEYS"
	minimumAgentSecretLength          = 32
	legacyPublicPlatformEncryptionKey = "CRATER_SYSTEM_SECRET_KEY_DEFAULT" // #nosec G101 -- rejected public legacy key.
)

type AgentContextClaims struct {
	UserID        uint          `json:"userId"`
	IdentityScope AgentIdentity `json:"identityScope"`
	jwt.RegisteredClaims
}

func SignAgentContext(userID uint, identity AgentIdentity, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if err := validateAgentSecret(secret, AgentContextSigningKeyEnv); err != nil {
		return "", err
	}
	now := time.Now()
	claims := AgentContextClaims{
		UserID:        userID,
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
	if !token.Valid || claims.UserID == 0 {
		return nil, bizerr.Auth.TokenInvalid.New("invalid or expired Agent context")
	}
	if _, err := ParseAgentIdentity(string(claims.IdentityScope)); err != nil {
		return nil, bizerr.Auth.TokenInvalid.Wrap(err, "invalid Agent identity")
	}
	return claims, nil
}

func validateAgentSecret(secret, name string) error {
	if len(secret) < minimumAgentSecretLength {
		return bizerr.ServiceError.ServiceUnavailable.New(name + " must contain at least 32 characters")
	}
	return nil
}
