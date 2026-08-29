---
status: completed
authors: [tn]
risk: low
reversibility: reversible
evidence_policy: risk-based-v2
created_at: 2026-08-29
priority: p3
depends_on: []
change_kind: tooling
initial_context:
  specification:
    - docs/development/specification-first-workflow.md
    - WORK_ITEM_FORMAT.md
  typespec: []
  source:
    - mise.toml
    - backend/jobs/domain/job.go
  tests:
    - tools/check/src/mise-config.test.ts
  stop_before_reading:
    - frontend
    - spec
    - docs/contexts
spec_impact: { kind: none, reason: "開発時の道具を 1 つ外し、証拠契約の書き方を道具名から義務の記述へ戻す変更であり、製品の振る舞いも公開契約も変えない。" }
---

# 変更耐性の証拠を、道具の名前ではなく義務の記述で定める

## Motivation

[[wi-411-evaluate-go-mutation-testing-pilot]] は Gremlins `0.6.0` を `mise.toml` へ固定し、`mise run test-go-mutation-package` を追加した。**評価そのものは正しかったが、評価の成果物として残すべきだったのは道具ではなく知見だった。**

外す理由は「使っていないから」ではない。固定から今日まで 6 日で、その間に完了した work item 27 件のうち `high` は 1 件（[[wi-424-threat-modeling-as-specification]]、文書の変更であり Go の純粋論理を触らない）だけである。**適用条件に当たる変更が一度も発生していないだけ**で、6 日という観測窓から使用頻度を結論することはできない。バックログには `wi-53`、`wi-114`、`wi-228`、`wi-261` のように判定論理が純粋関数に閉じる `high` の項目が残っている。

決め手は、**入れ直す費用が抱え続ける費用より安い**ことである。導入時に高かったのは「効くのか、いくらかかるのか、何を見つけるのか」の評価であり、その答えは wi-411 に永続的に残る。入れ直しは `[tools]` 1 行とタスク 1 個で済み、評価をやり直す必要は無い。一方で抱え続ける側が払うのは、**後方互換性を保証しない `0.x` 依存のバージョン追跡**、`go:` バックエンド由来の弱い固定（[[wi-291-dependency-vulnerability-management-and-disclosure-policy]] が受け入れ水準として言及している）、`mise tasks` に常時並ぶ 1 行、そしてそれを固定する検査であり、これらは使わない週にも毎週かかる。

もう 1 つ、道具とは独立に直すべきものがある。**証拠契約が特定のコマンド名を本文に埋めている。** `docs/development/specification-first-workflow.md:61` と `WORK_ITEM_FORMAT.md:138` は `mise run test-go-mutation-package -- <package> <git-ref>` を名指ししており、道具を出し入れするたびに規範文書が揺れる。`WORK_ITEM_FORMAT.md` は将来このリポジトリの外へ持ち出す前提の方法論文書であり、そこにリポジトリ固有の `mise` タスク名が入っているのは、その前提と食い違う。

## Scope

- `mise.toml` から Gremlins の宣言と `test-go-mutation-package` タスクを外す。
- `tools/check/src/mise-config.test.ts` の `mise change-resistance boundary` を削除する。
- `docs/development/specification-first-workflow.md:61` の `high`/`critical` 行と `WORK_ITEM_FORMAT.md:138` を、**義務を述べて道具を名指さない**形に書き換える。
- wi-411 が残した未網羅 2 件（`backend/jobs/domain/job.go:174-175`）を同値変異と判定して閉じる。

## Out of Scope

- 変異試験という手法そのものの否定。wi-411 の評価結果は有効なままであり、`high` の Go 純粋論理を変える work item が来たら、その項目の中で入れ直す判断をしてよい。
- fuzz の道具立て（`test-go-fuzz`、`test-go-fuzz-all`）。適用条件も費用も別であり、本 work item は触らない。
- `docs/development/specification-first-workflow.md:215` の「the mutation run is what exposed the table as empty」。過去に実際に観測したことの記録であり、道具の有無とは独立に残る。
- 文書が存在しない `mise` タスク名を挙げていないかの機械検査。Design の却下した選択肢を参照。
- [[wi-421-model-checking-and-deterministic-simulation]] が扱うモデル検査と決定的シミュレーション。並行性が対象で、変異試験の代替ではない。

## Design

