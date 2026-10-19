---
status: in_progress
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

[分離方針の再検討](../done/wi-18494-isolate-identity-credential-operations-behind-consumer-ports.md)は、Repository 群の複製を避け、所有者へ資格情報の削除を集約する方針を選んだ。
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

### 着手時の確定

基準 revision は `aa41edb6a` とする。
IdManagement の公開済み `user/ports` に `UserCredentialPurger`、`PasswordHasher`（Hash のみ）、`PasswordHistoryWriter`、`UserDeviceRevoker` と、違反コードを返すエラーの契約を置く。
資格情報の削除は `authentication.Module.UserCredentials()` が構築する所有者の操作の `PurgeUserCredentials(context.Context, string) error` が担当し、既存の保存先の nil 条件と順序をそのまま移す。
IdManagement がこの操作自体を省略する経路は作らず、未配線はエラーにする。
部分的な fixture は、持っている保存先を明示した Authentication の Module を渡す。
本番の Module は既存の NewMemoryModule と NewPostgresModule が資格情報の保存先を構築する。

ポリシーの検証は `ValidateInitialPassword(context.Context, string) error` を関数として必須で受け取り、Authentication の既存のポリシー計算へ接続する。
HTTP の本人確認は要求側の `AccountIdentity`（`RequireStepUpSub(*echo.Context) (string, error)` と `WriteAuthenticationError(*echo.Context, error) (bool, error)`）へ結ぶ。
Authentication の既存の本人確認とエラー変換を再利用し、IdManagement へ sentinel と鮮度の計算を複製しない。
再認証の操作は既存の IdManagement と同じく、共通の認証器が返す pending を文脈への射影より前に判定する。
所有者の別の本人確認ヘルパーは先に文脈へ射影するため、その単純な委譲では任意の認証結果の pending が落ちることを RED で確認し、元の順序を所有者へ移した。
ポリシーのエラーは `PasswordPolicyViolations() []string` の構造的な契約で表示する。

Tx の履歴 writer は、既存のルートの `PasswordHistoryWriterInTx` の返り値を要求側の writer に狭めて直接適合させる。
同関数は組み立て地点だけが使い、Authentication のルートは既に User のポートへ依存しているため、委譲だけの変換関数と公開パッケージを増やさずに同じ Tx を保てる。

既存の管理、削除、CSV と本人確認の表明を維持して配線だけを更新する。
新しい削除操作の順序、途中失敗と再実行、未配線、検証の拒否順を追加の検査で固定する。
既存の HTTP の未解析操作の照合では、構成入口の変更に影響する282操作のすべてにテストの参照があり、参照の不足はない。
局所検査の後に `test-go-changed` で逆依存を検査する。
このタスクは契約証拠を集める full モードではないため、未解析操作の実行証拠の収集には `test-go-race` も使う。
変更した純粋な分岐の変異は基準 revision との差分へ絞り、資格情報の削除と正式な配線の切断は手動注入で検出する。

着手時に import、構築経路と保証の被覆を再取得し、エラー、本人確認と必須依存の最終シグネチャを実装前に記す。
保証の被覆が欠けていれば、公開状態に応じた要件または特性化テストを先に追加する。
テストの配線を変える必要があるため、テスト無変更の構造変更の例外を適用せず、実際の差分で RED と変更耐性の手順を選ぶ。

## タスク

- [x] T001 [Design] 全入口、保証の被覆と構築条件を確認し、残るシグネチャを確定する。
- [x] T002 [Test] 所有者の削除の7検査を RED から GREEN にし、未配線の削除が匿名化してしまう失敗を固定して修正した。パスワード、ロールとハッシュの順序、本人認証の拒否と構築時の不足を別に検査する。
- [x] T003 [App] 所有者の削除操作、要求契約と結線を実装する。既存の User の139検査、管理 HTTP の35検査と、パスワード、ワークフロー、プロビジョニングの既存検査を維持する。
- [x] T004 [Docs] 公開範囲と設計を同期し、消えた `IdManagement → Authentication` の1件だけを台帳から除き、依存図を生成する。残存する循環負債は16件である。
- [x] T005 [Verify] HTTP、途中失敗と再実行、拒否後の無作用、Tx と正式な経路、規範差分を確認する。生存変異の観測を補って13件を再検査し、配線と表示の14故障も検出する。

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

