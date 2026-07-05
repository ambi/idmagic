---
id: wi-126-admin-and-account-ui-consistency-and-navigation-policy
title: 管理コンソール／マイページの UI 一貫性・ナビゲーションポリシー整備とユーザー詳細遷移バグ修正
created_at: 2026-07-05
authors: [tn]
status: in_progress
risk: medium
---

# Motivation
管理コンソール・マイページ・ログイン画面を実際に操作して見つかった、明確な
バグ 1 件と、UX 上の一貫性・情報設計の改善点をまとめて起票する。個々は小粒だが
「詳細 → 編集のナビゲーション」「一覧行アクションの見せ方」「編集はモーダルか
専用画面か」「画面ごとのページタイトル」など **横断的な UI ポリシー**に収束する
論点が多く、都度アドホックに直すと再び揺らぐ。まず全体を 1 つの WI に集約し、
統一方針として決めてからまとめて反映する。

最優先はユーザー一覧 → 詳細のリンク切れ (機能停止バグ)。それ以外は UX 改善と
ポリシー整備。実装は論点ごとに PR 分割してよい (下記 §Scope の番号単位)。

## 確認済みのバグ根因
ユーザー一覧から詳細を開くと `/admin/users/undefined` に遷移し「認証を続行
できません」エラーになる。バックエンドの admin user レスポンスは `sub` から
`id` にフィールド名が変わっている (`internal/identitymanagement/adapters/http/
admin_user_handler.go:45` が `json:"id"`、`internal/shared/spec/users.go:16` も
`json:"id"`) が、フロントの `AdminUser` 型はまだ `sub` (`ui/src/types.ts:9`) で
残っており、一覧の詳細リンク `detailHref=.../admin/users/${user.sub}`
(`ui/src/features/admin-users/AdminUsersPage.tsx:931`) の `user.sub` が
`undefined` に解決される。詳細ルートのパラメータ名も `$sub`
(`ui/src/routes/admin/users_/$sub.tsx`) のまま。右ペイン／詳細の「Subject ID」
行 (`AdminUsersPage.tsx:718,1002`) も同じ理由で空になる。sub→id の移行時に
フロント側が未追随だった箇所。

# Scope
UI の画面設計・コピー・ナビゲーション整備が中心。normative な SCL / permission /
API 契約は原則変更しない (バグ修正はすでに変わったバックエンドへのフロント追随)。
実装時に用語統一で SCL の語彙に触れる可能性がある箇所は §9 に切り出す。

