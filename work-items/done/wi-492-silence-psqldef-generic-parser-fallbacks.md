---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p3
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: スキーマの意味も外部から観測できる振る舞いも変えない。DDL の書き方だけを psqldef の汎用パーサーが読める形へ揃える。
  references: []
spec_impact:
  kind: none
  reason: >-
    制約の論理も列の意味も変えない。書き換えた 6 つの CHECK は元の双条件と論理的に同値であり、
    引用符を付けた 1 列は同じ小文字識別子へ解決する。
initial_context:
  specification: []
  typespec: []
  source:
    - infra/schema/postgres.sql
    - infra/schema/README.md
  tests:
    - infra/schema/check-convergence.sh
  stop_before_reading:
    - backend
    - frontend
---

# psqldef の汎用パーサーが postgres.sql で代替パーサーへ落ちる箇所を減らす

## Motivation

`mise run check-schema` は収束を検証して合格するが、その途中で `psqldef` が
`Generic parser failed on full SQL, using pgquery fallback` という警告を毎回 6 回出す。収束の判定自体は代替パーサーが
処理するので結果は正しい。問題は読み手のほうにある。ゲートの出力が毎回 6 個の長大な警告で埋まると、そこに本物の
診断が混ざったとき誰も見分けられない。ゲートは、失敗していないときは静かでなければ、失敗したときに読まれない。

警告は 1 件ではなく 3 種類あった。汎用パーサーはファイル全体の解析で最初の構文エラーを報告して止まるため、1 つ直すと
次が現れる。したがって「2 表の問題」という当初の見立ては誤りで、実際には予約語 1 列、双条件 6 つ、そして
`UNLOGGED` が 9 表あった。

## Scope

- `authorization_detail_types.schema` を引用符付き識別子にする。汎用パーサーが裸の `schema` を予約語として拒否する。
- `(boolExpr) = boolExpr` の形をした 6 つの `CHECK` を、論理的に同値な `AND` / `OR` の形へ書き換える。
- 書き換えた制約が元と同値であること、および実際に拒否と受理を行うことを PostgreSQL 上で確認する。

## Out of Scope

- `CREATE UNLOGGED TABLE` の 9 表。`UNLOGGED` は短命なプロトコル状態に対する意図した耐久性の選択であり、
  パーサーの警告を消すために外すと実行時の性質を落とすことになる。ツールの静けさと引き換えにしてよい性質ではない。
  警告はこの 1 種類だけ残る。
- `psqldef` の更新や差し替え。汎用パーサーの対応範囲は上流の問題であり、この作業項目はこちら側の書き方だけを扱う。
- `postgres.sql` に残る SQL コメント。`infra/schema/README.md` の規則はコメントを禁じているが、これは別の逸脱であり、
  この変更とは独立して扱う。

## Design

双条件 `(P) = (Q)` は、汎用パーサーが括弧で囲まれた真偽値どうしの `=` を扱えないために落ちる。同値な書き換えは
`(P AND Q) OR (NOT P AND NOT Q)` である。関係する列はすべて `NOT NULL` か、`IS NULL` / `IS NOT NULL` を通るため
NULL が式の値になることはなく、三値論理による差は生じない。書き換え後の形はむしろ読みやすい。「両方あるか、両方ないか」
という意図がそのまま字面に出る。

同値性は書き換えの正しさそのものなので、目視ではなく PostgreSQL 上で網羅的に確かめる。関係する列の定義域は小さい
(NULL の 2 値、`health` の 3 値、真偽値) ので、全組み合わせに対して旧式と新式を `IS DISTINCT FROM` で突き合わせれば
証明になる。

## Plan

1. 3 種類の警告を 1 つずつ潰し、そのつど `mise run check-schema` を走らせて次に現れる種類を確認する。
2. 書き換えた式の同値性を PostgreSQL 上で網羅的に検証する。
3. 制約が実際に拒否と受理を行うことを、実表への挿入で確認する。

## Tasks

- [x] T001 [Schema] `authorization_detail_types.schema` を引用符付き識別子にする。
- [x] T002 [Schema] 双条件の `CHECK` 6 つを同値な `AND` / `OR` の形へ書き換える。
- [x] T003 [Verify] 同値性を網羅的に検証し、制約の拒否と受理を実表で確認し、`mise run check-schema` と `mise run verify` を通す。

## Verification

- `mise run check-schema`
- `mise run verify`
- 書き換えた 3 種類の式が、関係する列の全組み合わせで元の式と同じ真偽値を返す。
- 片側だけ埋めた `footer_link_1` の行が拒否され、両側を埋めた行が受理される。

## Risk Notes

リスクは low。制約を弱める書き換えをすると、これまで拒否していた行が通るようになる。これが唯一の危険であり、
同値性の網羅検証がそこを直接見ている。`UNLOGGED` には手を触れないので、耐久性の性質は変わらない。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` は規範の差分なしを報告する。`postgres.sql` の DDL の書き方だけが変わり、列の意味、制約の論理、
  外部から観測できる振る舞いはどれも変わっていない。`psqldef` の汎用パーサーが代替へ落ちる箇所は 3 種類から 1 種類
  (`UNLOGGED` の 9 表) へ減り、`check-schema` の出力の警告は 6 件から 4 件になった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-schema`
  - **Requirement**: N/A: 規範上の振る舞いを変えないスキーマ記法の変更であり、対応する REQ が無い。
  - **Observed Failure**: 収束検査自体は変更前から合格していたので、失敗していたのは検査ではなく出力である。
    変更前の実行は `Generic parser failed on full SQL, using pgquery fallback` を 6 回出し、
    `near 'schema'` と `(health = 'quarantined'::text) = (quarantined_at IS NOT NULL)` の 2 種類の構文エラーを報告していた。
    これが実際に観測できた不良である。
  - **Detection Reason**: 警告の本数と種類は `check-schema` の出力から機械的に数えられる。1 種類直すたびに次の種類が
    現れたことが、この検査が書き方の違いへ実際に反応していることを示している。
- **Unit RED Evidence**:
  - **Test**: PostgreSQL 上での式の同値検証。
  - **Requirement**: N/A: 規範上の振る舞いを変えないスキーマ記法の変更であり、対応する REQ が無い。
  - **Observed Failure**: 該当なし。書き換えの前に失敗する検査は存在しなかったので、代わりに書き換えの後で
    同値性を網羅的に確かめた。3 種類の式すべてで不一致は 0 件だった
    (NULL の対 4 通り、`profile_id` × `is_default` 4 通り、`health` × `quarantined_at` 6 通り)。
  - **Detection Reason**: `IS DISTINCT FROM` による突き合わせは、両辺が NULL の場合も一致として扱うので、
    三値論理のずれを見落とさない。定義域が小さいため全数検証であり、標本ではない。
- **Change-Resistance Results**:
  制約を弱める向きの誤りが起きたときに検出できるかを、実表で直接確かめた。片側だけ埋めた `footer_link_1_label` の
  行は `violates check constraint "tenant_brandings_footer_link_1_complete"` で拒否され、両側を埋めた行は受理された。
  弱い制約に書き換えていたら前者が通っていた。`pg_get_constraintdef` で適用後の 4 制約を読み戻し、意図した
  `AND` / `OR` の形で保存されていることも確認した。
- **Verification Results**:
  - `mise run check-schema` - passed
  - `mise run verify` - passed
