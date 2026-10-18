// Package tenantcontext は処理の対象 ID と任意の HTTP 位置情報を一つの文脈で運ぶ。
package tenantcontext

import (
	"context"
	"strings"
)

// Scope の実装側が、ID に付随する設定の有無と寿命を所有する。
type Scope interface {
	TenantID() string
}

type contextKey struct{}

func WithScope(ctx context.Context, scope Scope) context.Context {
	return context.WithValue(ctx, contextKey{}, scope)
}

func Current(ctx context.Context) Scope {
	scope, _ := ctx.Value(contextKey{}).(Scope)
	return scope
}

type identity string

func (id identity) TenantID() string { return string(id) }

// WithID は親の設定や HTTP 位置情報を引き継がず、対象 ID だけを運ぶ。
func WithID(ctx context.Context, id string) context.Context {
	return WithScope(ctx, identity(id))
}

// TenantID は未配線の処理を default へ落とさない。拒否の判断はここに集める。
func TenantID(ctx context.Context) string {
	if scope := Current(ctx); scope != nil {
		if id := scope.TenantID(); id != "" {
			return id
		}
	}
	panic("tenancy: no resolved tenant in context")
}

// HTTPLocation の値を決める規則は提供側にあり、このパッケージは解釈しない。
func HTTPLocation(ctx context.Context) (issuer, prefix string, hostOnlyCookies bool) {
	if scope, ok := Current(ctx).(interface {
		HTTPLocation() (string, string, bool)
	}); ok {
		return scope.HTTPLocation()
	}
	return "", "", false
}

func Issuer(ctx context.Context, fallback string) string {
	issuer, _, _ := HTTPLocation(ctx)
	if issuer != "" {
		return issuer
	}
	return strings.TrimSuffix(fallback, "/")
}

func URLPrefix(ctx context.Context) string {
	_, prefix, _ := HTTPLocation(ctx)
	return prefix
}
