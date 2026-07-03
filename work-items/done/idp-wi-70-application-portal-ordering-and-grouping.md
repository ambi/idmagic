---
id: idp-wi-70-application-portal-ordering-and-grouping
title: "利用者ポータルのアプリケーション一覧に並び替えと分類を入れる"
created_at: 2026-06-27
authors: ["tn"]
status: completed
risk: low
---
# Motivation
[[wi-69-application-catalog-aggregate-and-assignment]] で利用者ポータルに
アプリケーション一覧 (icon 付きタイル) が入る。割当アプリが増えると、固定順では
目的のアプリを探しにくくなる。Okta の End-User Dashboard も Entra ID の
My Apps も、利用者がタイルを並べ替え、管理者がセクション (collection / category) で
分類してポータルを整理できる。

本 WI はポータル一覧の見つけやすさを担う UX 機能を分離して扱う。並び順 (利用者の
手動並び替えと既定の整列規則) と分類 (管理者が定義するカテゴリ/セクション) を導入する。
アプリケーション本体・割当・アイコンは wi-69 が所有し、本 WI はその上の表示順序と
グルーピングだけを足す。

# Scope
- **decision**: ADR 追補または軽量 ADR: 並び順の所有者を決める。既定整列 (アルファベット / 最近利用) は計算で出し、利用者の手動並び替えは per-user の表示設定として永続化する。 カテゴリは管理者が tenant 単位で定義し、Application に 0..N 個割り当てる。
- **scl**: ApplicationCatalog に ApplicationCategory / ApplicationOrdering を追加し、 per-user の手動順を IdentityManagement (User の portal preference) 側に置くか ApplicationCatalog に置くかを ADR で確定して反映する。, interface: 管理者の Category CRUD と Application への付与、利用者の ReorderMyApplications (手動順保存)。ListMyApplications に並び順とカテゴリを含める。, [object Object], [object Object]
- **go**: カテゴリの persistence (categories / application_categories テーブル、tenant scope)。, per-user の手動並び順の persistence と、未設定時の既定整列ロジック。
- **http**: /admin/application-categories の CRUD と Application への付与。, /api/account/applications/order の取得/更新。
- **ui**: [object Object], [object Object]

# Out of Scope
- アプリケーション本体・割当・アイコンの実装 ([[wi-69-application-catalog-aggregate-and-assignment]])。
- 属性/利用頻度に基づく自動レコメンド並び替え。初期は手動順 + 単純な既定整列のみ。
- ポータルのテーマ/ブランディング全般。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: 複数アプリを割当 → カテゴリを作成し付与 → 利用者ポータルでセクション表示と 手動並び替えが保存・再現されることを確認する。

# Risk Notes
ポータル表示順とグルーピングの UX 機能で、認証/認可の wire behavior には影響しない。
カテゴリ表示が割当境界を越えてアプリを露出しないことだけ確認する。
wi-69 の Application / Assignment 実装に依存するため、後続として実装する。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  利用者ポータルのアプリ一覧に「並び替え」と「分類」を入れた。並び順 (slice 1) は
  ApplicationCatalog が tenant_id + user_sub をキーに per-user の手動順を所有し、既定は
  name 昇順。手動順を割当済み visible アプリの上に重ね、未割当 id は除外、手動順に無い
  アプリは name 昇順で末尾に付ける。分類 (slice 2) は管理者が tenant 単位で
  ApplicationCategory を定義し Application に 0..N 個付与する。利用者ポータルはカテゴリ
  定義 (position 昇順) でタイルをセクション表示し、未分類は「その他」に集める。所有関係は
  ADR-069 が定める。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