- **ui**:
  1. **[bug/P0] ユーザー詳細遷移の sub→id 追随**: フロントの `AdminUser` 型と
     admin users 系 API ラッパ (`ui/src/api/admin.ts`) の `sub` を、バックエンドが
     返す `id` に合わせて統一する。一覧の詳細リンク・選択キー・各種操作
     (`getAdminUser` / update / disable / delete / restore / required_actions /
     グループ追加) が参照する `user.sub` を `user.id` に直し、詳細ルートの
     パラメータ `$sub` を `$id` にリネームする。`/admin/users/{id}` で詳細が
     開けること、deep link でも開けることを確認する。エージェント等、他エンティティ
     でも同種の sub/id 揺れがないか併せて点検する。
  2. **「Subject ID」表記の見直し**: 右ペイン／詳細の識別子ラベルが
     「Subject ID」で、意味が伝わりにくく値も空になっている。値を `id` に直す
     ことに加え、ラベルを利用者に分かる表現 (例: 「ユーザーID」) に改める。
  3. **ページタイトル (`document.title`)**: 管理コンソール・マイページ・ログイン
     画面のタイトルが一律「IdMagic」(`ui/index.html`)。画面ごとに現在地が分かる
     タイトル (例: 「ユーザー | IdMagic 管理コンソール」) を設定する。ルーティング
     機構 (TanStack Router, `ui/src/routes/`) に沿ってルート単位でタイトルを
     与える仕組みを用意する。
  4. **ダッシュボード再設計**: `admin-dashboard` の表示内容が有意でない。
     Okta / Entra ID / Google Cloud IAM / Keycloak / OneLogin の管理ダッシュボードを
     参考に作り直す。少なくとも (a) ユーザー数・アプリケーション数など単なる
     カウントの価値を再検討、(b) クイックリンクから「署名鍵を確認」のような
     ほぼ使わない導線を外し実務的な導線に絞る、(c) 直近の監査イベントは残す
     余地はあるが `RefreshTokenIssued` のような内部イベント名の生表示をやめ
     人間可読の説明にし、クリックで当該イベントの意味のある文脈へ遷移させる
     (単なる一覧表示で終わらせない)。
  5. **「一覧を更新しました」トーストの改善**: 一覧更新後のメッセージ
     (`'一覧を更新しました。'`、`AdminUsersPage.tsx:326` ほか admin-agents /
     admin-groups / admin-applications / admin-consents / admin-keys /
     system-tenants で共通) が残り続け、かつ挿入で一覧表が下にずれる。レイアウトを
     押し下げない・自動で消える控えめな見せ方 (トースト等) に改める。
  6. **編集／追加はモーダルでなく専用画面に**: ユーザー・グループの編集
     (および登録) が現状モーダル。独立した編集画面に移行する。
  7. **一覧行アクションの統一**: 一覧右側の「詳細」「編集」「削除」ボタンが、
     画面によって直接表示されたり三点リーダーメニューに隠れたりと揺らいでいる。
     見せ方を全画面で統一する (方針: 表示してよい)。
  8. **詳細 → 編集ナビゲーションの統一ポリシー適用**: 「まず現在の設定を見せる
     画面 (詳細) → それを編集する画面は別画面」を UI ポリシーとして全画面に
     適用する。一覧があるものは 一覧 → 詳細 → 編集、および 一覧 → 編集を許すが、
     最初に出る一覧／詳細画面がそのまま編集画面になっている状態を無くす (追加も
     同様)。未追随の Entra 連携ページ (`admin-entra-federation`)、SCIM アクセス
     トークン (`admin-settings` 配下) を優先的に是正する。
  9. **マイページ「アカウント情報」の導線・URL**:
     - メールアドレスが「アカウント情報」に表示されるのに同画面では編集できず、
       別の「メールアドレス」画面でのみ編集できる (確認フローが要る特殊属性の
       ため)。アカウント情報からメール編集画面への導線を追加し気づけるようにする。
     - 「アカウント情報」で「編集」を押しても URL が `/account/profile` のまま
       変わらないのに、見た目は別ページ遷移のように変化するため、ブラウザの
       戻るで想定外に前ページへ戻ってしまう。URL を編集画面用に変える (§8 の
       詳細→編集ポリシーと整合) か、同一ページ内変化だと明確に見せるか、いずれかに
       統一する。
- **decision**:
  - 本 WI の横断ポリシー (詳細→編集ナビゲーション / 一覧行アクションの見せ方 /
    編集は専用画面 / 画面ごとのページタイトル / 用語統一) を、今後追加される
    画面にも効くよう ADR 化するか検討する。既存の UI 一貫性系 WI
    ([[idp-wi-39-admin-detail-pages]] / [[idp-wi-37-admin-ui-consistency-and-localization-polish]] /
    [[idp-wi-16-admin-shell-visual-consistency]]) を踏まえ、重複しない粒度で残す。
- **documentation**:
  - 上記 UI ポリシーを「今後の UI 画面にも反映される」場所に明文化する。SCL に
    載せるべき normative 事項か、リポジトリの Markdown ドキュメント (UI ガイド
    ライン) が適切かを判断して配置する。判断の初期方針: 画面構成・遷移・コピーの
    規約は Markdown の UI ガイドライン + (必要なら) ADR、語彙の正本 (§9) のみ SCL。
- **spec (§9 用語統一、必要時のみ)**:
  - 「監査イベント」「監査ログ」が UI 内で混在している。正本語彙を 1 つに決めて
    統一する。SCL に該当語彙 (audit event) の定義があればそれに合わせ、UI コピーを
    そろえる。SCL 側の語が UI と食い違う場合のみ SCL を最小限更新する。

# Out of Scope
- 監査イベント基盤そのものの機能追加 (新しいイベント種別・検索軸など)。本 WI は
  ダッシュボードでの見せ方と用語統一に閉じる ([[idp-wi-44-audit-event-store-and-search]] /
  [[idp-wi-46-authentication-event-attribute-emit-and-correlation-search]] の範囲は触らない)。
- admin 詳細画面そのものの新設 (すでに [[idp-wi-39-admin-detail-pages]] で完了)。
  本 WI は「詳細→編集」遷移ポリシーの徹底と未追随画面の是正に閉じる。
- 新規バックエンドエンドポイント・集約の追加。バグ修正はフロントの sub→id 追随のみ。
- 認証・権限境界・テナント分離の変更。
- ログイン画面の視覚デザイン刷新 ([[idp-wi-89-tenant-login-branding]] の範囲)。本 WI は
  ログイン画面については §3 のページタイトルのみ扱う。

