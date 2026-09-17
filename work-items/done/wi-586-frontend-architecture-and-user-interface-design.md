---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-16
priority: p1
depends_on: [wi-583-normalize-design-document-terminology]
change_kind: docs
evidence_policy: risk-based-v3
spec_impact:
  kind: none
  reason: "現行のフロントエンド構造と画面の設計規則を書き下すだけで、実装も画面の振る舞いも変えない。"
documentation_impact:
  level: none
  reason: "変わるのは開発者が読む設計文書だけで、利用者が観測する振る舞い、API、設定キーはどれも変わらない。"
  references: []
initial_context:
  specification:
    - DOCUMENTATION_GUIDE.md
    - docs/design/application/README.md
    - docs/design/application/user-interface.md
    - docs/design/application/api-guidelines.md
    - docs/structure.md
    - docs/architecture/logical.md
    - docs/architecture/runtime.md
  typespec: []
  source:
    - frontend/README.md
    - frontend/src/features/README.md
    - frontend/src/router.tsx
    - frontend/src/routes/admin/users.tsx
    - frontend/src/routes/admin/route.tsx
    - frontend/src/routes/-guards.ts
    - frontend/src/api/core.ts
    - frontend/src/lib/usePaginatedList.ts
    - frontend/src/lib/lengthLimits.ts
    - frontend/src/lib/i18n/context.tsx
    - frontend/src/lib/i18n/errorMessage.ts
    - frontend/src/components/ui/page-navigation.tsx
    - frontend/src/components/ui/load-more.tsx
    - frontend/src/components/ui/toast.tsx
    - frontend/src/components/StepUpDialog.tsx
    - frontend/src/features/admin-users/AdminUserDialogs.tsx
    - frontend/vite.config.ts
    - frontend/Caddyfile
    - frontend/Dockerfile
    - tools/workspace/src/document-layout.ts
  tests:
    - frontend/src/devProxy.test.ts
  stop_before_reading:
    - frontend/src/api/admin.ts
    - frontend/src/types.ts
    - frontend/src/routeTree.gen.ts
---

# フロントエンドのアーキテクチャを独立した設計文書にし、UI 設計を書き足す

## Motivation

