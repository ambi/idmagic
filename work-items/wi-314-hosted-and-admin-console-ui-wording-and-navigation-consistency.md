---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-01
depends_on: []
---

# ホスト認証画面と管理コンソールの文言・レイアウト・参照編集分離を実機レビューの指摘に沿って改善する

## Motivation

トップページ・ログイン・同意・管理コンソール各画面を実際に触った上でのユーザーレビューにより、
以下の3系統の問題が多数見つかった。

1. **一般ユーザー・管理者に意味の伝わらない文言**（開発者向け説明文がそのままエンドユーザーに
   露出している、英語と日本語が混在している、冗長・不統一な表現など）。
2. **`frontend/ARCHITECTURE.md` の「UI navigation and consistency policy」
   （ADR-086, [[wi-126-admin-and-account-ui-consistency-and-navigation-policy]]、
   [[wi-268-admin-ui-improvements]]、[[wi-309-external-identity-provider-admin-ui-consistency]]
   で継続整備）に違反する箇所の残存** — 一覧/詳細画面（参照専用）から直接編集できてしまう、
   専用ルートではなくモーダルやインラインフォームで作成・編集させている、ボタン表記や
   アイコン運用が画面ごとに不統一など。
3. **同意拒否フローの実害あるバグ** — 「許可しない」を押すと汎用エラーが足されるだけで
   画面から離脱できず、しかも「許可して続行」ボタンが再度押せる状態のまま残ってしまう。

事前に対象画面のコードを調査し、指摘が現状のコードと一致することを確認済み（file mapping は
下記 Design 節に記載）。個々の指摘は軽微だが、合わせるとホーム画面・認証フローという
最初にユーザーが触れる導線の信頼感、および管理コンソールの「参照と編集を混同しない」という
既定方針の一貫性に直結するため、まとめて1つの work item として解消する。

なお本文中で「バックエンド側の変更が必要」と個別に指摘された2件は、本 WI のスコープから
明示的に外し、別 WI として切り出した:
[[wi-315-group-contact-and-custom-attributes]]（グループへのメールアドレス・カスタム属性追加）、
[[wi-316-oidc-client-secret-lifecycle-management]]（OIDCクライアントシークレットの複数保持・
有効期限管理）。

## Scope

- `frontend/src/features/auth-flow/`（`HomePage.tsx`, `LoginPage.tsx`, `ConsentPage.tsx` と
  各 `*.i18n.ts`）
- `frontend/src/features/admin-dashboard/`（`AdminDashboardPage.tsx` と i18n）
- `frontend/src/features/admin-users/`（`AdminUsersListPage.tsx`, `AdminUserDetailPage.tsx`,
  `AdminUserEditPage.tsx`, `AdminUsersShared.tsx` と i18n）
- `frontend/src/features/admin-groups/`（`AdminGroupsListPage.tsx`, `AdminGroupDetailPage.tsx`,
  `AdminGroupDetailCard.tsx` と i18n）
- `frontend/src/features/admin-agents/`（`AdminAgentsListPage.tsx`, `AgentEditorDialog.tsx`,
  `AdminAgentDetailCard.tsx`, `AdminAgentDetailPage.tsx` と i18n）
- `frontend/src/features/admin-lifecycle-workflows/`（`AdminLifecycleWorkflowsPage.tsx` と i18n）
- `frontend/src/features/admin-applications/`（一覧・詳細・編集・SAMLフォーム・
  プロビジョニング各コンポーネントと i18n 一式）
- `frontend/src/features/admin-authz-detail-types/`（`AdminAuthorizationDetailTypesPage.tsx`）
- `frontend/src/features/admin-mcp-resource-servers/`（`AdminMcpResourceServersPage.tsx`）
- `frontend/src/features/admin-audit-events/`（`AdminAuditEventsPage.tsx`）
- `frontend/src/features/admin-keys/`（`AdminKeysPage.tsx`）
- `frontend/src/features/admin-tenants/`（`AdminTenantAttributesPage.tsx`）
- `frontend/src/routes/admin/`配下、上記機能に対応する新規ルートファイル
  （`applications_/$applicationId.new.tsx` 等、モーダル撤去に伴い追加するもの）