最終検査は、DB 必須モードの `test-go-changed -- aa41edb6a` と、比較基準 `VERIFICATION_BASE_REVISION=aa41edb6a` を渡す `test-go-race`、`mise run lint-go` と `mise run check aa41edb6a` を使う。
HTTP の表現、入力と応答、拒否後の状態と正式な配線は、変更していない契約テストと、所有者の違反コードを返す追加テストで観測する。
UI、ブラウザーの操作と HTTP の経路は変えず、この構造変更の検証はバックエンドの契約と配線の境界で行う。

### 変更耐性

変更行を対象とした自動変異は、Authentication の共通操作、初回パスワード、HTTP の依存、ポリシーのエラー、ルートと、IdManagement の User の操作、HTTP、ルートで調べる。
User の削除エラーの握りつぶしと、本人確認の正常経路を常に拒否する変異が生き残ったため、途中失敗と再実行、成功、未認証、期限切れと取得失敗の観測を補って再検査する。
自動変異が生成されない呼び出しの配線、表示コードと鮮度の判定は、次の14件を一時的に注入して検査し、一件ごとに元のソースへ戻す。

- 所有者のパスワード履歴、TOTP、信頼済み端末、WebAuthn、復旧コードの配線をそれぞれ外す5件。
- 初回パスワードの判定の逆転と、表示する違反コードの誤りの2件。
- 未認証、再認証とセッション不在の応答をそれぞれ200にする3件と、鮮度の判定を逆転する1件。
- 管理 HTTP と保持期間の掃除からの削除操作をそれぞれ外す2件。
- CSV の履歴 writer に渡す Tx を失わせる1件。

ソースのビルド失敗は検出の証拠にせず、テストが実行されて失敗することを確認する。
自動変異は構文の範囲だけを扱い、生成されなかった箇所の保証を意味しない。

## リスク

一つの削除操作を原子的な確定と誤認すること、所有者がイベントを発行して既存の発行先を迂回すること、CSV の履歴を別の Tx で書くこと。
enum を文字列へ写しただけで意味の依存が消えたと扱わず、判定の担当、公開する知識と変更箇所をレビューする。

## 要約

IdManagement から Authentication への本番 import をすべて除去し、循環負債を17件から16件に減らした。
資格情報の削除は Authentication がパスワード履歴、TOTP、信頼済み端末、WebAuthn、復旧コードを既存順に消す一操作へ集約した。
IdManagement は User のライフサイクル、前後の削除、PendingPurge、使用量、イベントと通知の編成を維持する。
認証情報の種類を増やしても IdManagement の削除列と保存先の依存を増やさない構成にした。

初回パスワードの検証、ハッシュ化、履歴、端末失効と本人確認は、必要最小限の要求契約と所有者の操作を組み立て地点で結ぶ。
保存型と enum の写し、委譲だけの変換層、新しい公開パッケージは追加していない。
ポリシーとロールの拒否順、ハッシュを作る順、端末のイベントの発行先、pending を文脈へ射影する前の拒否、再認証の鮮度と HTTP のエラーを維持した。
通常の作成と CSV の履歴は書き込みだけの契約へ狭め、CSV の履歴には同じ Tx を渡す。
完全削除を一括の Tx に変えず、途中確定と失敗後の再実行を維持した。

管理 HTTP と管理 API 外の配線を同期し、未配線の操作を黙って省かず、経路の構築時と実行前に拒否する。
既存テストの期待値を保って配線を更新し、削除の順序とエラー伝播、再実行、拒否後の無作用、違反コードと本人確認の境界を検査に加えた。
七つの視点で、担当、公開する知識、変更の局所性、必須の配線と不要な変換の有無をレビューした。
現在状態の文書と依存図を同期し、計測で消えた負債の1件だけを台帳から除いた。

最終の全 Go の race 検査は6,550件成功、失敗0件、未実行0件で、41か所の DB 検証が成功した。
スキップ3件は、変更していない `FuzzClientIPAllowed` が不正な CIDR の2 seed を除外した記録と親の記録であり、DB と今回の境界のスキップはない。
HTTP 契約の参照を持つ190テストも同じ実行で成功した。
静的解析には未解析の HTTP 操作が残るため、すべての操作を静的に検証済みとは扱わない。
変更行への自動変異13件と、配線、判定、表示、Tx の手動故障14件をすべて検出し、最後の自動変異に生存、未被覆、コンパイル不能はなかった。
生成された変異が0件の箇所は、手動の故障の観測と変更しない既存テストで補った。

`lint-go`、`check aa41edb6a`、境界と負債の検査、文書生成が成功し、`spec-diff aa41edb6a` は規範仕様の変更なしだった。
公開前なので公開済み API との互換性比較は行っていない。
本項目に未実施事項はなく、他の循環の解消、使用量の再集計と削除の原子性の変更は対象外のままである。