# Initial Context
- `ui/src/features/admin-users/AdminUsersPage.tsx` (sub→id バグの中心、行アクション、右ペイン)
- `ui/src/types.ts` / `ui/src/api/admin.ts` (`AdminUser` 型・API ラッパの sub→id)
- `ui/src/routes/admin/users_/$sub.tsx` (詳細ルートのパラメータ名)
- `internal/identitymanagement/adapters/http/admin_user_handler.go` / `internal/shared/spec/users.go` (バックエンドの `id` 契約)
- `ui/index.html` / `ui/src/routes/` (ページタイトル)
- `ui/src/features/admin-dashboard/` (ダッシュボード)
- `ui/src/features/admin-entra-federation/` / `ui/src/features/admin-settings/` (詳細→編集ポリシー未追随)
- `ui/src/features/account/` (マイページ アカウント情報・メール編集導線)

# Affected Guarantees
- admin RBAC / tenant isolation: 既存の admin-guarded API を再利用するのみで
  権限境界・テナント分離を変えない。
- 内部表現の非露出: ラベルや監査イベント表示で内部識別子・内部イベント名を
  生で出さない ([[idp-wi-19-rich-user-attributes]] / [[idp-wi-37-admin-ui-consistency-and-localization-polish]] 継続)。
- backwards compatibility: admin API レスポンスは不変 (フロントが `id` に追随するのみ)。

# Verification
- `just verify-ui` (format-check / lint / typecheck / build を包含)
- `just yaml-check-work-items`
- `just check-ids`
- 手動 1: `/admin/users` で一覧からユーザーを選び詳細を開くと `/admin/users/{id}` に
  遷移し詳細が表示される (`undefined` にならない)。deep link でも開ける。
- 手動 2: 右ペイン／詳細の識別子行に `id` の値が表示され、ラベルが分かりやすい
  表現になっている。
- 手動 3: 管理コンソール・マイページ・ログインの各画面でブラウザタブのタイトルが
  現在地を表す。
- 手動 4: 一覧更新後のメッセージが一覧表を押し下げず、一定時間で消える。
- 手動 5: ユーザー／グループの編集・追加が専用画面で開く。一覧行の
  「詳細／編集／削除」導線が全画面で統一されている。
- 手動 6: Entra 連携・SCIM アクセストークンで、最初に現在設定の閲覧画面が出て
  編集は別画面に分かれている。
- 手動 7: マイページ「アカウント情報」からメール編集への導線があり、編集時に
  URL 変化と画面遷移の見え方が整合している。
- 手動 8: ダッシュボードのクイックリンク・直近監査イベント表示が実務的で、
  内部イベント名の生表示が無い。「監査イベント／監査ログ」の表記が統一されている。

# Risk Notes
大半は UI の画面設計・コピー・遷移の整備で影響は局所的。ただし範囲が広く論点が
多いため、①sub→id バグ修正 (P0)、②ページタイトル・トースト・行アクション、
③編集の専用画面化と詳細→編集ポリシー適用 (Entra / SCIM 含む)、④ダッシュボード
再設計、⑤マイページ導線、⑥用語統一とポリシー文書化、のように PR を分割して
段階リリースするのが安全。sub→id 修正は他エンティティにも同種の揺れがないか
点検し、取りこぼしを防ぐ。用語統一 (§9) で SCL 語彙に触れる場合は scl-change の
手順に従い derived artifact を再生成する。

# Progress

本 WI はアンブレラのため段階的に実装する。各フェーズ完了時にここへ記録し、全
フェーズ完了時に `status: completed` + `Completion` を追記して `done/` へ移す。

## 2026-07-05 — §1 (P0 バグ) + §2 完了 (UI のみ)

sub→id 移行のフロント未追随を修正。バックエンドはユーザー識別子を `sub` から
`id` に移行済みだったが、フロントの型が旧名のままで、ユーザー一覧→詳細が
`/admin/users/undefined` になっていた。調査の結果、同じ移行が admin コンソールの
複数エンティティで未追随・破損しており、§1 の「他エンティティ点検」に従いまとめて
是正した。

- **§1 型追随** (`ui/src/types.ts`): `AdminUser.sub`→`id`、
  `AdminGroupMember.user_sub`→`user_id`、`AdminConsent.sub`→`user_id`、
  `AdminAgent.owner_sub`→`owner_user_id` (backend の
  `adminUserResponse.id` / `admin_group_handler` `user_id` /
  `admin_consent_handler` `user_id` / `admin_agent_handler` `owner_user_id` に一致)。
