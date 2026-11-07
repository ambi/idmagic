package handlers_http

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type authorizeRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	StateParam          string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	Prompt              string
	MaxAge              *int
	AcrValues           string
}

var singleValueAuthorizationParameters = map[string]struct{}{
	"client_id": {}, "redirect_uri": {}, "response_type": {}, "scope": {}, "state": {}, "nonce": {},
	"code_challenge": {}, "code_challenge_method": {}, "prompt": {}, "max_age": {}, "acr_values": {},
	"request_uri": {}, "authorization_details": {},
}

func validateAuthorizationParameterCardinality(values url.Values) error {
	for key, values := range values {
		if _, securitySensitive := singleValueAuthorizationParameters[key]; securitySensitive && len(values) != 1 {
			return fmt.Errorf("authorization parameter %s must occur exactly once", key)
		}
	}
	return nil
}

// parseAuthorizeRequest は要求に現れた parameter だけを読む。必須の判定は parameter の
// 有無で行い、空の値は現れたものとして扱う。
func parseAuthorizeRequest(values url.Values) (authorizeRequest, error) {
	if err := validateAuthorizationParameterCardinality(values); err != nil {
		return authorizeRequest{}, err
	}
	var issues validationIssues
	required := func(key string) string {
		if !values.Has(key) {
			issues.add(key, "is required")
		}
		return values.Get(key)
	}
	request := authorizeRequest{
		ClientID:            required("client_id"),
		RedirectURI:         values.Get("redirect_uri"),
		ResponseType:        required("response_type"),
		Scope:               values.Get("scope"),
		StateParam:          values.Get("state"),
		Nonce:               values.Get("nonce"),
		CodeChallenge:       required("code_challenge"),
		CodeChallengeMethod: required("code_challenge_method"),
		Prompt:              values.Get("prompt"),
		AcrValues:           values.Get("acr_values"),
	}
	// 省けるかは登録件数と openid の有無で決まるので、use case が判断する。
	if values.Has("redirect_uri") && !spec.IsURL(request.RedirectURI) {
		issues.add("redirect_uri", "must be a valid URL")
	}
	if values.Has("max_age") {
		maxAge, err := strconv.Atoi(values.Get("max_age"))
		switch {
		case err != nil:
			issues.add("max_age", "number is invalid")
		case maxAge < 0:
			request.MaxAge = &maxAge
			issues.add("max_age", "number must be greater than or equal to 0")
		default:
			request.MaxAge = &maxAge
		}
	}
	return request, issues.err()
}

type registerClientRequest struct {
	ClientName                        string         `json:"client_name"`
	ClientType                        string         `json:"client_type"`
	RedirectURIs                      []string       `json:"redirect_uris"`
	GrantTypes                        []string       `json:"grant_types"`
	ResponseTypes                     []string       `json:"response_types"`
	TokenEndpointAuthMethod           string         `json:"token_endpoint_auth_method"`
	Scope                             string         `json:"scope"`
	JWKS                              map[string]any `json:"jwks"`
	JwksURI                           *string        `json:"jwks_uri"`
	TlsClientAuthSubjectDN            *string        `json:"tls_client_auth_subject_dn"`
	RequirePAR                        bool           `json:"require_pushed_authorization_requests"`
	DpopBoundAccessTokens             bool           `json:"dpop_bound_access_tokens"`
	FapiProfile                       string         `json:"fapi_profile"`
	BackChannelLogoutURI              *string        `json:"backchannel_logout_uri"`
	BackChannelLogoutSessionRequired  bool           `json:"backchannel_logout_session_required"`
	FrontChannelLogoutURI             *string        `json:"frontchannel_logout_uri"`
	FrontChannelLogoutSessionRequired bool           `json:"frontchannel_logout_session_required"`
}

var (
	registrableClientTypes = []string{"public", "confidential"}
	registrableAuthMethods = []string{
		"client_secret_basic",
		"client_secret_post",
		"private_key_jwt",
		"tls_client_auth",
		"none",
	}
	registrableFapiProfiles = []string{"none", "fapi_2_security_profile"}
)

// validateRegisterClientRequest は空の値を未指定として扱い、指定された値だけを検査する。
// 違反は Go のフィールド名で報告する。
func validateRegisterClientRequest(request *registerClientRequest) error {
	var issues validationIssues
	if utf8.RuneCountInString(request.ClientName) > spec.LengthDisplayName {
		issues.add("ClientName", fmt.Sprintf("must be at most %d characters", spec.LengthDisplayName))
	}
	issues.oneOf("ClientType", request.ClientType, registrableClientTypes)
	for i, uri := range request.RedirectURIs {
		if uri != "" && !spec.IsURL(uri) {
			issues.add(fmt.Sprintf("RedirectURIs[%d]", i), "must be a valid URL")
		}
	}
	for i, grant := range request.GrantTypes {
		if grant != "" && !spec.GrantType(grant).Valid() {
			issues.add(fmt.Sprintf("GrantTypes[%d]", i), "grant_type is not in enum")
		}
	}
	for i, responseType := range request.ResponseTypes {
		issues.oneOf(fmt.Sprintf("ResponseTypes[%d]", i), responseType, []string{"code"})
	}
	issues.oneOf("TokenEndpointAuthMethod", request.TokenEndpointAuthMethod, registrableAuthMethods)
	if uri := deref(request.JwksURI); uri != "" {
		if !spec.IsURL(uri) {
			issues.add("JwksURI", "must be a valid URL")
		}
		if !isHTTPSWithoutUserinfoOrFragment(uri) {
			issues.add("JwksURI", "jwks_uri must be https and must not contain userinfo or fragment")
		}
	}
	issues.oneOf("FapiProfile", request.FapiProfile, registrableFapiProfiles)
	for field, uri := range map[string]*string{
		"BackChannelLogoutURI":  request.BackChannelLogoutURI,
		"FrontChannelLogoutURI": request.FrontChannelLogoutURI,
	} {
		if value := deref(uri); value != "" && !spec.IsURL(value) {
			issues.add(field, "must be a valid URL")
		}
	}
	if request.TokenEndpointAuthMethod == "tls_client_auth" && deref(request.TlsClientAuthSubjectDN) == "" {
		issues.add("", "tls_client_auth requires tls_client_auth_subject_dn")
	}
	return issues.err()
}

func isHTTPSWithoutUserinfoOrFragment(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" && parsed.User == nil && parsed.Fragment == ""
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// validationIssues は「フィールド: 文言」を集め、辞書順に「; 」で結んだ error にする。
// フィールド名は呼び出し側が渡した綴りのまま使う。
type validationIssues []string

func (v *validationIssues) add(field, message string) {
	if field != "" {
		message = fmt.Sprintf("%s: %s", field, message)
	}
	*v = append(*v, message)
}

func (v *validationIssues) oneOf(field, value string, options []string) {
	if value != "" && !slices.Contains(options, value) {
		v.add(field, fmt.Sprintf("string must be one of %v", options))
	}
}

func (v validationIssues) err() error {
	if len(v) == 0 {
		return nil
	}
	sort.Strings(v)
	return errors.New(strings.Join(v, "; "))
}
