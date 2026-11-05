package domain

import (
	"cmp"
	"fmt"
	"slices"
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
// semantics: attributes omitted from the request body default (
// chooses the "reset to default" interpretation RFC 7644 §3.5.1 allows).
type UserWrite struct {
	UserName   string
	GivenName  string
	FamilyName string
	Formatted  string
	Email      string
	Active     bool
	// EmployeeNumber, Department, and ManagerValue are the
	// RFC7643-ENTERPRISE-EXTENSION adoption:partial subset (wi-247).
	// ManagerValue is the manager's SCIM id as given on the wire; usecases
	// resolve it to a tenant-scoped internal User reference.
	EmployeeNumber string
	Department     string
	ManagerValue   string
}

// ProjectCanonicalEmail validates every SCIM emails element and projects the
// multi-valued transport representation onto IdMagic's single canonical email
// (REQ-SOURCING-006). Selection order is primary, work, then wire order.
func ProjectCanonicalEmail(emails []any) (string, error) {
	if len(emails) == 0 {
		return "", nil
	}

	candidates := make([]string, 0, len(emails))
	primaryIndex := -1
	workIndex := -1

	for i, raw := range emails {
		email, ok := raw.(map[string]any)
		if !ok {
			return "", newMutationError("invalidValue", "emails[%d] must be an object", i)
		}

		value, ok := email["value"].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return "", newMutationError("invalidValue", "emails[%d].value must be a non-empty string", i)
		}

		var emailType string
		if rawType, exists := email["type"]; exists {
			emailType, ok = rawType.(string)
			if !ok {
				return "", newMutationError("invalidValue", "emails[%d].type must be a string", i)
			}
		}

		primary := false
		if rawPrimary, exists := email["primary"]; exists {
			primary, ok = rawPrimary.(bool)
			if !ok {
				return "", newMutationError("invalidValue", "emails[%d].primary must be a boolean", i)
			}
		}
		if primary {
			if primaryIndex >= 0 {
				return "", newMutationError("invalidValue", "emails must not contain more than one primary element")
			}
			primaryIndex = i
		}
		if workIndex < 0 && strings.EqualFold(emailType, "work") {
			workIndex = i
		}

		candidates = append(candidates, value)
	}

	if primaryIndex >= 0 {
		return candidates[primaryIndex], nil
	}
	if workIndex >= 0 {
		return candidates[workIndex], nil
	}
	return candidates[0], nil
}

// ParseUserWrite extracts a POST/PUT body into a fully-resolved UserWrite.
// userName is required; its absence is a *MutationError (invalidValue).
func ParseUserWrite(body map[string]any) (UserWrite, error) {
	for _, unsupported := range []string{"phoneNumbers", "addresses"} {
		if _, exists := body[unsupported]; exists {
			return UserWrite{}, newMutationError("invalidValue", "%s is not a supported User attribute", unsupported)
		}
	}

	userName, _ := body["userName"].(string)
	if strings.TrimSpace(userName) == "" {
		return UserWrite{}, newMutationError("invalidValue", "userName is required")
	}

	w := UserWrite{UserName: userName, Active: true}

	if nameMap, ok := body["name"].(map[string]any); ok {
		w.GivenName, _ = nameMap["givenName"].(string)
		w.FamilyName, _ = nameMap["familyName"].(string)
	}
	var err error
	w.Formatted, err = projectCanonicalName(body)
	if err != nil {
		return UserWrite{}, err
	}

	if rawEmails, exists := body["emails"]; exists {
		emails, ok := rawEmails.([]any)
		if !ok {
			return UserWrite{}, newMutationError("invalidValue", "emails must be an array")
		}
		var err error
		w.Email, err = ProjectCanonicalEmail(emails)
		if err != nil {
			return UserWrite{}, err
		}
	}

	if active, exists := body["active"].(bool); exists {
		w.Active = active
	}

	if extRaw, exists := body[EnterpriseUserSchemaURN]; exists {
		ext, ok := extRaw.(map[string]any)
		if !ok {
			return UserWrite{}, newMutationError("invalidValue", "%s must be an object", EnterpriseUserSchemaURN)
		}
		if v, exists := ext["employeeNumber"]; exists {
			s, err := parseEnterpriseStringAttr(v, "employeeNumber")
			if err != nil {
				return UserWrite{}, err
			}
			w.EmployeeNumber = s
		}
		if v, exists := ext["department"]; exists {
			s, err := parseEnterpriseStringAttr(v, "department")
			if err != nil {
				return UserWrite{}, err
			}
			w.Department = s
		}
		if v, exists := ext["manager"]; exists {
			s, err := parseManagerScimID(v)
			if err != nil {
				return UserWrite{}, err
			}
			w.ManagerValue = s
		}
	}

	return w, nil
}

