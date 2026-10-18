---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で OAuth2 → ApiTokens を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# 組み込みリソースの公開情報と失効の識別情報を OAuth2 へ渡す

## 動機

[依存の向きの決定](wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`OAuth2 → ApiTokens` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

組み込みリソースの公開スコープと、失効要求の組み込み client_id の例外を、OAuth2 の正式な構築入力へ置き換える。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

スコープ語彙の共有への移設、外部設定による認証例外の追加、署名と検証の担当変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

`backend/oauth2/internal/token/usecases/protected_resource_metadata.go` と `internal/handlers_http/token_handler.go:handleRevoke`、[API トークン](../../docs/modules/api-tokens/README.md)、[OAuth2](../../docs/modules/oauth2/README.md)を起点にする。

D1、D4、D5、D6 を適用する。
業務のスコープ語彙と組み込みクライアント識別子は ApiTokens が引き続き担当する。
OAuth2 が必要とするリソース情報と識別情報の入力を定義し、正式な組み立て地点から渡す。
ApiTokens → OAuth2 の署名と検証の依存を残す。

全スコープを shared に移す案は D5 に反し、同じ識別子を OAuth2 にハードコードする案は管理 API の知識を複製する。
組み込みリソースの設定を一般の外部入力で上書きできる案も採らない。
管理スコープを追加する変更は ApiTokens の宣言と配線から導き、OAuth2 のスコープ一覧を手で更新しない。

空の resource で返すメタデータと、登録済みリソースの照会を分ける。
失効の例外は既存の client_id 一致、Authorization ヘッダーなし、client_secret なしの条件に限定し、一般のクライアント認証を弱めない。
必須の構築入力を API、worker、batch、テストの各組み立てで照合する。

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

- `mise run test-go-package -- ./backend/oauth2/internal/token/usecases` の `protected_resource_metadata_test.go` と、HTTP の組み込み client の失効と通常 client の認証検査を起点にする。
  正式な組み立てから公開スコープが届くことを `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。
