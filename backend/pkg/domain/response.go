package domain

type SignupResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Role         string `json:"role"`
}

/*
	type SignupUsecase interface {
		Create(c context.Context, user *User) error
		GetUserByEmail(c context.Context, email string) (User, error)
		CreateAccessToken(user *User, secret string, expiry int) (accessToken string, err error)
		CreateRefreshToken(user *User, secret string, expiry int) (refreshToken string, err error)
	}
*/

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	Role         string `json:"role"`
}

/*type LoginUsecase interface {
	GetUserByEmail(c context.Context, email string) (User, error)
	CreateAccessToken(user *User, secret string, expiry int) (accessToken string, err error)
	CreateRefreshToken(user *User, secret string, expiry int) (refreshToken string, err error)
}*/