// 既存の formatted の明示値を保ち、null と省略だけを displayName へ委ねる。
// 選ばれない属性の型も検証し、同じ入力を経路ごとに異なる意味で受け付けない。
func projectCanonicalName(body map[string]any) (string, error) {
	display, err := parseNameValue(body["displayName"], "displayName")
	if err != nil {
		return "", err
	}
	name, _ := body["name"].(map[string]any)
	formatted, err := parseNameValue(name["formatted"], "name.formatted")
	if err != nil {
		return "", err
	}
	if name["formatted"] != nil {
		return formatted, nil
	}
	return display, nil
}

func parseNameValue(value any, path string) (string, error) {
	if value == nil {
		return "", nil
	}
	name, ok := value.(string)
	if !ok {
		return "", newMutationError("invalidValue", "%s must be a string or null", path)
	}
	return name, nil
}

// parseEnterpriseStringAttr validates an enterprise extension string
// attribute (employeeNumber, department).
func parseEnterpriseStringAttr(raw any, field string) (string, error) {
	s, ok := raw.(string)
	if !ok {
		return "", newMutationError("invalidValue", "%s must be a string", field)
	}
	return s, nil
}

// parseManagerScimID normalizes the enterprise extension "manager" value to
// the manager's SCIM id. RFC 7643 §4.3 defines manager as a complex
// attribute with a "value" sub-attribute, but some IdPs (e.g. Entra ID) send
// a bare string in PATCH/PUT bodies; both forms are accepted.
func parseManagerScimID(raw any) (string, error) {
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", newMutationError("invalidValue", "manager value must be a non-empty string")
		}
		return v, nil
	case map[string]any:
		value, ok := v["value"].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return "", newMutationError("invalidValue", "manager.value must be a non-empty string")
		}
		return value, nil
	default:
		return "", newMutationError("invalidValue", "manager must be a string or an object with a value field")
	}
}

// UserAttr enumerates the RFC7643-CORE-RESOURCES User attribute paths this
// server accepts in a PATCH operation (RFC7644-PATCH adoption:partial).
type UserAttr string

const (
	UserAttrUserName       UserAttr = "userName"
	UserAttrName           UserAttr = "name"
	UserAttrGivenName      UserAttr = "name.givenName"
	UserAttrFamilyName     UserAttr = "name.familyName"
	UserAttrFormatted      UserAttr = "name.formatted"
	UserAttrEmails         UserAttr = "emails"
	UserAttrActive         UserAttr = "active"
	UserAttrEmployeeNumber UserAttr = "employeeNumber"
	UserAttrDepartment     UserAttr = "department"
	UserAttrManager        UserAttr = "manager"
)

var userPatchAttrs = map[string]UserAttr{
	"username":        UserAttrUserName,
	"name":            UserAttrName,
	"name.givenname":  UserAttrGivenName,
	"name.familyname": UserAttrFamilyName,
	"name.formatted":  UserAttrFormatted,
	"displayname":     UserAttrFormatted,
	"emails":          UserAttrEmails,
	"active":          UserAttrActive,
	// enterprise extension attributes accept both the bare name and the
	// URN-qualified path RFC 7644 §3.10 recommends for extension schemas.
	"employeenumber": UserAttrEmployeeNumber,
	"urn:ietf:params:scim:schemas:extension:enterprise:2.0:user:employeenumber": UserAttrEmployeeNumber,
	"department": UserAttrDepartment,
	"urn:ietf:params:scim:schemas:extension:enterprise:2.0:user:department": UserAttrDepartment,
	"manager": UserAttrManager,
	"urn:ietf:params:scim:schemas:extension:enterprise:2.0:user:manager": UserAttrManager,
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

		targets, err := patchTargets(op, opMap)
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			parsed, err := parseUserPatchOp(op, target.path, target.value)
			if err != nil {
				return nil, err
			}
			ops = append(ops, parsed)
		}
	}
	return ops, nil
}

