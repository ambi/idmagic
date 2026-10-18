---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p2
depends_on: [wi-28013-separate-application-catalog-from-access-policy]
change_kind: refactor
spec_impact: { kind: none, reason: "OIDC 管理の重複する入力型と委譲を除去し、HTTP の応答、シークレットの公開範囲、クォータ、永続状態、イベントと部分確定の順序を維持する。" }
---

# OIDC カタログ管理の重複する契約と変換を除去する

## 動機

[wi-60634](../done/wi-60634-administer-oidc-applications-through-application-ports.md) は Application から OAuth2 の保存知識を除いたが、9操作の要求契約、同じ内容の入力と結果、178行の変換層を追加した。
作成、更新とシークレットの操作の多くは、既存の公開管理操作への項目ごとの委譲である。
Catalog の分離後は OAuth2 への依存を自然に持てるため、この間接化の必要性を再検討する。

## 対象範囲

Catalog の OIDC の作成、設定とクレームの更新、関連付け、削除、シークレットの発行、ローテーションと失効、設定表示。
Catalog の要求型と `backend/shared/http/server_http/applicationdeps/oidc_clients.go` の管理用の変換を対象にする。

## 対象外

Application が使う小さな `ClientNames` の読み取り契約。
OAuth2 のプロトコル、クライアントの不変条件、HTTP 契約と原子性の変更。

## 設計

D1、D2、D3、D4、D6、D7、D8 に従い、[カタログ](../../docs/modules/application/features/catalog.md)と OAuth2 の公開管理操作を照合する。
第一候補は Catalog が OAuth2 の `ClientAdministration` とその入力、結果、エラーを直接利用し、ワイヤ変換を HTTP の近くへ集める案とする。
所有者の検証と更新を Catalog へ戻す案は採らない。
表示はシークレットのハッシュを公開しない専用の射影を維持し、その担当が Catalog か OAuth2 かを実際の利用箇所と変更シナリオで比べる。
同じ値を往復させるだけの型、エラーと変換は除去し、フィールド追加の際に同期する宣言と実装を実例で比較する。
作成と関連付け、通常設定とクレームの別確定、イベントと削除の順序を保つ。

## 計画

分離後の Catalog の入口、OAuth2 の管理操作と表示の利用箇所を読み、直接利用案と必要最小限の射影案を比較してから実装する。

## タスク

- [ ] T001 [Design] 必要な知識、入力と結果の重複、変更の波及を比較する。
- [ ] T002 [App] 所有者の公開管理操作を利用し、不要な型と委譲を除去する。
- [ ] T003 [Verify] 既存の HTTP、シークレットの非公開、拒否後の状態と確定順を検査する。
- [ ] T004 [Docs] 現在の分担と公開範囲を同期する。

## 検証

Catalog の既存 HTTP テストと OAuth2 の管理操作のテストを起点とし、`test-go-changed`、`lint-go`、`check`、`spec-diff` を使う。
型変換の除去による値の脱落、クォータと失効の省略、シークレットの再表示を独立した故障として検査する。
着手時に実際の差分に応じた RED と変更耐性の手順を確定する。

## リスク

委譲を除去する際に所有者の検証を飛ばしたり、表示用の結果へハッシュを混ぜたりすること。
削減する行数だけで採否を決めず、変更の局所性と公開する内部知識を比較する。
