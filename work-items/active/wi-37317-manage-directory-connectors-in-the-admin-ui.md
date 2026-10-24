---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-24
priority: p3
depends_on: [wi-46712-enroll-a-directory-connector-and-authenticate-its-calls, wi-23043-ingest-directory-users-from-connector-batches]
change_kind: feature
affected_spec:
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-002 }
---

# 管理画面で Directory Connector を登録し、同期の状態を確かめられるようにする

## 動機

[[wi-46712-enroll-a-directory-connector-and-authenticate-its-calls]] と [[wi-23043-ingest-directory-users-from-connector-batches]] は管理 API だけを提供する。
テナントの管理者が API を直接呼ばずに Connector を導入し、同期の失敗に気付けるよう、管理画面が要る。

## 対象範囲

- Connector の作成、登録トークンの表示と再発行、無効化の画面。
- Connector の状態、最終接続時刻、直近の IngestionRun の件数と失敗の一覧。
- 衝突などで失敗した項目の表示。

## 対象外

- 失敗した項目を既存の User へ結び付ける操作。
- 属性の対応付けの編集。対応付けは Connector の設定が持つ。

## 設計

画面の配置と文言は、既存の管理画面のアプリケーションの一覧と詳細の形に合わせる。
登録トークンは作成直後に一度だけ表示し、再表示の手段を持たない。

## タスク

- [ ] T001 [Spec] 画面が使う管理 API に不足がないかを確かめ、要件を書く。
- [ ] T002 [Acceptance] 作成からトークンの表示、無効化までを UI の受け入れテストで RED にする。
- [ ] T003 [UI] 画面を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run typecheck-ui`、`mise run lint-ui`、`mise run test-ui-e2e-file`。e2e はサンドボックスの外で実行する。

## リスク

- 登録トークンが画面の履歴やログに残ると、偽の Connector を登録できる。一度だけの表示と、キャッシュしない応答を確かめる。
