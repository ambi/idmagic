---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-08-15
priority: p1
depends_on: [wi-53-rebac-fine-grained-authorization]
change_kind: feature
affected_spec:
  - { path: spec/contexts/authorization/models.tsp, symbol: IdMagic.Contract.FgaCheckRequest }
  - { path: spec/contexts/authorization/SPECIFICATION.md, requirement: REQ-AUTHORIZATION-001 }
  - { path: spec/contexts/authorization/SPECIFICATION.md, requirement: REQ-AUTHORIZATION-005 }
  - { path: spec/contexts/system/SPECIFICATION.md, requirement: REQ-SYSTEM-010 }
---

# 関係ベース細粒度認可 (ReBAC) の管理 UI を管理コンソールへ載せる

## Motivation
[[wi-53-rebac-fine-grained-authorization]] が `Authorization` Context と `/api/admin/v1/authorization/*` の 6 本を通したが、管理 UI は意図的に範囲外とした。そのためテナント管理者が ReBAC を運用する手段は HTTP を直接叩くことしかなく、実務では 3 点が詰まる。

**認可モデルを JSON で書き下ろすしかない。** 書き換え規則 3 種の組み合わせ制約は多い。`direct` は `computed_relation` を持てず、`tuple_to_userset` は tupleset 対象に素のオブジェクト型を要求し、`computed_userset` の連なりは循環できない。誤りは 422 の英語 1 行としてしか返らず、どの型のどの関係が問題なのかを本文から読み取ることになる。

**拒否の理由を追えない。** ReBAC の運用コストの大半はここにある。`CheckAccess` は `relation_path` と `reasons` を返すが、`relationship_permits_actor_chain` や `subject_form_not_declared` といった規則名を人が読んで意味を補う必要がある。「なぜこのエージェントはこの文書を読めないのか」に答えられなければ、細粒度認可は運用されずに放置される。

**関係タプルが目に見えない。** 一覧 API はあるが、型と関係の語彙は登録済みモデルの中にあり、両者を突き合わせながら手でクエリを組み立てることになる。

本 WI は、モデルの発行・タプルの編集・判定の説明までを管理コンソールに載せ、ReBAC を API の知識なしに運用できる状態にする。

## Scope
- **仕様**: TypeSpec の `FgaCheckRequest` と `ListAccessibleResourcesHttpRequest` に `granted_scopes` / `required_scopes` を追加し、契約を実装に合わせる。
- **API クライアント**: `frontend/src/api/admin.ts` に 6 操作のクライアント、`frontend/src/types.ts` に対応する型。
- **純関数**: 認可モデルのフォーム状態と API の `resource_types` を相互変換する関数、およびクライアント側検証。単体テスト付き。
- **画面**: 認可モデル (詳細・編集)、関係タプル (一覧・追加)、アクセス判定デバッガ。
- **配線**: ルート、`adminNav`、ページタイトル、i18n 辞書、判定理由の翻訳辞書。

## Out of Scope
- backend の API 変更。`granted_scopes` / `required_scopes` の TypeSpec 追記は既存実装に契約を合わせるだけで、ハンドラーの振る舞いは変えない。
- 関係タプル一覧のカーソルページネーション。追加するなら backend 側の変更であり、[[wi-159-admin-resource-cursor-pagination]] が確立した署名済みキーセットカーソルと `Link` ヘッダーの契約に合わせる別 WI とする。ここで UI 独自のページ送りを作らない。本 WI は絞り込み前提で割り切る。
- タプルの CSV 一括投入と JSON 差分の貼り付け。移行や大量投入は API 直呼びで足りる。
- 認可モデルの図としての可視化 (関係グラフのダイアグラム)。
- `ApiTokenScope` への `authorization-model:*` の追加。管理 API のスコープ配線は [[wi-320-agent-management-api-scope-wiring]] が扱う。

## Design

### 画面とルート

| ルート | 役割 |
| --- | --- |
| `/admin/authorization/model` | 現在の版の読み取り専用ビュー。リソース型と関係をツリーで示し、書き換え規則を可読な形へ展開する。`?version=` で過去の版も引く。 |
| `/admin/authorization/model/edit` | 発行フォーム。フォームビルダーと JSON の 2 モード。 |
| `/admin/authorization/relation-tuples` | 絞り込み検索付きの一覧。行内に削除。オブジェクト単位の波及削除もここから。 |
| `/admin/authorization/relation-tuples/new` | タプル 1 件の追加フォーム。 |
| `/admin/authorization/check` | アクセス判定デバッガ。`CheckAccess` と `ListAccessibleResources`。 |

