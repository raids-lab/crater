package api

type AdminUser struct {
	ID           uint           `json:"id,omitempty"`
	Name         string         `json:"name"`
	Nickname     string         `json:"nickname,omitempty"`
	Role         uint8          `json:"role,omitempty"`
	Status       uint8          `json:"status,omitempty"`
	ExtraBalance *float64       `json:"extraBalance,omitempty"`
	Space        string         `json:"space,omitempty"`
	Attributes   *UserAttribute `json:"attributes,omitempty"`
}

func (c *Client) ListAdminUsers(base bool) ([]AdminUser, error) {
	if !base {
		return fetchAllListPages[AdminUser](c, AdminUsersPrefix, "-id")
	}
	var result Response[[]AdminUser]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(AdminUsersPrefix + "/baseinfo")
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}
