---
depends_on: [wi-96-bulk-user-import-csv]
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
---

# Docker なしの標準開発環境で durable worker を稼働させる

## Motivation
`just dev` は API と UI を memory mode で起動するため、別プロセスの worker と JobRepository を共有できず、CSV import などの非同期機能を画面から確認できない。Docker Compose の完全スタックは正しいが、日常開発には起動負荷が大きい。Docker や常駐サービスを要求せず、本番と同じ API / worker 分離と PostgreSQL durable queue を再現する標準開発経路が必要である。

## Scope
- `spec/contexts/jobs.yaml` の `invariants` / `scenarios` に標準開発環境の共有 queue と worker 可用性を追加する。
- ADR-099 に embedded PostgreSQL + miniredis を使う開発限定構成を追記する。
- Docker 不要の dev-infra executable を追加し、embedded PostgreSQL、schema、miniredis を起動する。
- `just dev` / `dev.sh` で dev-infra、API、worker、Vite を lifecycle 管理する。
- 従来の軽量 memory 経路を `just dev-memory` として残し、durable jobs 非対応を明示する。
- README に初回 download、ポート、各 dev recipe の用途を記載する。

## Out of Scope
- production の PostgreSQL / Valkey を embedded 実装へ置換すること。
- Kafka、OpenTelemetry Collector、SMTP など Docker Compose が提供する完全統合環境。
- 開発データの再起動間永続化。

## Plan
- `backend/cmd/idmagic-dev-infra` が embedded PostgreSQL を `127.0.0.1:55432`、miniredis を `127.0.0.1:56379` で起動し、schema 適用後に ready file を作成する。
- `dev.sh` は ready file を待ってから API と worker を共通の postgres/valkey URL で別プロセス起動する。子プロセス異常終了と signal を全体 shutdown に伝播する。
- miniredis は既存 Valkey adapter の Lua / TTL / single-use テストが通る開発・テスト限定互換実装として扱う。
- `just dev-memory` は API + Vite のみを従来どおり起動し、非同期 job 非対応を起動時に表示する。

## Tasks
- [x] T001 [SCL/Decision] Jobs の開発時保証を追加し、ADR-099 を同期する。
- [x] T002 [Infra] embedded PostgreSQL + miniredis dev-infra と単体テストを実装する。
- [x] T003 [Dev] `just dev` の API / worker / UI lifecycle と `just dev-memory` を実装する。
- [x] T004 [Docs] README に Docker 不要開発環境と制約を記載する。
- [x] T005 [Verify] job end-to-end smoke と `just verify` を通す。

## Verification
- `just yaml-check`
- `just verify-go`
- `just verify-ui`
- `just verify`
- 手動: `just dev` で CSV dry-run Job が Queued → Succeeded へ進むことを確認する。

## Risk Notes
embedded PostgreSQL は初回だけバイナリ取得が必要で、miniredis は Valkey の完全互換ではない。起動失敗は fail-fast とし、miniredis は既存 adapter 契約テストで必要コマンドを固定する。production と完全に同じ運用構成の確認は `just dev-compose` が引き続き担う。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  Docker を使わず embedded PostgreSQL と miniredis を共有 TCP endpoint として起動し、API・worker・Vite を別プロセスで lifecycle 管理する標準 `just dev` を実装した。従来の API + UI memory mode は `just dev-memory` に分離し、durable jobs 非対応を明示した。fresh PostgreSQL で default tenant より先に signing key を作成していた bootstrap 順序も修正した。
- **Affected Guarantees State**:
  標準開発環境では API と worker が同一の PostgreSQL JobRepository を共有し、Docker なしでも enqueue された Job が worker により終端状態まで処理される。memory mode は明示的な限定経路であり durable jobs を提供しない。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just test-go` — passed（embedded PostgreSQL 上の enqueue → runner → Succeeded を含む）
  - `just verify` — passed（lint、race tests、UI format/lint/typecheck/unit/build を含む）
  - `ADDR=:18081 just dev` — API、worker、embedded PostgreSQL、miniredis、Vite の起動と graceful shutdown を確認
  - `ADDR=:18082 just dev-memory` — API + Vite のみの起動と durable jobs 非対応表示を確認
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: macOS、Docker 不使用、embedded PostgreSQL 18.3、miniredis
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。自動テスト結果と標準出力で確認。
