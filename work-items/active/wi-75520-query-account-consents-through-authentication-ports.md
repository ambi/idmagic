---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で Authentication → OAuth2 を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# 本人の同意管理を Authentication のポートで要求する

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`Authentication → OAuth2` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

本人の同意一覧、表示値、取り消し、対象がない場合のエラー変換と組み立てを対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

同意取り消しのクォータ修正、account HTTP 経路の移設、認証再開の変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

`backend/authentication/internal/handlers_http/account_consents_handler.go`、`internal/deps_http/deps.go`、`wiring.go` と、[同意](../../docs/modules/oauth2/features/consent.md)を起点にする。
既存の `backend/authentication/internal/handlers_http/account_handlers_test.go` は本人の同意一覧と取り消しを検査する。

D2、D4、D6 を適用する。
Authentication に本人の同意操作と表示値を定義し、組み立て地点の変換を通じて OAuth2 の ConsentManager に接続する。
同意の状態、失効、イベントと永続化は OAuth2 が担当し、本人の sub の確定、ブラウザー要求の検証と account HTTP 経路は Authentication に残す。
OAuth2 → Authentication の認証操作の依存は残す。

同意の Repository を Authentication に写す案は失効とイベントを省ける経路を作る。
HTTP の担当を OAuth2 へ移す案はルーターと本人認証の責務まで動かすため採らない。
同意の保存形式を変更するときは OAuth2 と表示値への変換で閉じる。
認証再開の移設は既に完了しており、この辺の原因は同意管理である。

対象 sub は本人認証の結果に固定し、URL からは client_id だけを受け取る。
同意の取り消しで関連する失効が走ること、not found と no-store の応答を保つ。
[wi-23517](wi-23517-release-the-consent-quota-when-the-user-revokes-a-consent.md) のクォータ修正とは分ける。

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

- `mise run test-go-package -- ./backend/authentication/internal/handlers_http` の `TestHandleListAccountConsentsReturnsGrantedConsentsOnly`、`TestHandleRevokeAccountConsentRequiresBrowserVerification`、`TestHandleRevokeAccountConsentSucceeds` と `account_consent_refusal_effects_test.go` を起点にし、OAuth2 の同意操作と正式な配線は `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。
