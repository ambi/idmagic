---
status: completed
archived_from: d1b45ea7499229939b03ccd6d5c4ed3a940a6712
---

# 大型 bounded context に feature 垂直スライス層を導入する規約を定め、idmanagement をパイロット変換する

feature 垂直スライスの規約を ADR-130 として決定し、idmanagement を `user`/`group`/`agent`
の 3 feature へ**全層**（`domain`/`ports`/`usecases`/`adapters/http`/
`adapters/persistence/{memory,postgres}`）変換した。
- **Phase 1**: `domain`/`ports`/`usecases`/`adapters/persistence/memory` を `git mv` で
再配置し、import path を一括置換。共有型（`enums.go`/`events.go`）は context ルートの
`domain/` に、feature 非依存ヘルパーと feature 横断エラー変数は新設
`backend/idmanagement/usecases/helpers.go` に残した。
- **Phase 2**（レビューでの指摘を受けて追加実施）: 当初 Go の言語制約・コード生成単位を
理由に分割対象外としていた `adapters/http` と `adapters/persistence/postgres` も、
再検討の結果別の設計で分割できると判明し実施した。
- `adapters/http`: `Deps` 構造体を独立した leaf package `httpdeps` へ切り出し、
ハンドラをメソッドからフリー関数（`func handleX(d Deps, c) error`）へ変換して
feature パッケージへ移した。`Deps` 型自体は分割せず、外部の `idmhttp.Deps{...}`
構築コード（bootstrap・約30のテストファイル）を無変更に保った。
- `adapters/persistence/postgres`: `sqlc.yaml` を feature 単位の複数エントリへ分割し
`sqlc generate` で再生成。feature 横断のテスト fixture ヘルパーと `textOrNil` は
各 feature パッケージへ複製した。`lifecycle_workflows` テーブルは IdGovernance
context 所有（wi-237/ADR-117 以前の歴史的経緯）で features のいずれにも属さないため、
context ルート共有のまま維持した。
- ADR-130・`REGENERATIVE_ARCHITECTURE.md` §3.8・`ARCHITECTURE.md`
（module ledger + Go Package Conventions 節）を Phase 2 の実態に合わせて更新した。

全文は `git show d1b45ea7499229939b03ccd6d5c4ed3a940a6712:<このファイルのパス>` で読む。