// parseUserPatchOp validates one operation on one path against the User
// attribute allowlist and projects its value.
func parseUserPatchOp(op, path string, value any) (UserPatchOp, error) {
	lowerPath := strings.ToLower(path)
	if readOnlyResourceAttrs[lowerPath] {
		return UserPatchOp{}, newMutationError("mutability", "attribute %q is readOnly", path)
	}
	attr, ok := userPatchAttrs[lowerPath]
	if !ok {
		return UserPatchOp{}, newMutationError("invalidPath", "attribute %q is not a supported PATCH path", path)
	}
	if op == "remove" {
		return UserPatchOp{Op: op, Attr: attr, Value: value}, nil
	}

	switch attr {
	case UserAttrFormatted:
		parsed, err := parseNameValue(value, path)
		if err != nil {
			return UserPatchOp{}, err
		}
		value = parsed
	case UserAttrName:
		name, isObject := value.(map[string]any)
		if !isObject {
			return UserPatchOp{}, newMutationError("invalidValue", "name value must be an object")
		}
		if _, err := parseNameValue(name["formatted"], "name.formatted"); err != nil {
			return UserPatchOp{}, err
		}
	case UserAttrActive:
		if _, isBool := value.(bool); !isBool {
			return UserPatchOp{}, newMutationError("invalidValue", "active value must be a boolean")
		}
	case UserAttrEmails:
		emails, isArray := value.([]any)
		if !isArray || len(emails) == 0 {
			return UserPatchOp{}, newMutationError("invalidValue", "emails value must be a non-empty array")
		}
		projected, err := ProjectCanonicalEmail(emails)
		if err != nil {
			return UserPatchOp{}, err
		}
		value = projected
	case UserAttrEmployeeNumber, UserAttrDepartment:
		if _, isString := value.(string); !isString {
			return UserPatchOp{}, newMutationError("invalidValue", "%s value must be a string", attr)
		}
	case UserAttrManager:
		projected, err := parseManagerScimID(value)
		if err != nil {
			return UserPatchOp{}, err
		}
		value = projected
	}
	return UserPatchOp{Op: op, Attr: attr, Value: value}, nil
}

// GroupWrite is a fully-resolved Group attribute set after applying
// POST/PUT full-specification semantics.
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
	if members, exists := body["members"]; exists {
		var err error
		w.MemberScimIDs, err = ParseGroupMemberScimIDs(members)
		if err != nil {
			return GroupWrite{}, err
		}
	}
	return w, nil
}

// ParseGroupMemberScimIDs validates the User-only Group membership subset
// before any mutation is applied (REQ-SOURCING-005).
func ParseGroupMemberScimIDs(value any) ([]string, error) {
	members, ok := value.([]any)
	if !ok {
		return nil, newMutationError("invalidValue", "members value must be an array")
	}

	scimIDs := make([]string, 0, len(members))
	for i, raw := range members {
		member, ok := raw.(map[string]any)
		if !ok {
			return nil, newMutationError("invalidValue", "members[%d] must be an object", i)
		}
		if rawType, exists := member["type"]; exists {
			memberType, isString := rawType.(string)
			if !isString || !strings.EqualFold(memberType, "User") {
				return nil, newMutationError("invalidValue", "members[%d].type must be User", i)
			}
		}
		scimID, ok := member["value"].(string)
		if !ok || strings.TrimSpace(scimID) == "" {
			return nil, newMutationError("invalidValue", "members[%d].value must be a non-empty string", i)
		}
		scimIDs = append(scimIDs, scimID)
	}
	return scimIDs, nil
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

		targets, err := patchTargets(op, opMap)
		if err != nil {
			return nil, err
		}
		for _, target := range targets {
			parsed, err := parseGroupPatchOp(op, target.path, target.value)
			if err != nil {
				return nil, err
			}
			ops = append(ops, parsed)
		}
	}
	return ops, nil
}