証拠契約の 2 か所は、コマンド名を消すのではなく**義務の記述へ書き換える**。現在の `high`/`critical` 行は「`mise run test-go-mutation-package` または明示的な故障注入を使う」と書いているが、要求している中身は「変更した純粋論理を体系的に変異させて、試験がそれを殺すことを見る」である。後者の書き方なら、道具が入っている週も入っていない週も同じ文で通り、実際にこれまで使われてきた故障注入も最初から含む。`WORK_ITEM_FORMAT.md` の 1 文も同じ理由で同じ形にする。

wi-411 の未網羅 2 件は同値変異として閉じる。報告された `job.go:174-175` は、当時も今も `DefaultBackoffBase = 30 * time.Second` と `DefaultBackoffCap = 30 * time.Minute` の定数宣言である。ここへの算術変異を殺す試験は、**定数の値を試験側に書き写すだけ**で、誤実装を 1 つも区別しない。試験を書かずに判定を記録するのが正しい対応であり、「未網羅を残したまま忘れる」状態を終わらせる。

### 却下した選択肢

- **`mise.toml` に理由のコメントを足して残す。** 隣の `govulncheck` は `go:` バックエンドを選ぶ理由を 3 行で説明しており（`mise.toml:14-16`）、Gremlins にはそれが無い。しかし説明を書き足すのは抱え続ける前提の対処であり、抱えるべきかという問いに答えていない。
- **`[tools]` の固定だけ外し、タスクは `go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0` で残す。** `mise install` の費用は消え、版は固定されたまま残る。採らないのは、**効いている費用がインストール時間ではない**からである。`jdx/mise-action` はキャッシュするので CI の実測差は小さい。払っているのはタスク 1 行、検査 1 ブロック、規範文書 2 か所という認知の費用で、この案はそのどれも減らさない。
- **`check-command-map` を拡張し、文書が挙げる `mise run <task>` も実在検査の対象にする。** 本 work item が直している種類の食い違いをそのまま捕まえる。採らないのは、[[wi-434-reduce-the-repository-check-set-to-what-earns-its-place]] が置いた物差しに照らして理由を言えないからである。`check-command-map` は自身の doc コメントで「pipeline を黙って壊す方向だけを見る」と範囲を宣言している。文書が死んだタスク名を挙げる誤りは pipeline を壊さず人を誤らせるもので、しかも本 work item の書き換え後は**文書がタスク名を挙げなくなる**ため、検査対象そのものが無くなる。今入れる根拠は無い。

## Plan

1. `mise.toml` の宣言とタスク、`mise-config.test.ts` の該当 describe を消す。
2. 証拠契約の 2 か所を書き換える。書き換え後の文が `high` の Go 純粋論理に対して何を要求しているかが、道具を知らない読み手に伝わることを確かめる。
3. 残った参照が無いことを確かめる。`rg -n 'gremlins|test-go-mutation-package'` の結果が `work-items/done/` の記録だけになる状態を期待値とする。
4. `mise run verify` を通す。

## Tasks

- [x] T001 [Tooling] Gremlins の宣言、`test-go-mutation-package`、`mise-config.test.ts` の `mise change-resistance boundary` を削除した。宣言を先に消して検査が赤になることを観測してから、検査を消した。
- [x] T002 [Spec] `docs/development/specification-first-workflow.md` の梯子の `high`/`critical` 行と `WORK_ITEM_FORMAT.md` の Change-Resistance Results を、道具を名指さない形へ書き換えた。
- [x] T003 [Verify] 参照の残りが `work-items/done/` の記録と本 work item だけであることを確かめ、`mise run verify` を通した。wi-411 の未網羅 2 件を同値変異と判定した。

## Verification

### 着手前に宣言する RED

減らす変更であり、`mise run verify` は前後とも緑である。緑は**消しすぎたときにも出る**ので、それ単独では証拠にならない。

- **Acceptance RED の代替**: `mise run test-tools` が赤になること。`mise-config.test.ts` は Gremlins の固定を検査で押さえているため、`mise.toml` の宣言を先に消すと `mise change-resistance boundary` が落ちる。**この赤は、道具が検査に支えられて存在していたことの直接の観測**であり、検査を消す前に一度見ておく。
- **Unit RED の代替**: 書き換えた証拠契約に対する故障注入。`docs/development/specification-first-workflow.md` の `high`/`critical` 行から変更耐性の要求を丸ごと落とすと、`medium` の行と区別が付かなくなる。**書き換えが「言い換え」なのか「弱体化」なのかは緑では判別できない**ので、書き換え前後の文が同じものを要求していることを、wi-424 が実際に出した証拠（`ROOT_DOCUMENTS` の登録行を外して `check-spec` が exit 0 で通ることの観測）が新しい文でも受理されるかで確かめる。

