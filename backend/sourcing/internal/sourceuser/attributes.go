// Package sourceuser は、取り込み元が解析した User の属性を、正規の User へ写す。
// SCIM と Directory Connector が同じ写し方を使い、同期する属性を一か所で増やせるようにする。
package sourceuser

import (
	"strings"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// User.Attributes のうち、取り込み元が管理するキー。
const (
	AttrEmployeeNumber = "employee_number"
	AttrDepartment     = "department"
	AttrManagerSub     = "manager_sub"
)

// Attributes は、取り込み元が管理する User の属性である。空の文字列は値がないことを表す。
// ManagerSub が nil なら、その取り込み元は manager を管理せず、既存の値を変えない。
type Attributes struct {
	PreferredUsername string
	Name              string
	GivenName         string
	FamilyName        string
	Email             string
	EmployeeNumber    string
	Department        string
	ManagerSub        *string
}

// Apply は user の管理する属性を a で置き換える。値のない属性は空にするか消す。
func (a Attributes) Apply(user *userdomain.User) {
	user.PreferredUsername = a.PreferredUsername
	user.Name, user.GivenName, user.FamilyName = new(a.Name), new(a.GivenName), new(a.FamilyName)
	user.Email = new(a.Email)
	SetString(user, AttrEmployeeNumber, a.EmployeeNumber)
	SetString(user, AttrDepartment, a.Department)
	if a.ManagerSub != nil {
		SetString(user, AttrManagerSub, *a.ManagerSub)
	}
}

// Matches は user の管理する属性が a と同じかを返す。空の文字列と値のないことを区別しない。
func (a Attributes) Matches(user userdomain.User) bool {
	same := user.PreferredUsername == a.PreferredUsername &&
		deref(user.Name) == a.Name && deref(user.GivenName) == a.GivenName && deref(user.FamilyName) == a.FamilyName &&
		deref(user.Email) == a.Email &&
		stringOrEmpty(user.Attributes, AttrEmployeeNumber) == a.EmployeeNumber &&
		stringOrEmpty(user.Attributes, AttrDepartment) == a.Department
	if a.ManagerSub != nil {
		same = same && stringOrEmpty(user.Attributes, AttrManagerSub) == *a.ManagerSub
	}
	return same
}

// SetString は user.Attributes[key] に文字列を保存し、空白だけの値ならキーを消す。
// 置き換えや削除で古い値を残さないためである。
func SetString(user *userdomain.User, key, value string) {
	if user.Attributes == nil {
		user.Attributes = make(map[string]userdomain.AttributeValue)
	}
	if strings.TrimSpace(value) == "" {
		delete(user.Attributes, key)
		return
	}
	user.Attributes[key] = userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: new(value)}
}

// String は文字列型の User.Attributes の値を読む。
func String(attrs map[string]userdomain.AttributeValue, key string) (string, bool) {
	v, ok := attrs[key]
	if !ok || v.Type != idmdomain.AttributeTypeString || v.String == nil {
		return "", false
	}
	return *v.String, true
}

func stringOrEmpty(attrs map[string]userdomain.AttributeValue, key string) string {
	value, _ := String(attrs, key)
	return value
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
