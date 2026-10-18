---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で IdManagement → OAuth2 を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# IdManagement の OAuth2 操作と照会を要求側のポートで分離する

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`IdManagement → OAuth2` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

User の同意、リフレッシュトークン、device code、承認要求の完全削除、Agent とクライアントの束縛と照会、本人データのエクスポートを対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

使用量再集計、削除の原子性、Agent の資格情報の意味と HTTP エクスポート項目の変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[ユーザーのライフサイクル](../../docs/modules/identity-management/features/user.lifecycle.md)の REQ-IDMANAGEMENT-050、[アカウント](../../docs/modules/identity-management/features/account.md)の REQ-IDMANAGEMENT-018、[Agent](../../docs/modules/identity-management/features/agent.md)の REQ-IDMANAGEMENT-074、078 と、`backend/idmanagement/internal/user/usecases/admin_users.go`、`internal/agent/usecases/admin_agents.go`、`internal/user/handlers_http/account_data_handler.go` を起点にする。

D2、D4、D6、D7 を適用する。
IdManagement に削除、Agent のクライアント操作、本人同意照会のポートと必要な値を定義し、OAuth2 の公開操作へ組み立て地点で接続する。
OAuth2 → IdManagement の主体照会は残す。
同意とトークンの保存型を IdManagement へ渡さず、状態所有者の検証と書き込みは OAuth2 に残す。

完全削除は同意、refresh、セッション、認証情報、device code、承認要求の既存の順を崩さないよう、Authentication 向けの別項目とも境界を照合する。
PendingPurge の進行、各保存先の確定、失敗後の再実行を維持する。
すべてを一操作の Tx にする案と非同期イベントに変える案は、現行の確定と完了の条件を変えるため採らない。
Agent の管理を OAuth2 へ移す案は主体の責務を分散させる。
トークンの保存方式を変える場合は OAuth2 の削除操作の内部へ閉じる。

束縛時の同一テナント確認と一つの Agent への束縛、本人以外をエクスポートできない条件を保つ。
[wi-83071](wi-83071-make-user-deletion-rerunnable-and-recount-quota-usage.md) の使用量再集計は混ぜない。
2つの逆向きの辺は別々に解消できるが、削除の順は両項目で共有する既存の操作列を基準にする。

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

- `mise run test-go-package -- ./backend/idmanagement/internal/user/usecases` の削除と再実行の検査、`./backend/idmanagement/internal/agent/usecases` の束縛検査、本人データの HTTP 検査を起点にし、OAuth2 と正式な配線を `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。