- `frontend/ARCHITECTURE.md`（新たに確定した規約があれば追記）

## Out of Scope

- [[wi-315-group-contact-and-custom-attributes]] にて切り出した、グループの属性拡張
  （メールアドレス・カスタム属性）とそれに伴うバックエンド変更。
- [[wi-316-oidc-client-secret-lifecycle-management]] にて切り出した、クライアントシークレットの
  複数保持・有効期限管理とそれに伴うバックエンド変更。
- プロビジョニング属性マッピングを JSON テキストエリアから GUI ビルダーへ置き換える設計そのもの
  （クレームマッピング UI の再利用可否を含む）は、バックエンドのマッピング表現・SCIM 連携仕様
  への影響有無を要調査のため、本 WI では「JSON 直書きは一般ユーザーに厳しい」という課題認識と
  日本語化のみ扱い、GUI 化の設計・実装は別途調査した上で後続 WI に切り出す。
- 署名鍵画面が JWKS（ID Token/Access Token）のみを扱い、SAML/WS-Federation 用の署名鍵管理画面が
  存在しないという指摘は、調査の結果 IdMagic 自身が発行する SAML/WS-Fed アサーションの署名鍵を
  管理する画面自体が存在しないギャップと判明した。管理画面新設はバックエンドの鍵管理設計にも
  関わるため、本 WI では現状の説明文言の是正（「JWKS」に限定した説明であることを明示する）
  に留め、鍵管理画面の新設は別 WI 化を検討する。
- 管理コンソール全体のデザインシステム刷新。

## Design

### 系統1: エンドユーザー向け文言の平易化

- **トップページ** (`HomePage.tsx` / `InformationalPages.i18n.ts`)。「IdMagic は起動しています」
  「ログイン画面は...」「/login を直接開くことはできません」は運用者・開発者向けの説明であり、
  一般ユーザーが迷い込んだ際に意味を持たない。トップページの役割（このサービスが何なのか、
  ここから何もできないこと）を一般ユーザー向けに書き直す。
- **ログインページ** (`LoginPage.tsx` / `LoginPage.i18n.ts`)。見出し「アカウントにログイン」→
  「ログイン」等へ簡潔化（一般ユーザーにアカウント以外へのログインという選択肢はない）。
  ボタン「ログインして続行」→「ログイン」（「ログイン」と「続行」は同義で冗長）。
- **同意ページ** (`ConsentPage.tsx` / `ConsentPage.i18n.ts`)。「許可は組織のポリシーに従って
  保存され、後から管理者またはアプリ側で取り消せます」は一般ユーザーに意味を持たない実装都合の
  説明であるため削除するか、「後から取り消せます」程度の平易な一文に置き換える。
  ボタン「許可して続行」→「許可」（ログインページと同じ理由）。
- **管理コンソール各画面の見出し説明文**。ユーザー一覧の
  「組織のID、アクセスロール、アカウント状態を一元管理します。」（`AdminUsersPage.i18n.ts`
  `pageDescription`）は「ID」がアカウントを指す独自表現になっており、かつロールは
  ロール専用画面がある以上「一元管理」とは言えない。アプリケーション一覧の
  「OIDC・WS-Federation・Web リンク・サービス (M2M) を 1 か所で登録し、利用者への割り当てを
  管理します」は SAML の記載が漏れており、かつ「アプリケーション」という主語が説明文から
  消えている。グループ一覧の「複数のロールをまとめ、所属ユーザーに一括で付与します。」は
  他画面（一覧に何が表示されるかを説明する形式）と方向性が異なり、グループの主な用途
  （ユーザーの集合を扱うこと）ではなくロール一括付与という副次機能を主役にしている。
  これらは各画面の役割を正しく言い当てる説明文に個別に書き直す。
- **「起動 URL」**（`AdminApplicationsPage.i18n.ts` `launchUrlFieldLabel`）。プログラム起動を
  連想させるため「アプリケーション URL」等へ変更する。
