---
name: implement-work-item
description: Implement a chosen Regenerative Architecture work item end to end — inner layers to outer, tests per layer, then UI, verify green, add completion, move to done, commit. Use when starting to build a selected work item, or when the user asks to implement / build a work item — e.g. "implement wi-NN", "wi-NN を実装して", "wi-NN をやって", "ワークアイテムを実装". Companion to scl-change (spec first).
---

# ワークアイテムの実装フロー

対象ワークアイテムが決まってから、SCL の内層から外層・UI・完了記録・コミットまでを
一定の順序で回す。**手順を毎回考え直さない**——この順序と検証ゲートに従う。

## 0. コンテキスト衛生（最初に決める）

大きなトークン消費と思考の遅さは、たいてい「無駄な全文読み」と「1 セッションに文脈を
積み過ぎ」で起きる。着手前に次を徹底する。

- **メタドキュメントは全文で読まない。** RA / SCL は節単位のリファレンス。必要な節だけを
  `rg '^#{2,3} ' <file>` で探し、その行域だけ読む（節マップは `CLAUDE.md §1`）。
- **feature の現状仕様は該当 `scl.yaml` だけ読む**（例 `<app>/spec/scl.yaml`）。
  リポジトリ全体を舐めない。
- **コードベース横断の探索はサブエージェント（`Explore`）に委譲し、結論だけ受け取る。**
  検索の生出力で本スレッドの文脈を埋めない。
- **中規模以上なら実装前に Plan / Tasks を正本化する。** 複数 scenario、RA 3 層以上、
  DB migration / 認可 / 外部契約 / 破壊的変更、1 セッションで終わる確信がない作業、
  複数サブシステム検証は中規模以上として扱う。ワークアイテムに `# Plan` と `# Tasks` が
  なければ先に追記し、過大なら work-item 分割を提案する。
- **Tasks の checkbox は進捗正本。** タスクを完了したら同じ work-item の checkbox を更新する。
  途中でセッションが切れても、次の AI は `# Tasks` と直近の検証結果から再開する。
- **層の区切りは自然なコンパクト点。** 1 つの層を実装・検証してグリーンにしたら、
  必要に応じ `/clear` して次の層に入ると、文脈が単調増加しない。

## 1. 内層から外層へ（各層でテストを書く）

RA の 5 層（`REGENERATIVE_ARCHITECTURE.md §3`）を内側から。**先に SCL、後で実装**。

1. **Spec Core (SCL)** — `scl-change` Skill に従い `scl.yaml` を先に更新。触れた節を
   ワークアイテムの `scope` に列挙。`just yaml-check` で検証する。並列 worktree の
   work-item branch では派生物を commit せず、必要な場合だけ確認用に `just scl-render` を
   実行する。生成物の同期 commit は integration branch / merge queue / main 直前で行う。
2. **Decision Record & Architecture** — 非自明な設計判断があれば `new-adr` Skill で ADR を残す。
   コア構造（コンテキスト・モジュール・スタック・ディレクトリ規約）に触れたら、`new-architecture`
   Skill に従い該当 `ARCHITECTURE.md` を現状へ同期する（書式は `ARCHITECTURE_FORMAT.md`）。同期は
   努力目標ではなく検証ゲートであり、`just yaml-check` の Architecture 整合検査（modules パス実在・
   realizes の SCL 要素解決・contexts 整合）を通すこと。
3. **Application Logic** — ドメイン／ユースケース実装。単体テストを同時に書く。
4. **Adapter Layer** — HTTP / 永続化などの境界実装 + テスト。
5. **Runtime & Infra** — 設定・配線・デプロイ関連（必要なら）。
6. **UI** — 最後。React + Tailwind + Radix/shadcn（`CLAUDE.md §2` Default Tooling）。

各層を終えたらその層のテストを green にしてから次へ進む（層をまたいで未検証を溜めない）。
`# Tasks` がある場合は、層の完了ごとに対応 checkbox を更新する。

## 2. 検証ゲート（全部グリーンで完了）

コマンドは `just`（`justfile` = 人間 / AI 共通のコマンドマップ）を使う。

- SCL / YAML: `just yaml-check`
- Go: app repo / app recipe の lint + race テスト
- UI: app repo / app recipe の format / lint / typecheck / build
- 一括: `just verify`

Codex 環境では `golangci-lint` が managed filesystem sandbox 内で
`context loading failed: no go files to analyze` になり、`GOLANGCI_LINT_CACHE`
や対象 package 指定だけでは回避できない。app repo 側の `golangci-lint` を
直接・間接に起動する検証は、必ず
承認付きで sandbox 外実行にする。

## 3. 完了処理（手順 5〜6）

1. ワークアイテムに `completion` を追記し `status: completed` にする。証跡の粒度は
   `new-work-item` Skill §1.3 に従う。
2. ファイルを `work-items/done/`（または該当コンテキストの `.../work-items/done/`）へ移す。
   id は変えない。
3. `commit` Skill でコミット（Conventional Commits・subject / body とも英語）。
   ユーザから指示があるまで push しない。
