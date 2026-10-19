---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: docs
spec_impact: { kind: none, reason: "資格情報操作の分離方針と現行の確定境界を整理する。本番コード、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順は変更しない。" }
---

# IdManagement の資格情報操作の分離方針を再検討する

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)は `IdManagement → Authentication` を除く向きを選んだ。
当初の計画は保存型の複製を退けていたが、要求する削除操作の粒度と、パスワードの検証、履歴、本人確認の具体的な契約は未確定だった。
認証方式の追加で両側の依存と変換を増やさず、既存の確定順を保てる分離を、責務の移設も含めて比較する。
この項目は境界の再検討として完了し、実装は具体化した後続項目へ渡す。

## 対象範囲

資格情報操作、パスワードの規則と履歴、本人確認を、それぞれの状態所有者、確定順と変更理由から比較する。
要求側へ Repository 群を複製する案、所有者の操作へ集約する案、管理の責務を別モジュールへ分ける案を比較する。
現在状態の文書に残る完全削除のトランザクションの誤記を直し、実装する項目へ具体的な契約と検証境界を渡す。

## 対象外

本番コード、テスト、TypeSpec、責務表、公開パッケージ、テーブル所有者の変更。
循環負債の削減と依存図の再生成、HTTP 経路の移設、認証方式の追加、削除の原子性の変更。
使用量の再集計は [wi-83071](wi-83071-make-user-deletion-rerunnable-and-recount-quota-usage.md) の対象を維持する。
イベントの発行後に進捗の保存が失敗する窓も、本項目では是正しない。

## 設計

### 入力と維持する保証

次の仕様と実装を境界の比較の入力にする。
本項目では既存テストの表明を読んで検証の入口を選ぶが、テストを実行済みとは扱わない。