- **アプリケーション一覧・エージェント一覧のリソース間で表現が食い違う箇所**。
  グループ詳細のメンバー選択プレースホルダは「ユーザーを選択…」
  (`AdminGroupDetailCard.tsx`)である一方、ユーザー詳細のグループ選択は「グループを選択して
  追加...」であり、体言止め+末尾の違い（「...」の数、「追加」の有無）が生じている。
  文言テンプレートを統一する。
- **エージェント削除ボタン**「エージェントを削除」。他リソースの削除ボタンは目的語を含まない
  短い表記（wi-268 T003 で統一済み）のため、「削除」に揃える。
- **アイコン画像アップロード欄**（`AdminApplicationEditPage.tsx` の素の
  `<Input type="file">`）が英語のブラウザデフォルト文言 "Choose File No file chosen" を
  そのまま表示している。カスタムのファイル選択 UI（ボタン + 選択済みファイル名表示）に
  置き換え、日本語ラベルを付ける。
- **SAML NameID 形式**（`nameIdFormatPersistent` / `nameIdFormatUnspecified` /
  `nameIdFormatEmail`）。選択肢のラベルに SAML 正式表現
  （`urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified` 等）を併記する
  （例:「Unspecified（未指定・urn:oasis:...:unspecified）」）。
- **`signResponseLabel`** の丸括弧注記「(Okta / Entra の "Sign Response")」を削除する。
- **クレームマッピング表の列見出し**（`claimTypeFieldLabel: 'claim 名'` /
  `claimRuleSourceFieldLabel: 'source'` / `claimRuleSourceKeyFieldLabel: '属性'` /
  `claimRuleRequiredLabel: '必須 (required)'`）を完全に日本語化する
  （例:「クレーム名」「取得元」「属性」「必須」）。英語併記の丸括弧も削除。
  `claimMappingRulesHelp`（「required にすると、値が解決できないとき fail-closed で
  sign-in / 発行自体を拒否します」）を英語混じりでない平易な日本語
  （例:「必須にすると、値を取得できない場合はサインインまたは発行そのものを拒否します。」）
  に書き換える。
- **プロビジョニング設定の用語**。「グループ連携」と「グループを下流へプッシュする」の違いが
  分からないという指摘は、実装を確認した上でこの2つが本当に別概念か（別画面/別トグルか、
  同じ設定の見出しと説明文が重複しているだけか）を`AdminApplicationProvisioningSettings.tsx`
  で切り分け、別概念でなければ表記を一本化する。「ユーザー停止」
  (`deactivateUsersLabel`) は無効化を指すなら「ユーザーの無効化」に統一する。
  「409 競合時の照合属性」と補足説明「externalId が未設定の下流リソースと突合する属性パス」
  (`conflictMatchAttributeFieldLabel`/Help) は、両者の言葉遣いを揃えた上で
  「新規作成時に、externalId が未登録の既存リソースと同一人物/同一グループとみなして
  紐付けるために比較する属性」のように一般ユーザーが読んで動作を推測できる説明に書き直す。
  「認証情報を更新する」(`rotateCredentialToggle`) は具体的に何を再送信/再取得するのかが
  分かる文言に変える。属性マッピングの補足説明文の未翻訳部分を日本語化する。
- **監査イベントの種別表示**（`AdminAuditEventsPage.tsx` は `e.type` を生値のまま表示）。
  ダッシュボードが使っている `friendlyEventName(event.type, locale)`
  (`AdminDashboardPage.tsx`) を監査イベント画面でも再利用し、一覧・詳細双方の種別表示を
  日本語化する。

### 系統2: 参照画面と編集画面の分離（ADR-086 / policy #1・#2 の徹底）

`frontend/ARCHITECTURE.md`「UI navigation and consistency policy」は既に
「一覧・詳細は読み取り専用、編集は専用ルート」「作成・編集にモーダルを使わない」を
明文化しており、[[wi-268-admin-ui-improvements]] と [[wi-309-external-identity-provider-admin-ui-consistency]]
で users/groups/identity-providers に適用済み。今回の指摘は、この方針がまだ及んでいない
残り領域を洗い出したものであり、新しい方針の考案は不要。判明した違反箇所は次の通り。

