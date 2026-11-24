package api

import "encoding/json"

type AdminUser struct {
	raw          json.RawMessage
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
	path := AdminUsersPrefix
	if base {
		path += "/baseinfo"
	}
	var result Response[[]AdminUser]
	resp, err := c.httpClient.R().
		SetSuccessResult(&result).
		SetErrorResult(&result).
		Get(path)
	if err != nil {
		return nil, &NetworkError{Cause: err}
	}
	if err := errorFromResponse(resp, result.Code, result.Message); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// Preserve new server fields through filtering and pagination in JSON mode.
func (u *AdminUser) UnmarshalJSON(data []byte) error {
	type alias AdminUser
	var value alias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*u = AdminUser(value)
	u.raw = append(json.RawMessage(nil), data...)
	return nil
}
func (u AdminUser) MarshalJSON() ([]byte, error) {
	if len(u.raw) > 0 {
		return u.raw, nil
	}
	type alias AdminUser
	return json.Marshal(alias(u))
}
func (u AdminUser) Field(key string) any {
	var data map[string]any
	raw, _ := u.MarshalJSON()
	_ = json.Unmarshal(raw, &data)
	if value, ok := data[key]; ok {
		return value
	}
	if attributes, ok := data["attributes"].(map[string]any); ok {
		return attributes[key]
	}
	return nil
}
