package domain

type SignupRequest struct {
	Name     string `form:"name" binding:"required"`
	Email    string `form:"email" binding:"required,email"`
	Password string `form:"password" binding:"required"`
}

type SignupResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

/*
	type SignupUsecase interface {
		Create(c context.Context, user *User) error
		GetUserByEmail(c context.Context, email string) (User, error)
		CreateAccessToken(user *User, secret string, expiry int) (accessToken string, err error)
		CreateRefreshToken(user *User, secret string, expiry int) (refreshToken string, err error)
	}
*/
type LoginRequest struct {
	Email    string `form:"email" binding:"required,email"`
	Password string `form:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

/*type LoginUsecase interface {
	GetUserByEmail(c context.Context, email string) (User, error)
	CreateAccessToken(user *User, secret string, expiry int) (accessToken string, err error)
	CreateRefreshToken(user *User, secret string, expiry int) (refreshToken string, err error)
}*/