- **ユーザー一覧右ペインの「強制アクション」**（`AdminUsersListPage.tsx` が
  `UserRequiredActionsSection` 経由でその場から `handleRequiredAction` を呼び、
  set/clear API を直接叩いている）。一覧右ペインでは表示のみとし、変更操作は
  ユーザー詳細画面（またはユーザー編集画面）に一本化する。
- **ユーザー詳細画面の「所属グループの追加」「強制アクションの変更」「セッションの終了」**
  （いずれも `AdminUsersShared.tsx` の `UserGroupsSection` / `UserRequiredActionsSection` /
  `UserSessionsSection` として詳細画面に同居）。これらは属性編集ではないため詳細画面に
  置くこと自体は許容しうる設計だが、指摘の通り「グループ追加はできるが離脱ができない」
  「操作のたびに詳細画面と編集画面のどちらを開くべきか迷う」という非対称・非一貫性が
  実害である。本 WI では、詳細画面上のこれら3操作をユーザー編集画面
  (`AdminUserEditPage.tsx`) に移し、詳細画面は完全な読み取り専用にする方針で統一する
  （離脱操作の欠落もこの移設と合わせて追加する）。
- **グループ詳細画面のメンバー追加**（`AdminGroupDetailCard.tsx`）も同様に、詳細画面から
  グループ編集画面へ移す。
- **エージェント一覧・詳細の追加/編集モーダル**（`AgentEditorDialog.tsx` が
  `AdminAgentDetailCard.tsx`（一覧右ペイン）と `AdminAgentDetailPage.tsx`（詳細）の両方から
  呼ばれている）。users/groups/applications と同じ構成
  （一覧 `agents.tsx` → 詳細 `agents_/$agentId.index.tsx` → 編集
  `agents_/$agentId.edit.tsx` → 作成 `agents_/new.tsx`）に分割し、`AgentEditorDialog`
  はダイアログとしての利用をやめて編集/作成それぞれの専用ページに展開する。
  一覧右ペインの「バインド変更」も同様に参照専用へ変更し、変更操作は編集画面に移す。
- **アプリケーション一覧の追加モーダル**（`CreateApplicationDialog.tsx`）。同じ構成で
  `applications_/new.tsx` 専用ルートへ分割する。
- **OAuth2 認可詳細タイプ画面・MCP リソースサーバー画面**（`AdminAuthorizationDetailTypesPage.tsx` /
  `AdminMcpResourceServersPage.tsx` はいずれも `editing`/`showForm` の boolean で同一画面内に
  インラインフォームを出し入れしており、モーダルではないが URL が変わらない点で同種の問題）。
  一覧/詳細（または一覧/編集）を専用ルートに分割する。件数・利用頻度が他リソースより
  少ない画面のため、詳細画面を割愛し「一覧（参照専用）+ 追加 `/new` + 編集 `/$id/edit`」の
  3ルート構成に簡略化してよい。
- **ユーザー属性画面の追加モーダル**（`AttributeEditorDialog.tsx`）。同様に専用ルート
  （`tenant/attributes/new.tsx` 等）へ分割する。
- **ユーザー属性画面の編集・削除ボタン**（アイコンのみ、`aria-label` はあるが可視テキストなし）。
  他の一覧画面（例: エージェントの「削除」）と同じ、テキスト付きボタンに揃える。

### 系統3: レイアウト改善

- **ユーザー編集画面の縦長フォーム**（`AdminUserEditPage.tsx`）。ユーザー詳細画面
  (`AdminUserDetailPage.tsx`) が採用している2列レイアウトと合わせ、編集画面も同じ
  グリッド構成に揃える。これにより「詳細画面と編集画面のレイアウトを極力揃える」という
  指摘にも同時に応える。
- **OIDC アプリケーション詳細の区切り文字不統一**（リダイレクト URI = 改行、スコープ = 半角
  スペース、グラント種別 = カンマ `grant_types.join(', ')`）。共通の `UriList`
  （改行区切り表示、`AdminApplicationsShared.tsx`）にスコープ・グラント種別の表示も揃え、
  すべて改行区切りの表示に統一する（値の保存形式は変更せず、表示コンポーネントのみ変更）。
