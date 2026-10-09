---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-92970-adopt-information-hiding-modular-design]
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: Go のパッケージの配置、責務表、構造の文書、検査だけを変え、製品の利用者と運用者が観測する振る舞い、設定、API は変わらない。
  references: []
spec_impact: { kind: none, reason: "一つのモジュールのパッケージを Go の internal/ へ移し、責務表の公開方式と公開パッケージを書き換えるだけである。HTTP の応答、認証方式、永続状態、ドメインイベント、外向きの通知は変えない。" }
initial_context:
  specification: []
  typespec: []
  source:
    - docs/domain/structure.md
    - docs/design/architecture/logical.md
    - tools/check/src/boundary-fitness.ts
    - tools/check/src/specification-rules.ts
    - backend/authorization/module.go
    - backend/authorization/internal/handlers_http/routes.go
    - backend/shared/http/server_http/routes.go
    - backend/cmd/internal/bootstrap/memory.go
    - backend/cmd/internal/bootstrap/postgres.go
  tests:
    - backend/authorization/internal/handlers_http/routes_test.go
    - backend/shared/http/server_http/routes_contract_test.go
  stop_before_reading: [frontend, spec]
---

# 依存の少ないモジュールを一つ選び、Go の internal/ へ移す

## 動機

モジュール設計の項目は、非公開の実装の最終的な配置を Go の `internal/` と定め、全モジュールを公開方式 `legacy` のまま残した。
`legacy` は `domain` と `ports` の命名で公開を判定するので、コンパイラは非公開の実装への import を拒否しない。
移行の手順（外側に残るパッケージの分類、利用元の変更、組み立て地点の結線、公開パッケージの宣言）は文書にあるが、実際のモジュールで確かめていない。
最初の一つで手順の不足を見つけてから、ほかのモジュールへ広げる。

## 対象範囲

- 依存の少ないモジュールを一つ選ぶ。
  ほかのモジュールからの import の数、そのモジュールへの `private-import` の数、組み立て地点からの import の数を比べ、選んだ理由を設計に記録する。
