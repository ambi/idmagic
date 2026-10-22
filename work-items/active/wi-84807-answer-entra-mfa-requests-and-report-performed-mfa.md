---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-23
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/ws-federation/features/passive-sign-in.md, requirement: REQ-WSFEDERATION-002 }
---

# Entra の MFA の要求に応え、実施した MFA を Entra に伝える

## 動機

Entra は、ドメインの `federatedIdpMfaBehavior` に従い、条件付きアクセスが MFA を求めるときにフェデレーションした IdP の MFA を受け入れる。
推奨の値 `enforceMfaByFederatedIdp` では、IdP が MFA をしていなければ、Entra は要求を IdP へ戻して MFA を求める（[internalDomainFederation](https://learn.microsoft.com/en-us/graph/api/resources/internaldomainfederation)、[AD FS での MFA の構成](https://learn.microsoft.com/en-us/entra/identity/authentication/howto-mfa-adfs)）。
IdP は、MFA をしたことを、Authentication Methods References のクレームの `http://schemas.microsoft.com/claims/multipleauthn` で伝える。

idmagic はどちらも行わない。

- Entra の定型設定のクレームは UPN と ImmutableID だけで、ログインセッションが TOTP や WebAuthn で MFA を済ませていても伝えない。Entra は自分で MFA をやり直す。
- `wauth` の未知の値はヒントとして扱い、要求された方式を満たさずに発行する。`enforceMfaByFederatedIdp` では、MFA を求められても MFA をしないトークンを返し、利用者はサインインできない。

## 対象範囲

- Entra の定型設定を持つ RP へ発行するとき、ログインセッションの `amr` が MFA を示せば、`http://schemas.microsoft.com/claims/authnmethodsreferences` に `http://schemas.microsoft.com/claims/multipleauthn` を含める。
- Entra が MFA を求める `wauth` を受けたとき、ログインセッションが MFA を済ませていなければ、MFA を求めるログインへ誘導する。
- Entra の構成の結果で、`federatedIdpMfaBehavior` に渡す値を案内する。

## 対象外

- 社内のネットワークからの要求を示すクレーム（`insidecorporatenetwork`）。社内の IP の範囲の設定を持つ wi-98213 の後に検討する。
- Entra 以外の RP の `wauth` の扱いの変更。

## 設計

MFA の判定は、`amr` からの決定論的な計算にする。

```go
// PerformedMFA は、ログインセッションの amr が二つ目の要素の認証を含むかを返す。
func PerformedMFA(amr []string) bool
```

ログインへの誘導は、受動的なサインインが再認証に使う 303 の経路を使い、ログインの画面に MFA を求める指示を渡す。
指示の渡し方は、SAML の `RequestedAuthnContext` と OIDC の `acr_values` の扱いと揃えるため、T001 で Authentication の既存の手段を確かめて決める。

## 計画

1. T001 で、Entra が MFA を求めるときの `wauth` の値を一次資料か実テナントで確かめ、ログインへ MFA を求める手段を決める。

## タスク

- [ ] T001 [Plan] `wauth` の値と、ログインへ MFA を求める手段を決める。
- [ ] T002 [Spec] REQ-WSFEDERATION-002 と Entra の定型設定のクレームの表を改める。
- [ ] T003 [Acceptance] MFA を済ませたセッションのトークンにクレームがないことと、MFA を求める `wauth` で MFA なしに発行することを RED で確かめる。
- [ ] T004 [App] クレームの発行と、MFA を求めるログインへの誘導を実装する。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run check`、`mise run check-spec-impact`。
- 実テナントでの `enforceMfaByFederatedIdp` の確認は wi-79 に足す。

## リスク

- MFA を示すクレームを誤って発行すると、Entra は MFA をしていない利用者を MFA 済みとして扱う。`PerformedMFA` をミューテーションテストの対象にする。