- **クライアントシークレットのローテーション UI の位置**
  （`AdminApplicationEditPage.tsx` でクライアント ID の `CopyableField` 直後に
  `ClientSecretRotationPanel` を配置）。操作体系がクライアント ID の他の項目
  （表示・コピーのみ）と全く異なるため、編集フォームの主要項目群から独立したセクション
  （見出し・枠線・警告トーンで視覚的に分離）に切り出す。UI の枠組みだけの変更であり、
  ローテーション機構自体（Entra ID 方式への刷新）は
  [[wi-316-oidc-client-secret-lifecycle-management]] で扱う。
- **プロビジョニング設定画面が詳細画面からしか開けない**
  （`ProvisioningNavButton` は `AdminApplicationDetailActions.tsx` のみで使われ、
  `AdminApplicationEditPage.tsx` からは導線がない）。編集画面にも同じ導線を追加する。
- **個別（オンデマンド）プロビジョニングの対象 ID 入力**
  （`AdminApplicationProvisioningOnDemand.tsx` の素のテキスト `<Input>`）。
  ユーザー/グループを検索して選択できるピッカーに置き換える。

### 系統4: 同意拒否フローの修正（バグ）

`ConsentPage.tsx` の `handleConsent('deny')` は、失敗時に汎用エラーメッセージを設定し
`setSubmitting(false)` するだけで、成功時の遷移も画面ロックも行わない。結果として
「許可しない」を押した後も「許可して続行」ボタンが押せる状態のまま残り、押しても
（同意が既に確定的に失敗しているにもかかわらず）同じエラーで止まり続ける。
`handleConsent('deny')` が成功した場合はクライアントへリダイレクト（拒否を示す
`error=access_denied` 等、OAuth2/OIDC の標準的なエラーレスポンス）で画面から離脱させる。
拒否操作そのものが失敗した場合（ネットワークエラー等）はリトライ可能な状態を維持してよいが、
現状のように「エラー文言だけ追加されて実際には何が起きたか分からない」表示は避け、
拒否が成立したのか通信が失敗しただけなのかをエラーメッセージで区別する。

## Plan

1. まず系統1（文言修正）を i18n ファイル中心に横断的に片付ける。挙動変更を伴わないため
   リスクが低く、並行して他系統の作業を進めやすい。
2. 系統4（同意拒否バグ）を独立した小さな修正として先に直す（ユーザー影響度が最も高い）。
3. 系統2（参照/編集分離）は画面ごとに、[[wi-268-admin-ui-improvements]] /
   [[wi-309-external-identity-provider-admin-ui-consistency]] で確立済みのルート分割パターン
   （一覧 → `$id.index.tsx` 詳細 → `$id.edit.tsx` 編集 → `new.tsx` 作成）を機械的に適用する。
   `adminNav.ts` へのエントリ追加は不要（既存ナビ項目からのリンク先が変わるだけ）。
   `src/routes/-page.tsx` の `PAGE_TITLES` に新規ルート分を追加する。
4. 系統3（レイアウト）は各画面の CSS/構造変更のみで、系統2のルート分割と同じ画面に
   触れる場合はまとめて実施する。
5. 各画面の変更後、既存のコンポーネントテスト（`renderWithRouter` ヘルパー使用）を更新し、
   新規に抽出した presentational component にはユニットテストを追加する
   （`frontend/ARCHITECTURE.md` の Container/Presentation split 方針に従う）。
6. 振る舞い（SCL）に影響する変更はない（表示・ナビゲーション・文言のみ）ため
   `spec/scl.yaml` の更新は不要。同意拒否バグの修正も、クライアントへのエラーリダイレクトは
   既存の OAuth2/OIDC 標準エラーレスポンスの範囲内であり、新規の意味変更ではない。

## Tasks

- [x] T001 [App] トップページ・ログインページ・同意ページの文言を一般ユーザー向けに書き直す
      （`HomePage.tsx`/`LoginPage.tsx`/`ConsentPage.tsx` と各 i18n）。
