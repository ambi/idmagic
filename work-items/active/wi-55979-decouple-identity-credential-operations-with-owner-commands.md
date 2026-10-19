---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-20
priority: p3
depends_on: [wi-18494-isolate-identity-credential-operations-behind-consumer-ports]
change_kind: refactor
spec_impact: { kind: none, reason: "資格情報の削除、失効、パスワードの検証と履歴、本人確認を所有者へ接続する。HTTP の応答、認証と認可の判定、永続状態、イベント、既存の確定順と再実行を維持する。" }
---

# 所有者の操作で IdManagement と Authentication の循環を解消する

## 動機

[分離方針の再検討](wi-18494-isolate-identity-credential-operations-behind-consumer-ports.md)は、Repository 群の複製を避け、所有者へ資格情報の削除を集約する方針を選んだ。
循環の原因は削除だけではなく、パスワードの規則、CSV の履歴、HTTP の本人確認にもある。
この項目は `IdManagement → Authentication` を生む本番 import をすべて取り除く。

## 対象範囲

IdManagement のルートと HTTP の依存、User の作成、CSV のハッシュ化と履歴、無効化と完全削除、本人確認とエラーの変換。
Authentication の所有者の操作、必要最小限の要求契約、組み立て地点の結線、公開範囲と設計、実測で消えた負債の削除と依存図の生成。

## 対象外

User のライフサイクル、管理 HTTP、CSV の別モジュールへの移設。
パスワードの規則と形式、削除の原子性、使用量の再集計、認証方式の追加と他の組の循環の解消。

## 設計

### 保証と境界

