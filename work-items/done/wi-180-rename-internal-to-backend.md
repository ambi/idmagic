---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-11
---

# トップレベル `internal/` を `backend/` に、`ui/` を `frontend/` に改名し、`cmd/` を `backend/cmd/` へ移す

## Motivation

`internal/` は Go 特有のディレクトリ名で他言語の開発者には意味が伝わりにくく、兄弟の
`ui/` との対比も非対称。Go の `internal/` は「module 外からの import を compiler が
拒否する非公開境界」を表すが、**idmagic は単一モジュールのアプリケーション
（`cmd/idmagic`・`cmd/idmagic-relay` のバイナリ）であり、外部から import される
ライブラリではない**。よってトップレベル `internal/` の import 境界は実質的に何も
守っておらず、Go 儀礼として残っているだけ。ADR-068 項目4 は「`internal/` を維持し
`src/` へ置き換えない」と決めていたが、その論拠（module 外 import 拒否）はアプリには
当てはまらない。役割ベースで明快な `frontend/` ↔ `backend/` の対比へ移行する。

## Scope

- 機能・振る舞い変更なし。**SCL 規範振る舞いは不変**のため `spec/scl.yaml` 編集は不要
  （`scl-render` も不要）。
- `decisions/`: 新規 ADR を追加（ADR-068 項目4 を supersede）。
- `ARCHITECTURE.md`: ディレクトリ構成マップの同期（CLAUDE.md 規約により必須）。
- Go ソース全体の import path（`github.com/ambi/idmagic/internal/...` →
  `.../backend/...`、約 405 ファイル / 1,130 import）。
- ビルド・設定: `justfile`, `deploy/docker/Dockerfile`,
  `deploy/docker/docker-compose.dev.yaml`, `sqlc.yaml`, `.golangci.yml`,
  `.github/workflows/idmagic-ci.yaml`。
- CI 検証対象のパス文字列: `backend/shared/spec/assurance_manifest.go`（旧
  `internal/shared/spec/assurance_manifest.go`、55箇所以上）。

## Out of Scope

- SCL（`spec/scl.yaml`）の規範定義の変更。
- 歴史的 ADR / 過去 work-item 内の `internal/...` 参照の一括書き換え（ADR-068 の前例に
  従い監査記録として保持する）。
- context-locality 再構成そのもの（ADR-089/090/091, wi-172〜179）の内容変更。本 work-item は
  トップレベルのディレクトリ改名のみ。

## Plan

移行後のルート構成:

```
idmagic/
  backend/          <- 旧 internal/
    cmd/            <- 旧 cmd/
      idmagic/ idmagic-relay/
    application/ authentication/ bootstrap/ identitymanagement/
    oauth2/ relay/ saml/ scim/ shared/ tenancy/ wsfederation/
  frontend/         <- 旧 ui/
  decisions/  deploy/  spec/  tools/  work-items/
```

- module path `github.com/ambi/idmagic` は不変。`/internal/` セグメントのみ `/backend/` に変わる。
- ディレクトリ移動は履歴保持のため `git mv` を使う。
- import path は module prefix が一意なので安全に一括置換できる。
- **却下した代替案**: `src/`（一般的すぎ、cmd もソースでは？という非対称、ADR-068 で既に却下）、
  `app/`（ui/frontend との対比が backend ほど明快でない）、現状維持（ユーザーの違和感が残る）。
- **タイミング**: context-locality ロールアウト（wi-173〜179）が進行中。本改名は純機械的な
  移動なので、**単一の atomic commit** として並行 work-item ブランチが少ないタイミングで実施し、
  未マージのブランチは後で rebase して衝突を最小化する。
- `ra` CLI / `tools/` は `internal` をハードコードしておらず影響なし（確認済み）。UI を
  Go の `//go:embed` で埋め込む箇所も無い（確認済み）。

## Tasks

