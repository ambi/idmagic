---
status: in_progress
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
    - docs/contexts/jobs/internals.md
  typespec: [IdMagic.Contract.JobStatus]
  source:
    - backend/jobs/domain/job.go
    - backend/jobs/ports/repository.go
    - backend/jobs/db_memory/repository.go
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
- [ ] T002 [Acceptance] `states.md` と Quint 側の片方だけを変えた fixture が同期検査で失敗する RED を両方向について観測する。
- [ ] T003 [Tooling] 評価用の Quint 基礎モジュール生成または双方向同期検査と、型検査、固定 seed シミュレーション、モデル検査の `mise` タスクを試作する。
- [ ] T004 [Model] Jobs のリース模型、三つの安全性、一つ以上の witness を書き、時刻、worker 選択、永続化を明示的な状態または入力として分離する。
- [ ] T005 [Acceptance] 安全性を壊した模型で期待する反例を観測し、修正後に TLC と Apalache の有界検査を成功させる。
- [ ] T006 [Test] Quint の JSON トレースで Jobs のメモリ永続化アダプターを駆動し、現行実装と故意に誤らせたテスト用実装を区別する Go テストドライバーを試作する。
- [ ] T007 [Verify] macOS と Linux の再現性、cold と warm の実行時間、取得物、キャッシュ、診断、変更量を測り、工程ごとのワークフロー差分を試作する。
- [ ] T008 [Decision] 採否、ゲート配置、再評価条件を Completion に記録し、評価用資材を撤去する。採用時だけ恒久導入用 work item を起票して `wi-421` を同期する。
- [ ] T009 [Verify] `mise run verify` と work item の検査を通し、独立した検証者が公式文書、試作結果、採否を照合する。

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
