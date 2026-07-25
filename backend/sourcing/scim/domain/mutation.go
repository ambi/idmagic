package domain

import (
	"fmt"
	"strings"
)

// MutationError signals that a SCIM create/replace/patch request violates
// the resource contract this server supports (missing required attribute,
// unsupported PATCH path, invalid operation/value, or a write to a readOnly
// attribute). Callers map ScimType directly to the SCIM protocol error
// "scimType" (RFC 7644 §3.12 Table 9).
type MutationError struct {
	ScimType string
	msg      string
}

func (e *MutationError) Error() string { return e.msg }

// NewMutationError builds a MutationError carrying the given RFC 7644
// §3.12 scimType. Exported for usecases-layer validation (e.g. member
// resolvability) that can't be expressed as a pure parse-time rule.
func NewMutationError(scimType, format string, args ...any) *MutationError {
	return &MutationError{ScimType: scimType, msg: fmt.Sprintf(format, args...)}
}

func newMutationError(scimType, format string, args ...any) *MutationError {
	return NewMutationError(scimType, format, args...)
}

// UserWrite is a fully-resolved User attribute set (RFC7643-CORE-RESOURCES
// adoption:partial subset) after applying POST/PUT full-specification
// semantics: attributes omitted from the request body default (ADR-122
// chooses the "reset to default" interpretation RFC 7644 §3.5.1 allows).
type UserWrite struct {
	UserName   string
	GivenName  string
	FamilyName string
	Formatted  string
	Email      string
	Active     bool
}

// ParseUserWrite extracts a POST/PUT body into a fully-resolved UserWrite.
// userName is required; its absence is a *MutationError (invalidValue).
func ParseUserWrite(body map[string]any) (UserWrite, error) {
	userName, _ := body["userName"].(string)
	if strings.TrimSpace(userName) == "" {
		return UserWrite{}, newMutationError("invalidValue", "userName is required")
	}

	w := UserWrite{UserName: userName, Active: true}

	if nameMap, ok := body["name"].(map[string]any); ok {
		w.GivenName, _ = nameMap["givenName"].(string)
		w.FamilyName, _ = nameMap["familyName"].(string)
		w.Formatted, _ = nameMap["formatted"].(string)
	}

	if emails, ok := body["emails"].([]any); ok && len(emails) > 0 {
		if firstEmail, ok := emails[0].(map[string]any); ok {
			w.Email, _ = firstEmail["value"].(string)
		}
	}

	if active, exists := body["active"].(bool); exists {
		w.Active = active
	}

	return w, nil
}

// UserAttr enumerates the RFC7643-CORE-RESOURCES User attribute paths this
// server accepts in a PATCH operation (RFC7644-PATCH adoption:partial).
type UserAttr string

const (
	UserAttrUserName   UserAttr = "userName"
	UserAttrName       UserAttr = "name"
	UserAttrGivenName  UserAttr = "name.givenName"
	UserAttrFamilyName UserAttr = "name.familyName"
	UserAttrFormatted  UserAttr = "name.formatted"
	UserAttrEmails     UserAttr = "emails"
	UserAttrActive     UserAttr = "active"
)

var userPatchAttrs = map[string]UserAttr{
	"username":        UserAttrUserName,
	"name":            UserAttrName,
	"name.givenname":  UserAttrGivenName,
	"name.familyname": UserAttrFamilyName,
	"name.formatted":  UserAttrFormatted,
	"emails":          UserAttrEmails,
	"active":          UserAttrActive,
}

var readOnlyResourceAttrs = map[string]bool{"id": true, "meta": true, "schemas": true}

// UserPatchOp is one validated PATCH operation targeting a supported User
// attribute (RFC 7644 §3.5.2).
type UserPatchOp struct {
	Op    string // add | replace | remove
	Attr  UserAttr
	Value any
}