フロントエンドの設計は、いま三か所に断片として存在する。
[ユーザーインターフェース設計](../../docs/design/application/user-interface.md)は 12 行で、機能ディレクトリの配置を一文、情報構造を二文、国際化を二文書いて終わる。
[構造](../../docs/structure.md#フロントエンドのコンポーネント構造)の該当節は 4 行で、`features/` と `components/` の区別だけを述べる。
[ランタイムアーキテクチャ](../../docs/architecture/runtime.md)はフロントエンドゲートウェイを実行単位として一行で挙げる。

実装は `frontend/src/features/` に 26 個の機能スライスを持ち、ビュー、局所コンポーネント、ヘルパー、テスト、`*.i18n.ts` の辞書をスライス内に同居させる Vertical Slice の構造になっている。
ルーティングは `frontend/src/routes/` のファイルベースで、サーバー状態は `frontend/src/api/` の呼び出しと `usePaginatedList` のようなフックが持つ。
この構造は実装を読めば分かるが、**何をスライスに入れ、何を `components/`、`lib/`、`api/` へ出すか**の境界規則はコードから復元できない。新しい機能を足す人は、既存スライスの真似をするしかない。

UI 側も足りない。
表示状態（読み込み中、空、エラー、権限なし、部分的失敗）、フォームと検証、破壊的操作の確認、一覧のページング操作、通知、日時と数値の表示形式といった、画面を跨いで揃えるべきものが書かれていない。

## Scope

- 新しい文書 `docs/design/application/frontend.md`「フロントエンド設計」を作り、フロントエンドのアーキテクチャを持たせる。
- [ユーザーインターフェース設計](../../docs/design/application/user-interface.md)を、画面を跨ぐ UI の規範として書き足す。
- [構造](../../docs/structure.md#フロントエンドのコンポーネント構造)のフロントエンド節を、コードの配置と依存方向に絞る。
- [論理アーキテクチャ](../../docs/architecture/logical.md)と[ランタイムアーキテクチャ](../../docs/architecture/runtime.md)から、フロントエンド設計への到達経路を置く。
- [アプリケーション設計の索引](../../docs/design/application/README.md)と `DOCUMENTATION_GUIDE.md` §4.12 の記述を追従させる。
- `frontend/README.md` と `frontend/src/features/README.md` が持つ設計の記述（ライブラリの選定理由、ルーティング、コンテナと表示用コンポーネントの分割、画面の設計指針、ナビゲーションの規約、ローカライゼーション）を新しい二文書へ移し、README には開発時に実行する手順だけを残す。同じ規則を二か所に置かないためである。
- 正準文書の閉じた集合（`tools/workspace/src/document-layout.ts`）へ `frontend.md` を加える。

## Out of Scope

- 実装の変更。既存スライスの再配置、依存の整理、コンポーネントの抽出は含めない。
- 画面ごとの URL、表示項目、文言、色や余白の値。実装とローカライズ辞書から導ける（`DOCUMENTATION_GUIDE.md` §3.9）。
- アクセシビリティの規範そのもの。[全体の標準仕様](../../docs/standards.md)が持ち、自動検査は [[wi-292-wcag22-accessibility-conformance-and-automated-checks]] が扱う。
- デザイントークンの値とテーマ。[[wi-196-tenant-admin-console-theming]] が扱う。
- API ガイドライン。[[wi-585-rewrite-api-rules-as-complete-api-guidelines]] が扱う。

## Design

三つの文書の責務を次のとおり分ける。
いまフロントエンドの記述が薄いのは、書く場所が決まっていないためである。

| 文書 | 持つもの |
| --- | --- |
| `docs/structure.md` | ディレクトリの配置と依存の向き。コードの構造だけ |
| `docs/design/application/frontend.md` | スライスの境界規則、ルーティング、データ取得と状態、ビルドと配信、テストの置き場所 |
| `docs/design/application/user-interface.md` | 画面を跨ぐ情報構造、操作、表示状態、フォーム、アクセシビリティ、国際化 |

`frontend.md` を新設するのは、構造と設計を分けるためである。
`structure.md` は「その境界がコードのどこにあり、どちら向きに依存しているか」を持つ文書であり、状態管理やデータ取得の設計はその問いに答えない。
一方 `user-interface.md` は利用者から見た規範を持つ文書であり、ディレクトリ構造はその問いに答えない。
いまフロントエンドの設計は、どちらの問いにも属さないために両方から落ちている。

`frontend.md` が持つものを挙げる。

- **Vertical Slice の境界規則**。`frontend/src/features/<feature>/` に入るもの（ビュー、局所コンポーネント、表示ロジック、テスト、`*.i18n.ts`）と、出るもの（`components/` の横断部品、`lib/` の横断ヘルパー、`api/` のバックエンド呼び出し）の判定基準。スライス間の直接 import を許すかどうか。
- **ディレクトリの木構造の実例**。`admin-users` のような実在するスライス一つを木で示し、各ファイルがどの役割かを注記する。
- **仕様上の機能とスライスの対応**。スライス名が Bounded Context 名と一致しない理由（スライスは画面の単位、Context はモデルの単位）。
- **ルーティング**。`frontend/src/routes/` のファイルベース構成、`@tanstack/router-generator` による生成物、スライスのビューを route が呼ぶ向き、認証が要る経路の守り方。
- **サーバー状態と画面状態**。`frontend/src/api/` のモジュールが返す型の出どころ、一覧の取得とページングを担うフック、キャッシュを持たない選択とその理由。
- **ゲートウェイとの関係**。同一オリジンで配信される理由、開発時の Vite プロキシと本番の Caddy の対応、`/realms/:tenant_id` 配下の転送。
- **ビルドと配信**。Vite のビルド成果物、静的アセットの配置、CSP を SPA の HTML にだけ付ける理由。
- **テストの置き場所**。スライス内の `*.test.tsx` と `frontend/src/test/` の使い分け。テストダブルの方針は [テスト方針](../../docs/development/testing.md#テストダブル)が持つため再掲しない。

`user-interface.md` が持つものを挙げる。

- 画面の情報構造（対象テナント、対象資源、現在の状態、実行できる操作、操作結果の識別順序）。既存の記述を残す。
- **表示状態**。読み込み中、空、エラー、権限なし、部分的失敗のそれぞれで何を見せるか。
- **破壊的操作**。確認の要否の判定、確認に何を含めるか、取り消せる操作と取り消せない操作の見せ方。
- **フォームと検証**。いつ検証するか、サーバーが返した違反をどこへ出すか、文字列長の上限が入力の補助でしかないこと（[API ガイドライン](../../docs/design/application/api-guidelines.md)が正本）。
- **一覧**。ページング操作、絞り込みと並べ替えの見せ方、件数の扱い。
- **通知とエラー表示**。エラーコードから文言を引く仕組み、辞書に無いコードのときに英語の `detail` を出すこと。
- **国際化**。`ja` と `en` を同じ変更で更新すること、辞書がスライス内にあること、日時と数値の表示形式。
- **アクセシビリティ**。規範は `docs/standards.md` が持ち、ここは実装がそれを満たす手段（セマンティック HTML、キーボード操作、可視フォーカス、ラベルとエラーの関連、状態通知）を持つ。

採らない案を二つ記録する。
一つは、フロントエンドのアーキテクチャを `docs/structure.md` に全部置く案である。`structure.md` はコードの配置と依存方向を持つ文書であり、データ取得や表示状態の設計を混ぜると、この文書が何に答えるのかが曖昧になる。
もう一つは、`user-interface.md` を拡張して構造も持たせる案である。読者が違う。UI の規範は画面を作る人が読み、スライスの境界規則はコードを足す人が読む。

## Plan

1. 現行の `frontend/src/` を読み、スライス、`components/`、`lib/`、`api/`、`routes/` の実際の分かれ方を観測する。
2. 観測から境界規則を言語化し、既存の配置で説明できない例を洗い出す。
3. `frontend.md` を書き、木構造の実例を置く。
4. `user-interface.md` を、画面を跨ぐ規範として書き足す。
5. `structure.md` のフロントエンド節を配置と依存方向へ絞り、`frontend.md` を参照させる。
6. 索引と `DOCUMENTATION_GUIDE.md` §4.12 を追従させる。

## Tasks

- [x] T001 [Design] 現行の `frontend/src/` の分かれ方を観測し、境界規則を言語化する。
- [x] T002 [Docs] `docs/design/application/frontend.md` を作る。
- [x] T003 [Docs] `docs/design/application/user-interface.md` を書き足す。
- [x] T004 [Docs] `docs/structure.md` のフロントエンド節を配置と依存方向へ絞る。
- [x] T005 [Docs] 論理・ランタイムアーキテクチャ、アプリケーション設計の索引、`DOCUMENTATION_GUIDE.md` §4.12 を追従させる。
- [x] T006 [Verify] リンク、仕様、UI 依存、全体検証を通す。

## Verification

- `mise run check-links`
- `mise run check-spec`
- `mise run check-ui-dependencies`
- `mise run verify`

## Risk Notes

観測した現状をそのまま規則として書くと、既存の逸脱まで規則になる。境界規則は「こう置く」と決め、現行で説明できない配置は逸脱として名指す。既存の状態を規則に合わせて書き換えない。

木構造の実例は、ファイルが増減した時点で古くなる。実例には実在するスライスを一つだけ使い、全スライスの一覧は置かない。

ライブラリ名と版を文書へ写すと、更新のたびに古くなる。`package.json` が正本であり、文書が持つのは「ファイルベースのルーターを使う」「サーバー状態のキャッシュライブラリを持たない」といった設計上の選択と理由だけにする。

`user-interface.md` にアクセシビリティの規範そのものを書くと、`docs/standards.md` と二重になる。規範は参照し、ここには実装手段だけを置く。

## Completion

- **Completed At**: 2026-09-18
- **Summary**:
  フロントエンドの設計が、`docs/design/application/frontend.md` と `docs/design/application/user-interface.md` の二文書に分かれて置かれるようになった。
  `frontend.md` は、コードの置き場所を決める判定表、ディレクトリ間の依存の向き、機能スライスどうしを参照しない規則、`admin-users` の木構造の実例、機能スライスと Bounded Context が一致しない理由、ルーティングと認証ガード、サーバー状態のキャッシュを持たない選択と状態の置き場所、コンテナと表示用コンポーネントの分割、ビルドと配信、テストの置き場所を持つ。
  `user-interface.md` は、情報構造とナビゲーションに加えて、表示状態、破壊的な操作、フォームと検証、一覧、エラーの文言、国際化、`docs/standards.md` の WCAG 2.2 の各行を満たす実装手段を持つ。
  `frontend/README.md` と `frontend/src/features/README.md` にあった設計の記述は二文書へ移し、README には開発時に実行する手順だけを残した。
  README は削除ボタンに `variant="outline" tone="danger"` を使うと書いていたが、ボタンに `tone` はなく、実装は `variant="destructive"` を使っていたので、実装に合わせて書いた。
  `docs/structure.md` のフロントエンドの節はディレクトリの表と依存の向きに絞り、論理アーキテクチャ、ランタイムアーキテクチャ、アプリケーション設計の索引、`DOCUMENTATION_GUIDE.md` §4.12 からの参照を加えた。
  正準文書の閉じた集合に `frontend.md` を加えた。
  Motivation はサーバー状態を `usePaginatedList` のようなフックが持つと書いていたが、観測するとこのフックと `LoadMoreButton` はどの画面からも使われておらず、一覧のページングは検索パラメーターと `PageNavigation` が持っていた。
  規則に従わない現行の箇所は、逸脱として文書に名指しした。
  機能スライス間の参照 2 件、`lib/` から `components/` への参照、使われていないページング補助、`window.confirm` による確認 4 か所、`useFormatters` を通さない日時の整形である。
  依存の向きを検査する仕組みはなく、その旨を文書に書いた。
  `standards.md` の見出しへのフラグメント付きリンクは、描画された仕様ページでアンカーが解決されず `check-rendered-spec` が拒否したので、フラグメントなしのリンクにした。
  `mise run spec-diff` は規範の変更を報告しない。
- **Acceptance RED Evidence**:
  - **Test**: N/A: 設計文書の追加であり、規範のプロダクト要求を持たない。代わりに `mise run check-spec` の文書検査が失敗した
  - **Requirement**: N/A: 規範のプロダクト要求を持たない
  - **Observed Failure**: `docs/design/application/frontend.md: not a canonical document; docs/design/application/ holds only README.md, api-guidelines.md, design-guidelines.md, user-interface.md`
  - **Detection Reason**: 正準文書の集合が閉じており、登録していない文書を拒否する
- **Unit RED Evidence**:
  - **Test**: N/A: コードの振る舞いを変えない
  - **Requirement**: N/A: 規範のプロダクト要求を持たない
  - **Observed Failure**: N/A: 単体の境界がない
  - **Detection Reason**: N/A: 単体の境界がない
- **Change-Resistance Results**:
  文書と正準文書の一覧だけの変更であり、変異テストの対象になる Go のコードはない。
  文書の記述は、`frontend/src/` の import、`Caddyfile`、`vite.config.ts`、部品の実装と突き合わせて確かめた。
- **Verification Results**:
  - `mise run check-spec` - 登録前は failed、登録後は passed
  - `mise run check-links` - passed
  - `mise run check-terminology` - passed（225 文書）
  - `mise run check-work-item-references` - passed
  - `mise run check-ui-dependencies` - passed
  - `mise run verify` - passed
  - `mise run spec-diff` - 規範の変更なし
