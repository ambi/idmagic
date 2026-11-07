---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-07
priority: p3
depends_on: [wi-17165-migrate-google-uuid-to-standard-uuid]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/features/agent-observation.md, requirement: REQ-IDMANAGEMENT-099 }
---

# urn:uuid: 接頭辞の ID を PostgreSQL の照会前に存在しない行として扱う

## 動機

[標準 uuid への移行](../done/wi-17165-migrate-google-uuid-to-standard-uuid.md)で、PostgreSQL の `uuid` 列を照会する前の検査を `sharedpg.IsUUIDText`（標準 `uuid.Parse` の成否）へまとめた。
標準 `uuid.Parse` は小文字の `urn:uuid:` 接頭辞を受け付けるが、PostgreSQL の `uuid` 型の入力は urn 形式を受け付けない。
そのため `urn:uuid:<UUID>` を渡すと検査を通り、照会が 22P02 で失敗する。
各リポジトリの検査は、UUID でない値を存在しない行として扱う意図で置かれている。
たとえば観測の確認で `agent_id` に `urn:uuid:…` を渡すと、REQ-IDMANAGEMENT-099 が求める 404 と `agent_not_found` ではなくエラー応答になる。
移行前の google/uuid の `Validate` も同じ値を通しており、移行で生じた欠陥ではない。

## 対象範囲

- `sharedpg.IsUUIDText` を、PostgreSQL の `uuid` 型が受け付けない urn 形式を偽とする判定にする。
- 観測の確認で `urn:uuid:` 接頭辞の `agent_id` が 404 と `agent_not_found` になることを確かめる。

## 対象外

- PostgreSQL が受け付けるが標準 `uuid.Parse` が拒否する表記（4 桁ごとの任意のハイフンなど）の受理。現在も存在しない行として扱われ、偽陰性になるだけで失敗はしない。
- ID の正規形以外の表記を一律に拒否するかどうかの方針。

## 設計

候補は二つある。

- 判定で `urn:` 接頭辞を大文字小文字を問わず拒否する。PostgreSQL が受け付ける表記との差が最小になる。
- 判定を通った値を `uuid.Parse` の結果の正規形に直して照会へ渡す。urn 形式でも行が見つかるようになり、受け付ける表記が広がる。

前者を第一候補とする。後者は利用者が観測する受理範囲を広げるため、選ぶなら仕様の判断を先に行う。

## 計画

1. 観測の確認の HTTP 境界で、`urn:uuid:` 接頭辞の `agent_id` が 404 にならないことを RED で確かめる。
2. `sharedpg.IsUUIDText` を直し、判定の両側を単体テストで固定する。

## タスク

- [ ] T001 [Test] `urn:uuid:` 接頭辞の `agent_id` で確認を要求し、404 と `agent_not_found` を期待する RED を確認する。
- [ ] T002 [App] `sharedpg.IsUUIDText` が urn 形式を拒否するようにする。

## 検証

- `mise run test-go-package -- ./backend/shared/storage/db_postgres` と、観測の確認の PostgreSQL 統合テスト。
- `mise run lint-go`。

## リスク

判定を使うほかのリポジトリ（tokenvault、sharedsignals、user、sourcing）でも、urn 形式の値はエラーから存在しない行の扱いへ変わる。
いずれも検査の意図に沿う変化である。