// patchTarget is one path and the value one PATCH operation applies to it.
type patchTarget struct {
	path  string
	value any
}

// patchTargets returns the paths one operation applies to. An operation
// without a path applies each member of its value object as if the member
// name were the path (RFC 7644 §3.5.2.1, §3.5.2.3); the members of the
// Enterprise extension object become URN-qualified paths. A remove needs a
// path (RFC 7644 §3.5.2.2). Only an absent or null path counts as omitted.
func patchTargets(op string, opMap map[string]any) ([]patchTarget, error) {
	if rawPath := opMap["path"]; rawPath != nil {
		path, _ := rawPath.(string)
		return []patchTarget{{path: path, value: opMap["value"]}}, nil
	}
	if op == "remove" {
		return nil, newMutationError("noTarget", "remove requires a path")
	}
	value, isObject := opMap["value"].(map[string]any)
	if !isObject {
		return nil, newMutationError("invalidValue", "value of an operation without a path must be an object")
	}

	targets := make([]patchTarget, 0, len(value))
	for name, member := range value {
		if !strings.EqualFold(name, EnterpriseUserSchemaURN) {
			targets = append(targets, patchTarget{path: name, value: member})
			continue
		}
		ext, isObject := member.(map[string]any)
		if !isObject {
			return nil, newMutationError("invalidValue", "%s must be an object", name)
		}
		for subName, subValue := range ext {
			targets = append(targets, patchTarget{path: name + ":" + subName, value: subValue})
		}
	}
	if len(targets) == 0 {
		return nil, newMutationError("invalidValue", "value of an operation without a path must name at least one attribute")
	}
	sortPathlessTargets(targets, value)
	return targets, nil
}

// sortPathlessTargets fixes the order the members of one value object apply
// in, since a decoded JSON object does not keep the wire order. displayName
// and name.formatted write the same canonical name, so displayName goes
// first when the value assigns a non-null formatted and last otherwise,
// which matches the precedence POST and PUT use (projectCanonicalName).
func sortPathlessTargets(targets []patchTarget, value map[string]any) {
	formattedAssigned, namesCompete := false, false
	for name, member := range value {
		switch strings.ToLower(name) {
		case "name":
			namesCompete = true
			nameObject, _ := member.(map[string]any)
			formattedAssigned = formattedAssigned || nameObject["formatted"] != nil
		case "name.formatted":
			namesCompete = true
			formattedAssigned = formattedAssigned || member != nil
		}
	}
	rank := func(path string) int {
		if !namesCompete || !strings.EqualFold(path, "displayName") {
			return 1
		}
		if formattedAssigned {
			return 0
		}
		return 2
	}
	slices.SortFunc(targets, func(a, b patchTarget) int {
		return cmp.Or(cmp.Compare(rank(a.path), rank(b.path)), cmp.Compare(strings.ToLower(a.path), strings.ToLower(b.path)))
	})
}

// parseGroupPatchOp validates one operation on one path against the Group
// attribute allowlist.
func parseGroupPatchOp(op, path string, value any) (GroupPatchOp, error) {
	lowerPath := strings.ToLower(path)
	if readOnlyResourceAttrs[lowerPath] {
		return GroupPatchOp{}, newMutationError("mutability", "attribute %q is readOnly", path)
	}
	attr, ok := groupPatchAttrs[lowerPath]
	if !ok {
		return GroupPatchOp{}, newMutationError("invalidPath", "attribute %q is not a supported PATCH path", path)
	}
	if attr == GroupAttrMembers {
		if _, err := ParseGroupMemberScimIDs(value); err != nil {
			return GroupPatchOp{}, err
		}
	}
	return GroupPatchOp{Op: op, Attr: attr, Value: value}, nil
}