- 選んだモジュールの本番パッケージを、ルートパッケージ、公開パッケージ、`internal/` のどれかに分類して移す。
- 責務表の公開方式を `internal` にし、公開パッケージを列挙する。
- 組み立て地点からの非公開パッケージへの直接の import を、ルートパッケージの操作へ移す。
- 手順に不足があれば、[構造](../../docs/domain/structure.md#公開範囲と-internal)を直す。

## 対象外

- 二つ目以降のモジュールの移行。[外から非公開パッケージへの import がないモジュールの移行](../wi-97546-move-modules-without-private-callers-behind-go-internal.md)と[残りのモジュールの移行](../wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)で行う。
- モジュールの分割と統合。

## 設計

公開パッケージを決めることは公開範囲の宣言であり、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。
現在の利用元が import しているから公開する、という理由だけで公開パッケージに加えない（D3）。

### 移すモジュールの選択

着手時の revision `213ad69ce` で、ほかのモジュールから非公開パッケージへ import されていない 12 モジュールを比べた。
数えたのは `go list` の本番とテストの import であり、組み立て地点は `backend/cmd`、`server_http`、`testing_stack` である。

| モジュール | パッケージ数 | import するほかのモジュール | 組み立て地点から import されるパッケージ |
| --- | --- | --- | --- |
| Seeding | 3 | なし | `domain`、`manifests_yaml`、`usecases`（ルートパッケージなし） |
| Authorization | 9 | なし | ルート、`db_memory`、`db_postgres`、`handlers_http`、`principals_idmanagement` |
| IdGovernance | 8 | なし | ルート、`db_memory`、`db_postgres`、`domain`、`usecases`、`handlers_http` |
| DataKeys | 8 | Authentication、OAuth2 | ルート、`db_memory`、`db_postgres`、`usecases`、`handlers_http` |
| そのほか 8 個 | 7〜11 | 1〜4 個 | 4〜14 |

Authorization を選ぶ。
ほかのモジュールからの import がなく、ルートパッケージ、`domain`、`ports`、`usecases`、二つの永続化アダプター、`handlers_http`、ほかのモジュールへのアダプター、`testing_contract` という、残りのモジュールと同じ配置を持つ。
Seeding は依存が最も少ないが、ルートパッケージ、HTTP のアダプター、永続化を持たないので、確かめた手順が残りのモジュールへ当てはまらない。
IdGovernance は組み立て地点が `domain` と `usecases` まで直接使っており、最初の一つとしては結線の変更が広い。

### 判断の記録

| 項目 | 内容 |
| --- | --- |
| 入力 | [構造](../../docs/domain/structure.md#公開範囲と-internal)の最終形と移行規則、責務表の Authorization の行、上の import の計測 |
| 適用した制約 | D3（非公開の処理を公開に変えない）。D4 と D7 は、import の向きと原子性を変えないので該当しない |
| 候補 A | `domain` と `ports` を公開パッケージとして残す。ほかのモジュールが使っていないので、公開する理由がない。採らない |
| 候補 B | 公開パッケージを持たず、ルートパッケージが組み立て地点へ、永続化の選択と経路の登録を操作として公開する。採る |
| 変更シナリオの波及 | 認可のモデルや永続化の実装を変えても、組み立て地点はルートパッケージの操作だけを使うので変わらない。ほかのモジュールが認可の判定を使う要求が出たら、その時点で公開する型と操作を D3 で決める |
| 残る仮定 | 認可の判定を OAuth2 の AuthZEN 経由で使う現在の構成（`oauth2/ports.Authorizer`）は、Authorization の import を必要としない |

### 移した後の形

- `backend/authorization/`（ルートパッケージ）は `Module` に加え、`NewMemoryModule()`、`NewPostgresModule(pool)`、`Module.RegisterRoutes(g, RouteDeps)` を持つ。組み立て地点はこの三つだけを使う。
- `domain`、`ports`、`usecases`、`db_memory`、`db_postgres`、`handlers_http`、`principals_idmanagement`、`testing_contract` は `backend/authorization/internal/` へ移す。パッケージ名は変えない。
- `Module` のフィールドは `internal/ports` の型のまま残す。組み立て地点は型を名指さずに値を受け渡せるので、フィールドを隠す必要はない。`handlers_http` のテストは同じモジュールの内側からフィールドを直接組む。
- `sqlc.yaml` のクエリと出力の位置を新しいパスへ変える。

### 検査で見つかった手順の不足

`tools/check/src/specification-rules.ts` の `codeSlices` は `backend/<モジュール>/<名前>/<層>` を機能スライスとして読む。
`internal/` の下の `domain` は `internal` という名前の機能スライスとして読まれ、対応する仕様のディレクトリがないとして拒否される。
`internal` の区画を飛ばして、`backend/<モジュール>/internal/<名前>/<層>` を `<名前>` の機能スライスとして読むように直す。

## タスク

- [x] T001 [Design] 移すモジュールを選び、パッケージの分類と公開パッケージを判断手順で決める。
- [x] T002 [App] パッケージを移し、利用元と組み立て地点を書き換える。
- [x] T003 [Docs] 責務表を書き換え、手順の不足を構造の文書へ反映する。
- [x] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 利用元が非公開の実装に依存していると、移行のために公開パッケージを広げたくなる。
  広げる代わりに、必要な振る舞いを公開操作で表せるかを D3 で比べる。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は main に対して規範仕様の差分がないことを示した。
  Authorization の本番パッケージ 8 個（`domain`、`ports`、`usecases`、`db_memory`、`db_postgres`、`handlers_http`、`principals_idmanagement`、`testing_contract`）を `backend/authorization/internal/` へ移し、責務表の公開方式を `internal`、公開パッケージをなしとした。
  ルートパッケージは `NewMemoryModule`、`NewPostgresModule`、`Module.RegisterRoutes` を持ち、組み立て地点（`cmd/internal/bootstrap`、`server_http`）はこの三つだけを使う。
  移行で見つかった手順の不足を二つ検査へ反映し、[構造](../../docs/domain/structure.md#公開範囲と-internal)に移行の手順として書いた。
  `codeSlices` は `internal` の区画を飛ばして機能スライスを読み、主要ユースケースの証拠の検査は、完了した記録のテストのパスを `internal/` の下で読み直す。
  故障の注入で見つけた、組み立てを固定するテストの欠けを二つ埋めた。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-boundaries`。
  - **要件**: N/A: 製品の振る舞いを変えないパッケージの移動であり、規範となる製品要件はない。
  - **観測した失敗**: 責務表を `internal` にし、移動前のコードに当てると、外側に残る 8 パッケージの入力診断と、組み立て地点から非公開パッケージへの import 4 件（`private-import:System->Authorization`）で失敗した。移動後は成功した。Go のビルドも、組み立て地点に残した `internal/` の import を `use of internal package ... not allowed` で拒否した。
  - **検出できる理由**: 検査はモジュールの外側に残る本番パッケージと、組み立て地点からの import をパッケージ単位で列挙するので、移し忘れと結線の残りを区別して示す。
- **単体 RED の証拠**:
  - **テスト**: `tools/check/src/specification-rules.test.ts` の「reads a slice below internal/ by the name under it, and the flat layers as no slice」と、`tools/check/src/primary-use-case-evidence.test.ts` の「follows a recorded test path that moved below its module internal/」。
  - **要件**: N/A: 製品要件のない検査の変更である。
  - **観測した失敗**: 前者は `internal` を機能スライスの名前として返して失敗した。後者は移したテストを `test path does not exist` として拒否して失敗した。
  - **検出できる理由**: 前者は `internal` の下の平らな層を機能スライスにしないことも表明する。後者は読み直した先のファイルでテスト名を照合することも表明するので、存在だけを見る誤実装を区別できる。
- **変更耐性の結果**:
  `test-go-mutation` はルートパッケージに変異を見つけなかった（配線だけで演算子がない）。
  配線の故障を手で注入した。

  | 注入した故障 | 結果 |
  | --- | --- |
  | `server_http` から経路の登録を外す | `TestAssembledRoutesMatchGeneratedOpenAPI` が失敗した |
  | 主体の解決から Agent の記録を外す | `handlers_http` のテスト 6 件が失敗した |
  | 主体の解決から User の記録を外す | 最初は生き残った。代行チェーンの User を IdManagement の記録で判定するサブテストを足し、失敗するようにした |
  | `NewMemoryModule` でモデルに別の保管庫を渡す | 最初は生き残った。テナントに一つの書き込みの版を共有する性質を契約テストに足し、`NewMemoryModule` が組む Repository に契約を当てるテストを足して、失敗するようにした |

  `NewPostgresModule` は構造体を二つ組むだけであり、PostgreSQL の契約はアダプター単体のテストが当てる。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check` - 成功
  - `mise run verify` - 成功
