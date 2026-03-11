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
}

func (e *BizError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
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
	be := &BizError{Code: c, Message: msg}
	// 1. errors.Mark 将 be 标记为 cause 的一部分，使得 errors.Is(err, be) 成立
	// 2. errors.WithStack 加上堆栈
	// 3. errors.WithMessage 加上新的描述
	err := errors.Mark(cause, be)
	err = errors.WithMessage(err, be.Error())
	return errors.WithStack(err)
}
