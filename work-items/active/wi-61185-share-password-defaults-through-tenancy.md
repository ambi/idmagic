---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p2
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: パスワードの既定値の定義場所だけを変える。認証時の長さ・履歴・期限の判定、設定 API の応答、上書きの拒否、永続状態とイベントを維持する。
---

# パスワードの既定値をテナント設定と同じ公開パッケージから共有する

## 動機

`tools/check/boundary-debt.json` の `module-cycle:Tenancy->Authentication` は、Tenancy の管理ハンドラー2ファイルがパスワードの既定値4個を参照するために生じる。
Authentication はテナントの上書きを解決するために Tenancy に依存する。
既定値の取得だけのために、設定を管理する側から認証の実装へ戻る参照をなくす。
この項目は wi-21934 の再計測から切り出した、単独で取り込める構造変更である。

## 対象範囲

- 既定の最小長、最大長、履歴件数、有効期限を `backend/tenancy/domain` に一度だけ定義する。
- Authentication の既存の定数名は参照によって保ち、Tenancy の管理 API は自モジュールの定数を使う。
- 解消した負債だけを除き、依存図と担当モジュールの設計を同期する。

## 対象外

- 既定値、上書きの許容範囲、パスワード検証規則の変更。
- 属性スキーマ、User の操作、プロトコルの編成が生むほかの循環の解消。
- テストの期待結果と構築方法の変更。

## 設計

入力は REQ-TENANCY-019、REQ-AUTHENTICATION-010、REQ-AUTHENTICATION-024 と、`PasswordPolicyOverride`、`DefaultPasswordPolicySnapshot`、`tenantPolicyFloor` である。
変更は定数の参照先だけであり、SQL、トランザクション、通知、イベントの呼び出し順には及ばない。
定数はテナントの設定の既定値として公開し、適用後の検証、期限と履歴の判断は Authentication に残す。

| 案 | D1・D4・D5 との適合 | 既定値を変える場合の波及 | 採否 |
| --- | --- | --- | --- |
| 現状を維持する | 既定値を一度だけ定義できるが、設定とその利用側の循環が残る | Authentication と設定の表示を確認する | 不採用 |
| Tenancy の既存の domain に定数を置き、Authentication の定数を参照にする | 設定とその上書きの型が同じ値を持つ。既存の Authentication → Tenancy の向きを保つ | 定義は一か所。既存の認証と管理 API のテストを共に実行する | 採用 |
| 呼び出し元が既定値を注入する | 循環は切れるが、固定値4個に省略・誤配線の余地と構築の契約を増やす | 本番とテストの組み立てすべてを変える | 不採用 |
| 共有ライブラリへ移す | パスワードの業務上の既定値は D5 の技術的な共有の条件に当たらない | 業務の値を共有基盤が所有する | 不採用 |

新しい公開パッケージやモジュールは作らず、テーブルの所有者も変えない。
残る仮定は、パスワードの既定値が既存要件どおり製品共通の固定値であることである。
実行時に差し替える設定を導入するときは、その所有者と構築契約を別途判断する。

## 計画

1. 管理 API と認証ポリシーの既存テストを変えず、定数の定義と参照だけを移す。
2. 境界検査が不要になった負債を指摘することを確認し、該当する1件だけを除く。
3. 設計と生成される依存図を更新し、影響するパッケージと逆依存を検査する。

## タスク

- [x] T001 [Design] 境界の比較と仕様影響を確定する。
- [x] T002 [App] 既定値を集め、既存テストを保って参照を移す。
- [x] T003 [Docs] 負債台帳、依存図と現在の設計を同期する。
- [x] T004 [Verify] 変更の範囲に合う検査を通す。

## 検証

- `mise run check-work-items` と `mise run check-links` で記録と設計の参照を検査する。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet -- main` で1件の解消と新しい違反がないことを検査する。
- `mise run test-go-changed -- main` で変更していない認証、設定 API と逆依存のテストを race 付きで実行する。
- `mise run lint-go` で Go の静的検査を実行する。
- `mise run spec-diff` と `mise run check-spec-impact -- main` で規範の変更がないことを確かめる。
- テストを変更しない構造変更として、振る舞いを保つ変更の証拠を使う。

## リスク

既定値を複製すると、設定画面の表示と認証時の判定がずれる。
数値リテラルを Tenancy の一か所だけに残し、Authentication の既存の名前は定数参照にする。

## 要約

`spec-diff` は規範仕様の差分なしを示した。
パスワードの既定値4個を Tenancy の domain に集め、Authentication の既存の定数名をその参照として保った。
認証の判定、設定 API の応答と上書きの拒否、永続状態とイベントは変えていない。
Tenancy から Authentication への参照をなくし、循環依存の負債を53件から52件に減らした。
固定値のための注入ポートと新しい共有ライブラリは増やさず、既存テストも変更していない。
ほかの循環と属性スキーマの所有は対象外であり、wi-21934 に残る。
