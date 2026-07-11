// Package kernel は複数 bounded context から真に published language として参照される、
// ごく小さな型・定数のみを持つ (ADR-089)。何を収録するかは spec/scl.yaml
// context_map の publishes/depends_on.uses を規範とする。
package kernel

// DefaultTenantID は既定テナントの不変 UUID 代理キー (ADR-085)。所有権は
// tenancy/domain (DefaultTenantID として re-export) にあるが、shared/spec の AuthZEN
// policy 述語は tenancy/domain を import すると import cycle になる
// (tenancy/domain が spec.Validate を使うため) ため、ここに置く (wi-179, ADR-089)。
const DefaultTenantID = "00000000-0000-4000-8000-000000000000"

// DefaultRealm は既定 realm slug。URL `/realms/{realm}/` 等の公開語彙に現れる。
const DefaultRealm = "default"
