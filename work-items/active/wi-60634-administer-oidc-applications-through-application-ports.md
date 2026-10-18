---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で Application → OAuth2 を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# OIDC アプリケーションの管理を Application のポートで要求する

## 動機

[依存の向きの決定](wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`Application → OAuth2` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

OIDC と service の作成、表示、設定更新、クレーム、シークレットの発行と失効、Application との関連付け、削除を対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

クライアント設定の公開契約、作成失敗の補償保証、シークレットの上限と有効期間の変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[カタログ](../../docs/modules/application/features/catalog.md)の REQ-APPLICATION-001、002、005、006、007 と、`backend/application/internal/handlers_http/application_provisioning.go`、`client_secret_lifecycle.go`、`backend/application/client_display_names.go` を起点にする。

D2、D4、D6、D7 を適用する。
Application はカタログの編成と管理の要求を担当し、保存型を含まない管理操作と表示値のポートを定義する。
OAuth2 の公開するクライアント管理操作へ組み立て地点で変換し、クライアントの不変条件と更新は OAuth2 に残す。
OAuth2 → Application の割り当てとポリシー評価は残す。

Repository を写す案は Client の保存型と検証を漏らし、管理 API 全体を OAuth2 へ移す案はカタログの編成を分散させるため採らない。
クライアントシークレットの表現の変更は OAuth2 と表示への変換で閉じ、Application が秘密の保存形式を知らない境界を選ぶ。

現行のクライアント作成、Application 作成、関連付けの順と各失敗時の状態を照合する。
すべての失敗に補償があるとは仮定しない。
秘密を一度だけ返す経路とクォータ、イベントの回数も保つ。

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

- `mise run test-go-package -- ./backend/application/internal/handlers_http` の `client_secret_examples_test.go`、`client_secret_refusal_effects_test.go`、`protocol_config_examples_test.go` を起点に、OAuth2 の管理操作と実際の配線を `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。
