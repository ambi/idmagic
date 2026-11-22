# IdManagement のページ

この文書は、IdManagement が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[IdManagement のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 主な利用手順

枠は利用者に見えるページや表示、矢印は次へ進む操作を示す。破線は戻り道、枠の破線はシステム外の表示である。

### ユーザーを作成して編集する

ユーザーを作成し、プロフィールとロールを確認して変更する。

```mermaid
%%{init: {"flowchart": {"padding": 12, "rankSpacing": 32, "nodeSpacing": 36}, "themeVariables": {"fontSize": "14px"}}}%%
flowchart LR
  %% ui-module-flow
  accTitle: ユーザーを作成して編集する
  accDescr: ユーザーを作成し、プロフィールとロールを確認して変更する。
  view0["<b>ユーザー一覧</b><br/><span class='flow-description'>ユーザー名・状態・ロール<br/>検索と絞り込み</span>"]:::uiFlowView
  view1["<b>ユーザーを追加</b><br/><span class='flow-description'>プロフィール<br/>初期パスワード・ロール</span>"]:::uiFlowView
  view2["<b>ユーザー詳細</b><br/><span class='flow-description'>プロフィール・ロール<br/>所属・認証状態</span>"]:::uiFlowView
  view3["<b>ユーザーを編集</b><br/><span class='flow-description'>プロフィール・ロール<br/>変更内容</span>"]:::uiFlowView
  view0 -->|"追加"| view1
  view1 -->|"作成"| view2
  view0 -->|"ユーザーを選択"| view2
  view2 -->|"編集"| view3
  view3 -->|"保存"| view2
```

- 既存のユーザーは一覧から詳細へ進み、同じ編集の手順を使う。
- 入力不備、権限不足、保存の失敗は各ページの操作表に示す。

ページの詳細：[ユーザー一覧](admin-users.md)、[ユーザー詳細](admin-user-detail.md)、[ユーザーを追加](admin-user-create.md)、[ユーザーを編集](admin-user-edit.md)

### CSV を確認して取り込む

取り込み前に内容を確認し、確認後にだけ変更を適用する。

```mermaid
%%{init: {"flowchart": {"padding": 12, "rankSpacing": 32, "nodeSpacing": 36}, "themeVariables": {"fontSize": "14px"}}}%%
flowchart LR
  %% ui-module-flow
  accTitle: CSV を確認して取り込む
  accDescr: 取り込み前に内容を確認し、確認後にだけ変更を適用する。
  view0["<b>ファイルの選択</b><br/><span class='flow-description'>CSV ファイル<br/>テンプレートの取得</span>"]:::uiFlowView
  view1["<b>内容のプレビュー</b><br/><span class='flow-description'>適用前の変更内容<br/>行ごとの問題</span>"]:::uiFlowView
  view2["<b>適用の確認</b><br/><span class='flow-description'>対象・影響の確認<br/>確認ダイアログ</span>"]:::uiFlowView
  view3["<b>取り込み結果</b><br/><span class='flow-description'>成功と失敗の件数<br/>失敗した行と項目</span>"]:::uiFlowView
  view0 -->|"プレビュー"| view1
  view1 -->|"適用"| view2
  view2 -->|"確認して実行"| view3
  view2 -.->|"キャンセル"| view1
```

- 四つの表示は同じ CSV インポートページ内で切り替わる。
- 結果では成功と失敗の件数、失敗した行と項目を確認できる。

ページの詳細：[ユーザーの CSV インポート](admin-user-import.md)

## Agent 管理

| ページ | URL | 目的 |
| --- | --- | --- |
| [Agent 一覧](admin-agents.md) | `/admin/agents` | Agent を選び、状態と責任者を確認する |
| [Agent の棚卸し](admin-agent-inventory.md) | `/admin/agents/inventory` | 責任者、資格情報、利用状態の問題を見つける |
| [Agent を登録](admin-agent-create.md) | `/admin/agents/new` | 責任者と初期の資格情報・アクセスを指定する |
| [Agent 詳細](admin-agent-detail.md) | `/admin/agents/$agentId` | Agent の責任者、状態と利用を管理する |
| [Agent を編集](admin-agent-edit.md) | `/admin/agents/$agentId/edit` | 表示情報と責任者を更新する |
| [Agent の資格情報](admin-agent-credentials.md) | `/admin/agents/$agentId/credentials` | 認証に使う資格情報を作成または関連付ける |
| [Agent のリソース](admin-agent-resources.md) | `/admin/agents/$agentId/resources` | リソースへの割り当てを確認する |
| [Agent のリソースを編集](admin-agent-resources-edit.md) | `/admin/agents/$agentId/resources/edit` | 対象リソースと関係の割り当てを変更する |
| [Agent のアクセス概要](admin-agent-access-summary.md) | `/admin/agents/$agentId/access` | Agent に関連する認証、委任、リソースへの権限を確認する |

## ユーザー管理

| ページ | URL | 目的 |
| --- | --- | --- |
| [ユーザーの CSV エクスポート](admin-user-export.md) | `/admin/users/exports` | 出力ジョブを開始し、完了した CSV を取得する |
| [ユーザー一覧](admin-users.md) | `/admin/users` | ユーザーを検索し、対象の確認と管理の入口を示す |
| [ユーザー詳細](admin-user-detail.md) | `/admin/users/$id` | 現在のプロフィール、権限、所属と認証の状態を確認する |
| [ユーザーを追加](admin-user-create.md) | `/admin/users/new` | 初期資格情報とプロフィールを指定してユーザーを作成する |
| [ユーザーを編集](admin-user-edit.md) | `/admin/users/$id/edit` | プロフィール、ロールとカスタム属性を一回の保存で変更する |
| [ユーザーの CSV インポート](admin-user-import.md) | `/admin/users/import` | CSV をプレビューしてから確認を経て適用し、行ごとの結果を確認する |

## グループ管理

| ページ | URL | 目的 |
| --- | --- | --- |
| [グループ一覧](admin-groups.md) | `/admin/groups` | グループを選び、ロールと所属人数を確認する |
| [グループ詳細](admin-group-detail.md) | `/admin/groups/$groupId` | グループの属性とメンバーを確認する |
| [グループを作成](admin-group-create.md) | `/admin/groups/new` | グループの基本情報とカスタム属性を登録する |
| [グループを編集](admin-group-edit.md) | `/admin/groups/$groupId/edit` | 属性、ロール、所属と動的ルールを管理する |
| [グループの CSV インポート](admin-group-import.md) | `/admin/groups/import` | CSV の内容を確認してグループを取り込む |
| [グループメンバーの CSV インポート](admin-group-member-import.md) | `/admin/groups/$groupId/members/import` | 指定グループの所属を CSV で追加または除外する |
| [グループの CSV エクスポート](admin-group-export.md) | `/admin/groups/exports` | グループ情報を CSV で取得する |
| [グループメンバーの CSV エクスポート](admin-group-members-export.md) | `/admin/groups/$groupId/members/exports` | 指定グループの所属を CSV で取得する |

## プロフィール

| ページ | URL | 目的 |
| --- | --- | --- |
| [本人のプロフィール](account-profile.md) | `/account/profile` | 本人の基本情報と公開された属性を確認する |
| [本人のプロフィールを編集](account-profile-edit.md) | `/account/profile/edit` | 本人に編集が許可された基本情報と属性を更新する |
| [本人のメールアドレス](account-emails.md) | `/account/emails` | 現在の確認状態を確認し、新しいアドレスの確認を依頼する |
| [メールアドレスの確認](email-verify.md) | `/account/email/verify` | メールに含まれるトークンで変更を確認する |
| [本人のデータ](account-data.md) | `/account/data` | 本人のデータを JSON として取得する |
