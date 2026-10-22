package ports

import "fmt"

// UniqueAttributeConflictError は、保存しようとした User の unique の属性の値を、同じテナントの
// 削除されていないほかの User がすでに持っていることを表す。保存は User を変えない。
type UniqueAttributeConflictError struct{ Key string }

func (e *UniqueAttributeConflictError) Error() string {
	return fmt.Sprintf("attribute %q: the value is used by another user", e.Key)
}

// AttributeValuesNotUniqueError は、unique に変えようとした属性の値を、同じテナントの削除されて
// いない二人以上の User が持っていることを表す。スキーマの保存はスキーマを変えない。
type AttributeValuesNotUniqueError struct{ Key string }

func (e *AttributeValuesNotUniqueError) Error() string {
	return fmt.Sprintf("attribute %q: users share a value, so it cannot be unique", e.Key)
}
