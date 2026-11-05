---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-06
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/oauth2/features/client.md, requirement: REQ-OAUTH2-035 }
  - { path: docs/modules/tenancy/features/quota.md, requirement: REQ-TENANCY-036 }
---

# 単独のクライアントの削除も、Application の削除と同じく同意とトークンを片付ける

## 動機

[wi-10874](../done/wi-10874-inventory-promises-kept-by-every-entry-path.md) の棚卸しで、OAuth2 クライアントを消す二つの経路が別々に書かれていることが分かった。

| 経路 | 同意とトークン | `consents` の使用量 |
| --- | --- | --- |
| Application の削除（`application/internal/db_postgres/deletion.go`） | 消す（`DeleteConsentsForClient` など） | 消した件数だけ減らす |
| 単独のクライアントの削除（`DeleteAdminOAuth2Client`） | 扱わない | 減らさない |

`consents.client_id` と `refresh_tokens.client_id` の外部キーは `ON DELETE RESTRICT` なので、同意か未失効のリフレッシュトークンを持つクライアントの削除は外部キーの違反で失敗する。
[REQ-OAUTH2-035](../../docs/modules/oauth2/features/client.md) は削除に 204 を返すと定めるので、利用されたクライアントを管理者が消せないのは違反である。
同意を持たないクライアントの削除が通っても、同じ集約を消す二つの経路で片付けの範囲が異なる。

## 対象範囲

- 単独のクライアントの削除で、Application の削除と同じ参加者（同意、リフレッシュトークン、クライアントのセッションなど）を消し、`consents` を消した件数だけ減らす。

## 対象外

- 組み込みのクライアントの削除の拒否（REQ-OAUTH2-080）。変えない。
- 消したリフレッシュトークンの失効の外部への伝播。

## 設計

### 所有の形

候補 A を採る。
OAuth2 がクライアントの削除の操作を一つだけ公開し（現在の `DeleteForApplicationWithState` を基にする）、Application の削除と管理 API の削除の両方がそれを通る。
`ClientRepo.Delete` をこの操作の外から呼べなくする。
使用量の減算もこの操作の中に置く。

## タスク

- [ ] T001 [Test] 同意を持つクライアントを管理 API で消すと失敗することを RED にする。
- [ ] T002 [App] 削除の操作を一つにし、二つの経路を通す。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/client/...`
- `mise run test-go-package -- ./backend/application/...`

## リスク

- Application の削除の確定の順と補償を変えないよう、既存の削除の参加者のテストを保つ。
