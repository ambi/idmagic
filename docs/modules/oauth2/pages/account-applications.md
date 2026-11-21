# 本人の同意と Agent への委任

この文書は、本人の同意と Agent への委任の操作、入力項目と状態を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2](../README.md) / [ページ一覧](README.md) / 本人の同意と Agent への委任

分類：[アカウントポータル](../../../design/application/pages/account/README.md) / [同意と承認](../../../design/application/pages/account/consent/README.md)

`account-applications`　`/account/applications`

アプリケーションへの同意を取り消し、Agent の委任を管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [OAuth2](../README.md) |
| 利用者 | 認証済みの本人 |
| 入口 | 本人のポータル |
| 利用条件 | 対象テナントの本人のセッションで参照する。CSRF と再認証は共通規則に従う。 |
| 表示内容 | 同意したアプリケーションとスコープ、Agent ごとの委任・期限・利用履歴、委任の追加 |

利用手順は[OAuth2 のページ一覧](README.md)にまとめる。

## フォームと入力項目

項目ごとのバリデーション規則は別の作業で定義する。

### Agent への委任

識別子：account-applications/delegation。表示条件：委任の追加欄。確定：委任。

| 項目 | 識別子 | 入力の種類 | 型 | 初期値 | 表示条件 | データとの対応 |
| --- | --- | --- | --- | --- | --- | --- |
| 委任先 Agent | account-applications/delegation/agentId | select | string | 空 | 対象のフォームを操作するとき | 委任可能 Agent 一覧の agent_id |
| 委任するスコープ | account-applications/delegation/scopes | text | string | 空 | 対象のフォームを操作するとき | 空白で区切って scopes 配列へ変換する |

## パラメーター

ページ固有のパラメーターはない。

## 実装と照合していない入力

次のフォーム部品は静的な `name` 属性を持たないため、上の入力項目との対応を機械的に確かめていない。ページを変更するときは、これらの部品を入力項目と突き合わせる。

| ファイル | 部品の数 |
| --- | --- |
| [`features/account/AgentDelegationsSection.tsx`](../../../../frontend/src/features/account/AgentDelegationsSection.tsx) | 2 |

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| revokeConsent | ready | 同意を取り消す | 取り消し成功 | IdMagic.OAuth2.Operations.RevokeMyConsent | アプリケーションを同意一覧から除く | [本人の同意と Agent への委任](account-applications.md) / ready | 対象の識別子と操作の入力値 |
| grant | ready | Agent に委任 | 付与成功 | IdMagic.OAuth2.Operations.GrantMyAgentDelegation | 委任を一覧へ反映し入力を消す | [本人の同意と Agent への委任](account-applications.md) / ready | 対象の識別子と操作の入力値 |
| revokeDelegation | ready | 委任を取り消す | 取り消し成功 | IdMagic.OAuth2.Operations.RevokeMyAgentDelegation | 委任を一覧から除く | [本人の同意と Agent への委任](account-applications.md) / ready | 対象の識別子と操作の入力値 |
| failure | ready | 更新 | API の拒否または通信失敗 | UI 内の操作 | 問題を表示して未確定の操作を残す | [本人の同意と Agent への委任](account-applications.md) / error | 対象の識別子と操作の入力値 |

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

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/oauth2/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[OAuth2のページ一覧へ戻る](README.md)