- [x] T002 [App] 同意拒否 (`handleConsent('deny')`) 成功時にクライアントへエラーリダイレクトし、
      画面に留まったまま「許可して続行」が再度押せてしまう状態を解消する。
      調査の結果、成功時のリダイレクト自体は既存コードで動作済みと判明（回帰テスト
      `redirects to the client when denying consent succeeds` を追加し確認）。実際のギャップは
      失敗時のメッセージが allow/deny で区別されず「何が起きたか分からない」点だったため、
      deny 専用の再試行メッセージ (`denyError`) を追加した。
      RED: `shows a deny-specific retry message when denying fails at the network level` を
      先に fail 確認 → GREEN（`AuthFlowPages.test.tsx`）。
- [x] T003 [App] 管理コンソール各画面の説明文（ユーザー一覧・アプリケーション一覧・グループ一覧
      ほか）を、各画面の実際の役割を言い当てる表現に書き直す。
- [x] T004 [App] 用語・表記統一: 「起動 URL」→ アプリケーション URL 系の表現、グループ/ユーザー
      選択プレースホルダの統一、「エージェントを削除」→「削除」。
- [x] T005 [App] アイコン画像アップロード欄をカスタム UI に置き換え、日本語ラベルを付ける
      (`AdminApplicationEditPage.tsx`)。
      RED: `shows the chosen icon file name instead of the native file input text` を先に
      fail 確認（PNG シグネチャ欠如で validateApplicationIconFile が弾いていた）→ 実 PNG
      シグネチャに直し GREEN（`AdminApplicationEditPage.test.tsx`）。
- [x] T006 [App] SAML NameID 形式の選択肢に正式な urn 表記を併記し、`signResponseLabel` の
      丸括弧注記を削除し、クレームマッピング表の列見出しとヘルプ文言を完全に日本語化する。
- [x] T007 [App] プロビジョニング設定の用語（グループ連携/プッシュ、ユーザー停止、409 照合属性、
      認証情報を更新する）を整理し、日本語化されていない補足説明を翻訳する。
      調査の結果、「グループ連携」(feature_flags.push_groups) と「グループを下流へプッシュする」
      (group_push 設定の有無) は UI 上別々にトグルできるが、バックエンドは push_groups を
      一切参照しておらず (grep 済み)、実質的な唯一のゲートは group_push の有無だった。
      feature flags 一覧から push_groups チェックボックスを撤去し、保存時に
      `feature_flags.push_groups` を group_push の有効状態へ自動的に同期させて一本化した
      （API 契約・SCL は変更なし、フロントエンドの送信内容のみ）。
      RED: `has a single group-push control that also drives the push_groups feature flag`
      を先に fail 確認（重複チェックボックスが2つ検出された）→ GREEN
      (`AdminApplicationProvisioningSettings.test.tsx`、新規作成)。
- [x] T008 [App] 監査イベント画面の種別表示にダッシュボードと同じ `friendlyEventName` を適用する。
      一覧・詳細双方の `e.type` 表示を置き換え、既存テストの生の type 文字列アサーションも
      `friendlyEventName` 経由の期待値に更新 (`AdminAuditEventsPage.test.tsx`)。
- [x] T009 [App] ユーザー一覧右ペインの「強制アクション」変更操作を撤去し参照専用にする。
      `UserRequiredActionsSection` の `onToggle` を任意化し、未指定時は付与済みアクションのみを
      バッジ表示する参照専用モードを追加。一覧右ペインからは `onToggle` を渡さない。
- [x] T010 [App] ユーザー詳細画面のグループ追加・強制アクション変更・セッション終了をユーザー
      編集画面へ移設し、グループ離脱操作を新設する。詳細画面は完全な読み取り専用にする。
      `UserGroupsSection`/`UserSessionsSection` に既存の `allowEditing` を活用・拡張し、詳細画面は
      すべて `allowEditing={false}` で参照専用化。編集画面 (`AdminUserEditPage.tsx`) に2列
      グリッドで3セクションを追加し、強制アクション用の独自 busy/notice state を新設。
      グループ離脱は既存の `removeAdminGroupMember` API を再利用して新設 (`leaveGroup` ボタン)。
      RED: `AdminUserEditPage.test.tsx` に `lets an admin toggle a required action without
      leaving the edit screen` を追加 → GREEN。回帰確認用に `AdminUserDetailPage.test.tsx`
      を新規作成（読み取り専用であることを検証）。
