package config

type TokenConf struct {
	ContextTimeout         int    `mapstructure:"CONTEXT_TIMEOUT"`
	AccessTokenExpiryHour  int    `mapstructure:"ACCESS_TOKEN_EXPIRY_HOUR"`
	RefreshTokenExpiryHour int    `mapstructure:"REFRESH_TOKEN_EXPIRY_HOUR"`
	AccessTokenSecret      string `mapstructure:"ACCESS_TOKEN_SECRET"`
	RefreshTokenSecret     string `mapstructure:"REFRESH_TOKEN_SECRET"`
}

func NewTokenConf() *TokenConf {
	return &TokenConf{
		ContextTimeout:         2,
		AccessTokenExpiryHour:  1,
		RefreshTokenExpiryHour: 168,
		AccessTokenSecret:      ***REMOVED***,
		RefreshTokenSecret:     ***REMOVED***,
	}
}
