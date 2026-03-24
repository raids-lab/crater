package resputil

import (
	"github.com/raids-lab/crater/internal/bizerr"
)

const (
	// 旧的
	// General

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	InvalidRequest bizerr.BizCode = 40001

	// Token (4010x)

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	TokenExpired bizerr.BizCode = 40101
	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	TokenInvalid bizerr.BizCode = 40102

	// Auth/Login (4011x)

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	InvalidCredentials bizerr.BizCode = 40111
	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	LdapError bizerr.BizCode = 40112
	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	LdapUserNotFound bizerr.BizCode = 40113
	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	LegacyTokenNotSupported bizerr.BizCode = 40114

	// Registration/Uid (4012x)

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	MustRegister    bizerr.BizCode = 40121
	UidServiceError bizerr.BizCode = 40122
	UidNotFound     bizerr.BizCode = 40123

	// User is not allowed to access the resource

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	UserNotAllowed bizerr.BizCode = 40301

	// User's email is not verified

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	UserEmailNotVerified bizerr.BizCode = 40302

	// Container related

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	ServiceSshdNotFound bizerr.BizCode = 40401

	// Conflict

	ResourceStatusError bizerr.BizCode = 40902

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	ServiceError bizerr.BizCode = 50001

	// Indicates laziness of the developer
	// Frontend will directly print the message without any translation

	// Deprecated: 保留旧码以兼容历史代码，但不推荐新代码使用
	NotSpecified bizerr.BizCode = 99999
)
