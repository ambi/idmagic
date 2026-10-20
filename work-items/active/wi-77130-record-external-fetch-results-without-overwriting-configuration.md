---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/workloadidentity/features/trust-configuration.md, requirement: REQ-WORKLOADIDENTITY-008, impact: conforms }
  - { path: docs/modules/authentication/features/federation.md, requirement: REQ-AUTHENTICATION-025, impact: conforms }
---

# 外部取得の結果を、設定を上書きせずに記録する

## 動機

次の二つの管理操作は、設定を読んでから外部のメタデータを取得し、その後に設定を丸ごと保存する。

| 操作 | 実装 | 本当に変える値 |
| --- | --- | --- |
| トラストバンドルの JWKS の確認 | `backend/workloadidentity/internal/usecases/admin_trust_bundles.go:RefreshWorkloadTrustBundleJWKS` | `jwks_cached_at` |
| IdP 接続のメタデータの更新 | `backend/authentication/internal/federation/handlers_http/routes.go:refresh` | Discovery で得たエンドポイントなどのメタデータ |

外部の取得には数秒かかり得る。
その間に別の管理者が同じ設定を無効化すると、取得後の保存が読んだ時点の状態を書き戻し、無効にしたトラストバンドルや IdP 接続が有効に戻る（REQ-WORKLOADIDENTITY-008）。
同じ窓で、名前、シークレットなどの設定の変更も失われる。

管理者どうしの同じ項目の編集の競合は [wi-605](wi-605-detect-lost-updates-with-etag-and-if-match.md) が版で扱う。
この二つは、管理者が編集しない値だけを書く処理が設定全体を書き戻す問題であり、版の検出ではなく書く列を絞って直す。

## 対象範囲

- 二つの操作の保存を、変える列だけを書く専用の永続化操作へ置き換える。メモリと PostgreSQL の実装と共有の契約テストを揃える。
- 外部取得の間に無効化した設定が、取得後も無効のままであることを検査する。
- [一貫性の境界と永続化](../../docs/design/application/design-guidelines.md#一貫性の境界と永続化)の判定基準に、集約の一部だけを書く処理の規則を加える。

## 対象外

- 管理者どうしの同じ項目の編集の競合（wi-605）。
- Authentication が User を丸ごと保存する問題（[wi-98381](wi-98381-update-authentication-state-without-overwriting-the-user.md)）。
- 所有者の停止に伴う Agent の無効化（[wi-97231](wi-97231-disable-owned-agents-with-a-conditional-update.md)）。

## 設計

ログインセッションの `RevokeAuthenticationSession` と `TouchAuthenticationSession` と同じ形にする。
集約を丸ごと保存する操作は残し、管理者が編集しない値の記録だけを専用の操作にする。

- トラストバンドル：`RecordJWKSFetched(ctx, tenantID, id, at)` が `jwks_cached_at` だけを書く。
- IdP 接続：Discovery の結果の列だけを書く操作を設ける。どの列が Discovery の結果かは着手時に `RefreshDiscovery` が変える値から決める。

設計ガイドラインに加える規則：集約のうち、利用者や管理者が編集しない値（時刻、外部取得の結果、計数）だけを記録する処理は、その列だけを書く操作を使い、読んだ集約を丸ごと保存しない。
すべての保存を列ごとの更新にはしない。複数の値にまたがる不変条件の検査が SQL に散らばるためである。

## 計画

1. 外部取得を待つ間に無効化を挟むテストで、二つの操作の故障を RED にする。
2. 専用の永続化操作を実装し、GREEN にする。
3. 設計ガイドラインに規則を加える。

## タスク

- [ ] T001 [Acceptance] JWKS の確認と IdP のメタデータ更新が、間の無効化を書き戻す故障を RED にする。
- [ ] T002 [App] 専用の永続化操作を実装し、二つの操作の保存を置き換える。
- [ ] T003 [Docs] 設計ガイドラインに、集約の一部だけを書く処理の規則を加える。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- メモリと PostgreSQL の共有の契約テストで、専用の操作がほかの列を変えないことを確かめる。PostgreSQL の検査はサンドボックスの外で実行する。
- `mise run test-go-changed`、`mise run lint-go`、`mise run check-links`。

## リスク

IdP 接続の Discovery の結果と管理者の設定が同じ列を共有している場合、列を絞れない。
その場合は着手時に、メタデータを設定から分けるか、版の比較を使うかを決める。