### 完了時に通すもの

- `mise run verify`
- `mise run test-tools`

## Risk Notes

- **入れ直す判断が誰にも起きなくなる。** 入っていない道具は誰も手に取らない。これは意図した取引で、`high` の Go 純粋論理を変える work item が Design で費用を見積もり直すのが正しい入り口である。wi-411 の測定値（`backend/jobs/domain` で 15 変異、24.56 秒、検出 13 件）がその見積もりの出発点になる。
- **書き換えが証拠契約の弱体化になる。** 道具名を消すついでに要求まで薄くなる形が最も起きやすい失敗である。`high`/`critical` の行が `medium` の行より強い要求を述べていることを、書き換え後の文だけを読んで確認する。
- **同値変異の判定が、単に試験を書かない口実になる。** `job.go:174-175` については、判定の理由（定数の値を試験へ書き写すだけで誤実装を区別しない）を Completion に残す。理由を書けない未網羅は、同じ扱いにしてはならない。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` である。意味上の差は 2 つある。1 つは、[[wi-411-evaluate-go-mutation-testing-pilot]] が固定した Gremlins `0.6.0` と `test-go-mutation-package` を外し、`mise` の固定バージョンが 1 個減ったことである。もう 1 つは、変更耐性の証拠契約が**特定のコマンド名ではなく義務**を述べる形になったことで、`docs/development/specification-first-workflow.md` の梯子と `WORK_ITEM_FORMAT.md` の両方から `mise run test-go-mutation-package -- <package> <git-ref>` という repository 固有のタスク名が消えた。要求そのものは弱めていない（下の Unit RED を参照）。
- **Acceptance RED Evidence**:
  - **Test**: `tools/check/src/mise-config.test.ts` の `mise change-resistance boundary`
  - **Requirement**: N/A: 開発時の道具を外す変更であり、対応する規範シナリオが無い。
  - **Observed Failure**: `mise.toml` から Gremlins の宣言だけを先に消した時点で `expect(received).toBe(expected) / Expected: "0.6.0" / Received: undefined` で落ちた（11 pass、1 fail）。
  - **Detection Reason**: **この赤は、道具が検査に支えられて存在していたことの直接の観測**である。宣言を消しても何も鳴らないなら、その検査は最初から効いていなかったことになる。検査を消したのはこれを見た後である。
- **Unit RED Evidence**:
  - **Test**: 書き換えた梯子の `high`/`critical` 行への故障注入
  - **Requirement**: N/A: 上と同じ理由。
  - **Observed Failure**: 変更耐性を述べる 1 文を落とすと、その行の「完了前」の列は `Apply the medium requirements.` だけになり、**`medium` 行と要求が一致した**。
  - **Detection Reason**: 書き換えが「言い換え」なのか「弱体化」なのかは、`mise run verify` の緑では判別できない。区別できるのは、その 1 文が唯一の要求の担い手であることを、落として `medium` と見分けが付かなくなることで見る形だけである。書き換え後の文は `medium` の「代表的な誤実装 1 つ」に対して「変更した純粋論理すべてを体系的に変異させる」を要求しており、旧文の要求（変異試験または故障注入と、同値変異と道具の限界の記録）を欠落なく含む。実際にこれまで使われてきた [[wi-424-threat-modeling-as-specification]] の故障注入も、旧文と同じく受理される。
- **Change-Resistance Results**:
  `risk: low` のため要求されない。上の 2 つの故障注入がその役割を兼ねる。

  あわせて wi-411 が「全体ゲート化する前の具体的な試験候補」として残した未網羅 2 件を閉じる。報告された `backend/jobs/domain/job.go:174-175` は、当時も現在も `DefaultBackoffBase = 30 * time.Second` と `DefaultBackoffCap = 30 * time.Minute` の定数宣言である（wi-411 以降、`backend/jobs/domain/` にコミットは無い）。**ここへの算術変異を殺す試験は、定数の値を試験側に書き写すだけで、誤実装を 1 つも区別しない。** よって同値変異と判定し、試験は書かない。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run test-tools` - passed (251 tests, 0 fail)
  - `mise run spec-diff` - no normative specification change against main
