---
status: completed
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
- [x] T001 [SCL] `flows.AdminGroups` の `update_dynamic_rule` / `preview_dynamic_rule` の `does` 説明をビルダー操作の記述に更新した (`just yaml-check` 緑)。
- [x] T002 [UI] `dynamicRuleCel.ts` に属性→演算子絞り込みと CEL 生成の純ロジックを実装。RED: `dynamicRuleCel.test.ts` の `buildDynamicRuleExpression`「string equality expression matching hand-written CEL」を先に fail 確認 (scenario `AdminGroups.update_dynamic_rule`) → GREEN。`DynamicRuleEditor.tsx` に属性/演算子/値の条件行を AND 結合で組み立てる UI を実装。
- [x] T003 [UI] ビルダーが生成した CEL を preview/save/enable フローへ接続し、「詳細設定 (CEL 直接編集)」トグルで生 CEL テキストエリアに切替可能にした。
- [x] T004 [UI] 保存済み rule の式があるグループは詳細設定モードで開くようフォールバック (逆変換は Out of Scope)。RED: `DynamicRuleEditor.test.tsx`「falls back to advanced mode for a saved OR expression」を先に fail 確認 (scenario `AdminGroups.update_dynamic_rule` の T004) → GREEN。
- [x] T005 [Verify] `just yaml-check` (SCL 緑 / 既存の AdminUserEditPage 負債超過のみ残存・本 WI 無関係)、`just verify-ui` 緑、`just test-ui-unit` 421 件緑。

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

## Completion
- **SCL**: `spec/contexts/identity-management.yaml` の `flows.AdminGroups` の
  `update_dynamic_rule` / `preview_dynamic_rule` の `does` をビルダー操作
  (属性・演算子・値を選び AND 結合、または詳細設定で CEL 直接編集) の記述に更新。
- **UI (純ロジック, test-first)**: `frontend/src/features/admin-groups/dynamicRuleCel.ts`
  に組み込み属性 7 種 + テナントのカスタム属性から属性一覧を組み立て、型
  (string / boolean / date) で演算子を絞り込み、条件を AND 結合した CEL 式を生成。
  組み込み属性・参照形式 (`user.<key>`)・date→`timestamp("…T00:00:00Z")` 変換は
  backend `idmanagement/group/domain/dynamic_group_rule.go` に一致。
  `dynamicRuleCel.test.ts` (9 ケース) で手書き CEL との一致・エスケープ・型別演算子を検証。
- **UI (コンポーネント)**: `DynamicRuleEditor.tsx` を新設し、`AdminGroupEditPage.tsx`
  から動的ルール編集の状態一式を移管。ビルダー ⇔ 詳細設定 (生 CEL) をトグルで切替え、
  生成 CEL をプレビュー用に表示。生成した式は既存の `updateDynamicGroupRule` /
  `previewDynamicGroupRule` / `setDynamicGroupRuleEnabled` へそのまま渡す。
  ルート `groups_/$groupId.edit.tsx` で `TenantUserAttributeSchema` をロードして渡す。
  `DynamicRuleEditor.test.tsx` で「式なし→ビルダー」「保存済み OR 式→詳細設定へ
  フォールバック」を検証。
- **副次的整理**: 状態移管により `AdminGroupEditPage` の local-state を 12→5 に削減
  (complexity ratchet 負債 `wi234-ui-page-local-state-admin-group-edit-page` の解消方向)。
- **検証**: `just yaml-check` (SCL 緑)、`just verify-ui` (format/lint/typecheck/build 緑)、
  `just test-ui-unit` (421 件緑) を確認。

### 対応していないこと (Out of Scope)
- CEL コンパイラ・評価器・API の変更は行っていない (バックエンド不変)。
- OR・括弧・関数ネストを含む複雑な式のビルダー対応は未対応。詳細設定 (CEL 直接編集)
  で継続対応する。
- **既存の保存済み CEL 式のビルダー条件への逆変換 (パース) は未実装**。このため
  保存済みルールを開くと必ず詳細設定 (生 CEL) 表示になり、ビルダーは新規作成
  および既存式の作り直し (置き換え) 時に使う。T004 の「AND パターンに一致しない
  式のフォールバック」はこの逆変換禁止方針に包含される形で満たしている。
- 数値 (number) / 文字列配列 (string_array) 型のカスタム属性はビルダーの属性候補
  から除外し、詳細設定での CEL 直接編集に委ねる (Plan が列挙する string / boolean /
  date に限定)。

### 未実施の検証
- ライブスタック上での手動 E2E (管理者ログイン → 動的グループ編集画面) は未実施。
  代替として、WI の手動確認項目が要求する CEL 文字列
  (`user.preferred_username == "alice"`、`… && user.email_verified == true`、
  OR 式のフォールバック) を単体・コンポーネントテストで実行可能な形で検証済み。
