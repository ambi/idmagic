---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-25
priority: p3
change_kind: docs
affected_spec:
  - { path: docs/domain/ws-federation/scenarios.feature.md, requirement: REQ-WSFEDERATION-001 }
---

# テナントの一致しない API アクセストークンが WS-Federation の管理 API で受ける拒否を、具体例に書く

## 動機

`docs/domain/ws-federation/scenarios.feature.md` の `EX-WSFEDERATION-001-03` は、トークンのテナントとリクエスト先のテナントが一致しないとき「操作を `AccessDeniedError` で拒否する」と宣言している。

[[wi-554-back-ws-federation-examples-with-tests]] の実測では、製品が返すのは 401 `invalid_token` である。
`acme` レルムで発行した `wsfed:read` と `wsfed:write` のトークンを `default` レルムの `/api/admin/v1/wsfed/relying-parties`（一覧、登録、削除）と `/api/admin/v1/wsfed/entra-federation` へ提示すると、8 通りすべてが 401 になり、保存先の RP は変わらない。

これは `EX-SAML-005-03` と同じ経路の食い違いである。
[[wi-558-name-the-refusal-a-cross-tenant-api-token-actually-gets]] は SAML について、RFC 7662 の非開示に従う 401 を正とし、具体例を `InvalidAccessTokenError` へ合わせた。
同項目は `EX-WSFEDERATION-001-03` を Out of Scope とし「個別の記録で扱う」と書いたが、その記録は起票されていなかった。
この具体例は `tools/check/example-coverage-debt.json` に `blocked_by: wi-558` のまま残り、[[wi-496-burn-down-the-example-coverage-debt]] が台帳を空にできない。

## 対象範囲

- `EX-WSFEDERATION-001-03` の `Then` を、実装が返す拒否（401 の `InvalidAccessTokenError`）へ合わせる。
- `EX-WSFEDERATION-001-03` を名指しするテストを書き、`tools/check/example-coverage-debt.json` から外す。テストは RP の一覧、登録、削除と Entra フェデレーションの操作を列挙し、拒否応答と保存先の RP が変わらないことの双方を観測する。

## 対象外

- 管理発行トークンの照合そのものの設計変更。
- 403 `AccessDeniedError` へ実装を寄せる変更。wi-558 が却下した理由（トークン自体は有効だという事実を別テナントの提示者へ伝える）がそのまま当てはまる。

## 設計

wi-558 の結論を踏襲する。
判断の材料は RFC 7662 の非開示と、`apitoken/usecases` の `AuthenticateClaims` がリクエスト先テナントで jti を照合し、見つからなければ `active: false` を返す既存の設計である。
TypeSpec はすでに 401 の `IdMagic.Contract.AuthenticationRequiredResponse` を宣言しているので、TypeSpec と製品のコードは変わらない見込みである。

## 計画

1. `spec-change` で `EX-WSFEDERATION-001-03` の `Then` を書き換え、`mise run check-spec` を通す。
2. 台帳から外した状態で `mise run check-spec` が落ちることを確かめ、テストを書いて通す。

## タスク

- [ ] T001 [Spec] `EX-WSFEDERATION-001-03` の `Then` を 401 の `InvalidAccessTokenError` へ書き換える。
- [ ] T002 [Acceptance] 台帳から外した状態で `mise run check-spec` の RED を確認する。
- [ ] T003 [Test] 8 通りの提示を列挙するテストを書き、`//spec:covers EX-WSFEDERATION-001-03` を付ける。
- [ ] T004 [Verify] `mise run verify`。

## 検証

- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## リスク

- **実装を具体例に寄せるほうが速いので、そちらへ流れる。** 401 を 403 に変えることは非開示を崩す変更であり、速さを理由に選ばない。
