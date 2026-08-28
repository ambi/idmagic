---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-27
priority: p1
change_kind: docs
spec_impact: { kind: none, reason: "docs/structure.md へ公開言語の定義を書き層名を実装に合わせるが、規範シナリオ、規範 ID、TypeSpec シンボルを追加も変更もしない。" }
---

# Context 境界と作用の禁止を機械検査する適応度関数を足す

## Motivation

`docs/structure.md` は「Context 間は公開された言語とポートで接続する」と宣言し、`docs/README.md` の Context Map は関係の向きと種類（OHS/PL、C/S、ACL、Events）まで型付けしている。しかし `tools/check/src/check-boundaries.ts` が検査するのは層の方向（`domain` が `usecases`/`handlers_*`/`db_*` を import しないこと、`usecases` が外向きに import しないこと）と起動時設定の読み取り点だけであり、Context 間の依存は 1 行も見ていない。

実際の import グラフは宣言と一致していない。`backend/oauth2` は `authentication/mfa/usecases`、`authentication/session/usecases`、`authentication/totp/usecases`、`authentication/webauthn/handlers_http` を直接 import している。Context Map が `Authentication --OHS/PL--> OAuth2` の一方向だけを宣言しているのに対し、実装は `oauth2 → authentication` と `authentication → oauth2` の双方向であり、`idmanagement ↔ authentication`、`tenancy ↔ idmanagement` にも循環がある。公開言語であるはずの `domain` と `ports` を越えて、他 Context の `usecases` と `handlers_http` に到達している。

Modular Monolith の全体重は「モジュールの内部が外から見えないこと」に乗っている。それが慣習だけで支えられている限り、Context Map は現在の設計の記述ではなく努力目標にすぎず、`docs/development/specification-first-workflow.md` が求める「現在の設計は正本文書と work item だけから理解できる」状態を満たさない。同じ理由で、`domain` が純粋であるという性質もどこにも書かれておらず検査もされていないため、`domain` の中で `time.Now()` を呼ぶことは今のところ自由である。

`checkRefusalCoverage` と `security-refusal-debt.json`（`tools/check/src/security-controls.ts:307`）は、既存の違反を負債として明示管理しながら新規の違反を止めるという型を既に持っている。同じ型を境界に適用する。

## Scope

- **公開言語の定義**：`docs/structure.md` に、Context の外へ公開されるのは `domain` と `ports` だけであること、他 Context の `usecases`、`handlers_*`、`db_*` へ到達してはならないことを書く。
- **Context 間の禁止依存**：`check-boundaries` に、他 Context の非公開パッケージへの import を拒否する規則を足す。
- **循環の非存在**：Context 単位の依存グラフを組み立て、循環を拒否する。
- **Context Map との一致**：`docs/README.md` の Context Map の矢印を読み取り、実 import グラフが宣言に無い辺を持つ場合に拒否する。Context Map を機械可読な正本として扱えるようにする。
- **`domain` の作用禁止**：`domain` パッケージが `time.Now`、`math/rand`、`crypto/rand`、`os`、`net`、`database/sql` を参照することを拒否する。作用は引数として入るという `docs/development/specification-first-workflow.md` の規律を、変更時の手順ではなくシステムの現在の性質として固定する。
- **迂回の検出**：`backend/shared/` を経由して禁止された方向へ到達する経路を、直接の import と同じ扱いで拒否する。
- **負債の明示管理**：既存の違反は `tools/check/boundary-debt.json` に列挙し、新規の違反だけを落とす。負債ファイルに残る項目は、その Context 対と理由を持つ。
- **層名の修正**：`docs/structure.md` が層を `usecase/`（単数）と書いているが実体は `usecases/`（複数）なので、実装に合わせる。

## Out of Scope

- 既存の循環と越境の解消そのもの。本 work item は現状を凍結して新規の悪化を止めるところまでを担い、個々の解消は負債ファイルの項目ごとに別の work item が扱う。
- Context の分割・統合の判断。境界の引き直しは wi-416 が扱う。
- Go の `internal/` パッケージへの移行。言語機能による強制は検査器による強制と重複するため、負債の解消が進んだ段階で改めて判断する。
- フロントエンドの機能境界の検査。

## Design

検査の実装場所は既存の `check-boundaries` を拡張する。独立したツールを足さないのは、`mise run check-boundaries` が既に `docs/development/specification-first-workflow.md` のループにゲートとして載っており、読み手が探す場所を増やさないためである。

