---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-28
priority: p2
depends_on: [wi-86206-select-ci-test-groups-from-changed-inputs, wi-40431-generate-state-matrices-from-testdata]
change_kind: tooling
spec_impact:
  kind: none
  reason: "手元の検証と作業項目の統合で実行するテストの選び方と mise タスクを変える。製品の振る舞い、HTTP 契約、永続状態、イベント、配備構成は変更しない。"
---

# 変更が影響し得るテストだけを実行し、統合では両側の影響が重なる範囲だけを検証する

## 動機

`integrate-work-item` は、rebase で新しいツリーができ、どちらかの変更がコードに触れると、`mise run check` と `mise run verify` を実行する。
`verify` は `check` を含むので `check` は二重に実行され、`verify` は Go 全体の race テスト、UI の全検査、tools の全検査を実行する。
2026-10-28 の wi-18773 と wi-43903 の統合では、`verify` がそれぞれ約 137 秒と 128 秒かかった。
wi-43903 では rebase で入った変更が runbook の Markdown だけで、検証済みの結果を変え得る入力はなかった。

統合で再実行が必要なのは、両側の変更がどちらも入力に入る検査だけである。
各側の検証結果は、もう一方の変更が入力を変えていない検査については、rebase 後もそのまま有効である。
作業中の最終検査も、同じ考え方で変更が影響し得るテストだけを選べるが、その選択を機械的に行う手段がない。

既存の手段には次の欠陥がある。

- Go の検証の実行器（`tools/verification`）は常に `-count=1` を付け、Go の結果キャッシュを使わない。入力が変わらないパッケージも毎回実行する。
- `test-go-changed` が使う `tools/changed-packages` は `.go` のファイルだけを数える。同じディレクトリの `testdata`、`go:embed` するファイル、パッケージの外からテストが読むファイルの変更を見落とす。
- `test-go-changed -- <base>` は `git diff <base>...`（merge-base からの差分）を使う。rebase 後のブランチでは両側の変更が混ざり、受け入れた変更だけの影響を求められない。
- 逆依存は `go list` の `Deps` だけから求め、テストだけが import するパッケージの変更を見落とすおそれがある（実装前に確かめる）。

Markdown でもテストの入力になる。
`idmanagement/internal/handlers_http/user_lifecycle_model_test.go`、`tenancy/internal/handlers_http/tenant_lifecycle_model_test.go`、`signingkeys/domain/staged_test.go` は `docs/modules/` の機能仕様の状態遷移表を読む。
`shared/storage/db_postgres/schema_test.go` は `infra/schema/postgres.sql` を、`shared/http/server_http/ops_assets_test.go` はリポジトリの根からの相対パスでファイルを読む。
`integrate-work-item` は、以前「`work-items/` と `docs/` の Markdown だけなら `check-work-items` と `check-links`」としていたが、`docs/modules/` についてこの事実に反していた。
2026-10-28 に、暫定として、テストの入力にならない Markdown を wi-86206 と同じ `work-items/` と `docs/development/` に限った。
片側がその Markdown だけなら `verify` を省き、それ以外は `verify` を実行する。

文書の Markdown をテストの入力にすること自体は、wi-40431 がやめる。
状態遷移表の正本を `testdata` へ移し、テストが `docs/` を読むことを検査で拒否する。
本項目はそれを前提にし、`docs/` の Markdown をテストの入力として扱わない。
パッケージ外の入力として残るのは、スキーマのテストの `infra/schema/postgres.sql` のような文書でないファイルである。

## 対象範囲

- 変更集合から、層ごとに影響し得る検査を選ぶ選択器を作る。Go はパッケージ、UI と tools はテストファイルの単位で選ぶ。
- Go のパッケージの入力に、`.go` のほか、パッケージのディレクトリの非 Go ファイル、`go:embed` の対象、テストが読むパッケージ外のファイルを含める。逆依存はテストの import も含めて求める。
- 二つの変更集合の影響の共通部分を求め、統合の検証に使う。
- 選んだ検査と選ばなかった検査を、理由（入力が検証済みのツリーから変わっていない、など）とともに `.verification` の報告へ記録する。選ばなかった検査を成功として数えない。
- mise タスクを作り直す。
  - 作業中の最終検査の入口（たとえば `verify-changed -- [base]`）。
  - 統合の入口（たとえば `verify-integration -- <検証済みのソースの revision>`）。
  - `test-go-changed` は選択器を使う形に改め、既存の作業項目の参照を壊さない。
  - `verify` は全体のゲートとして残す。
- 影響が分からない入力（`go.mod`、`mise.toml`、検証基盤、未知のパス、宣言のない実行時の読み取り）では全体へ戻す。
- `integrate-work-item` と `implement-work-item` の検査の選び方、`docs/development/specification-first-workflow.md` の「最終検査の選択」、`docs/development/testing.md` の Go の通常テストと race 検査の節を、新しい入口に合わせる。
- `integrate-work-item` の暫定の規則を、選択器による規則へ置き換える。

## 対象外

- CI の検査群の選択と必須チェックの集約。wi-86206 が扱い、本項目はその入力の分類を共有する。
- テスト名の単位への分割と、被覆情報からのテストの選択。
- ブラウザー E2E の spec の選択。現在の「触れた画面を通る spec を選ぶ」判断を残す。
- 変異テストと fuzz の実行範囲。

## 設計

### 検査ごとの扱い

