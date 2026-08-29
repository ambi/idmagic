---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
evidence_policy: risk-based-v2
created_at: 2026-08-29
priority: p2
depends_on: []
change_kind: tooling
initial_context:
  specification:
    - docs/contexts/jobs/states.md#JobLifecycle
    - docs/contexts/jobs/scenarios.md#REQ-JOBS-004
    - docs/development/specification-first-workflow.md
    - docs/development/testing.md
    - WORK_ITEM_FORMAT.md
  typespec: [IdMagic.Contract.JobStatus]
  source:
    - backend/jobs/domain/job.go
    - backend/jobs/ports/repository.go
    - backend/jobs/db_memory/repository.go
    - mise.toml
    - tools/check/src/specification-doc.ts
    - tools/check/src/check-boundaries.ts
  tests: [backend/jobs/db_memory/repository_test.go]
  stop_before_reading:
    - backend/jobs/db_postgres
    - frontend
spec_impact:
  kind: none
  reason: Quint を仕様、設計、開発、テストへ組み込めるか評価する作業であり、この項目では製品の振る舞い、外部契約、規範シナリオ、規範 ID、TypeSpec シンボルを変更しない。
---

# Quint を仕様、設計、開発、テストの検証経路へ組み込めるか実証する

## Motivation

`wi-421-model-checking-and-deterministic-simulation` は Jobs のリースと DataKeys の鍵ローテーションを対象にモデル検査を導入し、記述言語には直接 TLA+ を使う方針を置いている。しかし、[Quint](https://quint.sh/docs/what-does-quint-do) は状態機械、性質、反例トレースを一つの言語とコマンド群で扱い、[TLC と Apalache を検査器として利用できる](https://quint.sh/docs/model-checkers)。TLA+ の検査基盤を保ちながら、チームが読み書きする表層言語と開発時の操作を変えられる可能性があるため、`wi-421` の実装前に比較する価値がある。

評価対象はモデル検査の実行だけではない。現行の仕様先行開発では、`states.md` が言語非依存の状態と遷移の正本であり、実装後の証拠は Go の受け入れテストと単体テストが担う。Quint のモデルだけを追加すると、仕様と模型と実装の三者が独立して変わり、検査が成功しても製品の正しさを示さない。[Quint のモデルベーステスト](https://quint.sh/docs/model-based-testing) は模型から JSON トレースを生成して実装へ与える方法と、実装のトレースを模型で検証する方法を説明しているが、2026-08-29 時点で案内されている Quint Connect は Rust 向けであり、Go 実装との接続費用はこのリポジトリで実証されていない。

Jobs のリースは評価対象に適している。正準文書には `JobLifecycle` と `REQ-JOBS-004` があり、実装は時刻を `ClaimBatch`、`Heartbeat`、`Complete`、`Fail` の引数として受け取る。複数の `worker`、リース失効、再取得、所有者だけに許す完了報告を有限の模型へ落とし込みやすく、模型と実装の不一致も観測できる。

## Scope

- Quint の現行版、ライセンス、配布形態、`mise` での版固定方法、TLC と Apalache の取得方法、キャッシュ、macOS と Linux での再現性を調べる。
- `docs/contexts/jobs/states.md`、`REQ-JOBS-004`、Jobs のリース機構から、二つの `worker`、一つの Job、明示的な論理時刻を持つ有限の Quint 模型を試作する。
- `JobLifecycle` の状態と遷移は `states.md` から生成するか機械比較し、手書きの Quint 模型が正準文書から黙って乖離しない経路を試作する。
- 安全性として「有効なリース所有者は同時に一人以下」「リースを持たない `worker` は完了または失敗を確定できない」「終端状態から `running` へ戻らない」を検査し、到達可能性の witness と期待した反例トレースを得る。
- 固定 seed のシミュレーションとモデル検査を `mise` タスクから実行し、型検査、シミュレーション、TLC と Apalache の有界検査について実行時間と導入物を記録する。
- Quint が出力する JSON トレースを、明示的な時刻を渡す Jobs のメモリ永続化アダプター用 Go テストドライバーへ入力し、模型の操作と実装の観測状態を各 step で照合する試作を行う。
- 仕様、設計、開発、テストの各段階で Quint を使う条件、証拠、失敗時の戻り先を `docs/development/specification-first-workflow.md` と `docs/development/testing.md` へどう追加するかを差分として試作する。
- 採否、検査を置くゲート、保守費用、再評価条件を記録する。採用する場合だけ恒久導入を別の work item として起票し、`wi-421` の形式仕様言語とタスクを評価結果へ同期する。

## Out of Scope

- Quint、TLC、Apalache を恒久的な依存関係または必須ゲートとして導入すること。評価用の模型、ドライバー、タスク、依存関係は Completion に証拠を残して撤去する。
- 製品仕様、Jobs の振る舞い、Go の本番コード、データベーススキーマを変更すること。模型が欠陥を見つけた場合は、正本と実装を直す別の work item を起票する。
- DataKeys の鍵ローテーションをモデル化すること。Quint の採否後に `wi-421` が扱う。
- 全ての状態機械への展開、形式仕様からの実装生成、運用環境のトレース収集、実行時の trace validation。
- 公平性の仮定を要する活性性の証明。最初の評価は安全性と witness に絞り、活性性は採用後の再評価対象とする。
- 暗号プロトコル自体の形式検証と、Quint の模型検査を Go 実装の証明として扱うこと。

## Design

Quint は既存の正本を置き換えず、正準文書から導かれる検証模型の候補として評価する。`states.md` が状態と遷移を所有し、`scenarios.md` と `internals.md` が安全性の意味と機構を所有する。試作では `states.md` の状態表と遷移表を読み取って Quint の基礎モジュールを生成し、リース所有者、期限、論理時刻、各操作を手書きの拡張模型へ置く。生成が Quint の構文または型検査と両立しない場合は、全遷移行を双方向に機械比較する方式を次に試し、目視対応だけには後退しない。

各工程で期待する役割と証拠を先に固定する。

| Stage | Quint role | Evidence |
|---|---|---|
| Specification | 正準の状態遷移と規範シナリオから有限の模型と安全性を導く | 正本の変更を模型の型検査または同期検査が検出する |
| Design | 実装前に witness、シミュレーション、モデル検査を行う | 到達可能な正常トレースと、壊した安全性に対する反例トレース |
| Development | 固定 seed と保存した反例を再生し、設計変更を短い周期で確かめる | 同じ入力から同じトレースが得られ、反例を修正すると検査が成功する |
| Test | 模型の JSON トレースを Go のアダプターへ与えて観測状態を照合する | 現行実装が通り、故意に誤らせたテスト用実装をドライバーが拒否する |

直接 TLA+ を使う案は `wi-421` の既定案として残す。Quint は TLC と Apalache を利用できるため検査器の能力を捨てずに表層言語を変えられるが、その利点が版固定、追加の変換層、診断、Go との接続費用を上回るかは試作で測る。Quint を即時採用する案は、正本との同期方法と実装接続が未確認なので採らない。シミュレーターだけを使う案も、探索したトレースについてしか性質を確認できないため採用条件を満たさない。

評価用ファイルは `backend/jobs/model/` と必要最小限のテストドライバーへ置き、既存の検証を置換しない。採用時の恒久配置も同じ場所を第一候補とする。`spec/` は TypeSpec が所有する契約の場所であり、Quint 模型を置くと別種の正本に見えるため使わない。

## Decision Criteria

Quint を採用するには、次の条件を全て満たす。

- `JobLifecycle` とリース再取得を有限の模型で表現でき、三つの安全性と一つ以上の witness を、未解決の型エラーや検査器固有の意味差なしに検査できる。
- リース所有者の検査、期限の検査、終端状態のいずれかを故意に壊すと期待した反例が出て、修正後は同じ検査が成功する。
- `states.md` の状態または遷移を模型だけ更新し忘れた差分と、模型だけを更新した差分の両方を `mise` の同期検査が拒否する。
- 一つ以上の模型生成 JSON トレースが Jobs のメモリ永続化アダプターを駆動し、手書きの期待状態を複製せずに各 step を照合できる。リースを持たない完了を許すテスト用の誤実装は同じドライバーで失敗する。
- Quint と検査器の版、取得物、キャッシュを `mise` から再現でき、開発者に未管理の大域インストールを要求しない。macOS と Linux の fresh checkout で同じ模型、性質、固定 seed が同じ成否を返す。
- 型検査と固定 seed のシミュレーションは通常の編集周期に置ける。モデル検査が基準環境で 60 秒以内なら標準の pull request ゲート候補とし、超える場合は明示的な検査タスクまたは定期実行へ分け、`mise run verify` を非決定的または長時間の検査にしない。
- ワークフロー文書が、模型検査は模型の性質を検査すること、Go との適合はトレース駆動テストが別に検査すること、正本の意味変更は仕様段階へ戻すことを区別して説明できる。

一つでも満たせない場合は直接 TLA+ を使う `wi-421` の方針を維持し、失敗した条件と再評価条件を Completion に記録する。性能だけが 60 秒の境界を超えた場合は Quint 自体を不採用にせず、必須ゲートへ置かない判断とする。

## Plan

1. 公式文書と配布物を基に Quint、TLC、Apalache の版固定、実行、出力、ライセンス、キャッシュを棚卸しする。
2. `states.md` から基礎状態機械を導く同期経路を先に作り、片側だけを変えた fixture で失敗を観測する。
3. Jobs のリース模型、安全性、witness を追加し、固定 seed のシミュレーションで正常トレースを得る。
4. 安全性を故意に壊して反例を観測し、元へ戻して TLC と Apalache の検査を通す。
5. JSON トレースを実行する Go ドライバーを試作し、現行のメモリ永続化アダプターと故意に誤らせたテスト用実装を区別できることを確かめる。
6. 実行時間、導入物、診断の読みやすさ、模型と実装の変更量を記録し、工程ごとのワークフロー差分を試作する。
7. Decision Criteria に従って採否とゲート配置を決める。試作を撤去し、採用時だけ恒久導入用 work item を起票して `wi-421` を同期する。

## Tasks

- [x] T001 [Research] Quint、TLC、Apalache の現行版、配布、ライセンス、版固定、キャッシュ、対応環境、JSON トレース出力を確認する。
- [x] T002 [Acceptance] `states.md` と Quint 側の片方だけを変えた fixture が同期検査で失敗する RED を両方向について観測する。
- [x] T003 [Tooling] 評価用の Quint 基礎モジュール生成または双方向同期検査と、型検査、固定 seed シミュレーション、モデル検査の `mise` タスクを試作する。
- [x] T004 [Model] Jobs のリース模型、三つの安全性、一つ以上の witness を書き、時刻、worker 選択、永続化を明示的な状態または入力として分離する。
- [x] T005 [Acceptance] 安全性を壊した模型で期待する反例を観測し、修正後に TLC と Apalache の有界検査を成功させる。
- [x] T006 [Test] Quint の JSON トレースで Jobs のメモリ永続化アダプターを駆動し、現行実装と故意に誤らせたテスト用実装を区別する Go テストドライバーを試作する。
- [x] T007 [Verify] macOS と Linux の再現性、cold と warm の実行時間、取得物、キャッシュ、診断、変更量を測り、工程ごとのワークフロー差分を試作する。
- [x] T008 [Decision] 採否、ゲート配置、再評価条件を Completion に記録し、評価用資材を撤去する。採用時だけ恒久導入用 work item を起票して `wi-421` を同期する。
- [x] T009 [Verify] `mise run verify` と work item の検査を通す。起票時にあった「独立した検証者が照合する」という条件は落とした。`d05ccbac` が証拠契約から独立検証を外し、二人目の読み手はレビューが持つと定めたためである。

## T001 Research Results

調査日は 2026-08-29 とし、版と配布物は各プロジェクトの公式リリース、実行条件と出力形式は公式文書または対応するリリースタグのソース、ライセンスは各リリースタグの `LICENSE` だけで確認した。

### Verified facts

| Tool | Current version | Distribution | Runtime and platforms | License |
|---|---|---|---|---|
| Quint | 安定版は [`v0.32.0`](https://github.com/quint-co/quint/releases/tag/v0.32.0) であり、npm パッケージの版も [`0.32.0`](https://github.com/quint-co/quint/blob/v0.32.0/quint/package.json) である。 | `@informalsystems/quint` と、Deno でコンパイルした単一実行ファイルを配布する。公式の生成対象は [Linux `amd64`/`arm64`、macOS `amd64`/`arm64`、Windows `amd64`](https://github.com/quint-co/quint/blob/v0.32.0/.github/upload-binaries.sh) である。 | npm 版は [Node.js 18 以上](https://github.com/quint-co/quint/blob/v0.32.0/quint/package.json)を要求する。単一実行ファイルはリリース工程が `deno compile` で生成するため、評価では Node.js を追加しない取得経路として扱える。 | [Apache License 2.0](https://github.com/quint-co/quint/blob/v0.32.0/LICENSE) |
| TLA+ Tools / TLC | 現行の安定配布版は [`v1.7.4`](https://github.com/tlaplus/tlaplus/releases/tag/v1.7.4) である。TLC 単体の版番号ではなく、TLC を含む TLA+ Tools のリリース番号として固定する。 | コマンドライン用の `tla2tools.jar` と、Linux、macOS、Windows 向け TLA+ Toolbox を配布する。[`tla2tools.jar` は TLC を主クラスとして直接実行できる](https://github.com/tlaplus/tlaplus#use)。 | コマンドライン版は [Java 11 以上](https://github.com/tlaplus/tlaplus#use)を要求するプラットフォーム非依存の JAR であり、macOS と Linux のどちらでも同じ取得物を使える。Toolbox の公式 `v1.7.4` 配布物は macOS と Linux とも `x86_64` なので、今回の CLI 評価には使わない。 | [MIT License](https://github.com/tlaplus/tlaplus/blob/v1.7.4/LICENSE) |
| Apalache | 現行の安定版は 2026-08-26 公開の [`v0.62.2`](https://github.com/apalache-mc/apalache/releases/tag/v0.62.2) である。 | 公式リリースは版入りの `apalache-0.62.2.tgz` と `.zip`、版なしの別名、`sha256sum.txt` を提供し、[各リリースの Docker image](https://apalache-mc.org/docs/apalache/installation/docker.html) も提供する。 | 事前構築済みパッケージは [JDK 17 の Temurin または Zulu を推奨し、Linux と macOS では `bin/apalache-mc` を使う](https://apalache-mc.org/docs/apalache/installation/jvm.html)。公式のシステム要件は [4 GB 以上のメモリー](https://apalache-mc.org/docs/apalache/installation/index.html)を推奨する。 | [Apache License 2.0](https://github.com/apalache-mc/apalache/blob/v0.62.2/LICENSE) |

Quint `v0.32.0` の統合経路が使う検査器は、各検査器の現行安定版を独立に選ぶ構成ではない。Quint は既定の Apalache を [`0.56.1` に固定](https://github.com/quint-co/quint/blob/v0.32.0/quint/src/apalache.ts)し、`--apalache-version` で変更できるものの、[互換性を失う可能性を CLI の説明に明記する](https://github.com/quint-co/quint/blob/v0.32.0/quint/src/cli.ts)。したがって、Apalache `0.62.2` が現行版であることは、Quint `v0.32.0` から安全に差し替えられることを意味しない。

Quint が管理する Apalache 配布物は、[`QUINT_HOME`、未指定時は `~/.quint` の `apalache-dist-<version>`](https://github.com/quint-co/quint/blob/v0.32.0/quint/src/config.ts) に展開される。公式の配布試験は、初回に取得し、二回目に同じ配布物を再利用することを[検査している](https://github.com/quint-co/quint/blob/v0.32.0/quint/integration-tests/distribution/apalache.md)。したがって、cold 実行は GitHub からの取得と展開を含み、warm 実行はこの永続キャッシュを再利用する。

Quint `v0.32.0` の TLC backend は独立した `tla2tools.jar` を取得せず、[先に指定版の Apalache 配布物を確保する](https://github.com/quint-co/quint/blob/v0.32.0/quint/src/verify.ts)と、その配布物の [`apalache/lib/apalache.jar` をクラスパスにして `tlc2.TLC` を起動する](https://github.com/quint-co/quint/blob/v0.32.0/quint/src/tlc.ts)。Apalache `v0.56.1` は [`tla2tools` `1.7.4` を固定している](https://github.com/apalache-mc/apalache/blob/v0.56.1/project/Dependencies.scala)ため、Quint の既定経路で実際に評価する組み合わせは Quint `0.32.0`、Apalache `0.56.1`、TLA+ Tools/TLC `1.7.4` となる。TLC の作業用 `quint-tlc-*` ディレクトリは OS の一時ディレクトリに作られ、[終了時に削除される](https://github.com/quint-co/quint/blob/v0.32.0/quint/src/tlc.ts)ため、TLC backend の永続キャッシュも Apalache 配布物である。

### JSON trace output

Quint の実装接続に使える JSON トレースは、Quint IR を書く `--out` ではなく ITF を書く `--out-itf` で生成する。`quint run` は [`--out-itf`、`--n-traces`、`--seed` を備え](https://github.com/quint-co/quint/blob/v0.32.0/docs/content/docs/quint.md#command-run)、`--mbt` を付けると各 step に `mbt::actionTaken` と `mbt::nondetPicks` を追加するが、[このメタデータ機能は実験的で変更されやすい](https://github.com/quint-co/quint/blob/v0.32.0/docs/content/docs/quint.md#the---mbt-flag)。したがって、T006 の Go ドライバー入力は `quint run --mbt --seed=<fixed> --out-itf=<pattern>` で生成し、メタデータの形を Quint `0.32.0` に固定して解釈する。

ITF は `vars`、状態列 `states`、任意の `loop` と `#meta` を持つ JSON object であり、整数は JSON number ではなく `{ "#bigint": "<digits>" }`、集合は `#set`、写像は `#map`、variant は `tag` と `value` で表す。[公式の ITF 定義](https://apalache-mc.org/docs/adr/015adr-trace.html)に従い、Go 側では通常の `int64` へ直接デコードせず、範囲を検査してから変換する必要がある。

`quint verify --out-itf` は [Apalache backend にだけ対応する](https://github.com/quint-co/quint/blob/v0.32.0/docs/content/docs/quint.md#command-verify)。TLC backend は反例を標準出力へ表示するが Quint から ITF を保存できないため、T006 の正常トレース生成には `quint run --mbt` を使い、Apalache の反例保存には `quint verify --backend=apalache --out-itf` を使い、TLC の成否と診断は別の証拠として保存する。

### Proposed mise pinning and cache policy

次は調査結果から導く評価用の提案であり、T003 で実際の取得と実行を検証するまで採用済みの構成ではない。

```toml
[tools]
"github:quint-co/quint" = "0.32.0"
java = "temurin-17.0.20+101"
```

Quint には、公式リリースの単一実行ファイルを OS と architecture に応じて選ぶ `github` backend を第一候補とする。mise の `github` backend は [OS と architecture から asset を自動選択し、単一実行ファイルの OS/architecture suffix を自動で除く](https://mise.jdx.dev/dev-tools/backends/github.html#asset-autodetection)ため、Quint の四つの macOS/Linux asset と対応する。さらに `github` backend は [`mise.lock` に版、URL、checksum、size を記録できる](https://mise.jdx.dev/dev-tools/mise-lock.html#backend-support)ため、npm backend より取得物の固定が強い。npm 版を使う場合の代案は `"npm:@informalsystems/quint" = "0.32.0"` と Node.js 18 以上の追加であるが、npm backend の lock は版だけを記録し、[Quint 自体も Node.js を実行時に要求する](https://github.com/quint-co/quint/blob/v0.32.0/quint/package.json)ため、導入物が増える。

Java は Apalache の推奨に合わせて JDK 17 を選び、調査日に `mise latest java@temurin-17` が返した正確な版 `temurin-17.0.20+101` を候補にした。mise は [Temurin の vendor prefix と major versionを指定でき、active installation の `JAVA_HOME` を設定する](https://mise.jdx.dev/lang/java.html)ため、同じ JDK が Apalache と TLC の双方を満たす。

各 `mise` 検証 task は `--apalache-version=0.56.1` を明記し、`QUINT_HOME` を評価専用の一定パスへ設定する。これにより、Quint の暗黙の既定値が将来変わっても検査器の版を task から確認でき、CI は `QUINT_HOME/apalache-dist-0.56.1` を明示的な cache key として扱える。ただし、Apalache 配布物の取得自体は Quint が担い、mise の `github` backend と `mise.lock` の checksum 検証を通らないため、T003 では取得物の SHA-256 を別途検査するか、固定 URL と checksum を持つ mise tool stub へ移す必要がある。

Apalache `0.62.2` と TLA+ Tools/TLC `1.7.4` の standalone 配布物を mise で別に固定しても、Quint が内部で使う検査器は置き換わらない。現行 Apalache との比較が必要なら、[`github` backend の `asset_pattern`、`bin_path`、checksum](https://mise.jdx.dev/dev-tools/backends/github.html#tool-options)または checksum を記録できる `http` backend を使う独立 task とし、Quint の統合検査とは結果を分ける。

この調査だけで確認できた対応範囲は、Quint の単一実行ファイルと Temurin 17 が提供される macOS `amd64`/`arm64` と Linux `amd64`/`arm64` である。fresh checkout の実行結果、Apalache が同梱する Z3 の各 architecture での動作、cold と warm の所要時間、`mise.lock` が生成する各 platform の checksum は未測定であり、T003 と T007 の実証対象として残す。

## Verification

- **Intended Acceptance RED**: `N/A: tooling evaluation with no normative product requirement`。`T002` で、`states.md` だけから状態または遷移を除いた fixture と Quint 側だけから除いた fixture の双方に対し、評価用同期検査が不一致を報告して失敗することを観測する。
- **Intended Unit RED**: `N/A: 恒久的な製品ロジックを追加せず、評価用資材は完了時に撤去する`。代替として `T006` で、リース所有者でない `worker` の完了を許すテスト用誤実装を模型生成トレースのドライバーが拒否することを観測する。
- 片側だけを変更した二つの同期 fixture が、想定した不一致を報告して失敗する。
- 安全性を壊した模型が期待した反例を返し、元の模型では同じ性質の検査が成功する。
- 模型生成トレースの Go ドライバーが現行実装を受理し、故意に誤らせたテスト用実装を拒否する。
- 評価用 `mise` タスクで Quint の型検査、固定 seed シミュレーション、TLC と Apalache の有界検査を再現できる。
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

## Risk Notes

最大のリスクは、検査済みの模型が正本または Go 実装から乖離したまま成功し、保証が増えたように見えることである。正本からの生成または双方向同期検査と、模型生成トレースを使う実装テストの両方を採用条件にし、どちらか一方だけでは Quint を採用しない。

Quint は表層言語、変換、検査器という複数層を追加する。反例の位置が原模型へ正しく対応しない場合や、Apalache の自動取得が版固定と再現性を損なう場合は、直接 TLA+ より学習しやすく見えても保守費用が下がらない。検査器ごとの成否と診断を分けて記録する。

模型の状態空間を小さくする抽象化は、実装の詳細を落とす。二つの `worker`、一つの Job、有限の時刻という境界を明示し、その範囲外の正しさを主張しない。モデル検査の成功とトレース駆動テストの成功も別々の証拠として扱う。

## T001 Corrections

試作の過程で、T001 の記録に 2 点の誤りが見つかった。

`quint run` は Apalache とは別に **Rust evaluator** を実行時に取得する。取得先は Quint 自身のリリース資産で、展開先は `QUINT_HOME/rust-evaluator-v0.6.0` である。T001 は Quint が取得する配布物を Apalache だけとして扱っていたが、版固定と検査の対象になる取得物は Quint 本体、Rust evaluator、Apalache 配布物の 3 つである。

`mise.lock` による取得物の固定は、このリポジトリでは現状成立しない。リポジトリに `mise.lock` が無く、`github` backend が checksum を記録できるという T001 の利点は、lockfile を導入するという別の方針決定を前提にしている。Quint を採用するなら、その決定も同時に要る。

## T002–T007 Prototype Results

### 配置と機構（T003、T004）

試作は次の 4 つで構成した。全て評価用であり、T008 で撤去した。

| 場所 | 役割 | 行数 |
|---|---|---|
| `tools/model-sync/src/` | `states.md` の `JobLifecycle` から Quint の基礎モジュールを生成し、`--check` でバイト比較する | 228 |
| `backend/jobs/model/joblifecycle.qnt` | 生成物。状態、事象、遷移表、`jobTargets`、`isJobTerminal` | 47 |
| `backend/jobs/model/jobs_lease.qnt` | 手書きのリース模型、安全性、witness | 276 |
| `backend/jobs/model/{itf.go,trace_driver_test.go}` | ITF デコーダーと Go テストドライバー | 626 |

生成モジュールは死重にならない。手書き模型は遷移先を自分で書かず `jobTargets(Queued, JobStarted)` のように生成モジュールへ問い合わせるので、`states.md` の `To` 列を変えると模型の遷移も変わる。生成できたのは状態、事象、遷移の対応関係までである。Guard 列は `attempts >= max_attempts` のように表が定義していないデータを指す自然言語であり、文字列として運ぶことしかできない。手書き模型が `MAX_ATTEMPTS` と結び付けている。

リースは worker ごとの付与期限の写像 `leases: str -> int` として書いた。実装は `lease_owner` 列 1 つなので、これは意図的に弱い符号化である。相互排他が「1 つの欄には 1 人しか書けない」という表現の帰結ではなく、「取得は前の付与を取り消す」という規則の帰結として検査対象になる。

効果は全て明示した。時刻は `tick` だけが進め、worker の選択は `nondet`、永続化は状態変数そのものである。模型のどこにも時計を読む式と識別子を生成する式が無い。

### 同期検査の RED（T002）

`mise run check-model-sync` は片側だけの変更を両方向で拒否した。

`states.md` から `running --JobCanceled--> canceled` の行だけを削ると、`line 33 / specification: ")" / model: "{ from: Running, event: JobCanceled, ... }"` を報告して失敗した。逆に `.qnt` へ `{ from: Succeeded, event: JobStarted, guard: "", to: Running }` を手で足すと、`line 29` の不一致を報告して失敗した。

後者について重要な観測がある。**この手書き改変は `mise run model-typecheck` を通過した。** 終端状態から `running` へ戻る遷移を模型へ足しても型は合う。同期を担保しているのは型検査ではなくバイト比較であり、Quint の型検査を同期の証拠として数えてはならない。

### 安全性、witness、反例（T005）

修正後の模型に対する結果。

| 性質 | 意味 | シミュレーション | TLC | Apalache |
|---|---|---|---|---|
| `leaseExclusivity` | 有効なリース付与を持つ worker は同時に 1 人以下 | 成立 | 成立 | 成立 |
| `commitRequiresLease` | 付与を持たない worker の完了と失敗は確定しない | 成立 | 成立 | 成立 |
| `terminalIsFinal` | 終端に達した Job は再び `running` にならない | 成立 | 成立 | 成立 |
| `reclaimWitness` | REQ-JOBS-004 の再取得後に別 worker が完了する | 到達 | — | — |
| `deadLetterWitness` | 試行上限に達して配信不能へ落ちる | 到達 | — | — |

TLC は網羅探索で 490 個の異なる状態、完全な状態グラフの深さ 10 を確認した。Apalache は 12 step の有界検査である。同じ規模の模型に対して、TLC のほうが強い結果を速く返した。

意図的な欠陥を 3 つ入れ、それぞれ期待した反例を得た。

1. **リース排他**：`claimExpired` の再取得条件を「誰も有効な付与を持たない」から「呼び出した worker が付与を持たない」へ弱め、取得が前の付与を取り消さないようにした。TLC が 3 状態の反例を返した。`worker-2` が取得（付与期限 2）、`worker-1` が同じ時刻に再取得、両者の付与が有効。Apalache も同じ反例を ITF で保存した。
2. **リースなし確定**：`complete` から `holdsLease(w)` を外した。反例は `worker-1` が取得して heartbeat した Job を `worker-2` が完了させる 4 状態である。
3. **終端の不可逆性**：`claimQueued` から `status == Queued` を外した。反例は `cancel` で `canceled` に達した Job を `claimQueued` が `running` へ戻す 3 状態である。

1 について、最初に試した変異は**等価変異**だった。取得が前の付与を取り消さないようにするだけでは `leaseExclusivity` は破れない。`claimExpired` が「誰も有効な付与を持たない」ことを要求している限り、取り消し忘れた付与は既に失効しているからである。この性質を実際に支えている条件は取り消しではなく再取得の判定であり、そこを壊して初めて反例が出た。等価変異を隠さず記録する。

### 模型駆動の実装テスト（T006）

`quint run --out-itf` が生成した 7 本のトレース（固定 seed のサンプル 5 本と witness 2 本）で、`db_memory.JobRepository` を各 step 駆動した。期待状態は手で書かず、ITF から読んだ。照合対象は `status`、`attempts`、`run_at`、有効な付与を持つ worker、その付与の期限、そして終端の不可逆性である。

さらに、模型が**拒否すると言う操作**を各 `running` 状態で実際に呼んだ。付与を持たない worker からの `Complete`、`Fail`、`Heartbeat` である。どの worker が権利を持たないかは模型が決めるので、否定側の事例も列挙ではなく導出である。拒否については、返った誤りと、拒否が状態を変えていないことの両方を表明した。

故意に誤らせた実装を 2 つ用意し、両方とも 7 本すべてで検出した。

| 誤実装 | 何を模したか | 捕まえた表明 |
|---|---|---|
| `Complete` が呼び出し元の主張する所有権を信じる | 条件付き更新の `WHERE` から `AND lease_owner = $2` が落ちた形 | 返った誤り |
| `Complete` が正しい番兵を返しつつ確定させる | 「拒否したと書いて、そのうえで実行する」形 | 拒否後に読み戻した状態 |

2 つ目は 1 つ目の表明では捕まらない。誤りの表明だけを持つドライバーは、拒否を宣言して実行する実装に対しても同じように通る。

### 再現性、費用、実行時間（T007）

macOS `arm64`（Apple Silicon）での実測。`quint` が自己申告する時間ではなく、`mise run` の経過時間である。

| タスク | 内容 | warm | cold |
|---|---|---|---|
| `check-model-sync` | 生成とバイト比較 | 0.10 秒 | — |
| `model-typecheck` | Quint の型検査 | 0.37 秒 | — |
| `model-simulate` | 固定 seed、20000 サンプル、12 step | 2.4 秒 | 4.8 秒 |
| `model-verify-tlc` | TLC の網羅検査 | 4.0 秒 | — |
| `model-verify` | Apalache の 12 step 有界検査 | 14.1 秒 | 18.7 秒 |
| `verify` | 試作を含む標準検証一式 | 30.2 秒 | — |

いずれも 60 秒の境界内である。性能は制約にならない。試作を置いたまま `mise run verify` が通ることも確認した。

取得物は 4 つ、合計約 560 MB である。

| 取得物 | 版 | 大きさ | 取得者 | このリポジトリが検証できる checksum |
|---|---|---|---|---|
| Quint 単一実行ファイル | 0.32.0 | 104 MB | mise `github` backend | `mise.lock` があれば。無い |
| Rust evaluator | v0.6.0 | 2.2 MB | Quint 自身 | 無し |
| Apalache 配布物 | 0.56.1 | 144 MB | Quint 自身 | 無し |
| Temurin JDK | 17.0.20.1+1 | 309 MB | mise | あり |

`QUINT_HOME` をリポジトリ内へ固定すると、Quint が取得する 2 つは開発者のホームから出て cache key として名前を付けられる。ただし名前が付くことと中身を検証できることは別である。

macOS `arm64` と Linux `arm64`（Ubuntu 24.04、Docker）で、同じ模型、同じ性質、同じ固定 seed が同じ成否を返した。正規化後のトレースの SHA-256 も 5 本すべて一致した。

ただし **Debian 12 では `quint run` が動かない**。Rust evaluator が GLIBC 2.39 を要求し、Debian 12 の 2.36 では `version 'GLIBC_2.39' not found` で終わる。型検査は通るので、失敗するのはシミュレーションとトレース生成である。CI の `ubuntu-latest` は現在 24.04（GLIBC 2.39）なので動くが、この下限はどこにも宣言されておらず、runner 像が 1 つ戻れば止まる。

### 記録すべき所見

**F1. `x' = a or b` は `(x' = a) or b` と解釈される。** 代入のつもりで書いた行が黙って「代入するか、さもなくば `b`」という論理式になる。10 行の模型で単離して確認した。`a' = a or (n == 0)` は latch せず、`a' = (a or (n == 0))` は latch する。同じファイルで括弧 1 組の違いであり、**両方とも型検査を通り、両方ともシミュレーターが受理する**。この評価では実際に踏んだ。`everTerminal` が一度も真にならない模型ができ、`terminalIsFinal` が空虚な不変条件のまま緑になっていた。気づいた経路は Apalache の拒否である。

  これは Quint 固有の欠陥ではない。TLA+ の `a' = a \/ b` も同じ結合順になる。Quint 固有なのは、型検査器がこれを捕まえないことと、速いシミュレーターが編集周期の中で緑を返し続けることで、検査器を呼ぶ動機が遠のくことである。

**F2. Apalache の診断は Quint のソースを指さない。** F1 の症状は `Assignment error: <[UNKNOWN]>: Missing assignments to: sawUnleasedCommit` であり、`.qnt` の行番号が無い。TLC の反例は `<claimQueued line 326, col 5 ... of module jobs_lease>` と行を出すが、これは**生成された TLA+ モジュール**の行であり、276 行の `.qnt` には存在しない。

**F3. `--out-itf` に `{}` の置換は無い。** Quint 0.32.0 はトレース番号を `--out-itf` に与えたパス**全体の最初のドットの直前**へ差し込む。`out{}.itf.json` は `out{}0.itf.json` になり、作業ディレクトリ名にドットがあると出力先が別の場所になる。

**F4. ITF の `#meta` に実時刻が入る。** 固定 seed でも同じファイルにならないので、トレースを収めて差分で読むには `#meta` を落とす正規化が要る。

**F5. `--mbt` のメタデータは状態 0 でずれる。** 初期状態の `mbt::actionTaken` が `"step"` になり、`mbt::nondetPicks` が次の step の選択を指す。この機能は公式に実験的と宣言されている。ghost 変数で `lastAction` と `lastActor` を持つほうが安定で、Go 側の解釈も単純だった。

**F6. TLC のほうが強い結果を速く返した。** この規模では TLC が網羅（490 状態、深さ 10）を 0.8 秒、Apalache が 12 step 有界を 13.3 秒である。ただし ITF で反例を保存できるのは Apalache だけなので、両方を残す理由はある。

**F7. Apalache は作業ディレクトリに `_apalache-out/` を残す。** 掃除は呼び出し側の責任である。

**F8. 模型定数は ITF に現れない。** `LEASE_DURATION`、`RETRY_BACKOFF`、`MAX_ATTEMPTS` は `pure val` であって `var` ではないので、Go ドライバーは値を再記述するしかなかった。状態機械より小さいが、これも本物の乖離面である。恒久導入するなら、`model-sync` が状態機械について閉じたのと同じように閉じる必要がある。

**F10. トレースドライバーの拒否試験は、既存のテストが持っていない表明を自動的に持つ。** `backend/jobs/db_memory/repository_test.go` の `TestComplete_WrongWorkerReturnsErrJobLeaseLost` は、返った誤りだけを表明していて、拒否が Job を変えていないことを読み戻していない。つまり 2 つ目の誤実装（正しい番兵を返しつつ確定させる）は、現行のテスト一式を通過する。現行の `db_memory` 実装自体は正しいので製品の欠陥ではないが、この欠陥種別に対する回帰検出が無い。

  この種別はリポジトリ全体で既に負債として追跡されており、`mise run report-security-test-gaps` は「拒否をテストしている 123 箇所のうち 87 箇所が無作用を読み戻していない」と報告する。Jobs のリース拒否はこの報告に現れない。宣言された security control ではないからである。注目すべきなのは、トレースドライバーがこの表明を**注釈も宣言も無しに**持ったことである。模型が「この worker は権利を持たない」と言う以上、拒否の確認は駆動の一部として自然に出てくる。負債の所有は既に別にあるので、ここから新しい work item は起票しない。

**F9. シミュレーターが最初に見つけたのは製品の欠陥ではなく模型の欠陥だった。** リース未付与を `0` で表したため `clock == 0` で両 worker が保持者と判定され、初期状態で `leaseExclusivity` が落ちた。`NO_GRANT = -1` にして解決した。安いフィードバックではあるが、これは模型の debug であって実装の検証ではない。

### 工程ごとのワークフロー差分（試作）

採用しなかったので適用していない。採用する場合に `docs/development/specification-first-workflow.md` と `docs/development/testing.md` へ入れるべき内容は次のとおりである。

`specification-first-workflow.md` の「3. The loop」には、`states.md` を変えたときの生成物の再生成と同期検査を、仕様段階のゲートとして 1 行足す。「5. Verification ladder」には、型検査と固定 seed のシミュレーションを 3 の層（narrowest test recipe）と同じ位置に、モデル検査を 4 の層（change-resistance）と同じ位置に置く。`mise run verify` には入れない。

同じ文書に、3 つの検査が別々のことを言っているという区別を書く。モデル検査は**模型の**性質を検査する。トレース駆動テストは模型と Go 実装の**適合**を検査する。正本の意味そのものを変える発見は、どちらでもなく仕様段階へ戻る。この 3 つを混ぜると、検査が通ったことが何を保証しているのか言えなくなる。

`testing.md` の水準表には行を足さない。模型駆動のトレースは新しい水準ではなく、**既存の「アダプター統合」の事例の作り方**だからである。代わりにテストダブルの節へ、期待状態を模型から読むこと、否定側の事例も模型が決めることを書く。

## Decision (T008)

**Quint は採用しない。** `wi-421` は表層言語も直接 TLA+ とする方針を維持する。

Decision Criteria に対する判定は次のとおりである。

| 条件 | 判定 | 根拠 |
|---|---|---|
| 有限の模型で 3 つの安全性と witness を、未解決の型エラーや検査器固有の意味差なしに検査できる | **不成立** | 表現力は足りた。しかしシミュレーターが受理する模型を Apalache が不正形として拒否する差があり（F1、F2）、しかもその差は最初に書いた模型で踏んだ |
| 安全性を壊すと期待した反例が出て、修正後は成功する | 成立 | 3 つ全てで確認。等価変異も記録した |
| `states.md` と模型の片側だけの差分を両方向で拒否する | 成立 | 両方向で観測した。型検査は通ってしまうので、バイト比較が担っている |
| 模型生成トレースがアダプターを駆動し、誤実装を拒否する | 成立 | 7 本のトレース、2 つの誤実装、異なる 2 つの表明が捕捉 |
| 版、取得物、cache を `mise` から再現でき、未管理の大域インストールを要求しない | **不成立** | 4 つの取得物のうち 2 つ（Rust evaluator、Apalache 配布物、合わせて 146 MB）を Quint 自身が取得し、このリポジトリが検証できる checksum が無い。`mise.lock` も無い。加えて Rust evaluator の GLIBC 2.39 要求が Debian 12 と Ubuntu 22.04 を宣言なく除外する |
| 型検査とシミュレーションが編集周期に載り、モデル検査が 60 秒以内 | 成立 | 0.37 秒、2.4 秒、4.0 秒（TLC）、14.1 秒（Apalache） |
| ワークフロー文書が 3 種類の検査を区別して説明できる | 成立 | 上に差分を試作した |

不成立が 2 つあるので、Decision Criteria の規定に従って不採用とする。性能は境界内なので、この判断は性能を理由としない。

判断の実質は再現性である。Quint の表現力、診断の読みやすさ、シミュレーションの速さは、直接 TLA+ を書くより明確に良かった。一方で、Quint を入れることは版が独立に動く 4 つの取得物と約 560 MB を入れることであり、そのうち 2 つはリポジトリの版固定の外にある。直接 TLA+ なら `tla2tools.jar` 1 つと JDK で足りる。この差は、模型を書きやすくなる分では埋まらない。

**この評価から `wi-421` が持ち帰るもの**は、Quint に依存しない 2 つの機構である。`states.md` から模型の基礎部分を生成してバイト比較する同期経路と、ITF トレースで Go のアダプターを駆動し、模型が拒否すると言う操作の無作用まで確かめるテストドライバーである。ITF は Apalache 自身の形式なので TLA+ を直接検査しても得られるはずだが、本評価は Quint 経由でしか確認していない。`wi-421` が最初に確かめる。

**再評価の条件**は次のいずれかである。

- Quint が Rust evaluator の GLIBC 下限を下げるか静的リンクし、かつ evaluator と Apalache 配布物の checksum を利用側が検証できるようにする。
- Quint が検査器と evaluator を `mise` が管理するパスから受け取れるようになり、自分では取得しなくなる。
- このリポジトリが別の理由で `mise.lock` を導入する。これだけでは Quint 自身の取得分が残るので、単独では十分でない。
- `wi-421` が直接 TLA+ で進めた結果、律速が模型を書くことそのものだと判明する。本評価の時点では、律速は模型の設計（何を状態に持ち、何を ghost にするか）であって記法ではなかった。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff main` は `no normative specification change against main` を返す。規範シナリオ、状態表、遷移表、標準の行、TypeSpec シンボルのいずれも変わっていない。宣言した `spec_impact: none` のとおりである。

  この work item が残した変更は work item 2 件だけである。`wi-435` に T001 の訂正、試作結果、採否の判断、再評価条件を記録した。`wi-421` は `depends_on` の解決に伴い、表層言語を「`wi-435` の結果しだい」から直接 TLA+ へ確定し、引き継ぐ 2 つの機構と、確かめるべき前提（`tla2tools.jar` の取得物が `mise.toml` だけで固定できること、Apalache が TLA+ を直接検査しても ITF を出せること）を書き足した。

  試作した資材は Out of Scope の定めどおり全て撤去した。撤去したのは `backend/jobs/model/`（Quint 模型 2 本、ITF デコーダー、Go トレースドライバー、生成トレース 7 本）、`tools/model-sync/`、`mise.toml` の `[tools]` 2 行と `[env]` と評価用タスク 8 個、`.gitignore` の 1 行である。撤去後の作業ツリーは work item 2 件の変更だけを含む。

- **Acceptance RED Evidence**:
  - **Test**: `mise run check-model-sync`（評価用タスク。`tools/model-sync/src/main.ts --check`）
  - **Requirement**: N/A: 道具の評価であり、対応する規範的な製品要件を持たない。
  - **Observed Failure**: 片側だけを変えた 2 つの fixture の両方で失敗した。`docs/contexts/jobs/states.md` から `| running | JobCanceled | — | canceled | |` の行だけを削ると、`fail backend/jobs/model/joblifecycle.qnt does not match ... line 33 / specification: ")" / model: "{ from: Running, event: JobCanceled, guard: "", to: Canceled },"` を出して非ゼロ終了した。逆に `joblifecycle.qnt` へ `{ from: Succeeded, event: JobStarted, guard: "", to: Running },` を手で足すと、`line 29` の不一致を出して非ゼロ終了した。どちらも復元すると `ok` に戻った。
  - **Detection Reason**: 受け入れ境界が適用できないので、代わりに失敗させた検査がこれである。妥当な誤り方は「`states.md` を直して模型の再生成を忘れる」と「模型だけ手で直す」の 2 つで、方向が逆なので片方向の検査では取り逃がす。生成結果とファイルのバイト比較は方向を持たないため両方で落ちる。この検査が担っていることの証拠として、**手書き改変のほうは `mise run model-typecheck` を通過した**ことも観測した。終端状態から `running` へ戻る遷移を足しても Quint の型は合うので、型検査を同期の証拠に数えることはできない。

- **Unit RED Evidence**:
  - **Test**: `go test ./backend/jobs/model/ -run TestModelTracesRejectWrongImplementations`（評価用テストドライバー）
  - **Requirement**: N/A: 恒久的な製品ロジックを追加せず、評価用資材は完了時に撤去した。
  - **Observed Failure**: 故意に誤らせた実装 2 つを、生成トレース 7 本すべてで検出した。1 つ目（`Complete` が呼び出し元の主張する所有権を信じる）は `state 1: Complete by "worker-2", which the model says holds no lease, returned <nil>; want jobs: lease lost`。2 つ目（正しい番兵を返しつつ確定させる）は `state 1: refused Complete by "worker-2" still changed the Job: status running -> succeeded, lease_owner worker-1 -> <nil>, lease_expires_at 2026-08-29T00:02:00Z -> <nil>, result written`。現行の `db_memory.JobRepository` は同じドライバーで 7 本すべて通過した。
  - **Detection Reason**: 誤実装を分けたのは、2 つが別々の表明にしか捕まらないからである。1 つ目は返った誤りで、2 つ目は拒否後に読み戻した状態でしか区別できない。誤りの表明だけを持つドライバーは、「拒否したと書いて、そのうえで実行する」実装に対しても通る。期待状態は手書きせず ITF から読み、どの worker が権利を持たないかも模型が決めるので、否定側の事例も著者の想定の下流にない。

- **Change-Resistance Results**:
  `medium` の要求は代表的な誤実装 1 つの検出だが、性質ごとに 1 つずつ、計 5 つの故意の欠陥を入れて結果を記録した。

  | 対象 | 入れた欠陥 | 検出 |
  |---|---|---|
  | 模型 `leaseExclusivity` | `claimExpired` の再取得条件を「誰も有効な付与を持たない」から「呼び出した worker が付与を持たない」へ弱め、取得が前の付与を取り消さないようにした | TLC が 3 状態の反例。Apalache も同じ反例を ITF で保存 |
  | 模型 `commitRequiresLease` | `complete` から `holdsLease(w)` を外した | TLC が 4 状態の反例 |
  | 模型 `terminalIsFinal` | `claimQueued` から `status == Queued` を外した | TLC が 3 状態の反例 |
  | Go アダプター | `Complete` が呼び出し元の主張する所有権を信じる | トレースドライバーが 7 本すべてで検出 |
  | Go アダプター | `Complete` が正しい番兵を返しつつ確定させる | トレースドライバーが 7 本すべてで検出 |

  欠陥を戻すと、5 つとも元の検査が成功に戻った。

  **等価変異**: `leaseExclusivity` について最初に試した「取得が前の付与を取り消さない」という単独の変異は、反例を生まなかった。`claimExpired` が「誰も有効な付与を持たない」ことを要求している限り、取り消し忘れた付与は既に失効しているからである。この性質を実際に支えているのは取り消しではなく再取得の判定であり、そこを合わせて壊して初めて反例が出た。

  **手法の限界**: 模型側の欠陥注入は、模型が表現している範囲についてしか何も言わない。範囲は worker 2 名、Job 1 件、レーン 1 つ、論理時刻上限 4、試行上限 2 である。テナント境界、バッチ取得、`dedup_key`、バックオフ曲線、PostgreSQL アダプターは模型にも駆動にも入っていない。また Apalache は 12 step の有界検査であり、TLC は網羅（490 状態、深さ 10）だが、いずれもこの有限の構成についての結果である。模型検査の成功とトレース駆動テストの成功は別々の証拠であり、後者が言えるのは「この 7 本のトレースが辿った範囲で、模型と `db_memory` アダプターの観測状態が一致する」ことだけである。

  この評価はこの過程で**自分の模型の欠陥を 2 つ**見つけている。リース未付与の番兵を `0` にしたために初期状態で両 worker が保持者と判定された件（シミュレーターが検出）と、`x' = a or b` が `(x' = a) or b` と解釈されて `everTerminal` が一度も真にならず `terminalIsFinal` が空虚な不変条件になっていた件（Apalache だけが検出）である。後者は型検査もシミュレーションも通過していた。製品の欠陥は 1 件も見つかっていない。

- **Verification Results**:
  - `mise run check-model-sync`（評価用、撤去済み）- 両方向の RED を観測し、復元後に通過
  - `mise run model-typecheck` / `model-simulate` / `model-verify-tlc` / `model-verify`（評価用、撤去済み）- 3 つの安全性が成立、2 つの witness が到達、5 つの故意の欠陥が期待どおり検出
  - Linux 再現性（Docker、Ubuntu 24.04 `arm64`）- 同じ模型、同じ固定 seed が同じ成否を返し、正規化後のトレースの SHA-256 が macOS `arm64` と 5 本すべて一致。Debian 12 では Rust evaluator が GLIBC 2.39 を要求して `quint run` が起動しない
  - `mise run verify`（試作を含む状態）- passed（30.2 秒）
  - `mise run verify`（試作を撤去した最終状態）- passed
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