feature は `frontend/src/features/admin-authorization/` にまとめる。既存の `admin-authz-detail-types` は RFC 9396 の authorization details 型を扱う別機能なので、名前が紛らわしくないよう UI 上の見出しでも「細粒度認可 (ReBAC)」と「OAuth2 認可詳細」を明確に分ける。

作成・編集を専用ルートに置き、一覧の操作を行内に出し、破壊的操作を確認ダイアログにするのは `spec/contexts/system/SPECIFICATION.md` の Admin Console Policy と UI navigation and consistency policy に従う。

### 認可モデルのフォーム ⇄ JSON

フォーム状態 (`ModelDraft` → `ResourceTypeDraft` → `RelationDraft` → `RewriteDraft`) と API の `resource_types` を、`toModelDraft()` / `toResourceTypes()` の純関数で相互変換する。`admin-groups/dynamicRuleCel.ts` と同じく feature 内の独立ファイルへ切り出し、単体テストの対象にする。

`DynamicRuleEditor` は生の CEL からビルダーへの逆変換を諦めているが、あれは CEL が自由文法だからである。認可モデルは閉じた構造化データなので双方向に変換でき、モードを往復しても情報を失わない。往復で等価になることをテストで固定する。JSON モードの内容が壊れている間はフォームへ戻せないので、切り替え時にパースし、失敗したらモードを維持してエラーを出す。

`direct_subject_types` は `user` (個別) / `group#member` (subject set) / `user:*` (ワイルドカード) の 3 形を取る。フォームでは「型を選ぶ」「型と関係を選ぶ」「型を選んで全員を許す」の 3 通りの追加操作として見せ、文字列の組み立ては変換関数に閉じ込める。

**クライアント検証はフォームで防げる範囲に限る。** 名前の書式、型名と関係名の重複、`rewrites` が空、`direct` の主体型が空、`computed_userset` の自己参照、宣言されていない型・関係の参照までをフォーム側で拾う。`computed_userset` の連なりの循環検出はサーバーに任せ、422 をそのまま見せる。深さ優先探索をクライアントへ二重実装すると、規則が増えたとき片方だけが古くなる。フェイルクローズの正本は 1 つに保つ。

**発行前に差分を見せる。** モデルの版は追記のみで過去の版を壊さないが、発行した瞬間から判定は新しい版で行われる。編集画面は発行ボタンの手前で、追加・削除される型と関係、および主体形の変更を差分として提示する。既存タプルがモデルの縮小によって判定に数えられなくなる (`subject_form_not_declared`) のはこの操作で起きるので、そこを黙って通さない。

### 判定デバッガ

`CheckAccess` と `ListAccessibleResources` は同じ画面に置く。入力 (主体・リソース型・関係・代行チェーン・スコープ) の大半が共通で、「この主体はこれを読めるか」と「この主体は何を読めるか」は同じ問いの両面だからである。

`relation_path` は `document#viewer` → `folder#viewer` という関係名の連なりなので、経路として順に並べる。識別子を含まない仕様なので、そのまま表示して情報漏洩にならない。

`reasons` の規則名は人間可読な説明へ翻訳する。対象は AuthZEN 側の `relationship_facts_present` / `relationship_permits_subject` / `relationship_permits_actor_chain` / `actor_chain_principals_active` / `scope_subset_of_client_scope` / `actor_and_resource_share_tenant` と、ドメイン側の `no_relation_path` / `unknown_relation` / `evaluation_depth_exceeded` / `relation_cycle_detected` / `subject_form_not_declared`。辞書は feature の `*.i18n.ts` に置く。`lib/i18n/errorMessage.ts` は HTTP のエラーコード用であり、判定理由はそれとは別の語彙である。**未知の理由コードはそのまま表示する。** backend が規則を増やしたときに UI が黙って理由を隠すほうが、英語の識別子が出るより悪い。

### 整合トークン

書き込み応答の `consistency` は画面の状態として保持し、直後の判定へ `minimum_consistency` として自動で渡す。不透明値なので通常は利用者へ見せない。`consistency_not_satisfied` が返ったときだけ「直前の書き込みがまだ反映されていません」と説明する。デバッガには契約を確かめるための任意入力欄を置く。

### 関係タプル一覧の規模

backend の一覧は `limit` だけを取り、上限 1000 で `Link` ヘッダーを返さないので、`lib/usePaginatedList.ts` は使えない。本 WI は絞り込み前提とし、**上限に達したことを画面に明示する**。消したはずのタプルが一覧に出ないまま「無い」と誤解されるほうが、件数の表示より危険だからである。関係タプルは一覧を眺めるデータではなく特定の主体・リソースについて引くデータであり、主な導線は判定デバッガになる。

