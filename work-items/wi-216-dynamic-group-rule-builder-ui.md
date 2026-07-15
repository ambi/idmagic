---
status: pending
authors: [tn]
risk: low
created_at: 2026-07-16
depends_on: []
---

# 動的グループの CEL ルールに属性ベースのビルダー UI を導入する

## Motivation
wi-149 で動的グループの membership ルールとして制限付き CEL を導入したが、UI は
CEL 式を直接テキストで書かせるだけで、wi-149 の Plan にあった「一般操作は
Builder、高度な条件は同じ CEL の直接編集で扱う」の Builder 側は未実装のまま
出荷された。管理者(非エンジニア含む)が「ユーザー名が alice に等しい」のような
単純な条件を組み立てるだけでも CEL の構文・利用可能な属性 key・演算子を把握する
必要があり、誤った式を書いて意図しない所属になるリスクがある。

本 WI は、属性・演算子・値を選んで組み立てる簡易ビルダー UI を追加し、生成した
CEL 式をこれまで通り DynamicGroupRule.expression として既存 API に渡す。CEL の
コンパイル・評価・プレビュー・有効化ロジックは変更しない。

## Scope
- **scl**:
  - `spec/contexts/identity-management.yaml` の `flows.AdminGroups.views.AdminGroups.does`
    のうち `update_dynamic_rule` / `preview_dynamic_rule` の `does` 説明を、
    ビルダー操作(属性・演算子・値を選ぶ)を反映した記述に更新する。
- **ui**:
  - `frontend/src/features/admin-groups/AdminGroupsPage.tsx` の動的ルール編集
    (編集画面、wi-* で detail から edit 専用に切り出し済み)に、条件行(属性 /
    演算子 / 値)を組み立てるビルダーを追加する。
  - 属性一覧は組み込み属性 (`id` / `preferred_username` / `name` /
    `given_name` / `family_name` / `email` / `email_verified`) とテナントの
    `TenantUserAttributeSchema` のカスタム属性から取得し、属性の型に応じて
    選べる演算子を絞り込む。
  - ビルダーが生成した CEL 式を既存の `previewDynamicGroupRule` /
    `updateDynamicGroupRule` にそのまま渡す。

## Out of Scope
- CEL コンパイラ・評価器・API (`backend/identitymanagement/domain/dynamic_group_rule.go` 等) の変更。
- OR・括弧・ネストを含む複雑な式のビルダー対応(詳細設定への CEL 直接編集で継続対応する)。
- 既存の保存済み CEL 式をビルダー条件へ逆変換(パース)すること。

## Plan
- 単一条件は「属性 + 演算子 + 値」の 3 点で組み立てる。演算子は属性型で絞り込む:
  文字列は `==` / `!=` / `contains` / `startsWith` / `endsWith`、真偽値は
  `true` / `false` 判定、日付は `before` / `after` / `==`(`timestamp()` 呼び出しに変換)。
- 複数条件は AND 結合のみ builder でサポートする。OR や関数のネストが必要な式は
  「詳細設定 (CEL を直接編集)」に切り替えて手書きする。
- 保存済み rule の式がビルダーの AND 結合パターンに一致しない場合(OR を含む、
  ビルダーが生成しない関数呼び出しがある等)は、無理にビルダー表示へ変換せず
  自動的に詳細設定(生 CEL テキストエリア)にフォールバックする。
- サーバ側の検証・評価・保存 API (`UpdateDynamicGroupRule` /
  `PreviewDynamicGroupRule` / `EnableDynamicGroupRule` /
  `DisableDynamicGroupRule`) は変更しない。ビルダーが生成した文字列は既存の
  `expression` フィールドとして渡すだけで、CEL 側の制約 (バイト数上限、参照数
  上限、許可された関数一覧) はそのまま適用される。

## Tasks
- [ ] T001 [SCL] `flows.AdminGroups` の `update_dynamic_rule` / `preview_dynamic_rule` の `does` 説明をビルダー操作の記述に更新する。
- [ ] T002 [UI] 属性・演算子・値を選ぶ条件行コンポーネントを実装し、AND 結合で複数条件を組み立てられるようにする。
- [ ] T003 [UI] ビルダーが生成した CEL 式をプレビュー・保存フローに接続し、「詳細設定」トグルで既存の生 CEL テキストエリアに切り替えられるようにする。
- [ ] T004 [UI] 保存済み rule がビルダーの AND 結合パターンで表現できない場合、詳細設定表示に自動フォールバックする。
- [ ] T005 [Verify] SCL・UI 検証、手動シナリオを確認する。

## Verification
- `just yaml-check`
- `just verify-ui`
- 手動: `preferred_username` が `alice` に等しいユーザーだけを対象にする条件をビルダーで組み立て、プレビュー結果が同等の手書き CEL (`user.preferred_username == "alice"`) と一致することを確認する。
- 手動: 複数条件 (例: 部署 = Engineering かつ email_verified = true) を AND で組み立てて保存し、動的所属に反映されることを確認する。
- 手動: OR を含む既存の CEL 式を保存済みのグループを開いたとき、ビルダーではなく詳細設定 (生 CEL) 表示に自動フォールバックすることを確認する。

## Risk Notes
UI のみの変更でバックエンドの評価ロジックには手を入れないため、既存の CEL
コンパイル時検証・プレビュー・fail-closed の安全網 (wi-149) がそのまま働く。
主なリスクはビルダーが意図と異なる CEL を生成することだが、演算子選択を属性の
型で絞り込むこと、保存前に必ずプレビューで差分確認できることで軽減する。