- [x] T001 [ADR] `decisions/ADR-092-*.md` を作成。`internal/`→`backend/`、`cmd/`→`backend/cmd/`、
      `ui/`→`frontend/` を決定し、ADR-068 項目4 を supersede（アプリでは import 境界の実益が無い
      という再評価）。ADR-068/070 には `superseded_by` 追記のみ。SCL 不変を明記。
- [x] T002 [Move] `git mv internal backend` / `git mv cmd backend/cmd` / `git mv ui frontend`。
- [x] T003 [Go] import path 一括置換:
      `grep -rl --include='*.go' 'github.com/ambi/idmagic/internal/' . | xargs sed -i '' 's#github.com/ambi/idmagic/internal/#github.com/ambi/idmagic/backend/#g'`。
- [x] T004 [Build] `justfile` 更新（ldflags の version パス、build ターゲット
      `./cmd/...`→`./backend/cmd/...`、UI 系レシピ約12個の `cd ui`→`cd frontend`）。
- [x] T005 [Build] `deploy/docker/Dockerfile`（ldflags `idmagic/internal/...`→`idmagic/backend/...`、
      build ターゲット、UI ステージの `ui/`→`frontend/`）と `docker-compose.dev.yaml` の `ui` サービス。
- [x] T006 [Config] `sqlc.yaml`（2箇所）、`.golangci.yml`（除外パス10箇所）、
      `.github/workflows/idmagic-ci.yaml`（`working-directory: ui`→`frontend`）を更新。
- [x] T007 [Config] `backend/shared/spec/assurance_manifest.go` のパス文字列 55箇所以上の
      先頭 `internal/`→`backend/`（coherence 検証のファイル存在チェック対象）。
- [x] T008 [Docs] `ARCHITECTURE.md`（`new-architecture` skill）を同期。`CLAUDE.md` の
      Repository Layout に `backend/` `frontend/` を明記。
- [x] T009 [Verify] 下記 Verification を実行し全緑を確認。

## Verification

- `just verify-go` — format-check / lint（golangci 除外パス）/ typecheck / build が緑。
- `just build-go` — 全パッケージビルドで新 import path 解決を確認。
- `just test-go` — テスト緑（assurance manifest のパス整合を含む）。
- `just check-ids` / `just yaml-check` — RA/SCL の ID・YAML 整合（SCL 不変なので影響無いはず）。
- `just verify-ui` — `cd frontend` 化した UI の format/lint/typecheck/build が緑。
- `just verify` — 全体スイートの最終確認。
- `grep -rn 'idmagic/internal/' --include='*.go' .` が空（旧 import 残存なし）。
- `git log --follow backend/oauth2/<某ファイル>` で履歴保持を確認。

## Risk Notes

- **広範だが機械的**: 405 ファイル・1,130 import と多いが、置換対象は一意な module import
  prefix と限られた設定/文字列であり、`just build-go`・`just test-go` で網羅的に検証できる。
- **進行中ロールアウトとの衝突**が主リスク。単一 atomic commit 化と、並行ブランチの
  rebase で軽減する。着手前に未マージの wi-173〜179 ブランチ状況を確認する。
- `assurance_manifest.go` のパス文字列と `.golangci.yml` の正規表現パターンは
  自動リネーム対象から漏れやすいので、専用タスク（T006/T007）で明示的に扱う。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: Go 実装と entry point を `backend/`、React UI を `frontend/` へ履歴を保って移動し、Go import path、ビルド・Docker・CI・sqlc・検証 manifest・運用文書を同期した。ADR-092 を追加し、ADR-068/070 の関連判断を supersede として記録した。SCL の規範振る舞いは不変。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just build-go` - passed
  - `just verify-go` - passed
  - `just verify-ui` - passed (38 test files, 197 tests)
  - `just verify` - passed
  - `rg 'github.com/ambi/idmagic/internal/' --glob '*.go' .` - no matches
