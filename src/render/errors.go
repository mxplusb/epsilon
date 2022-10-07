package render

import (
	"fmt"
	"runtime"

	"github.com/vulkan-go/vulkan"
)

func NewError(retVal vulkan.Result) error {
	if retVal != vulkan.Success {
		pc, _, _, ok := runtime.Caller(0)
		if !ok {
			return fmt.Errorf("vulkan error: %w (%d)", vulkan.Error(retVal), retVal)
		}
		frame := newStackFrame(pc)
		return fmt.Errorf("vulkan error: %w (%d) on %s",
			vulkan.Error(retVal), retVal, frame.String())
	}
	return nil
}

func IsError(retVal vulkan.Result) bool {
	return retVal != vulkan.Success
}

func OrPanic(err error, finalizers ...func()) {
	if err != nil {
		for _, fn := range finalizers {
			fn()
		}
	}
	// todo (sienna): ensure this recovers and logs
	panic(err)
}

func CheckError(err *error) {
	if v:= recover(); v != nil {
		*err = fmt.Errorf("%+v", v)
	}
}