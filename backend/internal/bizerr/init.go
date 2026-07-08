package bizerr

import (
	"errors"
	"reflect"
	"strconv"
)

const (
	badRequestBaseCode       = 40000
	authBaseCode             = 40100
	forbiddenBaseCode        = 40300
	notFoundBaseCode         = 40400
	methodNotAllowedBaseCode = 40500
	conflictBaseCode         = 40900
	payloadTooLargeBaseCode  = 41300
	rateLimitBaseCode        = 42900
	internalBaseCode         = 50000
	badGatewayBaseCode       = 50200
	serviceError             = 50300
	gatewayTimeoutBaseCode   = 50400
)

var (
	BadRequest       = &badRequestGroup{}
	Auth             = &authGroup{}
	Forbidden        = &forbiddenGroup{}
	NotFound         = &notFoundGroup{}
	Conflict         = &conflictGroup{}
	Internal         = &internalGroup{}
	RateLimit        = &rateLimitGroup{}
	MethodNotAllowed = &methodNotAllowedGroup{}
	PayloadTooLarge  = &payloadTooLargeGroup{}
	BadGateway       = &badGatewayGroup{}
	ServiceError     = &serviceErrorGroup{}
	GatewayTimeout   = &gatewayTimeoutGroup{}
)

// initErrors initializes all BizError groups.
//
// Design notes:
//   - Error definitions are declarative via struct tags
//   - reflect is used intentionally to minimize boilerplate
//   - Adding a new error requires only adding a struct field with `code` tag
//
// This trades a small amount of explicitness for:
//   - code-as-documentation
//   - linear-cost extensibility
//
//nolint:gochecknoinits // init functions are acceptable for package-level initialization
func init() {
	initGroup(BadRequest, badRequestBaseCode, "Bad Request")
	initGroup(Auth, authBaseCode, "Unauthorized")
	initGroup(Forbidden, forbiddenBaseCode, "Forbidden")
	initGroup(NotFound, notFoundBaseCode, "Not Found")
	initGroup(MethodNotAllowed, methodNotAllowedBaseCode, "Method Not Allowed")
	initGroup(PayloadTooLarge, payloadTooLargeBaseCode, "Payload Too Large")
	initGroup(Conflict, conflictBaseCode, "Conflict")
	initGroup(RateLimit, rateLimitBaseCode, "Rate Limit")
	initGroup(Internal, internalBaseCode, "Internal Server Error")
	initGroup(BadGateway, badGatewayBaseCode, "Bad Gateway")
	initGroup(ServiceError, serviceError, "Service Unavailable")
	initGroup(GatewayTimeout, gatewayTimeoutBaseCode, "Gateway Timeout")
}

func initGroup(g any, baseCode int, groupName string) {
	v := reflect.ValueOf(g)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		panic("bizerr: initGroup expects pointer to struct")
	}

	v = v.Elem()
	t := v.Type()

	// ----------------------------------------------------------------
	// 1. 初始化嵌入的 *BizError（分组基准）
	// ----------------------------------------------------------------
	baseErrField := v.FieldByName("Base") // 改为寻找 "Base" 字段
	if !baseErrField.IsValid() || baseErrField.Type() != reflect.TypeOf(&BizError{}) {
		panic("bizerr: group must have 'Base *BizError' field")
	}

	baseErr := &BizError{
		Code:    BizCode(baseCode),
		Message: groupName,
	}
	baseErrField.Set(reflect.ValueOf(baseErr))

	// ----------------------------------------------------------------
	// 2. 初始化各 BizCode 字段
	// ----------------------------------------------------------------
	bizCodeType := reflect.TypeOf(BizCode(0))

	for i := 0; i < v.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := v.Field(i)

		codeTag := fieldType.Tag.Get("code")
		if codeTag == "" {
			continue
		}

		// --- 类型校验（非常重要） ---
		if fieldType.Type != bizCodeType {
			panic("bizerr: `code` tag can only be used on BizCode fields")
		}

		code, err := strconv.Atoi(codeTag)
		if err != nil {
			panic("bizerr: invalid code tag: " + codeTag)
		}

		fieldValue.SetInt(int64(code))
	}
}

// FromError 辅助工具：从任意 error 中提取 *BizError
func FromError(err error) (*BizError, bool) {
	var bErr *BizError
	if errors.As(err, &bErr) {
		return bErr, true
	}
	return nil, false
}