// ParseUserPatchOps validates a PATCH body's Operations against the User
// attribute allowlist. Unknown paths are invalidPath, readOnly attribute
// paths (id/meta/schemas) are mutability, and unknown ops or
// type-incompatible values are invalidValue (RFC 7644 §3.12 Table 9).
func ParseUserPatchOps(body map[string]any) ([]UserPatchOp, error) {
	rawOps, _ := body["Operations"].([]any)
	if len(rawOps) == 0 {
		return nil, newMutationError("invalidValue", "Operations must be a non-empty array")
	}

	ops := make([]UserPatchOp, 0, len(rawOps))
	for _, rawOp := range rawOps {
		opMap, ok := rawOp.(map[string]any)
		if !ok {
			return nil, newMutationError("invalidValue", "each Operation must be an object")
		}
		op, _ := opMap["op"].(string)
		op = strings.ToLower(op)
		if op != "add" && op != "replace" && op != "remove" {
			return nil, newMutationError("invalidValue", "unsupported PATCH op %q", opMap["op"])
		}

		path, _ := opMap["path"].(string)
		lowerPath := strings.ToLower(path)
		if readOnlyResourceAttrs[lowerPath] {
			return nil, newMutationError("mutability", "attribute %q is readOnly", path)
		}
		attr, ok := userPatchAttrs[lowerPath]
		if !ok {
			return nil, newMutationError("invalidPath", "attribute %q is not a supported PATCH path", path)
		}

		value := opMap["value"]
		if attr == UserAttrActive && op != "remove" {
			if _, isBool := value.(bool); !isBool {
				return nil, newMutationError("invalidValue", "active value must be a boolean")
			}
		}

		ops = append(ops, UserPatchOp{Op: op, Attr: attr, Value: value})
	}
	return ops, nil
}

// GroupWrite is a fully-resolved Group attribute set after applying
// POST/PUT full-specification semantics (ADR-122).
type GroupWrite struct {
	DisplayName   string
	MemberScimIDs []string
}

// ParseGroupWrite extracts a POST/PUT body into a fully-resolved
// GroupWrite. displayName is required; its absence is a *MutationError
// (invalidValue). Omitted members default to an empty set (PUT replaces
// the full membership).
func ParseGroupWrite(body map[string]any) (GroupWrite, error) {
	displayName, _ := body["displayName"].(string)
	if strings.TrimSpace(displayName) == "" {
		return GroupWrite{}, newMutationError("invalidValue", "displayName is required")
	}

	w := GroupWrite{DisplayName: displayName, MemberScimIDs: []string{}}
	if members, ok := body["members"].([]any); ok {
		for _, mVal := range members {
			mMap, ok := mVal.(map[string]any)
			if !ok {
				continue
			}
			scimID, _ := mMap["value"].(string)
			if scimID != "" {
				w.MemberScimIDs = append(w.MemberScimIDs, scimID)
			}
		}
	}
	return w, nil
}

// GroupAttr enumerates the Group attribute paths this server accepts in a
// PATCH operation (RFC7644-PATCH adoption:partial).
type GroupAttr string

const (
	GroupAttrDisplayName GroupAttr = "displayName"
	GroupAttrMembers     GroupAttr = "members"
)

var groupPatchAttrs = map[string]GroupAttr{
	"displayname": GroupAttrDisplayName,
	"members":     GroupAttrMembers,
}

// GroupPatchOp is one validated PATCH operation targeting a supported Group
// attribute.
type GroupPatchOp struct {
	Op    string
	Attr  GroupAttr
	Value any
}

// ParseGroupPatchOps validates a PATCH body's Operations against the Group
// attribute allowlist, mirroring ParseUserPatchOps.
func ParseGroupPatchOps(body map[string]any) ([]GroupPatchOp, error) {
	rawOps, _ := body["Operations"].([]any)
	if len(rawOps) == 0 {
		return nil, newMutationError("invalidValue", "Operations must be a non-empty array")
	}

	ops := make([]GroupPatchOp, 0, len(rawOps))
	for _, rawOp := range rawOps {
		opMap, ok := rawOp.(map[string]any)
		if !ok {
			return nil, newMutationError("invalidValue", "each Operation must be an object")
		}
		op, _ := opMap["op"].(string)
		op = strings.ToLower(op)
		if op != "add" && op != "replace" && op != "remove" {
			return nil, newMutationError("invalidValue", "unsupported PATCH op %q", opMap["op"])
		}

		path, _ := opMap["path"].(string)
		lowerPath := strings.ToLower(path)
		if readOnlyResourceAttrs[lowerPath] {
			return nil, newMutationError("mutability", "attribute %q is readOnly", path)
		}
		attr, ok := groupPatchAttrs[lowerPath]
		if !ok {
			return nil, newMutationError("invalidPath", "attribute %q is not a supported PATCH path", path)
		}

		ops = append(ops, GroupPatchOp{Op: op, Attr: attr, Value: opMap["value"]})
	}
	return ops, nil
}
