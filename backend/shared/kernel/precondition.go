package kernel

import "errors"

// ErrPreconditionFailed は、条件付きの変更が現在の表現と一致せず、実行されなかったことを表す。
var ErrPreconditionFailed = errors.New("resource precondition does not match")
