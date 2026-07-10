---
status: suggested
authors: [tn]
created_at: 2026-07-10
---

# ADR-091: Module パターンによる分散 DI / ルーティング

## コンテキスト

[[ADR-047]] / [[ADR-070]] で HTTP アダプタとルーティングは context 所有＋横断 `support` /
`server` へ整理された。しかし依存注入（DI）と route 集約は依然として中央集権のままである。

- `internal/shared/adapters/http/server/routes.go` の `Deps` 構造体は 55+ field。
- `internal/bootstrap/deps.go` の `Dependencies` 構造体は 70+ field の神構造体で、
  memory / postgres_valkey の永続層差分をここで吸収している。
- 中央 `server.Register` が全 context の `RegisterRoutes` を配線する。

このため 1 エンドポイントの追加が、handler → 各 context の `routes.go` → 中央 `Deps` →
`bootstrap/deps.go` → 永続層組立 …と 8〜9 file を横断する（shotgun surgery）。痛点の
本質は「配線を書く手間」ではなく「所有権と locality が中央の巨大構造体に *集中*して
いる」ことである。

## 決定

各 context が自身の DI 組立とルート登録を所有する **Module パターン**を導入し、神構造体
と中央 `Register` を解体する。DI ライブラリは導入しない。

1. **各 context に `module.go` を置く**。`Module.Register(g *echo.Group, infra Infra)` が
   自 context の repository / usecase / handler を自前で組立て、自 context の route を登録
   する。context 固有の依存はこの中に閉じる。
2. **`Infra` を小さな純技術依存に絞る**。DB pool / crypto / eventsink / notification /
   tenant resolver / observability など、どの context にも固有でない共有アダプタのみを
   `Infra` として bootstrap が組み、各 Module へ渡す。
3. **bootstrap は Module 一覧を回すだけにする**。`bootstrap` は永続層バックエンド
   （memory / postgres_valkey）と `Infra` を選択・構築し、登録済み Module を順に
   `Register` する薄い構成へ縮約する。55+ / 70+ field の神構造体と中央 `Deps` は解体する。
4. **google/wire は将来オプションとする**。Module が境界と所有権を担い、必要になれば
   context 別 `ProviderSet` で wire を*重ね掛け*して module 内 constructor 配線の定型を
   codegen で削減できる（両者は排他でない）。まず依存ゼロの手動 Module で始め、
   constructor 配線が冗長化した時点で wire 採用を検討する。
5. **神ファイル分割を随伴させる**。各 context の Module 化に際し、`authorize_handler.go`
   (1,144 行) や `scim/usecases/usecases.go`(770 行) を feature 単位ファイルへ分解する
   （付随ハイジーン、振る舞いは不変）。

## 却下した代替案

- **uber/fx（および dig）**: 実行時リフレクションで依存解決し、エラーは起動時にしか出ない。
  「AI が 1 context を読めば理解でき、コンパイル時に安全」という本目標とトレース性・
  監査性（IdP）に反する。参考ベンチでも fx は wire 比で桁違いに遅い（Wire はゼロ
  オーバーヘッド）。fx.Module は bounded context に対応するが、その利点は手動 Module でも
  得られ、実行時マジックの代償に見合わない。
- **google/wire を最初から全面採用**: コンパイル時・ゼロオーバーヘッドで RA 再生成思想と
  整合するが、痛点の本質（中央集権の解消）は Module パターンが直接解く。wire は module 内
  配線の定型削減という *補完* に留まるため、まず手動 Module を骨格とし wire は後付け余地。
- **現状維持で神構造体だけ per-context サブ構造体へ分割**: 記述量は減るが、中央 `Register`
  と bootstrap への横断依存が残り、endpoint locality が達成されない。

## 影響

- `internal/shared/adapters/http/server/routes.go` の `Deps`（55+ field）と `Register` を
  解体し、router は各 Module の `Register` を回す薄い構成へ変わる。
- `internal/bootstrap/deps.go` の `Dependencies`（70+ field）を解体し、bootstrap は
  `Infra` 構築と Module 登録の薄い層になる。
- 各 context に `internal/<context>/module.go` が新設される。
- エンドポイント追加が該当 context ディレクトリ内に閉じ、中央神構造体と bootstrap の
  横断編集が消える。
- 新規ランタイム依存は導入しない（google/wire は将来の任意採用）。
- 振る舞い・SCL 規範・HTTP route・DB schema・公開 API は変更しない。
- [[ADR-047]] / [[ADR-070]] を extend する（HTTP 所有境界は不変、本 ADR は DI と route
  集約の所有権を各 context へ前進させる）。
