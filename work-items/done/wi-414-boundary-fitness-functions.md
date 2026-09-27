---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-08-27
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "docs/domain/structure.md へ公開言語と依存方向の現在の設計を書き、それを検査するが、規範シナリオ、規範 ID、TypeSpec シンボルを追加も変更もしない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 開発時の境界検査と現在のコード構造の説明だけを変更し、利用者向けの機能や運用手順は変わらない。
  references: []
initial_context:
  specification:
    - docs/design/architecture/logical.md
    - docs/domain/structure.md
  typespec: []
  source:
    - tools/check/src/check-boundaries.ts
    - tools/check/src/registry.ts
    - tools/check/src/runner.ts
    - tools/workspace/src/workspace.ts
    - mise.toml
  tests:
    - tools/check/src/worktree-exclusion.test.ts
  stop_before_reading:
    - frontend
---

# Context 境界と作用の禁止を機械検査する適応度関数を足す

## Motivation

`docs/domain/structure.md` は「Context 間は公開された言語とポートで接続する」と宣言し、`docs/design/architecture/logical.md` の Context Map は Supplier から Customer への関係と種類（OHS/PL、C/S、ACL）を型付けしている。しかし `tools/check/src/check-boundaries.ts` が検査するのは層の方向（`domain` が `usecases`/`handlers_*`/`db_*` を import しないこと、`usecases` が外向きに import しないこと）と起動時設定の読み取り点だけであり、Context 間の依存は 1 行も見ていない。

実際の import グラフは宣言と一致していない。`backend/oauth2` は `authentication/mfa/usecases`、`authentication/session/usecases`、`authentication/totp/usecases`、`authentication/webauthn/handlers_http` を直接 import している。Context Map の矢印は Supplier から Customer へ向くため、Go の利用側の import は原則として矢印の逆向きになる。その向きを踏まえても、実装は `oauth2 → authentication` と `authentication → oauth2` の双方向であり、`idmanagement ↔ authentication`、`tenancy ↔ idmanagement` にも循環がある。公開言語であるはずの `domain` と `ports` を越えて、他 Context の `usecases` と `handlers_http` に到達している。

Modular Monolith の全体重は「モジュールの内部が外から見えないこと」に乗っている。それが慣習だけで支えられている限り、Context Map は現在の設計の記述ではなく努力目標にすぎず、`docs/development/specification-first-workflow.md` が求める「現在の設計は正本文書と work item だけから理解できる」状態を満たさない。同じ理由で、`domain` が純粋であるという性質もどこにも書かれておらず検査もされていないため、`domain` の中で `time.Now()` を呼ぶことは今のところ自由である。

既存の違反は具体的な安定 ID と理由を持つ ledger に固定し、Git の基準 revision と比較する ratchet で新規の違反だけを止める。これにより、既存コードを一括で直さずに境界を縮める方向だけを許可する。

## Scope

- **公開言語の定義**：`docs/domain/structure.md` に、Context の外へ公開されるのは `domain` と `ports` だけであること、他 Context の `usecases`、`handlers_*`、`db_*` へ到達してはならないことを書く。
- **Context 間の禁止依存**：`check-boundaries` に、他 Context の非公開パッケージへの import を拒否する規則を足す。
- **循環の非存在**：Context 単位の依存グラフを組み立て、循環を拒否する。
- **Context Map との一致**：`docs/design/architecture/logical.md` の Context Map の Supplier → Customer の矢印を読み取り、Customer から Supplier への利用として説明できない実 import 辺を拒否する。Context Map を機械可読な正本として扱えるようにする。
- **`domain` の作用禁止**：`domain` パッケージが `time.Now`、`math/rand`、`crypto/rand`、`os`、`net`、`database/sql` を参照することを拒否する。作用は引数として入るという `docs/development/specification-first-workflow.md` の規律を、変更時の手順ではなくシステムの現在の性質として固定する。
- **迂回の検出**：`backend/shared/` を経由して禁止された方向へ到達する経路を、直接の import と同じ扱いで拒否する。
- **負債の明示管理**：既存の違反は `tools/check/boundary-debt.json` に列挙し、新規の違反だけを落とす。負債ファイルに残る項目は、その Context 対と理由を持つ。
- **層名の修正**：`docs/domain/structure.md` が層を `usecase/`（単数）と書いているが実体は `usecases/`（複数）なので、実装に合わせる。

## Out of Scope

- 既存の循環と越境の解消そのもの。本 work item は現状を凍結して新規の悪化を止めるところまでを担い、個々の解消は負債ファイルの項目ごとに別の work item が扱う。
- Context の分割・統合の判断。境界の引き直しは wi-416 が扱う。
- Go の `internal/` パッケージへの移行。言語機能による強制は検査器による強制と重複するため、負債の解消が進んだ段階で改めて判断する。
- フロントエンドの機能境界の検査。

## Design

検査の実装場所は既存の `check-boundaries` を拡張する。独立したツールを足さないのは、`mise run check-boundaries` が既に `docs/development/specification-first-workflow.md` のループにゲートとして載っており、読み手が探す場所を増やさないためである。