| 保証 | 一次情報と実装の入口 |
| --- | --- |
| 無効化と本人以外の管理 | [REQ-IDMANAGEMENT-010、046](../../docs/modules/identity-management/features/user.lifecycle.md#無効化と再有効化)、`SetUserDisabled`、`revokeTrustedDevicesOnDisable` |
| 完全削除と再実行 | [REQ-IDMANAGEMENT-050](../../docs/modules/identity-management/features/user.lifecycle.md#完全削除)、`DeleteUser`、`finishPurge`、`cascadeDeleteForSub`、[既知の欠陥](../../docs/modules/identity-management/risks.md) |
| CSV の行の不可分な確定 | [REQ-IDMANAGEMENT-004、058](../../docs/modules/identity-management/features/user-csv.md#インポート)、`ApplyUserImport`、メモリと PostgreSQL の `UserImportRowCommitter` |
| パスワードの規則と初回の履歴 | [パスワードのモデル](../../docs/modules/authentication/features/password.md#モデル)、`CreateUser`、`createUser`、`PasswordPolicyError` |
| 本人のプロフィールとメール変更 | [REQ-IDMANAGEMENT-019、053](../../docs/modules/identity-management/features/account.md#操作)、`requireAuthenticatedSub`、`requireStepUpSub`、`writeAccountError` |
| 信頼済み端末の失効と再認証 | [REQ-AUTHENTICATION-028、029](../../docs/modules/authentication/features/trusted-device.md#操作)、`UserDeviceRevoker`、`StepUpSatisfied` |

HTTP の入口は `spec/modules/identity-management/main.tsp` の `CreateAdminUser`、`DeleteAdminUser`、`UpdateUserProfile`、`RequestEmailChange` に照合する。
本人確認が返す `session_not_found` は同じ TypeSpec に含まれるため、IdManagement 内で直接生成していないことだけを理由に変換を消さない。

### import と作用の区別

逆向きの import は、IdManagement のルートの結線、HTTP の依存、User の作成と CSV、二つの CSV の確定アダプター、管理と本人の HTTP にある。
Authentication からの主体の照会と更新は、パスワード変更、MFA、WebAuthn、復旧、連携ログインなどが利用する。
その側の User モデルと Repository を置き換える案は、本項目が扱う資格情報操作だけでは閉じない。

| 操作 | 現在の順序と確定点 | 分離後も維持する境界 |
| --- | --- | --- |
| 管理者による作成 | 名前の空判定、テナント文脈からのポリシー解決と検証、ロール検証、ハッシュ生成、`createUser` の保存と履歴、イベントと通知 | 検証とハッシュ生成を一操作へまとめてロール拒否より前にハッシュを作らない。ポリシーの解決と違反コードの担当は Authentication に残す |
| CSV の作成 | 無作為な初期パスワードをハッシュ化し、User、必須操作、履歴、使用量、監査を行の mutation へまとめる | PostgreSQL は同じ `pgx.Tx` に束縛した履歴 writer を使う。履歴を独立した公開ユースケースで後から保存しない |
| 無効化 | User の保存、動的 Group の同期、端末の失効とそのイベント、UserDisabled、通知、所有 Agent の停止 | 端末ごとのイベントを返す既存操作を使い、IdManagement が既存の発行先へ送る。失効操作の内部で別の発行先へ送らない |
| 完全削除 | 所有 Agent の停止、Tombstone と PendingPurge の保存、同意、refresh、セッション、パスワード履歴、TOTP、信頼済み端末、WebAuthn、復旧コード、device code、承認要求の削除、使用量、UserDeleted と通知 | 連続する認証情報の削除だけを所有者の一操作へ集約する。セッションは前、OAuth2 の後半の削除は後に保つ。保存先ごとの確定と失敗後の再実行を維持する |
| 本人確認 | 変更時のブラウザー要求の検証、認証済みで pending でない本人の確定、必要な操作だけ再認証の鮮度判定 | 本人の sub を URL や本文から受け取らない。再認証の時間窓、エラーとその HTTP 変換を維持する |

完全削除では `PendingPurge` は資格情報ごとの進捗を持たない。
削除群の途中で失敗すると、再実行はその群の先頭から冪等な削除を繰り返す。
一つの公開操作にまとめることは、一つのトランザクションへ変えることを意味しない。
使用量の減算と進捗の保存、イベントの発行と進捗の保存も別々であり、既知の失敗窓を解決済みとは扱わない。

### 候補の比較

D1、D2、D3、D4、D6、D7、D8 を適用する。
暗号の方式、認証情報の保存形式、ポリシー解決と再認証の判定を所有者へ隠し、User のライフサイクルの編成を IdManagement に残す。

| 候補 | 制約と変更シナリオへの適合 | 判断 |
| --- | --- | --- |
| 現状維持 | 現在の保証は保てるが、認証情報の種類を追加すると所有者、IdManagement の削除列、依存のフィールドと配線が増える | 不採用。循環と保存先の知識が残る |
| Repository の写しと相互変換 | import は切れるが、IdManagement が使わない `Recent`、資格情報の保存型、列挙まで同期することになる。D2、D6 が求める必要最小限の操作にならない | この形では実装しない |
| 所有者の削除操作と細い要求契約 | 認証方式の追加を Authentication の削除操作へ閉じられる。履歴の保存形式の変更は所有者へ閉じ、CSV は書き込みのシグネチャだけに依存する。D4 の結線と D7 の行の確定を両立する | 採用。要求の意味が同じ操作は構造的に適合させ、値の変換が必要な場所だけ変換する |
| User の管理を別モジュールへ分離 | Authentication が利用する主体の照会と管理の編成を分けられるが、共通の User 作成、動的 Group、CSV、Agent、SCIM、ワークフローと保持期間の入口まで再配置する。資格情報操作の保存判断を隠すためだけには不要な境界が増える | 今回は採用しない。管理機能を独立して変更する要求が生じた場合に D8 で再比較する |
| Authentication 側の User 利用をポートで置換 | 資格情報操作を使う側は単純になるが、認証と更新に使う User の状態とエラーを多数の機能で写す必要がある | 対象外。波及が今回の辺の解消を超える |

後続の [wi-55979](wi-55979-decouple-identity-credential-operations-with-owner-commands.md) へ、Repository の写しを作らずに実装する方針を渡す。
暗号方式や削除手段を追加する変更では所有者だけを変え、既存の保証を新たに追加する変更ではその保証に必要な要求契約とテストも変える。
将来どの変更でも利用側が無変更になるとは主張しない。

### 文書の修正範囲

[設計ガイドライン](../../docs/design/application/design-guidelines.md#一貫性の境界と永続化)の「User の削除が一つのトランザクション」という例は、現行の[ユーザー設計](../../docs/modules/identity-management/features/user.md#信頼性)と実装に反している。
この例を現状に合わせ、CSV の行の原子性とアクショントークンの原子性を、それぞれ別の保証として残す。
wi-83071 からはこの文書修正だけを取り出し、使用量の補正の実装を完了扱いにしない。

## 計画

今回は文書だけを変更して境界の再検討を完了する。
実装、テストの配線の変更、負債の削減は wi-55979 に分離し、実装前に解決する問いを同項目へ具体化する。
本項目の完了は循環の解消を意味しない。

## タスク

- [x] T001 [Design] import の入口、更新元、確定順と再実行を照合し、D1〜D8 に基づき候補と変更シナリオを比較する。
- [x] T002 [Docs] 現行の確定境界の誤記を直し、実装する契約と検証を後続項目へ渡す。
- [x] T003 [Verify] 文書の形式、参照、境界と規範差分を検査する。

## 検証

- `mise run check-work-items`、`mise run check-links`：再検討と実装の項目の形式、依存と参照を検査する。
- `mise run check-spec`、`mise run check-terminology`、`mise run check-work-item-references`：設計の更新が文書の形式と用語に適合し、現在状態の文書が作業記録へ依存しないことを検査する。
- `mise run check-boundaries`：現行の負債を測り、文書の修正を実装の完了と混同しない。
- `mise run render-docs` と `mise run check-published-api-compat`：設計の表示と公開契約のゲートを確認する。
- `mise run spec-diff e6a511e69`：規範仕様を変えていないことを確認して要約へ反映する。

差分をすべて Markdown に限定する文書変更なので、RED、故障注入、Go とブラウザーの実行は本項目では行わない。
後続の実装の保証は、同項目で選ぶ実行の証拠で判断する。

## リスク

境界の計画を具体化しただけで、必須の作用、循環、失敗窓を解決済みと誤認すること。
現在状態の文書に未実装と残存する依存を明記し、負債台帳と生成した依存図は維持する。

## 要約

IdManagement と Authentication の循環の分離方針を再検討し、資格情報の削除は所有者の一操作へ集約し、パスワードの検証、ハッシュ、CSV の履歴と本人確認は必要最小限の要求契約で結ぶ方針を具体化した。
Repository の写し、管理モジュールの分離、認証側の主体照会の置換と現状維持を比較し、認証方式の追加が IdManagement の削除列と依存フィールドへ波及しない案を選んだ。
パスワードとロールの拒否順、端末失効のイベントの発行先、CSV の同じ Tx への参加、完全削除の途中確定と再実行を維持する境界として記した。

完全削除を一つのトランザクションとする設計ガイドラインの誤記を直し、既存の CSV とアクショントークンの原子性とは区別した。
使用量の再集計は wi-83071 に残し、循環を解消する実装は wi-55979 へ具体的な契約、正式な配線と検証の計画を渡した。
本番コード、テストと TypeSpec は変更せず、循環負債は17件のままである。
規範仕様の差分はない。

文書の形式、参照、用語、境界、文書サイトの生成と公開 API の互換性ゲートを確認した。
公開前のため公開済み契約との比較は行われていない。
今回の再検討に未実施事項はなく、コードと PostgreSQL、ブラウザーの実行は後続の実装の検証に残す。
