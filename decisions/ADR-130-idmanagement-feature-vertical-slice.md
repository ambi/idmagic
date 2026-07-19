---
status: accepted
authors: [tn]
created_at: 2026-07-20
---

# ADR-130: 大型 bounded context に feature 垂直スライス層を条件付きで導入する

## コンテキスト

`backend/<context>/{adapters,domain,ports,usecases}/` という層 × context の格子は、
context が小さいうちは十分に「ドメイン・仕様の構造をそのまま反映する」(RA §3.8) が、
context が複数の独立した sub-domain（feature）を抱えると破綻する。特に大型 context
（oauth2 ≈10.3k / idmanagement ≈8.3k / authentication ≈8.3k LOC）では、`domain/` や
`usecases/` に user・group・agent など無関係な sub-domain のファイルが同一ディレクトリに
平積みされ、境界が物理配置から読み取れなくなる（wi-254 Motivation）。

idmanagement を `user`/`group`/`agent` の 3 feature へ実際に垂直スライスするパイロット
変換（wi-254）を通じて、この再配置が「ファイル移動と import path の機械的置換」だけでは
済まないことが分かった。Phase 1 では次の 2 点をいったん分割対象外と判断したが、レビューで
再検討した結果、いずれも別の設計で分割可能と判明し、Phase 2 として実施した。

- **adapters/http**: `RegisterRoutes` が `Deps` 構造体のメソッド (`func (d Deps) handleX`)
  として実装されている。Go はメソッドを receiver 型と同一パッケージにしか定義できないため、
  `Deps` 型を素朴に feature ごとの embedded 部分構造体へ分割すると、admin_group_handler.go が
  `UserRepo` を、admin_agent_handler.go が `UserRepo`/`ClientRepo` を、admin_user_handler.go
  が `GroupRepo`/`ScimRepo`/`JobRepo` を参照するなど feature 横断の port 参照が多く、
  各部分構造体に同じ port フィールドを重複定義する必要が生じる。決定 4 のフリー関数化で
  この重複を避けて分割した。
- **adapters/persistence/postgres**: `sqlc` は 1 つの `queries/` ディレクトリから 1 つの
  `sqlcgen` パッケージを生成する設定（`sqlc.yaml`）になっており、feature ごとに分割するには
  codegen 設定自体の再設計が要る。加えて postgres テストの fixture ヘルパー
  （`seedTenant`/`seedUser`/`seedGroup` 等、`fixtures_test.go`）は agent/group 双方の
  テストから使われる横断的なヘルパーであり、Go の `_test.go` はパッケージ外からアクセスできない
  ため、分割すると機能横断ヘルパーの複製が要る。決定 4 で `sqlc.yaml` を feature 単位の
  3 エントリへ分割し、fixture ヘルパーは各 feature パッケージへ複製した。

さらに、`domain/` の一部型（`enums.go`/`events.go`）や `usecases/` の一部ヘルパー・エラー
変数（`normalizeRoles` 系のユーティリティ、`ErrUserNotFound`/`ErrInvalidRole`）は feature
横断で参照されており、機械的に 1 feature へ寄せることができなかった。

## 決定

1. **feature 垂直スライス層の条件付き導入**: `backend/<context>/<feature>/{domain,ports,
   usecases,adapters/{http,persistence/memory}}/` という feature 層は、**2 つ以上の
   feature を持つ context にのみ**導入する。単一 feature の context
   （signingkeys, tenancy, audit, jobs 等）に `backend/signingkeys/signingkeys/` のような
   context 名 = feature 名の stutter を作ることは、RA の「ディレクトリ構造が叫ぶ」原則に
   照らして何も伝えないため禁止する。
2. **成長トリガー規約**: context が 2 つ目の feature を獲得した時点で feature 層を導入する。
   これを明文化し、「導入しそびれ」を防ぐ。
3. **module.go は context ルートに 1 つのまま据え置く**（ADR-091 の DI 束方針と整合）。
   feature 層はソース配置のみの変更であり DI 組立の分割ではない。
