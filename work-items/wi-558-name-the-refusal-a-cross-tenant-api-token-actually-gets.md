---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: docs
spec_impact: { kind: none, reason: "本項目が決めるのは、シナリオの具体例が名指す拒否の型を実装に合わせるか、実装をシナリオに合わせるかである。決着したうえで規範を書き換えるなら、その時点で spec-change を通す。まだどちらとも決まっていないので、この記録の時点では規範を触らない。" }
---

# テナントの一致しない API アクセストークンが実際に受ける拒否を、具体例に書く

## Motivation

`docs/contexts/saml/scenarios.feature.md` の `EX-SAML-005-03` は、トークンのテナントとリクエスト先のテナントが一致しないとき「操作を `AccessDeniedError` で拒否する」と宣言している。

[[wi-550-back-saml-examples-with-tests]] が測ったところ、製品が返すのは 401 `invalid_token` である。`acme` レルムで発行した `saml:read` と `saml:write` のトークンを `default` レルムの `/api/admin/v1/saml/service-providers` へ提示すると、参照も登録も 401 になり、保存先には何も残らない。

拒否そのものは効いている。食い違っているのは拒否の型だけである。

そして 401 の側には理由がある。管理発行トークンの照合は `apitoken/usecases` の `AuthenticateClaims` が `FindByJTI(ctx, リクエスト先テナント, jti)` で行い、見つからなければ RFC 7662 に従って検証の内訳を漏らさず `active: false` を返す。`usecases.go` にはその旨の `//nolint:nilerr` 注記が置いてある。403 `AccessDeniedError` へ変えることは、「このトークン自体は有効だが、このテナントでは使えない」という事実を提示者へ伝えることになる。

つまりこれは写像の欠落ではなく、規範と実装のどちらを正とするかの判断である。

## Scope

- `EX-SAML-005-03` の `Then` を、実装が返す拒否と一致させるか、実装を具体例に合わせるかを決める。
- 決めた側へ寄せる。規範を直すなら `spec-change` を通し、実装を直すなら RFC 7662 の非開示との折り合いを記録に残す。
- 決着後、`EX-SAML-005-03` を名指しするテストを書き、`tools/check/example-coverage-debt.json` から外す。テストを置く場所は `backend/shared/http/server_http/saml_scope_examples_test.go` で、同じレルム越えの提示を組み立てる材料はそこに揃っている。

## Out of Scope

- SAML 以外の Context が同じ形で宣言している具体例。同じ判断が要るなら、この項目の結論を参照して個別に扱う。
- 管理発行トークンの照合そのものの設計変更。

## Verification

- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **実装を具体例に寄せるほうが速いので、そちらへ流れる。** 401 を 403 に変えるのは 1 行で済むが、それは「このトークン自体は有効である」ことを別テナントの提示者へ伝える変更になる。判断の材料は RFC 7662 の非開示と、そこへ寄せた既存の設計である。速さを理由に選ばない。
- **拒否が効いていることを、型が合っていないことと混同する。** 測定では参照も登録も 401 で止まり、保存先には何も残っていない。この項目が扱うのは型の食い違いだけであり、境界そのものは壊れていない。