- [x] T011 [App] グループ詳細画面のメンバー追加をグループ編集画面へ移設する。
      `AdminGroupDetailCard.tsx` からメンバー一覧/追加/除外ロジックを `GroupMembersSection` として
      抽出し、詳細画面 (`AdminGroupDetailPage.tsx`) には `allowEditing={false}` を明示指定
      （従来 default true のまま渡し忘れていたのが実際の違反箇所）。編集画面
      (`AdminGroupEditPage.tsx`) に `GroupMembersSection` を追加。
      回帰確認用に `AdminGroupDetailPage.test.tsx` を新規作成、`AdminGroupEditPage.test.tsx` に
      メンバー追加コントロールの存在を確認するテストを追加。
- [x] T012 [App] エージェントの追加/編集モーダル (`AgentEditorDialog`) を廃止し、一覧
      (`agents.tsx`) / 詳細 (`agents_/$agentId.index.tsx`) / 編集
      (`agents_/$agentId.edit.tsx`) / 作成 (`agents_/new.tsx`) の専用ルートに分割する。
      一覧右ペインのバインド変更を参照専用にする。
      `agents_/$agentId.tsx` を detail leaf からレイアウトルート (Outlet) に変換し、
      groups と同じ index/edit 分割構成にした。`AgentEditorDialog.tsx` は削除し、
      新設の `AdminAgentEditPage.tsx`（プロフィール編集 + 資格情報バインド/解除を統合）と
      `AdminAgentCreatePage.tsx`（旧・一覧内インラインモーダルを移設）に展開。
      `AgentDetailCard`（一覧右ペイン/詳細共用）から bind/unbind の書き込み UI を削除し
      読み取り専用化、Edit ボタンは `editHref` によるルート遷移に変更。選択切り替え時の
      ローカル state リセットは `useEffect` ではなく `key={agent?.id}` の再マウントに変更
      (biome の exhaustive-deps 指摘を受けての設計変更)。
      新規テスト: `AdminAgentCreatePage.test.tsx`、`AdminAgentEditPage.test.tsx`
      （プロフィール更新・bind/unbind・失敗時表示）。既存
      `AdminAgentsListPage.test.tsx` からモーダル前提のテストを移設し、参照専用である
      ことを検証するテストに置き換え。
- [x] T013 [App] アプリケーション追加モーダル (`CreateApplicationDialog`) を廃止し
      `applications_/new.tsx` 専用ルートに分割する。
      `CreateApplicationDialog.tsx` を `AdminApplicationCreatePage.tsx`（フォーム内容は
      同一のまま外枠のみモーダル→専用ページに変更）へ展開し削除。一覧の「追加」ボタンは
      `/admin/applications/new` へのリンクに変更。
      作成テストを `AdminApplicationCreatePage.test.tsx` へ移設。
- [x] T014 [App] OAuth2 認可詳細タイプ・MCP リソースサーバー各画面のインラインフォームを、
      一覧 (参照専用) + 追加 `/new` + 編集 `/$id/edit` の専用ルート構成に分割する。
      両画面ともフォーム本体を `*Shared.tsx` に抽出し（type/resource フィールドは編集時
      `locked` で不可視化ではなく disabled 化）、Create/Edit 専用ページから再利用。
      一覧画面は削除ボタンのみ残し参照専用化（削除は他リソースの一覧右ペインと同様に
      許容: ADR-086 policy が禁じるのは create/edit のインライン化であり削除ではない）。
      個別 GET エンドポイントが無いため、編集ルートの loader は一覧を取得して該当項目を
      検索する方式（`saml-idp-profiles_/$profileId.edit.tsx` の既存パターンを踏襲）。
      モーダル前提だった既存テストを Create/Edit 専用テストへ移設・新設。
- [x] T015 [App] ユーザー属性画面の追加モーダルを専用ルートに分割し、編集・削除ボタンを
      他画面と同じテキスト付きボタンに揃える。
      カスタム属性はテナントスキーマの配列全置換で保存される単一ドキュメントのため、
      `AdminTenantAttributeCreatePage.tsx` は一覧 loader と同じスキーマを取得し、新規属性を
      追加した配列全体を PATCH する。フォーム本体は `AdminTenantAttributesShared.tsx` に
      `AttributeFormFields` として抽出し、一覧画面の編集ダイアログ (既存のまま維持、
      Design 節が専用ルート化を明示したのは追加のみ) と再利用。編集・削除ボタンは
      アイコンのみ→アイコン+テキスト（`variant="outline"`/`variant="destructive"`、
      lifecycle-workflows 一覧の既存ボタン様式を踏襲）に変更。
