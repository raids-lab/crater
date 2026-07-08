package bizerr

import (
	"fmt"

	"github.com/cockroachdb/errors"
)

type BizCode int

const codeGroupDivisor = 100

// Group 获取该错误码所属的分组（前三位）
// 例如 40901 -> 409
func (c BizCode) Group() int {
	return int(c) / codeGroupDivisor
}

type BizError struct {
	Code    BizCode
	Message string // 自定义消息
	cause   error
}

func (e *BizError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *BizError) Unwrap() error {
	return e.cause
}

func (e *BizError) Cause() error {
	return e.cause
}

// Is 关键实现：支持 错误码、错误对象、以及“分组判断”
func (e *BizError) Is(target error) bool {
	// 1. 如果 target 是 *BizError 类型
	if t, ok := target.(*BizError); ok {
		// 如果 target 是一个组（后两位是 00），则判断 Group 是否一致
		if t.Code%codeGroupDivisor == 0 {
			return e.Code.Group() == t.Code.Group()
		}
		// 否则，精确匹配错误码
		return e.Code == t.Code
	}

	// 2. 如果直接比较 BizCode (可选增强)
	if t, ok := target.(interface{ GetCode() BizCode }); ok {
		return e.Code == t.GetCode()
	}

	return false
}

// --- 行为方法 ---

// New 创建业务错误
func (c BizCode) New(msg string) error {
	be := &BizError{Code: c, Message: msg}
	// errors.WithStack 会捕获当前位置的堆栈
	return errors.WithStack(be)
}

// Wrap 包装底层错误
func (c BizCode) Wrap(cause error, msg string) error {
	if cause == nil {
		return nil
	}
	be := &BizError{Code: c, Message: msg, cause: cause}
	return errors.WithStack(be)
}