Context Map を機械可読にする方法は 2 つある。採るのは Mermaid のフェンスをそのまま解析する案である。`docs/design/architecture/logical.md` の Context Map は既に `flowchart LR` で辺と種類を書いており、これを正本のまま読めば第二の台帳が生まれない。却下したのは、Context 間の許可された辺を YAML の一覧として別に持つ案である。正本と負債ファイルのほかに許可辺の台帳を持つと、三者が独立に古くなる。Mermaid の解析は脆いという欠点があるが、書式が崩れれば検査が落ちるので、黙って古くなることはない。

Context 名と Go パッケージ名の対応は同じ `logical.md` の「Context の責務」表（仕様上の Context 列と Go パッケージ列）から読み取る。この表も既に存在するため、新しい対応表を作らない。

主要なデータ型は `ContextDefinition`（Context 名、Go パッケージ接頭辞）、`ContextRelation`（Supplier、Customer、関係名）、`GoSource`（導入元ファイルとソース）、`BoundaryViolation`（安定 ID、種類、Context 対、経路）、`BoundaryDebtEntry`（Context 対、具体的な違反 ID 群、理由）とする。決定可能な中核操作は `parseLogicalArchitecture(markdown): ArchitectureModel`、`findBoundaryViolations({ modulePath, architecture, sources }): BoundaryViolation[]`、`reconcileBoundaryDebt(violations, debt): BoundaryDebtFinding[]`、`addedBoundaryDebtViolations(before, current): string[]` とし、ファイルの列挙と読み取り、Git の基準 revision の読み取りは検査側の作用境界に残す。

負債ファイルは Context 対または作用の種類ごとに項目をまとめ、その中へ具体的な違反 ID を列挙する。項目には「どの Context がどの Context の何に到達しているか」と理由を持たせ、理由の無い項目は拒否する。負債に載っていない違反が現れたら落ち、負債に載っているが既に解消された具体 ID が残っていても落ちる。後者を入れるのは、負債ファイルが解消の進捗と乖離しないようにするためである。Git ratchet は項目単位ではなく具体 ID 単位で基準 revision にない追記を拒否するため、既存の Context 対に別の経路を足すこともできない。負債ファイルは縮む方向にしか動かない、という性質を検査された性質にする。

初期値は 335 項目、855 個の具体的な違反 ID である。内訳は循環 118、domain の作用 20、非公開 package import 111、`shared` 経由 363、Context Map に無い辺 243 である。wi-417 で確定したとおり、Events は Go import の許可辺には含めていない。

`domain` の作用禁止は import 文の検査で行う。`time` パッケージ全体を禁じると `time.Duration` と `time.Time` が使えなくなるため、禁じるのは識別子 `time.Now` の呼び出しであり、パッケージの import ではない。

wi-417 は完了し、ドメインイベントは Context 間 import を生まない配信点とワイヤ表現で実現すると確定した。Context Map から Events 辺も削除済みであるため、イベントは Go import の許可辺として扱わない。

この作業はツールと現在設計の文書化であり、製品要件の Acceptance/E2E 境界は該当しない。Acceptance RED は `tools/check/src/check-boundaries.acceptance.test.ts` が仮のリポジトリに公開言語外の import を追加しても `checkBoundaries` が成功してしまう失敗で、Unit RED は `tools/check/src/boundary-fitness.test.ts` の解析関数が存在しない失敗で観測する。局所レシピは `mise run test-tools-file -- check/src/<test-file>`、層のゲートは `mise run check-boundaries`、埋め込みツール全体は `mise run test-tools`、`mise run typecheck-tools`、`mise run lint-tools` とする。

## Plan

1. 現在の Context 間 import グラフを取得し、Context Map の宣言との差分を一覧にする。この一覧が負債ファイルの初期値になる。Context の Go パッケージ対応と Supplier/Customer の関係は `logical.md` から導出する。
2. 公開言語の定義と層名の修正を `docs/domain/structure.md` へ入れる。
3. Context Map の解析、禁止依存、循環、`domain` の作用禁止、迂回検出を順に実装する。各規則は違反する fixture を先に用意し、規則を入れる前にその fixture が通ってしまうことを観測する。
4. 負債ファイルを初期値で投入し、`mise run check-boundaries` が現状の作業ツリーで通ることを確認する。
5. 意図的な新規違反（他 Context の `usecases` を import する、`domain` で `time.Now()` を呼ぶ、Context Map に無い辺を作る）を入れて、それぞれが落ちることを確認する。

## Tasks

