package domain

import (
	"encoding/base64"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Entry は検索が返した一件である。LDAP の属性名は大文字と小文字を区別しない。
type Entry struct {
	DN         string
	Attributes []Attribute
}

type Attribute struct {
	Name   string
	Values [][]byte
}

func (e Entry) first(name string) ([]byte, bool) {
	for _, attribute := range e.Attributes {
		if strings.EqualFold(attribute.Name, name) && len(attribute.Values) > 0 {
			return attribute.Values[0], true
		}
	}
	return nil, false
}

// Mapping は User の項目の属性名から LDAP の属性名への対応である。
type Mapping map[string]string

// 写せる属性の名前。preferred_username は必須である。
const (
	attrPreferredUsername = "preferred_username"
	attrName              = "name"
	attrGivenName         = "given_name"
	attrFamilyName        = "family_name"
	attrEmail             = "email"
	attrEmployeeNumber    = "employee_number"
	attrDepartment        = "department"
)

func defaultMapping(dialect Dialect) Mapping {
	if dialect == ActiveDirectory {
		return Mapping{
			attrPreferredUsername: "sAMAccountName", attrName: "displayName", attrGivenName: "givenName",
			attrFamilyName: "sn", attrEmail: "mail", attrEmployeeNumber: "employeeNumber", attrDepartment: "department",
		}
	}
	return Mapping{
		attrPreferredUsername: "uid", attrName: "cn", attrGivenName: "givenName",
		attrFamilyName: "sn", attrEmail: "mail", attrEmployeeNumber: "employeeNumber", attrDepartment: "departmentNumber",
	}
}

// EffectiveMapping は方言のデフォルト値に設定の上書きを重ねる。写せない属性名の上書きは拒否する。
func EffectiveMapping(dialect Dialect, overrides map[string]string) (Mapping, error) {
	mapping := defaultMapping(dialect)
	for name, ldapName := range overrides {
		if _, known := mapping[name]; !known {
			return nil, fmt.Errorf("ldap.attributes: unknown attribute %q", name)
		}
		if strings.TrimSpace(ldapName) == "" {
			return nil, fmt.Errorf("ldap.attributes: empty ldap attribute for %q", name)
		}
		mapping[name] = ldapName
	}
	return mapping, nil
}

const (
	adObjectGUID         = "objectGUID"
	adAccountControl     = "userAccountControl"
	adAccountDisabledBit = 0x2
	openLDAPEntryUUID    = "entryUUID"
)

// RequestedAttributes は検索で要求する属性である。対応付けに使うものだけを読む。
func RequestedAttributes(dialect Dialect, mapping Mapping) []string {
	requested := []string{openLDAPEntryUUID}
	if dialect == ActiveDirectory {
		requested = []string{adObjectGUID, adAccountControl}
	}
	return append(requested, slices.Sorted(maps.Values(mapping))...)
}

// IncompleteEntryError は、不変 ID かユーザー名に必要な属性がないエントリである。Connector は
// そのエントリを送らずに同期を続ける。
type IncompleteEntryError struct {
	DN        string
	Attribute string
}

func (e *IncompleteEntryError) Error() string {
	return fmt.Sprintf("ldap entry %q has no usable %s", e.DN, e.Attribute)
}

// MapEntry はエントリを User の項目へ写す (REQ-DIRECTORYCONNECTOR-004)。
func MapEntry(dialect Dialect, mapping Mapping, entry Entry) (Item, error) {
	item := Item{Enabled: true}
	missing := func(attribute string) (Item, error) {
		return Item{}, &IncompleteEntryError{DN: entry.DN, Attribute: attribute}
	}
	switch dialect {
	case ActiveDirectory:
		guid, ok := entry.first(adObjectGUID)
		if !ok || len(guid) != 16 {
			return missing(adObjectGUID)
		}
		item.ExternalID = base64.StdEncoding.EncodeToString(guid)
		control, ok := entry.first(adAccountControl)
		flags, err := strconv.ParseInt(string(control), 10, 64)
		if !ok || err != nil {
			return missing(adAccountControl)
		}
		item.Enabled = flags&adAccountDisabledBit == 0
	default:
		uuid, ok := entry.first(openLDAPEntryUUID)
		if !ok || len(uuid) == 0 {
			return missing(openLDAPEntryUUID)
		}
		item.ExternalID = string(uuid)
	}
	value := func(name string) string {
		raw, _ := entry.first(mapping[name])
		return string(raw)
	}
	if item.PreferredUsername = value(attrPreferredUsername); item.PreferredUsername == "" {
		return missing(mapping[attrPreferredUsername])
	}
	item.Name = value(attrName)
	item.GivenName = value(attrGivenName)
	item.FamilyName = value(attrFamilyName)
	item.Email = value(attrEmail)
	item.EmployeeNumber = value(attrEmployeeNumber)
	item.Department = value(attrDepartment)
	return item, nil
}