| 検査 | 選び方 | 理由 |
| --- | --- | --- |
| `check`（仕様、契約、境界、記録などの静的検査） | Markdown 以外の入力に触れたら全体 | 仕様と全ハンドラーを横断して読み、全体で数秒で終わるので、絞る価値がない |
| Go のテスト | 影響を受けるパッケージだけ | 最も時間がかかる |
| `lint-go` | 影響を受けるパッケージだけ。全体を読む linter があれば全体 | パッケージ単位の解析が大半である。有効な linter で確かめる |
| UI の単体テスト | 影響を受けるテストファイルだけ | `bun test --changed=<ref>` が import のグラフから選べる |
| UI の型検査、lint、整形、ビルド | `frontend/` に触れたら全体 | プログラム全体を読む |
| tools のテスト | 影響を受けるテストファイルだけ | UI と同じ |

### 影響の計算の候補

- **A. Go の結果キャッシュに任せる**：`-count=1` をやめ、`(cached)` の結果を「入力が同じなので再利用した」と報告する。
  - Go はテストが開いたファイルと読んだ環境変数を記録するので、`infra/schema/postgres.sql` のようなパッケージ外の読み取りも宣言なしに追える。
  - DB や LDAP などの外部状態は追えないので、外部に依存するテストはキャッシュの対象外にする必要がある。
  - 統合の検証済みのツリーは別の worktree で実行されている。worktree の絶対パスがキャッシュの鍵に入ると再利用できない。`-trimpath` で共有できるか確かめる。
  - 二つの変更集合の共通部分という考え方は持たない。入力が変わったものをすべて再実行するので、共通部分より広い。
- **B. 自前の影響計算を広げる**：`changed-packages` に、非 Go の入力、テストの import、実行時の読み取り、二つの変更集合の共通部分を加える。
  - 実行前に選択と理由を決定的に出せるので、報告と CI（wi-86206）で同じ計画を使える。
  - 実行時の読み取りの宣言漏れが選択漏れになる。テストの文字列定数の相対パスは静的に解決し、解決できない読み取りを持つパッケージは「Go 以外の変更があれば選ぶ」側に倒す。
- **C. B で選び、選んだパッケージの中では A を使う**。

実装前に、`-trimpath` の効果と、A のキャッシュが統合の典型例でどれだけ当たるかを測ってから選ぶ。
選択の誤りは検証されないコードを統合させるので、どの案でも、判定できない入力は全体へ戻す。

### 統合の共通部分

検証済みのソースのツリーを S、rebase 後のツリーを R、元の base を B とする。

- ソースの変更集合は `B..S`、受け入れた変更集合は `S..R`（二点の差分）である。
- それぞれの影響範囲を求め、その共通部分の検査だけを R で実行する。
- 受け入れた変更が `docs/` の Markdown だけなら、テストの影響範囲は空である（wi-40431 の検査が保証する）。文書でないパッケージ外の入力を含むなら、それを読むテストの影響範囲に入る。

### 選ばなかった案

- 統合で常に `verify` を実行する：両側が別々の層に触れる場合や、影響が重ならない場合にも全体の費用を払う。
- 統合で層の単位だけを見る（両側が `backend/` に触れたら Go 全体）：今回の調査の出発点だが、パッケージの単位で重なりを求められるので粗すぎる。

## 計画

1. `changed-packages` の逆依存の求め方、`bun test --changed` の比較の意味（二点か merge-base か、fixture の扱い）、`-trimpath` によるキャッシュの共有、有効な linter の解析範囲を確かめ、候補を選ぶ。
2. テストが読むパッケージ外のファイルを棚卸しし、静的に解決する範囲と、全体へ戻す範囲を決める。
3. 最小の Git workspace と go list の fixture で、選択器の RED（非 Go の入力、テストの import、パッケージ外の読み取り、共通部分、未知の入力）を確認し、実装する。
4. mise タスクを作り直し、報告に選んだ理由と選ばなかった理由を記録する。
5. スキルと開発文書を新しい入口に合わせる。
6. wi-18773 と wi-43903 の統合を再現し、選んだ範囲と所要時間を `verify` と比べて記録する。

## タスク

- [ ] T001 [Design] 計画 1 と 2 の調査を行い、候補と全体へ戻す条件を設計へ記す。
- [ ] T002 [Acceptance] 選択器の RED を確認する。
- [ ] T003 [App] 選択器、共通部分、報告を実装する。
- [ ] T004 [Tooling] mise タスクを作り直す。
- [ ] T005 [Docs] `integrate-work-item`、`implement-work-item`、最終検査の選択、Go のテストの節を更新する。
- [ ] T006 [Verify] 選択漏れの故障を注入して検出し、過去の統合を再現して時間を記録する。

## 検証

製品要件 ID は N/A（開発時の検査の選択）。

- 対照：Go だけ、UI だけ、tools だけ、`docs/` の Markdown、テストが読む文書でないパッケージ外のファイル、`testdata` と `go:embed`、テストだけの import、`go.mod`、未知の入力、削除と移動、両側が重ならない変更、重なる変更。
- 故障の注入：非 Go の入力を見落とす、共通部分を和集合や片側で取り違える、未知の入力を全体へ戻さない、選ばなかった検査を成功として数える。
- `mise run test-tools-file`、`typecheck-tools`、`lint-tools`、`check-command-map`、`check-agent-guidance`、`check-work-items`、`check-links`。

## リスク

- 選択漏れは、検証されないコードを統合させる。判定できない入力は全体へ戻し、`verify` は全体のゲートとして残す。
- テストが新しくパッケージ外のファイルを読み始めると、宣言漏れが選択漏れになる。静的に解決できない読み取りを検出して全体側へ倒す。
- 実行するテストが減っても、ビルドと依存の準備が支配的なら時間は減らない。準備と実行を分けて測る。
