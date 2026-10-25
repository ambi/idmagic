package domain

import (
	"fmt"
	"reflect"
)

// AttributeWriteAuthority は、過去の値の書き込み元ではなく、上流に許す書き込みの範囲である。
type AttributeWriteAuthority string

const (
	AttributeAuthorityUpstream AttributeWriteAuthority = "upstream"
	AttributeAuthorityLocal    AttributeWriteAuthority = "local"
)

func (a AttributeWriteAuthority) Effective() AttributeWriteAuthority {
	if a == "" {
		return AttributeAuthorityUpstream
	}
	return a
}

// UpstreamWritableAttributes は、参照キーのうち上流から変更され得るものを返す。
// User と値の有無に依存せず、未知のキーや分類もローカルへ格上げしない。
func UpstreamWritableAttributes(keys []string, defs []UserAttributeDef) []string {
	local := map[string]bool{}
	for _, def := range defs {
		if def.WriteAuthority == AttributeAuthorityLocal && !coreUserAttribute(def.Key) {
			local[def.Key] = true
		}
	}
	var upstream []string
	for _, key := range keys {
		if !local[key] {
			upstream = append(upstream, key)
		}
	}
	return upstream
}

// ValidateUpstreamAttributeChanges は、上流がローカル専用の値を変えることを拒否する。
// 投影済みの User はローカルの値も保持するため、同値の保持を禁止しない。
func ValidateUpstreamAttributeChanges(before, after map[string]AttributeValue, defs []UserAttributeDef) error {
	for _, def := range defs {
		if def.WriteAuthority != AttributeAuthorityLocal {
			continue
		}
		previous, had := before[def.Key]
		next, has := after[def.Key]
		if had != has || !reflect.DeepEqual(previous, next) {
			return fmt.Errorf("attribute %q is local-only", def.Key)
		}
	}
	return nil
}

func coreUserAttribute(key string) bool {
	switch key {
	case "id", "preferred_username", "name", "given_name", "family_name", "email", "email_verified":
		return true
	default:
		return false
	}
}
