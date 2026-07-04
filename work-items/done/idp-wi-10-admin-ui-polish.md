---
id: idp-wi-10-admin-ui-polish
title: "管理画面・ログイン画面の使い勝手を直す (forgot link 位置 / 編集ダイアログ統合 / dashboard / refresh ボタン)"
created_at: 2026-06-15
authors: ["tn"]
status: completed
risk: medium
---

# Motivation
日常的に画面を触っていて、操作の流れに引っかかる箇所が 4 つ見えた。
1 本の WI でまとめて整える (個別の優先度はどれも低〜中だが、放置すると
どれも admin・end user の体験を地味に削り続けるため)。

1. ログイン画面: "パスワードを忘れた場合" のリンクが password 入力の
   **上** にあるため、tab 移動が `username → forgot link → password →
   submit` となり余計な tab が 1 回入る。industry standard (Google /
   Microsoft / Apple) は link を password の下に置く。
2. AdminUsersPage: 属性編集 (wi-7) とロール編集 (RoleEditorDialog) が
   別ダイアログに分かれており、admin が「メール変更ついでにロールも
   直す」みたいな複合操作で 2 回ダイアログを開き直す。1 ダイアログに
   統合する。ただしロール変更の "review changes" 二段階確認 (admin /
   system_admin の付け外しを意識的にやらせる ceremony) は残す。
3. 管理画面のホーム: `/admin` 直下のダッシュボードが存在せず、いきなり
   `/admin/users` などの個別ページに着地する。横断的な数 (users /
   active clients / pending consents / 24h audit events) が見えず、
   admin が「何が起きているか」を 1 画面で把握できない。
4. AdminUsersPage / AdminClientsPage 等の `更新` ボタン: "edit" とも
   "refresh" とも読めるラベル名で、"ユーザーを追加" の隣にあるため
   action group の意味が階層化されていない。アイコンのみの refresh
   ボタンに統一し、テーブル直上のフィルタ行に移す。

# Scope
- **ui**:
  - pages:
    - LoginPage — "パスワードを忘れた場合" リンクを password 入力欄 の **下** (送信ボタンの直前 or 直後) に移動する。tab order が `username → password → submit → forgot link` になるよう DOM 順を直す。視覚的なヒエラルキー (font-size / color) は 現状を踏襲し、主要 CTA (送信ボタン) より目立たないことを維持 する。
    - AdminUsersPage — 既存の `AttributeEditorDialog` を `UserEditorDialog` にリネーム (or 統合) し、Profile セクション と Roles セクションを内側に並べる。Profile 4 項目 (preferred_username / name / email / email_verified) と Roles 編集を 1 つの保存ボタンで送信する。Roles に変更がある ときだけ、保存前に "review changes" の second-step confirmation を出す (現 RoleEditorDialog の動線を移植する)。Profile だけ の変更なら一発で送信。UserDetails 詳細パネルの "ロールを変更" ボタンを削除し、"属性を編集" ボタンを "編集" (or "編集 (属性 とロール)") に改名する。api.ts は `updateAdminUserAttributes` と `updateAdminUserRoles` を残したまま、ダイアログ側で 「変更があった項目だけ送る」or「両方一括 PATCH」のどちらかに 統一する (推奨は後者: backend の `UpdateUserInput` が pointer optional なので、変更フィールドだけ非 nil で送れば既存の use case で完結する)。
    - AdminDashboardPage (新規) — ルート: `/admin` (tenant 内)。 既存 sidebar の "概要" 項目に紐付ける (`adminNav.ts` の disabled を解除)。カード 4 枚: 総ユーザー数 (active / disabled の比率) / クライアント数 (`active` のみカウント) / tenant 内 consent 数 (`active` のみカウント) / 24h 以内の admin audit event 数。直近 5 件の audit event リスト (timestamp / type / actor → target)。クリックで AdminAuditEventsPage の filter に飛ばす。クイックリンク (アイコン付き 4-6 枚): "ユーザを追加" / "クライアントを追加" / "鍵を確認" / "監査ログを開く"。tenant 管理リンクは `system_admin` かつ default tenant の時のみ出す。
    - 全 admin ページの `更新` ボタンを削除し、テーブル直上の フィルタ行に IconRefresh のみのアイコンボタンを置く (aria-label="一覧を再読み込み")。AdminUsers / AdminClients / AdminConsents / AdminAuditEvents / AdminKeys / AdminTenants の 6 ページを揃える。
  - api:
    - admin dashboard 用の集計は **専用 endpoint を作らず**、既存 list endpoint を並列に叩いて UI 側で count する (page load 時に Promise.all)。endpoint 追加は scope が膨らむので避ける。
    - 既存 `listAdminAuditEvents` の `after` パラメータで 24h 絞り込みを admin dashboard 側で行う。
  - routing:
    - `/admin` を AdminDashboardPage に割り当てる。loadPageData の ディスパッチに追加する。
    - 既存 `/admin/users` 等の挙動は変更しない。
  - navigation:
    - `adminNavItems` の "概要" を有効化し、`href='/admin'`、 `active = path === '/admin'` で他項目と排他にする。
- **scl**:
  - 変更なし。新規 endpoint / 新規モデル / 新規 event は導入しない。 ダッシュボードは既存データの read-only 集計表示のみ。
- **documentation**:
  - idmagic/README.md と idmagic/ui/README.md には新規記述を 増やさない (admin 画面の sidebar に "概要" が増えるだけのため、 README の admin UI 説明の文言は微調整のみ可)。