- **§1 consumer 追随**: `AdminUsersPage`（選択キー・検索・詳細リンク・全操作・
  グループ追加）、`AdminGroupsPage`（メンバー一覧/追加/削除）、
  `AdminApplicationsPage`（割当ユーザーピッカー）、`AdminAgentsPage`（所有者
  入力/表示、作成/更新 body）、`api/admin.ts`（ユーザー系ラッパ引数 `sub`→`id`、
  consent `userID`、group member `userID`、agent input `owner_user_id`）。
- **§1 詳細ルート**: `ui/src/routes/admin/users_/$sub.tsx` を `$id.tsx` に
  リネームし `params.sub`→`params.id`。`routeTree.gen.ts` を router-plugin で再生成。
- **§2 ラベル**: 右ペイン・詳細の識別子行ラベル「Subject ID」→「ユーザーID」、
  consent 詳細「ユーザー (sub)」→「ユーザー (ID)」、agent「所有者 sub」→
  「所有者 (ユーザーID)」。値は各 `id`/`user_id` を表示。
- **対象外 (今回は触れない)**: account portal の `AccountProfile.sub` /
  `AccountSummary.sub` は backend が `id` を返すがフロントで `.sub` を読む箇所が
  無く実害なし (§9 マイページで扱う)。`PortalAccount.sub`・監査イベントの `sub`
  フィルタ/ペイロードは別機構 (backend も `sub` のまま) で未 drift。
- **検証**: `just verify-ui` (format-check / lint / typecheck / build) green。
  ユーザー一覧→詳細が `/admin/users/{id}` で開くことのブラウザ目視確認は未実施。

残り §3〜§9 は pending。

## 2026-07-05 — §3 + §5 + §7 完了 (UI のみ)

Risk Notes のフェーズ② (ページタイトル・トースト・行アクション) を実装。

- **§3 ページタイトル** (`ui/src/routes/-page.tsx`): `PageMarker` の `kind` を
  正本マップ `PAGE_TITLES` でタブタイトルへ対応づけ、`document.title` を設定する
  仕組みを追加。管理コンソール / マイページ / システム管理 / ログインの各画面に
  ポータル別接尾辞付きの現在地タイトル (例「ユーザー | IdMagic 管理コンソール」)
  を与える。既存の全ルートが `PageMarker` を通るため route ファイルの改変は不要。
  未定義 kind は素の "IdMagic" にフォールバック。`markErrorPage` も追随。
- **§5 トースト** (`ui/src/components/ui/toast.tsx` 新規): 成功通知を、レイアウトを
  押し下げない固定表示 (position: fixed) で数秒後に自動消去し、閉じるボタンでも
  消せる `Toast` を追加。各画面の成功 `notice` インライン表示 (14 ファイル・
  `<Alert variant="success">` および `AdminUsersPage` の `role="status"` div 2 箇所、
  計 17 箇所) を `Toast` に置換。エラーは持続表示したいので従来どおりインラインの
  `Alert` のまま。`onDismiss` は ref に逃がし、親再描画でタイマーがリセットされ
  続けないようにした。
- **§7 行アクション統一** (`ui/src/components/AdminPaneActions.tsx`): 右ペインの
  二次アクションを ⋮ ドロップダウンから直接ボタン表示へ変更 (`menu?: ReactNode` →
  `actions?: PaneAction[]`)。他の一覧画面 (署名鍵・テナント・認可詳細の種類) が
  既に直接ボタンなのに合わせ「表示してよい」方針で統一。破壊的操作は赤系
  (tone='danger')。callers: `AdminUsersPage` (復元/完全削除 または 無効化/削除)、
  `AdminGroupsPage`・`AdminAgentsPage`・`AdminApplicationsPage` (削除) を追随。
  `AdminRolesPage` は二次アクション無しで不変。
- **対象外 (フェーズ②では触れない)**: 詳細ページヘッダ (`AdminShell` の `actions`
  スロット) の ⋮ ケバブは list-view ではなく詳細画面の面なので §8 (詳細→編集
  ポリシー・フェーズ③) で扱う。§4 ダッシュボード・§6 編集専用画面・§9 マイページ
  導線・用語統一/文書化も未着手。
- **検証**: `just verify-ui` (format-check / lint / typecheck / test-ui-unit /
  build) green、`just yaml-check-work-items` / `just check-ids` OK。ブラウザ目視
  (手動 3/4/5) は未実施。

残り §4・§6・§8・§9・decision・documentation は pending。

## 2026-07-05 — §8 優先対象 (Entra / SCIM) 完了 (UI のみ)

§8「詳細→編集ポリシー」で WI が優先指定した未追随 2 画面 (Entra 連携・SCIM
アクセストークン) を、applications の詳細/編集分離を範として是正。「最初に出る
画面は現在の設定を見せ、追加/編集は別画面 (または明示アクションで切替)」に統一。

