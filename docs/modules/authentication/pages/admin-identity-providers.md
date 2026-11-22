# 外部 IdP の一覧

この文書は、外部 IdP の一覧の操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Authentication](../README.md) / [ページ一覧](README.md) / 外部 IdP の一覧

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [フェデレーション](../../../design/application/pages/admin/federation/README.md)

`admin-identity-providers`　`/admin/identity-providers`

外部認証の接続状態と設定を管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Authentication](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 接続方式、状態と選択対象の概要、接続テスト、有効化、無効化と削除確認 |

利用手順は[Authentication のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| refresh | ready | 再取得または接続テスト | 常時 | IdMagic.Authentication.Operations.ListIdentityProviderConnections、IdMagic.Authentication.Operations.TestIdentityProviderConnection | 接続状態またはテスト結果を表示する | [外部 IdP の一覧](admin-identity-providers.md) / ready | 対象の識別子と操作の入力値 |
| status | ready | 有効化または無効化・削除を確認 | 常時 | IdMagic.Authentication.Operations.ActivateIdentityProviderConnection、IdMagic.Authentication.Operations.DisableIdentityProviderConnection、IdMagic.Authentication.Operations.DeleteIdentityProviderConnection | 対象の状態と一覧を更新する | [外部 IdP の一覧](admin-identity-providers.md) / ready | 対象の識別子と操作の入力値 |
| create | ready | 作成 | 常時 | UI 内の操作 | 移動先を表示する | [外部 IdP を登録](admin-identity-provider-create.md) / ready | なし |
| detail | ready | 詳細 | 常時 | UI 内の操作 | 移動先を表示する | [外部 IdP の詳細](admin-identity-provider-detail.md) / ready | なし |
| edit | ready | 編集 | 常時 | UI 内の操作 | 移動先を表示する | [外部 IdP を編集](admin-identity-provider-edit.md) / ready | なし |

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
| empty | 対象がないことと、作成または検索条件を変える入口を示す。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/authentication/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Authenticationのページ一覧へ戻る](README.md)
