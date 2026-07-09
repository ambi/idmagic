---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-06-16
---

# 管理画面ヘッダと UserDetails の情報階層を直す (ブランドリンク / アバターメニュー / アクションの優先度)

## Motivation
日常的に管理画面を触っていて、UI の情報階層に 3 つの引っかかりが
ある。どれも単独では小さいが、まとめて直さないと "admin の頻度に
応じて適切なボタンが手前に来る" という基本的な階層が崩れたままに
なる。

1. **ブランド名がリンクになっていない**。
   左上の "RA Identity" は他のアプリ (Okta / Google Admin /
   Microsoft 365) と同様、テナント home (`/admin`) への戻りリンクで
   あるべきだが、現状は plain text。admin が深い詳細ページから
   ダッシュボードへ戻る動線が無い。

2. **右上のアバターがメニューを開かない**。
   アバター ( `(actorUsername).slice(0,1).toUpperCase()` の丸印) と
   username 表示は装飾としてあるだけで、click しても何も起きない。
   アバターメニューはエンドユーザの「自分自身に対する操作」が
   集まる場所なので、欠落していると "パスワード変更"・"ログアウト" の
   経路がヘッダ右側の単発アイコンに頼る形になる。

3. **UserDetails のアクション階層が頻度と逆転している**。
   詳細パネルの構成は上から順に Profile → Roles → アカウント状態
   (Disable / Enable, full-width destructive ボタン) → アカウントを
   削除 (full-width destructive ボタン) となっており、削除ボタンは
   一画面に収まらず必ずスクロールする位置にある。実際の利用頻度は
   編集 ≫ 無効化 > 削除 だが、UI のサイズと位置は逆順。

## Scope
- **ui**:
  - pages:
    - AdminUsersPage (および AdminShell) — 左上 Brand コンポーネントを `<a href="${tenantURL('/admin')}">` でラップする。ダッシュボード ページ自身では active を示せるよう href は維持しつつ aria-current 等の状態を付与。既存の `<Brand compact />` レンダリングは流用する。
    - 右上のユーザアバター + 名前ブロックを Radix UI の `DropdownMenu` でラップする。トリガは丸アイコン (+username の現在表示)。メニュー項目: actorUsername (header label, non-interactive) / アカウント概要 (`/account/password` への link) / パスワードを変更 (`/account/password` への link) / ログアウト (`/end_session` への link)。既存のログアウト用 ghost ボタン (header 右端の `IconLogout` を単体で出していたもの) は撤去する。
    - UserDetails のアクション階層を以下に変更する。 (a) Profile セクションの "編集" ボタンを primary に格上げ する (現状 ghost variant + h-7 + xs → default variant + 通常サイズ + icon + 主ラベル "編集")。位置は Profile ヘッダ 右に維持し、視覚的にすぐ目に入る大きさにする。 (b) 「アカウント状態」セクションを廃止し、代わりに UserDetails のヘッダ右端 (Profile ヘッダ行) に kebab menu (⋮) を置く。メニュー項目: アカウントを無効化 / 再有効化 (wi-13 で導入する DisableUserDialog に橋渡し) / アカウントを 削除 (現行 DeleteUserDialog)。危険ラベルは menu item に red text + アイコンを付ける。full-width な destructive ボタンは UserDetails から消える (kebab menu に押し下げ)。 (c) 詳細パネルの縦方向高さが下がる。これにより削除導線も 折り返さずに見える位置 (= kebab menu) に入る。
  - components:
    - 新規: ui/src/components/ui/dropdown-menu.tsx を追加する (Radix UI `DropdownMenu` の薄ラッパ、shadcn 流の styling)。 既存の `Card` / `Button` / `Alert` と同じトーンで揃える。
- **api**:
  - 変更なし。既存 `/api/auth/account` の `preferred_username` / `sub` をアバターメニューにそのまま使う。
- **routing**:
  - 変更なし。`/admin` (dashboard) は wi-10 part 3 で既に追加済。
- **navigation**:
  - 既存 `adminNavItems` に変更なし。"概要" の sidebar 項目は ブランドリンクの代替動線として残す。
- **documentation**:
  - idmagic/ui/README.md には新規記述を増やさない。

## Out of Scope
- end user 側 (`/account/*` 等) の右上アバターメニュー。
- dark mode toggle / theme switcher / language picker のメニュー追加。
- tenant switcher (system_admin が複数 tenant を行き来する経路は 将来別 WI)。
- 通知センター (バナー / トースト) のメニュー化。
- UserDetails の Profile セクション自体のレイアウト変更 (`DetailRow` の構造は維持)。
- "Danger zone" セクションを再度 UserDetails 内に再配置する案 (kebab menu に押し下げる方針を採用する)。
- DropdownMenu のキーボードナビゲーション拡張 (Radix のデフォルト 挙動に依存する)。
- mobile / narrow viewport 向けレイアウト最適化。

## Verification
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- `go test ./internal/adapters/http/...` (in: idmagic)
  - reason: backend は触らないが、UI と通信する `/api/auth/account` / `/api/admin/users` の handler テストが落ちていないことを確認。
- 手動 1 (Brand リンク): `/admin/users` から左上 "RA Identity" を クリック → `/admin` (dashboard) に遷移する。
- 手動 2 (アバターメニュー): 右上アバターをクリック → DropdownMenu が開き、ログアウト・パスワード変更が選べる。閉じるとフォーカスが 戻る。
- 手動 3 (UserDetails): ユーザを選択 → 編集ボタンが primary として 大きく表示・kebab menu (⋮) を開くと無効化 / 削除が出る。Disable / Delete は wi-13 / wi-8 のダイアログを経由する。

## Risk Notes
UI 側の純粋な polish。backend / SCL に影響しない。
DropdownMenu を新規追加するため、radix-ui/react-dropdown-menu の
依存追加が発生する。bundle size は数 KB 程度の見込みで、既存の
Radix Primitives ライブラリと整合する。

kebab menu に押し下げることで、テスト用に "Disable / Delete を
繰り返す" 開発フローが menu 操作 1 つ増える。これは本 WI で許容する
(動線安全性のトレードオフ)。

ブランドの "active" 状態をどう示すかは UX 判断。現状 `/admin`
自身では aria-current を付ける程度で済ませる。

## Completion
- **Completed At**: 2026-06-16
- **Summary**:
  AdminShell のブランドを `/admin` へのリンクにし、ユーザー名とアバターを
  Radix DropdownMenu のトリガに変更した。ログアウト単独アイコンは廃止し、
  アカウント概要、パスワード変更、ログアウトをメニューに集約した。
  UserDetails は編集を primary action にし、無効化・再有効化・削除を
  kebab menu に移した。
- **Verification Results**:
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui lint`
    - result: ok (42 files)
  - `bun --cwd idmagic/ui build`
    - result: ok
  - `GOCACHE=/tmp/idmagic-cache go test -race ./...` (in: idmagic)
    - result: ok
  - 実ブラウザでの DropdownMenu フォーカス復帰と目視確認は未実施。
- **Affected Guarantees State**:
  - admin RBAC、CSRF、ユーザー操作 API は不変
  - DropdownMenu のフォーカス・キーボード操作は Radix primitive に委譲
  - 無効化と削除は menu 選択後も既存確認ダイアログを経由