Context Map を機械可読にする方法は 2 つある。採るのは Mermaid のフェンスをそのまま解析する案である。`docs/README.md` の Context Map は既に `flowchart LR` で辺と種類を書いており、これを正本のまま読めば第二の台帳が生まれない。却下したのは、Context 間の許可された辺を YAML の一覧として別に持つ案である。`AGENTS.md` が「アーキテクチャ台帳を足さない。境界検査は経路から構造を推論し、禁止された依存だけを拒否する」と定めており、辺の一覧はまさにその台帳になる。Mermaid の解析は脆いという欠点があるが、書式が崩れれば検査が落ちるので、黙って古くなることはない。

Context 名と Go パッケージ名の対応は `docs/README.md` の索引表（Specification context 列と Go package 列）から読み取る。この表も既に存在するため、新しい対応表を作らない。

負債ファイルの形式は `security-refusal-debt.json` に倣う。項目には「どの Context がどの Context の何に到達しているか」と理由を持たせ、理由の無い項目は拒否する。負債に載っていない違反が現れたら落ち、負債に載っているが既に解消された項目が残っていても落ちる。後者を入れるのは、負債ファイルが解消の進捗と乖離しないようにするためである。

`domain` の作用禁止は import 文の検査で行う。`time` パッケージ全体を禁じると `time.Duration` と `time.Time` が使えなくなるため、禁じるのは識別子 `time.Now` の呼び出しであり、パッケージの import ではない。

未解決の論点として、Context Map の `Events` 関係が現在は直接の Go の import として実現されているのか、それとも共有のイベント基盤を介しているのかを確認する必要がある。前者であれば `Events` の辺は import の許可としては扱えず、wi-417 の結論を待つ。実装着手前に確定させる。

## Plan

1. 現在の Context 間 import グラフを取得し、Context Map の宣言との差分を一覧にする。この一覧が負債ファイルの初期値になる。
2. 公開言語の定義と層名の修正を `docs/structure.md` へ入れる。
3. Context Map の解析、禁止依存、循環、`domain` の作用禁止、迂回検出を順に実装する。各規則は違反する fixture を先に用意し、規則を入れる前にその fixture が通ってしまうことを観測する。
4. 負債ファイルを初期値で投入し、`mise run check-boundaries` が現状の作業ツリーで通ることを確認する。
5. 意図的な新規違反（他 Context の `usecases` を import する、`domain` で `time.Now()` を呼ぶ、Context Map に無い辺を作る）を入れて、それぞれが落ちることを確認する。

## Tasks

- [ ] T001 [Baseline] Context 間 import グラフと Context Map の宣言の差分を取得し、負債ファイルの初期値と `Events` 関係の実現方法を記録する。
- [ ] T002 [Spec] `docs/structure.md` に公開言語の定義を書き、層名を `usecases` へ修正する。
- [ ] T003 [Acceptance] 違反する fixture が現在の `check-boundaries` を通過することを観測する。
- [ ] T004 [Tooling] Context Map の解析と Context 名からパッケージ名への対応の導出を実装する。
- [ ] T005 [Tooling] 他 Context の非公開パッケージへの import、循環、Context Map に無い辺、`domain` の作用、`shared` 経由の迂回を検査する規則を実装する。
- [ ] T006 [Tooling] 負債ファイルの形式、理由の必須化、解消済み項目の検出を実装する。
- [ ] T007 [Verify] 現状で通り、5 種類の意図的な違反それぞれで落ちることを確認する。

## Verification

- `mise run check-boundaries` が現状の作業ツリーで通る。
- 他 Context の `usecases` を import する変更、`domain` で `time.Now()` を呼ぶ変更、Context Map に無い辺を作る変更、`shared` を経由して禁止方向へ到達する変更、負債ファイルの項目から理由を削る変更が、それぞれ落ちる。
- `mise run verify`

## Risk Notes

負債ファイルの初期値が大きい場合、検査が「現状追認の一覧」に見えて解消の圧力を生まないおそれがある。項目ごとに理由を必須にし、理由が「既存のため」となる項目を許さないことで、一覧を読めば何が設計上の負債で何が意図した例外かが分かる状態を保つ。

Context Map の Mermaid 解析は書式の変更に弱い。矢印とラベルの形が変われば検査が落ちるので、`docs/README.md` を編集する人が検査の存在に気づける失敗メッセージにする。

循環の拒否を入れると、既存の双方向依存を持つ Context 対に対して新しい辺を足せなくなる。これは意図した効果だが、進行中の作業を止める可能性があるため、負債に載っている対については循環の判定を保留する。
