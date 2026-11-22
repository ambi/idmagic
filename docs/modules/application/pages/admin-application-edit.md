# アプリケーションを編集

この文書は、アプリケーションを編集の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Application](../README.md) / [ページ一覧](README.md) / アプリケーションを編集

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [アプリケーション](../../../design/application/pages/admin/applications/README.md)

`admin-application-edit`　`/admin/applications/$applicationId/edit`

アプリケーションの表示情報、接続・認証設定、資格情報、割り当てとカテゴリを管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Application](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 一覧または詳細 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 基本情報とアイコンの追加・削除、方式別の接続設定と SAML 専用 IdP プロファイルの作成、サインイン規則、複数の保存先の検査と失敗時の入力保持、ローテーションできる OIDC シークレットの発行、初回だけの表示と失効確認、ユーザーとグループへの割り当ての追加と解除、カテゴリの作成・削除と割り当て |

利用手順は[Application のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| samlProfile | ready | 専用の IdP プロファイルを作成 | SAML アプリケーション | IdMagic.Saml.Operations.CreateSamlIdentityProviderProfile | 作成したプロファイルを選択肢に加えて選ぶ | [アプリケーションを編集](admin-application-edit.md) / ready | 対象の識別子と操作の入力値 |
| issue | ready | シークレットを発行 | シークレットをローテーションできる | IdMagic.Application.Operations.IssueApplicationClientSecret | 発行したシークレットを一度だけ表示する | [アプリケーションを編集](admin-application-edit.md) / secret | 対象の識別子と操作の入力値 |
| hide | secret | 控えたシークレットを閉じる | 常時 | UI 内の操作 | シークレットの表示を消す | [アプリケーションを編集](admin-application-edit.md) / ready | 対象の識別子と操作の入力値 |
| revoke | ready | 失効を確認 | 常時 | IdMagic.Application.Operations.RevokeApplicationClientSecret | 資格情報の状態を更新する | [アプリケーションを編集](admin-application-edit.md) / ready | 対象の識別子と操作の入力値 |
| assignments | ready | 割り当てを追加または解除 | サービス以外のアプリケーション | IdMagic.Application.Operations.AssignApplication、IdMagic.Application.Operations.UnassignApplication | 割り当てを更新する | [アプリケーションを編集](admin-application-edit.md) / ready | 対象の識別子と操作の入力値 |
| category | ready | カテゴリを管理 | サービス以外のアプリケーション | IdMagic.Application.Operations.CreateApplicationCategory、IdMagic.Application.Operations.DeleteApplicationCategory、IdMagic.Application.Operations.SetApplicationCategories | カテゴリと割り当てを更新する | [アプリケーションを編集](admin-application-edit.md) / ready | 対象の識別子と操作の入力値 |
| save | ready | 保存 | すべての変更の保存が成功 | IdMagic.Application.Operations.UpdateAdminApplication、IdMagic.Application.Operations.UploadApplicationIcon、IdMagic.Application.Operations.DeleteApplicationIcon、IdMagic.Application.Operations.UpdateApplicationOidcConfig、IdMagic.Application.Operations.UpdateApplicationWsFedConfig、IdMagic.Application.Operations.UpdateApplicationSamlConfig、IdMagic.Application.Operations.UpdateAppSignInPolicy | 更新後の詳細へ戻る | [アプリケーション詳細](admin-application-detail.md) / ready | なし |
| failure | ready | 保存 | 一部の保存先が失敗 | UI 内の操作 | 入力を保持して失敗した操作を表示する | [アプリケーションを編集](admin-application-edit.md) / error | 対象の識別子と操作の入力値 |
| cancel | ready | キャンセル | 常時 | UI 内の操作 | 移動先を表示する | [アプリケーション詳細](admin-application-detail.md) / ready | なし |

## 表示状態

| 状態 | 表示 |
| --- | --- |
| ready | 取得した内容を表示する。 |
| loading | ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。 |
| error | 表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。 |
| forbidden | 権限不足を表示し、未反映の状態を保つ。 |
| expired | 認証が必要なページでは再認可し、復帰できなければログインへ移る。 |
| success | 成功した操作を一度通知し、表示内容を再取得する。 |
| invalid | 入力不備の表示は共通フォーム規則に従う。項目ごとの規則は別に定義する。 |
| submitting | 実行中の操作を示し、二重送信を防ぐ。 |
| secret | 新しいシークレットと控えた後に閉じる操作を表示する。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/application/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Applicationのページ一覧へ戻る](README.md)