- [x] T001 [Baseline] Context 間 import グラフと Context Map の宣言の差分を取得し、負債ファイルの初期値と、wi-417 で確定した「Events は import 辺ではない」という前提を記録する。
- [x] T002 [Spec] `docs/domain/structure.md` に公開言語の定義を書き、層名を `usecases` へ修正する。
- [x] T003 [Acceptance] 違反する fixture が現在の `check-boundaries` を通過することを観測する。
- [x] T004 [Tooling] Context Map の解析と Context 名からパッケージ名への対応の導出を実装する。
- [x] T005 [Tooling] 他 Context の非公開パッケージへの import、循環、Context Map に無い辺、`domain` の作用、`shared` 経由の迂回を検査する規則を実装する。
- [x] T006 [Tooling] 負債ファイルの形式、理由の必須化、解消済み項目の検出を実装する。
- [x] T007 [Verify] 現状で通り、5 種類の意図的な違反それぞれで落ちることを確認する。

## Verification

- `mise run check-boundaries` が現状の作業ツリーで通る。
- 他 Context の `usecases` を import する変更、`domain` で `time.Now()` を呼ぶ変更、Context Map に無い辺を作る変更、`shared` を経由して禁止方向へ到達する変更、負債ファイルの項目から理由を削る変更が、それぞれ落ちる。
- `mise run verify`

## Risk Notes

負債ファイルの初期値が大きい場合、検査が「現状追認の一覧」に見えて解消の圧力を生まないおそれがある。項目ごとに理由を必須にし、理由が「既存のため」となる項目を許さないことで、一覧を読めば何が設計上の負債で何が意図した例外かが分かる状態を保つ。

Context Map の Mermaid 解析は書式の変更に弱い。矢印とラベルの形が変わる、または責務表が消えると検査が落ちるので、`docs/design/architecture/logical.md` を編集する人が原因を特定できる失敗メッセージにする。

循環の拒否を入れると、既存の双方向依存を持つ Context 対に対して新しい具体経路を足せなくなる。既存の循環辺は具体 ID として ledger に固定し、同じ Context 対であっても追加経路は ratchet が拒否する。

## Completion

- **Completed At**: 2026-09-28
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返した。規範シナリオ、規範 ID、TypeSpec シンボルは変更していない。`docs/domain/structure.md` に Context の公開言語、Supplier/Customer と Go import の向き、循環禁止、domain の作用禁止、shared 経由の迂回禁止を現在設計として記述し、`check-boundaries` が `docs/design/architecture/logical.md` から Context と関係を導出して検査するようにした。既存の 855 具体違反は理由つきの 335 項目へ固定し、基準 revision から具体 ID が増える変更を CI の ratchet が拒否する。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/check-boundaries.acceptance.test.ts`
  - **Requirement**: N/A: 製品の観測可能な振る舞いを変えない、リポジトリのアーキテクチャ検査であるため。
  - **Observed Failure**: 公開言語外の `Supplier/usecases` を `Customer/usecases` から import する仮リポジトリに対し、実装前の `checkBoundaries` は成功したため `Expected: false, Received: true` で失敗した。実装後は同じ fixture が `private-import:Customer->Supplier` を報告する。
  - **Detection Reason**: 実際のリポジトリ入口である `checkBoundaries` を通し、他 Context の内部 package を検出しない既存実装と、負債に無い越境を拒否する実装を区別する。最終形では非公開 import、未宣言辺、domain の作用、shared 経由、理由欠落の五つを独立した fixture で拒否する。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/boundary-fitness.test.ts`
  - **Requirement**: N/A: 同上。対象は検査器の決定可能な中核ロジックである。
  - **Observed Failure**: テストを先に追加した時点では `./boundary-fitness.ts` が存在せず、module not found で失敗した。実装後は Context Map 解析と壊れた正本の拒否、公開範囲、未宣言辺、循環、domain の作用、shared 経由、負債照合を扱う 12 件が成功した。
  - **Detection Reason**: ファイル探索や Git 読み取りから分離した純粋関数へ入力を与え、各禁止規則と stale debt の判定を個別に固定するため、入口の出力だけを文字列比較するより誤りの位置を特定できる。
- **Change-Resistance Results**:
  `publicPackage` を一時的に常に `true` を返すよう変更し、別 Context の `usecases` を公開扱いする代表的な誤実装を注入した。`rejects a cross-context import of a private package` は `Expected: false, Received: true` で失敗し、復元後は成功した。未宣言辺、domain の `time.Now`、shared 経由、理由欠落はそれぞれ別の受け入れ fixture が拒否し、Git ratchet は既存グループ内へ具体 ID を足す単体テストでも拒否を確認した。手動変異は公開判定の一例であり、TypeScript 全体への体系的な mutation score は測っていない。
- **Verification Results**:
  - `mise run test-tools-file -- check/src/check-boundaries.acceptance.test.ts check/src/boundary-fitness.test.ts check/src/boundary-debt-ratchet.test.ts` - passed（21 件）
  - `mise run test-tools` - passed
  - `mise run typecheck-tools` - passed
  - `mise run lint-tools` - passed
  - `mise run check-boundaries` - passed
  - `mise run check-boundary-debt-ratchet main` - passed（基準側に ledger が無いため初回導入として成功）
  - `mise run check main` - passed
  - `mise run check-spec` - passed
  - `mise run verify` - passed
