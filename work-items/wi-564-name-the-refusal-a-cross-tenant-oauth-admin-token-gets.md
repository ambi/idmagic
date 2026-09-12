---
depends_on: [wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: docs
spec_impact: { kind: none, reason: "本項目が決めるのは、具体例が名指す拒否の型を実装に合わせるか、実装を具体例に合わせるかである。判断そのものは wi-558 が SAML 側で先に決める。決着したうえで規範を書き換えるなら、その時点で spec-change を通す。" }
---

# テナントの一致しない OAuth2 管理 API トークンが実際に受ける拒否を、具体例に書く

## Motivation

`docs/contexts/oauth2/scenarios.feature.md` の `EX-OAUTH2-003-04` は、トークンのテナントとリクエスト先のテナントが一致しないとき「操作を `AccessDeniedError` で拒否する」と宣言している。

[[wi-538-back-oauth2-examples-with-tests]] が測ったところ、製品が返すのは 401 `invalid_token` である。`acme` レルムで発行した `oauth-clients:read` と `oauth-clients:write` のトークンを `default` レルムの `/api/admin/v1/clients` へ提示すると、参照も登録も 401 になり、`default` テナントのクライアント一覧は 1 件も動かない。`WWW-Authenticate` は `Bearer error="invalid_token"` を返す。

拒否そのものは効いている。食い違っているのは拒否の型だけである。

これは [[wi-550-back-saml-examples-with-tests]] が `EX-SAML-005-03` について測ったものと同じ食い違いであり、原因も同じである。管理発行トークンの照合は `apitoken/usecases` の `AuthenticateClaims` が `FindByJTI(ctx, リクエスト先テナント, jti)` で行い、見つからなければ RFC 7662 に従って検証の内訳を漏らさず扱う。403 `AccessDeniedError` へ変えることは、「このトークン自体は有効だが、このテナントでは使えない」という事実を提示者へ伝えることになる。

したがって判断は 1 つで、対象の具体例が 2 つある。[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] が SAML 側で判断を下し、本項目はその結論を OAuth2 側へ適用する。

## Scope

- wi-558 の結論に従って `EX-OAUTH2-003-04` を解決する。規範を直すなら `spec-change` を通し、実装を直すなら RFC 7662 の非開示との折り合いを記録に残す。
- 決着後、`EX-OAUTH2-003-04` を名指しするテストを書き、`tools/check/example-coverage-debt.json` から外す。テストを置く場所は `backend/shared/http/server_http/oauth2_scope_examples_test.go` で、レルム越えの提示を組み立てる材料 (`apiTokenStack` の `issue` と `oauthAdminRequest`) はそこに揃っている。

## Out of Scope

- wi-558 が下す判断そのもののやり直し。本項目は結論を適用する側である。
- 管理発行トークンの照合そのものの設計変更。
- REQ-OAUTH2-003 のほかの具体例。[[wi-538-back-oauth2-examples-with-tests]] が消化済みである。

## Verification

- `mise run check-spec` が、`EX-OAUTH2-003-04` を台帳から外した状態で通る。
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run verify`

## Risk Notes

- **SAML 側と別の結論を出す。** 同じ照合、同じ理由、同じ食い違いなので、2 つの具体例が別々の拒否の型を名指す状態は、規範の読み手に対して嘘になる。wi-558 の結論からずれるなら、ずれる理由を記録に書く。
- **拒否の型だけを直して、拒否が防いだ効果を観測しない。** 型を 403 へ変える実装も 401 のまま残す実装も、保存先を動かさないことまで読まなければ区別できない。
