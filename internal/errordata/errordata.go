package errordata

import (
	"context"
)

type key struct{}

var errorDataKey key

type ErrorData struct {
	Message string
}

func WithErrorData(ctx context.Context) context.Context {
	ed := &ErrorData{ Message: "" }
	return context.WithValue(ctx, errorDataKey, ed)
}

func GetErrorData(ctx context.Context) *ErrorData {
	val := ctx.Value(errorDataKey)
	ed, ok := val.(*ErrorData)
	if !ok {
		return nil
	}
	return ed
}

func (ed *ErrorData) SetMessage(msg string) {
	ed.Message = msg
}

func (ed *ErrorData) HasMessage() bool {
	return ed.Message != ""
}