[ユーザーのライフサイクル](../../docs/modules/identity-management/features/user.lifecycle.md)の REQ-IDMANAGEMENT-010、046、050、[User CSV](../../docs/modules/identity-management/features/user-csv.md)の REQ-IDMANAGEMENT-004、058、[アカウント](../../docs/modules/identity-management/features/account.md)の REQ-IDMANAGEMENT-019、053 を起点にする。
パスワードの意味は [Authentication のパスワード](../../docs/modules/authentication/features/password.md)、端末の失効と再認証は [REQ-AUTHENTICATION-028、029](../../docs/modules/authentication/features/trusted-device.md#操作)、HTTP の応答は `spec/modules/identity-management/main.tsp` の対応操作に照合する。
D1、D2、D3、D4、D6、D7、D8 を適用する。
所有者への集約、Repository の写し、管理モジュールの分離、逆向きの主体照会の置換、現状維持の比較と確定順は親項目の設計にある。

### 要求する操作

名称は実装時に確定するが、必要な情報と粒度は次のとおりとする。
同じ意味のメソッドは Go の構造的適合を使い、委譲だけの変換層を作らない。

| 契約 | シグネチャの形と担当 |
| --- | --- |
| 資格情報の削除 | `PurgeUserCredentials(ctx context.Context, sub string) error`。Authentication が履歴、TOTP、信頼済み端末、WebAuthn、復旧コードを既存順に消し、最初のエラーで止まる。IdManagement はセッション削除の後、device code 削除の前に呼ぶ |
| 初回のパスワードの検証 | `ValidateInitialPassword(ctx context.Context, password string) error`。Authentication が現在と同じテナント文脈からポリシーを解決する。ロール拒否より前の判定順を保ち、変更時だけの履歴照合や新しい辞書検査を加えない |
| ハッシュの生成 | `Hash(password string) (string, error)` だけを要求し、暗号方式と検証の実装は Authentication に残す。ロール拒否より前にハッシュを作らない |
| 履歴の書き込み | `Add(ctx context.Context, sub, encoded string, now time.Time) error`。`Recent`、履歴の保存型と削除を要求しない。通常の作成と CSV で同じ最小の writer の形を使う |
| Tx の履歴 writer | `func(pgx.Tx) <要求側の履歴writer>`。関数の戻り値の型は自動で適合しないため、既存の `authentication.PasswordHistoryWriterInTx` の結果を組み立て地点の関数で返す |
| 無効化での端末失効 | 既存の `RevokeAllForUser(context.Context, string, string, spec.TrustedDeviceRevokeReason, time.Time) ([]spec.DomainEvent, error)` と同じ形を要求する。保存モデルを写さず、イベントは既存の呼び出し側の発行先へ返す |
| 本人確認と再認証 | 既存の HTTP 認証の結果と本人の sub を使い、再認証の判定を Authentication の既存実装へ結ぶ。時間窓を複製しない |
| エラーの表示 | ポリシー違反コードだけを取得できるエラーの形などで HTTP へ写す。`password_policy` のコード順と応答、`step_up_required`、`session_not_found` を保ち、列挙やエラー構造体の写しを作らない |

User の保存、UserMutationCommitter、動的 Group、Agent、PendingPurge、クォータ、User のイベントと通知の編成は IdManagement に残す。
所有者の削除を一操作にまとめても保存先ごとに確定し、失敗後は同じ削除群を繰り返す。
認証情報の種類を追加するときに IdManagement の削除列と依存フィールドを増やさないことを確認する。

### 構築と経路

管理 HTTP の `server_http/routes.go` と `bootstrap.UserLifecycleDeps` の双方で同じ所有者の操作を渡す。
後者は SCIM、LifecycleWorkflow、Batch の保持期間の掃除が使う。
CSV の PostgreSQL の配線は `bootstrap/postgres.go` の `PostgresImportDeps.HistoryInTx` を変換し、同じ Tx を渡す。
必須の操作が nil なら成功として省く経路を増やさず、構築時に不足を拒否する。
機能選択で省略できる依存と、部分的なテストの fixture の都合を分ける。
HTTP の入力と認可、暗号方式、保存データは変えず、ロールバックにデータの変換を要しない。

## 計画

着手時に import、構築経路と保証の被覆を再取得し、エラー、本人確認と必須依存の最終シグネチャを実装前に記す。
保証の被覆が欠けていれば、公開状態に応じた要件または特性化テストを先に追加する。
テストの配線を変える必要があるため、テスト無変更の構造変更の例外を適用せず、実際の差分で RED と変更耐性の手順を選ぶ。

## タスク

- [ ] T001 [Design] 全入口、保証の被覆と構築条件を確認し、残るシグネチャを確定する。
- [ ] T002 [Test] 被覆の不足と正式な配線の故障を検出する証拠を用意する。
- [ ] T003 [App] 所有者の削除操作、要求契約と結線を実装する。
- [ ] T004 [Docs] 公開範囲と設計を同期し、実測で消えた違反だけを台帳から除き、依存図を生成する。
- [ ] T005 [Verify] HTTP、途中失敗と再実行、拒否後の無作用、Tx と正式な経路、規範差分を確認する。

## 検証

| 境界 | 既存の検査の入口と故障 |
| --- | --- |
| User の操作 | `TestDeleteUserAnonymizesAndCascades`、`TestCreateUpdateAndDisableUser` と再実行の既存検査。削除の漏れ、他の sub の削除、失敗の握りつぶし、発行順の変化 |
| 管理 HTTP | `TestAdminUserAPIPurgeDestroysWebAuthnCredentialsAndRecoveryCodes`。正式な配線の脱落を対象と対照の利用者の保存状態で検出する |
| CSV | `TestApplyUserImportCreateIncludesCredentialHistoryRequiredActionAndQuotaInOneCommit`、PostgreSQL の `TestUserImportRollsBackWhenHistoryOrAuditFails`。別 Tx の履歴保存、失敗した行の部分保存 |
| 本人確認 | プロフィールとメール変更の HTTP の既存検査。pending、本人以外、再認証の期限切れ、エラー変換と拒否後の無作用 |
| 共通の構築 | 管理 HTTP と SCIM、LifecycleWorkflow、保持期間の掃除が通る構築部品。必須の配線の省略を型検査が拒否できなければ、その経路の最小の実行境界で検出する |

局所の実行は `mise run test-go-test` と `mise run test-go-package`、逆依存と race は `mise run test-go-changed` を使う。
変更した Go の変異は `mise run test-go-mutation`、ツールで表せない結線と Tx の切断だけは手動注入を選ぶ。
PostgreSQL の共有契約は実 DB で実行し、スキップを成功の証拠にしない。
`lint-go`、`check-boundaries`、`check-boundary-debt-ratchet`、HTTP の契約と応答の検査、`check-work-items`、`check-links`、`spec-diff` を実際の差分から選ぶ。

## リスク

一つの削除操作を原子的な確定と誤認すること、所有者がイベントを発行して既存の発行先を迂回すること、CSV の履歴を別の Tx で書くこと。
enum を文字列へ写しただけで意味の依存が消えたと扱わず、判定の担当、公開する知識と変更箇所をレビューする。
