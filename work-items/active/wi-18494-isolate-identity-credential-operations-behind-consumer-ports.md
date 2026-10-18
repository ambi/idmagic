---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で IdManagement → Authentication を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# IdManagement の資格情報操作を要求側のポートで分離する

## 動機

[依存の向きの決定](wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`IdManagement → Authentication` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

パスワードの検証とハッシュ、履歴、MFA、WebAuthn、リカバリーコード、信頼済み端末、セッション、本人照会と完全削除の要求を対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

認証と認可の保証、パスワード形式、削除の原子性、使用量の再集計の変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[ユーザーのライフサイクル](../../docs/modules/identity-management/features/user.lifecycle.md)、[User CSV](../../docs/modules/identity-management/features/user-csv.md)、[アカウント](../../docs/modules/identity-management/features/account.md)と、`backend/idmanagement/internal/user/usecases/admin_users.go`、`user_import_apply.go`、`internal/user/db_postgres/user_import_committer.go`、`internal/user/handlers_http/account_handler.go` を起点にする。

D2、D4、D6、D7 を適用する。
IdManagement に必要な資格情報操作と本人照会の値を定義し、Authentication の公開操作へ組み立て地点で変換する。
User の作成、無効化と完全削除の編成は IdManagement に、認証情報の規則と保存は Authentication に残す。
Authentication → IdManagement の主体照会と更新は残す。

CSV は既存のトランザクション参加の書き込み口を変換し、User と認証情報の確定境界を維持する。
削除は現状の各保存先の確定と再実行を保ち、独立した一括 Tx に置き換えない。
Repository 群の保存型を複製する案と、User のライフサイクル全体を Authentication へ移す案は採らない。
認証方式を増やす変更は Authentication と必要な要求側の操作へ閉じ、IdManagement に認証方式ごとの保存知識を増やさない。

呼び出し順と再入を User 作成、CSV、無効化、完全削除、本人照会に分けて比較する。
必須の資格情報更新や失効を省略する未配線経路を増やさず、パスワード規則の拒否と同じ Tx の要否を既存仕様に照合する。

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

- `mise run test-go-package -- ./backend/idmanagement/internal/user/usecases` の `TestDeleteUserAnonymizesAndCascades` と CSV の既存検査、HTTP の本人認証検査を起点にする。
  PostgreSQL の CSV 確定境界も既存の共有契約に従って検査し、正式な配線は `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。
