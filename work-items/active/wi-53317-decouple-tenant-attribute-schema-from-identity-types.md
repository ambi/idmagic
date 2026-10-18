---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で Tenancy → IdManagement を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# テナント属性スキーマを IdManagement の保存型から切り離す

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`Tenancy → IdManagement` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

User と Group のスキーマの保存型、Repository、HTTP 表現、組み込み属性の照会、定義検証、動的 Group の参照保護を対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

属性定義の形式の変更、スキーマ更新の原子性の追加、テーブルの移管。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[属性スキーマ](../../docs/modules/tenancy/features/attribute-schema.md)の REQ-TENANCY-002、020、043 と、`backend/tenancy/internal/usecases/manage_user_attribute_schema.go`、`manage_group_attribute_schema.go`、`internal/handlers_http/admin_user_attribute_schema_handler.go` を起点にする。

D1、D2、D4、D6 を適用する。
Tenancy はスキーマの保存と管理 API を維持し、要求側の表現とポートを定義する。
IdManagement は属性定義の形式、組み込み属性と動的 Group の参照情報を引き続き定め、組み立て地点の変換で Tenancy のポートへ接続する。
定義検証と参照保護を通してから保存し、拒否時は保存とイベント発行を行わない既存の保証を保つ。
IdManagement → Tenancy のスキーマ照会は残し、表の所有者と保存形式は変えない。

全体を IdManagement へ移す案は保存と API の担当を変え、属性定義を shared へ移す案は D5 に反するため採らない。
Tenancy の型に IdManagement の別名を残す案では循環が消えない。
属性型を増やす変更では IdManagement の検証と境界の変換だけを見直し、Tenancy に同じ検証を複製しない。

保存型の表現を選ぶ際は、定義の意味と保存の担当を混同しない。
User と Group のスキーマを統合しない。

## 計画

着手時に既存の仕様、TypeSpec、名指した実装とテストを読み、操作列ごとの成功、拒否、途中失敗と再実行を照合する。
新しい型と操作のシグネチャ、変換の担当、必須依存と確定の境界をコードの前に記す。
この結果が上の保証維持と両立しない場合は、実装へ進む前に親項目の判断を見直す。
保証の追加や変更が必要になれば別の仕様変更として扱う。

## タスク

- [ ] T001 [Design] import の全入口と作用を再調査し、境界の型と操作、変換、確定順を設計へ記す。
- [ ] T002 [App] 要求側のポートと所有側の公開操作を結び、逆向きの辺を解消する。
- [ ] T003 [Docs] 公開範囲と現在の設計を同期し、実測で消えた負債だけを削除して依存図を再生成する。
- [ ] T004 [Verify] 既存の保証、正式な配線と境界を検査し、仕様差分を確認する。

## 検証

- `mise run test-go-package -- ./backend/tenancy/internal/usecases` と該当 HTTP テストで、`TestUpdateUserAttributeSchemaRejectsBuiltinCollision`、重複キーの拒否、動的参照の 409 と状態不変を確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。