### 却下した案

- **モーダルでのモデル編集**: Admin Console Policy が主要リソースの作成・編集に専用ルートを要求している。ディープリンクとブラウザーの戻る操作を壊さない。
- **JSON エディタのみ**: 実装は最も軽いが、書き換え規則の組み合わせ制約をすべて 422 に委ねることになる。フォームなら選択肢の段階で大半を潰せる。
- **フォームのみ**: 他環境からのモデル持ち込みと、大きなモデルの一括編集ができなくなる。
- **クライアント側での完全な検証**: 循環検出まで写すと、フェイルクローズの判断の正本が 2 か所になる。

## Plan
1. TypeSpec を先に直し、`just check-spec` を通す。
2. API クライアントと型を足す。
3. モデルの変換と検証を純関数として実装し、往復テストを付ける。
4. 画面を 3 系統・5 ルート実装する。コンテナと表示用コンポーネントを分け、抽出した単位に単体テストを付ける。
5. ルート、ナビ、ページタイトル、辞書を登録する。
6. `just verify-ui` と `just verify` を通し、一連の流れを手動で確認する。

## Tasks
- [ ] T001 [Spec] `FgaCheckRequest` と `ListAccessibleResourcesHttpRequest` に `granted_scopes` / `required_scopes` を追加し、`just check-spec` と `just check-api-compat` を通す。
- [ ] T002 [API] `frontend/src/types.ts` と `frontend/src/api/admin.ts` に 6 操作のクライアントと型を追加する。テスト: `frontend/src/api/admin.test.ts`。
- [ ] T003 [Pure] `toModelDraft()` / `toResourceTypes()` とクライアント検証を純関数として実装する。テスト: `authorizationModel.test.ts` の往復等価と検証境界 (REQ-AUTHORIZATION-001)。
- [ ] T004 [UI] 認可モデルの詳細ページと編集ページ (フォーム / JSON の 2 モード、発行前の差分提示) を実装する。テスト: `AdminAuthorizationModelPage.test.tsx` / `AdminAuthorizationModelEditPage.test.tsx`。
- [ ] T005 [UI] 関係タプルの一覧・追加フォーム・削除・オブジェクト波及削除の確認ダイアログを実装する。テスト: `AdminAuthorizationTuplesPage.test.tsx` (REQ-AUTHORIZATION-002, REQ-AUTHORIZATION-008)。
- [ ] T006 [UI] アクセス判定デバッガを実装し、`relation_path` の経路表示と `reasons` の翻訳、未知コードの素通しを検証する。テスト: `AdminAuthorizationCheckPage.test.tsx` (REQ-AUTHORIZATION-004, REQ-AUTHORIZATION-005)。
- [ ] T007 [Nav] ルート、`adminNav`、`shell.i18n`、ページタイトル、辞書を登録する。テスト: `adminNav.test.ts`。
- [ ] T008 [Verify] `just verify-ui` と `just verify` を通し、手動の一連の流れを確認する。

## Verification
- `just check-spec`
  - reason: TypeSpec の追記が契約として通ること。
- `just check-api-compat`
  - reason: 任意フィールドの追加がリリース済みワイヤ契約を壊さないこと。
- `just verify-ui`
- `just verify`
- 手動: モデルを発行 → タプルを 1 件追加 → 判定デバッガで許可と `relation_path` を確認 → 代行チェーンにエージェントを足して拒否と理由を確認 → オブジェクト波及削除 → 再判定で拒否になることを確認する。
- E2E は追加しない。`frontend/tests/e2e/README.md` の方針どおり、破壊的な CRUD の E2E はブラウザー操作自体にリスクがある場合に限る。判定の境界は Go 側のテストが既に持っている。

## Risk Notes
認可の中枢を操作する画面なので、UI の誤操作がアクセス範囲を広げうる。モデルの発行は過去の版を壊さないが、発行した瞬間から判定は新しい版で行われるため、編集画面は発行前に追加・削除される型と関係の差分を提示する。モデルの縮小で既存タプルが判定に数えられなくなる経路 (`subject_form_not_declared`) を黙って通さない。

クライアント側の検証とサーバー側の検証がずれると「フォームでは通るのに 422」が起きる。変換と検証を純関数へ切り出して単体テストで固定し、循環検出はサーバーへ一本化してロジックの正本を二重化しない。

タプル一覧が上限で黙って切れると、消したはずのタプルが見えないまま「無い」と誤解される。上限到達は必ず画面に出す。