- [x] T016 [App] ユーザー編集画面のフォームレイアウトを、ユーザー詳細画面と同じ2列グリッドに
      揃える。
      T010 でセクション移設と合わせて実施済み（`grid xl:grid-cols-3` +
      `xl:col-span-2` を両画面で共通化）。
- [x] T017 [App] OIDC アプリケーション詳細のリダイレクト URI/スコープ/グラント種別の表示を、
      共通 `UriList` を用いてすべて改行区切りに統一する（保存形式は変更しない）。
      スコープは既存の `parseList` (空白/カンマ区切りをパースする既存ヘルパー) で配列化して
      `UriList` に渡す。グラント種別は既に配列型のため `.join(', ')` を除去して直接渡す。
      保存/送信フォーマットには触れていない（表示のみ）。回帰テストを追加。
- [x] T018 [App] クライアントシークレットローテーション UI をクライアント ID 表示から
      視覚的に独立したセクションへ分離する。
      `ClientSecretRotationPanel` 自体は既に警告トーンの枠を持つが、OIDC 設定セクション内で
      クライアント ID のすぐ下・リダイレクト URI 等の入力群の直前に挟まっていたため、OIDC
      セクションの外（閉じタグの直後）に独立した `<section>` として括り出した。
- [x] T019 [App] プロビジョニング設定画面への導線をアプリケーション編集画面にも追加する。
      既存の `ProvisioningNavButton`（詳細画面の `AdminApplicationDetailActions` が使用）を
      編集画面のヘッダー actions にも追加。回帰テストを追加。
- [x] T020 [App] 個別（オンデマンド）プロビジョニングの対象 ID テキスト入力を、ユーザー/グループ
      検索ピッカーに置き換える。
      テナント内のユーザー/グループ件数が大きくない前提の一覧選択（`listAdminUsers`/
      `listAdminGroups` を取得し、既存の `Select` コンポーネントで選択）に置き換えた。
      対象種別 (ユーザー/グループ) 切り替え時に選択をリセットする。
      新規テスト `AdminApplicationProvisioningOnDemand.test.tsx`
      （選択して送信・種別切り替えでリセット・一覧取得失敗時のエラー表示）。
- [ ] T021 [App] 変更した各画面のコンポーネントテストを更新し、新規抽出した presentational
      component にユニットテストを追加する。
- [ ] T022 [Verify] 下記 Verification を通す。

## Verification

- `just verify-ui`（format-check / lint / typecheck / unit test / build）
- `just test-ui-e2e`
- `just check-work-items` / `just check-ids`
- 手動: (1) トップページ・ログイン・同意の各画面をブラウザで開き文言が改善されていることを
  確認する。(2) 同意画面で「許可しない」を押し、クライアントへ戻る（同一画面に留まらない）
  ことを確認する。(3) ユーザー一覧・グループ一覧・エージェント一覧・アプリケーション一覧・
  ユーザー属性一覧の各右ペイン/一覧画面から編集操作ができなくなっており、詳細/編集の
  専用ルートに導線が移っていることを確認する。(4) エージェント・アプリケーションの追加/編集が
  専用 URL のページで行われ、ブラウザの戻るボタンで一覧に戻れることを確認する。

## Risk Notes

表示・文言・ナビゲーション構造の変更が中心で、SCIM/OAuth2 等の外部契約やデータモデルには
触れないため技術的リスクは低い。ただし対象画面数が多く、参照/編集分離のルート分割は
画面ごとに一定の作業量があるため、1セッションで全 Task を終える前提を置かず、画面単位で
区切って進める。同意拒否フロー修正 (T002) は認可フローに触れる唯一の Task であり、
クライアントへのリダイレクトパラメータが RFC 6749 の `error=access_denied` 相当になっている
ことを確認する。
