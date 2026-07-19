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
済まないことが分かった。具体的には次の 2 点が Go の言語制約により分割不能だった。

- **adapters/http**: `RegisterRoutes` が `Deps` 構造体のメソッド (`func (d Deps) handleX`)
  として実装されている。Go はメソッドを receiver 型と同一パッケージにしか定義できないため、
  ハンドラファイルを feature ごとに別パッケージへ分割すると `Deps` 型ごと分割するか、
  メソッドをフリー関数へ書き換える大改修が要る。これは「物理配置のみの変更」という
  wi-254 の前提（`spec_impact: none`）を超える。
- **adapters/persistence/postgres**: `sqlc` は 1 つの `queries/` ディレクトリから 1 つの
  `sqlcgen` パッケージを生成する設定（`sqlc.yaml`）になっており、feature ごとに分割するには
  codegen 設定自体の再設計が要る。加えて postgres テストの fixture ヘルパー
  （`seedTenant`/`seedUser`/`seedGroup` 等、`fixtures_test.go`）は agent/group 双方の
  テストから使われる横断的なヘルパーであり、Go の `_test.go` はパッケージ外からアクセスできない
  ため、分割すると機能横断ヘルパーの複製が要る。

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
4. **adapters/http と adapters/persistence/postgres は feature 分割の対象外とし、
   context ルート共有のまま維持する**。上記コンテキストの言語制約・生成コード制約による。
   将来 `Deps` を feature ごとの embedded 部分構造体へ再設計する、または `sqlc` の
   多パッケージ生成へ移行するなら、それ自体を独立した ADR とスコープを持つ work item で扱う。
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
（`domain`/`ports`/`usecases`/`adapters/persistence/memory` を分割、`adapters/http`/
`adapters/persistence/postgres` は共有のまま）。`just build-go` / `just test-go` /
`just verify-go` は全緑。

## 却下した代替案

- **全 context に一律で feature 層を導入する**: 単一 feature context で stutter が
  発生し RA 原則に反するため却下（決定 1 で条件付き化）。
- **adapters/http も `Deps` を feature ごとの embedded 部分構造体
  （`type Deps struct { user.UserDeps; group.GroupDeps; ... }`）へ再設計して分割する**:
  技術的には可能だが、`Deps` の構成 API が flat から nested へ変わり、bootstrap や
  全テストの construction call site（数十箇所）を書き換える必要が生じる。これは
  「振る舞い不変の物理配置変更」という wi-254 のスコープを超えるため、本 wi では見送り、
  必要になれば独立した work item で扱う。
- **adapters/persistence/postgres を feature ごとに分割し、fixture ヘルパーを feature ごとに
  複製する**: `sqlc.yaml` の再設計と fixture 複製の二重コストに対して、decluttering の
  効果（postgres ファイルは 1 aggregate = 1 file で元々見通しが良い）が薄いため却下。
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