4. **adapters/http と adapters/persistence/postgres も分割する**。
   - `adapters/http`: `Deps` 構造体の定義・`RegisterRoutes`・feature 横断の統合テスト
     (`extra_identity_test.go`) は context ルート共有のまま維持する。ハンドラの実装は
     `func (d Deps) handleX(c *echo.Context) error` という**メソッド**から
     `func handleX(d Deps, c *echo.Context) error` という**フリー関数**へ変換し、
     `<feature>/adapters/http/` へ移す。フリー関数は receiver 型と同一パッケージである
     必要がないため、`Deps` 型を分割せずに実装コードだけを feature パッケージへ移せる。
     `routes.go` の登録は `g.GET(path, d.handleListGroups)` から
     `g.GET(path, grouphttp.HandleListGroups(d))` のような呼び出しへ変わるが、
     `Deps{}` を構築する外部呼び出し（bootstrap・約 30 のテストファイル）のフラットな
     field 名は無変更のまま。
   - `adapters/persistence/postgres`: `sqlc.yaml` の idmanagement 用エントリを feature
     単位の 3 エントリへ分割し、`queries/*.sql` と生成される `sqlcgen/` を feature
     ディレクトリへ移す。`fixtures_test.go`/`harness_test.go` の feature 横断 fixture
     ヘルパー（`seedTenant`/`seedUser`/`seedGroup`/`seedClient`/`testClock` 等）は
     Go の `_test.go` がパッケージをまたげない制約により、各 feature パッケージへ複製する。
5. **domain 層の共有型**: feature 横断で使われる型（enum、DomainEvent 等）は
   context ルートの共有 `domain/` に残す。feature 固有型のみ `<feature>/domain/` へ移す。
6. **usecases 層の共有ヘルパー**: feature に依存しない小さい utility 関数
   （role 正規化・時刻正規化・任意 emit 等）と、feature 横断で参照されるエラー変数
   （`ErrUserNotFound`/`ErrInvalidRole` 等）は、context ルートの共有 `usecases/` に残す。
   feature 固有の usecase 関数・型のみ `<feature>/usecases/` へ移す。
7. **package 名は各層名のまま**（`domain`/`ports`/`usecases`/`http`/`memory`）。Go は
   import パスで区別するため、同一 context の複数 feature を同時 import する箇所
   （feature 間の正当な依存や、共有パッケージへの参照)では named import が必要になる
   （`userdomain`, `groupdomain`, `idmusecases` 等）。既存コードの `idmports` 等の
   named import 慣習の延長。

idmanagement をパイロットとして `user`/`group`/`agent` へ実際に変換した
（`domain`/`ports`/`usecases`/`adapters/persistence/memory`/`adapters/http`/
`adapters/persistence/postgres` の全層を分割。`Deps` 型定義と `module.go` の DI 束のみ
context ルート共有）。`just build-go` / `just test-go` / `just verify-go` は全緑。

## 却下した代替案

- **全 context に一律で feature 層を導入する**: 単一 feature context で stutter が
  発生し RA 原則に反するため却下（決定 1 で条件付き化）。
- **adapters/http の `Deps` を feature ごとの embedded 部分構造体
  （`type Deps struct { user.UserDeps; group.GroupDeps; ... }`）へ再設計して分割する**:
  技術的には可能だが、feature 横断の port 参照（上記コンテキスト参照）により各部分構造体へ
  同じフィールドを重複定義する必要が生じ、かつ `Deps` の構成 API が flat から nested へ
  変わって bootstrap や全テストの construction call site（数十箇所）を書き換える必要が
  生じる。決定 4 のフリー関数化はこの両方を避けられるため、embedding 案は却下した。
- **adapters/persistence/postgres を分割せず据え置く**: Phase 1 時点ではいったんこの判断を
  したが、fixture ヘルパーの複製コスト（数十行）は sqlc 3 分割の実装コストに対して小さく、
  decluttering の効果（postgres ファイルが feature ごとに独立し、他 feature の変更が
  無関係な diff を生まない）の方が上回ると判断し、Phase 2 で撤回して分割した。
- **usecases の共有ヘルパーを feature ごとに複製する**: `normalizeRoles` 等は数行の
  純粋関数だが、3 箇所へ複製すると将来の修正が同期されなくなるリスクがあるため、
  共有 `usecases/` に 1 箇所だけ残す方を選んだ。

## 影響

- SCL (`spec/scl.yaml`) の規範定義・`context_map` の変更なし（`spec_impact: none`、
  純粋な物理配置変更）。
- `REGENERATIVE_ARCHITECTURE.md` §3.8 の構造例を本決定に合わせて更新する（wi-254 T002）。
- `ARCHITECTURE.md` の `## Go Package Conventions` と `modules[].path`（idmanagement 分）を
  feature 粒度へ更新する（wi-254 T005, `new-architecture` skill）。
- ADR-089/090/091（context-locality 系）と RA §3.8 を前提とする。