- **§8 Entra** (`ui/src/features/admin-entra-federation/AdminEntraFederationPage.tsx`,
  `ui/src/routes/admin/federation/entra_/new.tsx` 新規): 一覧画面
  (`/admin/federation/entra`) は「現在フェデレーション済みドメイン一覧＋エンドポイント
  リンク」の閲覧に限定し、configure フォームを専用の追加画面
  (`/admin/federation/entra/new`, `AdminEntraFederationAddPage`) へ分離。一覧ヘッダに
  追加画面への導線ボタン、追加画面に一覧へ戻る導線と PowerShell 手順の一度きり表示を
  配置。applications の flat route 規約 (`entra_/new.tsx`) に合わせ、route tree は
  router-plugin で再生成。
- **§8 SCIM** (`ui/src/features/admin-settings/AdminSettingsPage.tsx` `ScimTab`):
  設定はタブ構成のため GeneralTab/PasswordPolicyTab と同じ「閲覧→明示操作で編集」
  トグルに合わせ、既定で接続情報＋現行トークン一覧を見せ、「トークンを発行」ボタン
  押下で発行フォームを表示 (キャンセル可)。一覧画面が常時発行フォームを兼ねる状態を
  解消。
- **用語**: ユーザー指摘を受け「ドメインを federation する」等の英語動詞混在の
  半端訳をやめ、タイトル/ラベル/トーストを自然な日本語 (フェデレーション/
  ドメインフェデレーションを追加) に統一。真の術語 (relying party, claim preset,
  Microsoft 365 domain federation) のみ英語を残す。
- **§3 追随**: 追加画面用の kind `admin-entra-federation-add` を `PAGE_TITLES` に追加。
- **対象外 (今回は触れない)**: §6 ユーザー/グループの編集・追加のモーダル→専用画面化
  (大きめ・別セッション)、§8 の他画面 (詳細ページヘッダの ⋮ ケバブ等) への横展開、
  §4 ダッシュボード・§9 マイページ・decision/documentation。
- **検証**: `just verify-ui` (format-check / lint / typecheck / test-ui-unit /
  build) green、`just yaml-check-work-items` / `just check-ids` OK。ブラウザ目視
  (手動 6) は未実施。

残り §4・§6 (users/groups)・§8 横展開・§9・decision・documentation は pending。

## 2026-07-05 — §6 ユーザー編集の専用画面化 完了 (UI のみ)

§6 のうちユーザー編集をモーダルから専用画面へ移行 (登録・グループは対象外、別
セッション)。applications の詳細/編集ルート分離を範とし、詳細→編集ポリシー (§8)
にも整合させた。

- **編集画面** (`ui/src/features/admin-users/AdminUsersPage.tsx`): 従来の
  `UserEditorDialog` モーダルを `AdminUserEditPage` (専用画面) に変換。AdminShell +
  Card 内に同じフォーム (プロフィール/メール確認/ロール/カスタム属性) と、ロール変更
  時の確認ステップ (confirming) を同一画面で保持。保存は `updateAdminUser` 後に詳細
  画面へ遷移、キャンセル/戻るは詳細へ戻す。
- **ルート** (`ui/src/routes/admin/users_/$id.tsx` をレイアウト(Outlet)化、
  `$id.index.tsx` に詳細、`$id.edit.tsx` に編集を新設): `/admin/users/{id}` 詳細と
  `/admin/users/{id}/edit` 編集に分離。route tree は router-plugin で再生成。
- **導線**: 詳細ヘッダの「編集」ボタンを編集画面への `<a>` リンク化。一覧右ペインは
  `AdminPaneActions` に `editHref` を追加し、「編集」を (モーダルでなく) 編集画面への
  アンカーに変更 (§8 の「一覧→編集」許容に沿う)。`onEdit` コールバックはグループ/
  エージェント等のモーダル呼出し用に残置。
- **不要物整理**: 一覧 `AdminUsersPage` と一覧ルート loader から編集専用だった
  `attributeDefs` (schema fetch) を除去。編集/詳細画面は各 loader で schema を取得。
- **対象外 (今回触れない)**: ユーザー「登録」(CreateUserDialog) のモーダル、グループ
  編集/登録、§4・§9・decision/documentation。
- **検証**: `just verify-ui` (format-check / lint / typecheck / test-ui-unit /
  build) green・warning 0、`just yaml-check-work-items` / `just check-ids` OK。
  ブラウザ目視 (手動 5) は未実施。

残り §4・§6 (ユーザー登録 / グループ)・§8 横展開・§9・decision・documentation は
pending。
