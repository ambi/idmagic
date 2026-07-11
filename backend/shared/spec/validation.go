package spec

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	z "github.com/Oudwins/zog"
)

var tenantIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)

// attrKeyPattern は ADR-040 の属性キー命名規則: snake_case、英字始まり。
var attrKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

var tenantSchema = z.Struct(z.Shape{
	"ID": z.String().Min(1).Required(),
	"Realm": z.String().Min(1).Max(63).TestFunc(
		func(value *string, _ z.Ctx) bool {
			return value != nil && tenantIDPattern.MatchString(*value) && *value != "admin"
		},
		z.Message("tenant realm must be a URL-safe slug and must not be admin"),
	).Required(),
	"DisplayName": z.String().Min(1).Max(200).Required(),
	"Status": z.StringLike[TenantStatus]().TestFunc(
		func(value *TenantStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("tenant status is not in enum"),
	).Required(),
	"CreatedAt": z.Time().Required(),
	"UpdatedAt": z.Time().Required(),
})

var userSchema = z.Struct(z.Shape{
	"ID":                z.String().Required(),
	"PreferredUsername": z.String().Min(1).Max(100).Required(),
	"PasswordHash":      z.String().Required(),
	"Name":              z.Ptr(z.String().Max(200)),
	"GivenName":         z.Ptr(z.String().Max(100)),
	"FamilyName":        z.Ptr(z.String().Max(100)),
	"Email":             z.Ptr(z.String().Email()),
	"Roles":             z.Slice(z.String().Min(1)),
	"CreatedAt":         z.Time().Required(),
	"UpdatedAt":         z.Time().Required(),
})

var userAttributeDefSchema = z.Struct(z.Shape{
	"Key": z.String().TestFunc(
		func(value *string, _ z.Ctx) bool {
			return value != nil && attrKeyPattern.MatchString(*value)
		},
		z.Message("attribute key must be snake_case starting with a letter"),
	).Required(),
	"Type": z.StringLike[AttributeType]().TestFunc(
		func(value *AttributeType, _ z.Ctx) bool { return value.Valid() },
		z.Message("attribute type is not in enum"),
	).Required(),
	"Label":     z.String().Max(100),
	"ClaimName": z.Ptr(z.String().Min(1).Max(100)),
	"OIDCScope": z.Ptr(z.String().Min(1).Max(60)),
	"Visibility": z.StringLike[AttrVisibility]().TestFunc(
		func(value *AttrVisibility, _ z.Ctx) bool { return value.Valid() },
		z.Message("attribute visibility is not in enum"),
	).Required(),
})

var groupSchema = z.Struct(z.Shape{
	"ID":          z.String().Min(1).Max(64).Required(),
	"TenantID":    z.String().Min(1).Required(),
	"Name":        z.String().Min(1).Max(100).Required(),
	"Description": z.Ptr(z.String().Max(500)),
	"Roles":       z.Slice(z.String().Min(1)),
	"CreatedAt":   z.Time().Required(),
	"UpdatedAt":   z.Time().Required(),
})

var groupMemberSchema = z.Struct(z.Shape{
	"GroupID":   z.String().Min(1).Required(),
	"UserID":    z.String().Min(1).Required(),
	"CreatedAt": z.Time().Required(),
})

var agentSchema = z.Struct(z.Shape{
	"ID":          z.String().Min(1).Max(64).Required(),
	"TenantID":    z.String().Min(1).Required(),
	"Name":        z.String().Min(1).Max(100).Required(),
	"Description": z.Ptr(z.String().Max(500)),
	"Kind": z.StringLike[AgentKind]().TestFunc(
		func(value *AgentKind, _ z.Ctx) bool { return value.Valid() },
		z.Message("agent kind is not in enum"),
	).Required(),
	"OwnerUserID": z.String().Min(1).Required(),
	"Status": z.StringLike[AgentStatus]().TestFunc(
		func(value *AgentStatus, _ z.Ctx) bool { return value.Valid() },
		z.Message("agent status is not in enum"),
	).Required(),
	"Roles":     z.Slice(z.String().Min(1)),
	"CreatedAt": z.Time().Required(),
	"UpdatedAt": z.Time().Required(),
})

var agentCredentialBindingSchema = z.Struct(z.Shape{
	"AgentID":   z.String().Min(1).Required(),
	"ClientID":  z.String().Min(1).Required(),
	"CreatedAt": z.Time().Required(),
})

var authorizationRequestSchema = z.Struct(z.Shape{
	"ID": z.String().UUID().Required(),
	"State": z.StringLike[AuthorizationCodeFlowState]().TestFunc(
		func(value *AuthorizationCodeFlowState, _ z.Ctx) bool { return value.Valid() },
		z.Message("state is not in enum"),
	).Required(),
	"ClientID":    z.String().Required(),
	"RedirectURI": z.String().URL().Required(),
	"ResponseType": z.StringLike[ResponseType]().OneOf(
		[]ResponseType{ResponseTypeCode},
		z.Message("response_type must be code"),
	).Required(),
	"CodeChallenge": z.String().Required(),
	"CodeChallengeMethod": z.StringLike[CodeChallengeMethod]().OneOf(
		[]CodeChallengeMethod{CodeChallengeMethodS256},
		z.Message("code_challenge_method must be S256"),
	).Required(),
	"MaxAge":    z.Ptr(z.Int().GTE(0)),
	"CreatedAt": z.Time().Required(),
	"ExpiresAt": z.Time().Required(),
})

// ValidateAuthorizationRequest は oauth2 domain の認可要求検証を共有する。
func ValidateAuthorizationRequest(value any) error {
	return validate(authorizationRequestSchema, value)
}

var authorizationCodeRecordSchema = z.Struct(z.Shape{
	"Code":                   z.String().Required(),
	"AuthorizationRequestID": z.String().UUID().Required(),
	"ClientID":               z.String().Required(),
	"UserID":                 z.String().Required(),
	"RedirectURI":            z.String().URL().Required(),
	"CodeChallenge":          z.String().Required(),
	"CodeChallengeMethod": z.StringLike[CodeChallengeMethod]().OneOf(
		[]CodeChallengeMethod{CodeChallengeMethodS256},
		z.Message("code_challenge_method must be S256"),
	).Required(),
	"State": z.StringLike[AuthorizationCodeRecordState]().TestFunc(
		func(value *AuthorizationCodeRecordState, _ z.Ctx) bool { return value.Valid() },
		z.Message("state is not in enum"),
	).Required(),
	"IssuedAt":  z.Time().Required(),
	"ExpiresAt": z.Time().Required(),
})

// ValidateAuthorizationCodeRecord は oauth2 domain の認可コード検証を共有する。
func ValidateAuthorizationCodeRecord(value any) error {
	return validate(authorizationCodeRecordSchema, value)
}

var refreshTokenRecordSchema = z.Struct(z.Shape{
	"ID":                z.String().UUID().Required(),
	"Hash":              z.String().Required(),
	"FamilyID":          z.String().UUID().Required(),
	"ClientID":          z.String().Required(),
	"UserID":            z.String().Required(),
	"IssuedAt":          z.Time().Required(),
	"ExpiresAt":         z.Time().Required(),
	"AbsoluteExpiresAt": z.Time().Required(),
})

// ValidateRefreshTokenRecord は oauth2 domain の refresh token 検証を共有する。
func ValidateRefreshTokenRecord(value any) error { return validate(refreshTokenRecordSchema, value) }

var parRecordSchema = z.Struct(z.Shape{
	"RequestURI": z.String().Required(),
	"ClientID":   z.String().Required(),
	"IssuedAt":   z.Time().Required(),
	"ExpiresAt":  z.Time().Required(),
})

// ValidatePARRecord は oauth2 domain の PAR レコード検証を共有する。
func ValidatePARRecord(value any) error { return validate(parRecordSchema, value) }

var deviceAuthorizationSchema = z.Struct(z.Shape{
	"DeviceCodeHash": z.String().Required(),
	"UserCode":       z.String().Required(),
	"ClientID":       z.String().Required(),
	"State": z.StringLike[DeviceCodeFlowState]().TestFunc(
		func(value *DeviceCodeFlowState, _ z.Ctx) bool { return value.Valid() },
		z.Message("state is not in enum"),
	).Required(),
	"IntervalSeconds": z.Int().GT(0).Required(),
	"IssuedAt":        z.Time().Required(),
	"ExpiresAt":       z.Time().Required(),
})

// ValidateDeviceAuthorization は oauth2 domain の device authorization 検証を共有する。
func ValidateDeviceAuthorization(value any) error { return validate(deviceAuthorizationSchema, value) }

func validate(schema *z.StructSchema, value any) error {
	return zogError(schema.Validate(value))
}

// Validate は zog スキーマによるフィールド検証を行う。per-context domain パッケージが
// 自身のスキーマを検証するための汎用ラッパー (ADR-093)。
func Validate(schema *z.StructSchema, value any) error {
	return validate(schema, value)
}

// ZogError は zog の検証結果をメッセージ結合済みの error へ変換する。per-context domain
// パッケージが自身の Validate() 実装から呼び出す汎用ラッパー (ADR-093)。
func ZogError(issues z.ZogIssueList) error {
	return zogError(issues)
}

func zogError(issues z.ZogIssueList) error {
	if len(issues) == 0 {
		return nil
	}

	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue == nil {
			continue
		}
		message := issue.Message
		if message == "" && issue.Err != nil {
			message = issue.Err.Error()
		}
		if message == "" {
			message = issue.Code
		}
		if path := issue.PathString(); path != "" {
			message = fmt.Sprintf("%s: %s", path, message)
		}
		messages = append(messages, message)
	}
	if len(messages) == 0 {
		return errors.New("validation failed")
	}
	sort.Strings(messages)
	return errors.New(strings.Join(messages, "; "))
}