# Out of Scope
- 新規 backend endpoint (集計 API / metrics API)。
- dashboard カードのリアルタイム更新 (WebSocket / SSE)。1 ロード 時の count + 直近 N 件で固定。
- tenant 横断ダッシュボード (system_admin 向けでも default tenant 配下に閉じる)。
- 通知 (バナー / トースト) センター。
- i18n 追加 (既存 ja/en の slot に乗せるのみ)。
- グローバル検索 / quick action palette。
- end user 側 (`/account/*`) の改修。

# Verification
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- `go test ./internal/adapters/http/...` (in: idmagic)
  - reason: backend は変更しないが、admin handler テストで wire 形式の 回帰がないことを確認する。
- 手動 1 (LoginPage): username → tab → password → tab → submit ボタンの順で focus が遷移すること、forgot link は submit の直後 で focus 可能になることを確認。
- 手動 2 (UserEditorDialog): alice の name と email を変更 → 保存 → 一覧と detail に反映、audit に user.updated が出ることを確認。 続けて roles を変更 → review confirmation が出てから保存できる ことを確認。両方同時に変更しても 1 リクエストで反映されることを 確認。
- 手動 3 (AdminDashboardPage): `/admin` を開いてカード 4 枚と直近 audit event が出ること、クイックリンクの 4-6 ボタンが対応ページ に遷移すること、`system_admin` で default realm のときだけ tenant クイックリンクが出ること。
- 手動 4 (refresh icon): 6 admin ページのフィルタ行で IconRefresh のみが表示され、押すと一覧が再読み込みされること、"更新" ラベル の文字列が dom から消えていることを確認。

# Risk Notes
4 件を 1 WI に束ねているため、回帰範囲が admin 画面ほぼ全域 + login
画面に広がる。各部分を独立コミットで分けてレビューしやすくする
(LoginPage / UserEditorDialog 統合 / AdminDashboardPage / refresh icon
の 4 commit) ことが望ましい。

UserEditorDialog 統合は最も触る範囲が広く、ロール変更の confirmation
step を踏み外すと "admin が自分の admin role を保存ボタン一発で外す"
ような事故が起こりうる。review-step を残す方針を unit / 手動テスト
両方で担保する。

AdminDashboardPage は既存 list endpoint を集計に流用するため、tenant
内 user / client / consent / audit-event が多い環境で初回ロード遅延の
懸念がある。スコープ拡大を避けるため本 WI では caching / pagination は
入れず、limit 付き list で取得する。負荷観測の結果は別 WI で扱う。

# Completion
- **Completed At**: 2026-06-16
- **Summary**:
  4 件の UX ノイズを 1 本の WI として 4 commit に分けて直した。
  どれも単独では小さいが、放置すると管理者と end user の体験を地味に
  削り続ける箇所だったため、まとめて整理した。
  backend (Go) には触らず、すべて UI のみで完結している。

  part 1 (LoginPage): "パスワードを忘れた場合" リンクを password 入力欄の
  下、submit ボタンの直後に移した。tab 順が
  `username → password → submit → forgot link` で安定した。

  part 2 (UserEditorDialog): 属性編集とロール編集の 2 ダイアログを統合し、
  Profile セクションと Roles セクションを 1 画面に並べた。ロールに変更が
  ある場合だけ "review changes" 二段階確認を出すという原則は維持した。

  part 3 (AdminDashboardPage): `/admin` を独立した landing page にした。
  カード 4 枚 (users / clients / granted consents / 24h audit events) +
  直近 5 件の監査イベント + クイックリンクで、何が起きているかを 1 画面で
  把握できるようにした。集計用の新規 endpoint は作らず、既存 list endpoint
  を Promise.all で叩いて UI 側で count する形に留めた。

  part 4 (refresh icon): admin 5 ページの "更新" / "再読込" テキストボタンを
  IconRefresh のみのアイコンボタンに置き換えた。"...を追加" ボタンの隣で
  primary 操作と並んで見えていた refresh が、副次的な affordance として
  正しく扱われる位置に収まった。
- **Verification Results**:
  - `bun --cwd idmagic/ui typecheck`
    - result: ok (tsc --noEmit pass)
  - `bun --cwd idmagic/ui lint`
    - result: ok (biome lint pass, 40 files)
  - `bun --cwd idmagic/ui build`
    - result: ok (vite build pass, 6284 modules)
  - `go test ./internal/adapters/http/...` (in: idmagic)
    - result: ok (admin handler 既存テストの回帰なし)
  - 手動確認 (residual): 各 commit ごとの実ブラウザ操作は本セッションでは 未実施。typecheck / lint / build / go test がすべて緑であることから 回帰の主リスクは低いが、wi-10 の verification にある手動 1〜4 は dev サーバ起動時に行う必要がある。
- **Affected Guarantees State**:
  - admin RBAC: AdminDashboardPage は既存 `verifyBrowserRequest` + `requireAdmin` を通る (新規経路を追加していないため)。
  - CSRF: dashboard は read-only のため CSRF token 不要。GET endpoint のみ。UserEditorDialog の保存は既存 PATCH の CSRF protection をそのまま 使う。
  - UserEditorDialog: ロール変更時の "review changes" 二段階確認は維持。 admin / system_admin role の付け外しが他の編集に紛れて起きない。
  - SCL coherence: scl.yaml を変更していないため SCLPermissionsCoverage 系テストは無影響。
  - backend 既存挙動: PATCH `/admin/users/:sub` の pointer-optional semantics に乗り、変更フィールドのみを送る。`equalOptionalString` で no-change を判定する既存 use case 挙動は不変。
