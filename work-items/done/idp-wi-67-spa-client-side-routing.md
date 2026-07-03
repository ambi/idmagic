---
id: idp-wi-67-spa-client-side-routing
title: "管理コンソール / アカウントポータルの画面遷移を client-side routing 化して全リロードを無くす"
created_at: 2026-06-27
authors: ["tn"]
status: completed
risk: medium
---
# Motivation
管理コンソール (`/admin/*`) とアカウントポータル (`/account/*`) のサイドバー / パンくず
リンクは素の `<a href>` で、クリックごとに SPA 全体がフルリロードされる。ブート時の
`loadPageData()` が現在の URL を見てそのページのデータを取り直す設計のため、遷移ごとに
バンドル再評価とページデータの再取得 (例: ダッシュボードは毎回 4 本の API を並列取得)
が走り、体感的な遅延になっている。

この遅延は OIDC RP 化 ([[wi-66-portals-as-oidc-rp]]) より前から存在する全リロード方式が
主因で、OIDC とは独立の課題である。本 WI は admin / account の画面遷移を client-side
routing (SPA ナビゲーション) 化し、フルリロードとページデータ全取得を無くして遷移を
即時化する。

現状の SPA dispatcher は「ブート時に 1 つの PageData を解決し、`data.kind` に一致する
route だけを描画する」静的構造 ([[wi-22]] の SPA スモーク参照)。本 WI はこれを、URL 変化に
応じて per-route loader がデータを取得する動的ルーティングへ作り変える。`api/pageData` の
巨大な分岐を route 単位の loader に分解し、@tanstack/react-router の本来のルーティングを使う。

# Scope
- **decision**: ルーティング方式の決定: ブート時単発 PageData から、route 単位 loader + SPA ナビゲーションへ移行する方針と、サーバ駆動の `kind` 解決をどう廃止/縮小するかを 確定する (必要なら ADR)。
- **go**: サーバ側の HTML 配信が `kind` を前提にしている箇所があれば、SPA が任意の admin / account パスを直接開けるよう調整する (deep link / リロード耐性)。API 契約は不変。
- **ui**: サイドバー / パンくず / メニューのリンクを SPA ナビゲーション (router.navigate / Link) に置き換え、フルリロードを廃止する。, `api/pageData.ts` の単発分岐を route 単位の loader に分解し、遷移時はその route の データだけを取得する。共有コンテキスト (`/api/auth/account` など) はキャッシュ/共有して 重複取得を避ける。, OIDC RP のトークン取得 (ensureLoggedIn) を、ブートだけでなく SPA ナビゲーション時にも route guard として機能させる ([[wi-66-portals-as-oidc-rp]] と整合)。, 直接リロード・ブラウザ戻る/進む・deep link が壊れないことを担保する。
- **documentation**: ui/ARCHITECTURE.md の SPA dispatcher 記述を新ルーティングモデルに更新する。

# Out of Scope
- 認証ロジック自体の変更 ([[wi-66-portals-as-oidc-rp]] が所有)。本 WI は遷移方式に閉じる。
- auth-flow 画面 (login / consent / totp / device / callback) のルーティング刷新。まず admin / account に閉じる。
- サーバ側レンダリング (SSR) の導入。

# Verification
- [object Object]
- [object Object]
- [object Object]
- 手動: サイドバー遷移がフルリロードせず即時であること、deep link / リロード / 戻る進むが壊れないこと。

# Risk Notes
ルーティングモデルの作り替えは admin / account 全ページの描画経路に触れるため回帰面が
広い。route 単位 loader への分解は段階的に行い、各 route で deep link とリロード耐性を
確認する。OIDC トークン guard を SPA ナビゲーション経路にも通すこと。

# Completion
- **Completed At**: 2026-06-27
- **Summary**:
  SPA を TanStack Router の loader ベースの client-side routing に作り替えた。各パスは
  共有 loader (resolvePageData) で「その route のデータだけ」を取得し、PageView が
  解決済み PageData の kind に応じてページを描画する。サイドバー / パンくず / アカウント
  メニュー / ロゴのナビゲーションを `<Link>` に置き換え、画面遷移でドキュメントを再読込
  しない。OIDC ログイン guard (ensureLoggedIn) は loader 内で動くため、初期ロードと
  in-app 遷移の両方に適用される。pageData は window.location 依存を排し、遷移先 location
  (pathname / searchStr) を受け取る resolvePageData に分離した。markPage は Suspense 境界の
  内側で呼び、lazy chunk のロード完了 (ページ DOM 実在) と meta 表明を一致させた。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
