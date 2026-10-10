package service

import (
	"strings"

	"github.com/raids-lab/crater/internal/bizerr"
)

type AgentIdentity string

const (
	AgentIdentityUser  AgentIdentity = "user"
	AgentIdentityAdmin AgentIdentity = "admin"
)

func ParseAgentIdentity(value string) (AgentIdentity, error) {
	identity := AgentIdentity(strings.TrimSpace(value))
	if identity != AgentIdentityUser && identity != AgentIdentityAdmin {
		return "", bizerr.BadRequest.ParameterError.New("Agent identity is invalid")
	}
	return identity, nil
}
