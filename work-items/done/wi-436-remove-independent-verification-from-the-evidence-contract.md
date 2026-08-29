---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-08-29
depends_on: []
priority: p2
change_kind: tooling
evidence_policy: risk-based-v2
initial_context:
  source:
    - docs/development/specification-first-workflow.md
    - WORK_ITEM_FORMAT.md
    - tools/check/schemas/work-item.schema.json
    - tools/check/src/main.ts
    - .claude/skills/implement-work-item/SKILL.md
  tests:
    - tools/check/src/lib.test.ts
  stop_before_reading:
    - backend
    - frontend
    - spec
spec_impact:
  kind: none
  reason: 作業記録の証拠契約と、その機械検査の変更である。製品の振る舞い・インタフェース・保証を変えない。
---

# 独立検証を証拠契約から外す

## Motivation

`docs/development/specification-first-workflow.md` の証拠契約は、`risk: medium` 以上と `reversibility: irreversible` の作業項目に「実装者でない人間または fresh-context エージェントによる独立検証」を要求し、`tools/check/schemas/work-item.schema.json` がその記載を完了記録に強制している。

この要求は二重である。作業項目を立て、実装し、Pull Request を人間がレビューするという通常の流れの中で、実装していない読み手は既に差分を読んでいる。証拠契約が別途「独立検証を行った」と書かせても、その流れの外に新しい読み手が現れるわけではなく、**同じ 1 回のレビューを 2 か所に記録させているだけ**である。

記録が実質を伴わない要求は、埋めるための文言を生む。証拠契約が守ろうとしているのは「実装者の思い込みが検査を素通りしないこと」であり、それを担保しているのは Acceptance RED / Unit RED と change-resistance の各要求である。独立検証の記載欄はそこに何も足していない。

エージェントによる独立検証それ自体に価値がある場面はありうるが、それは「いつ起こすか」を別に設計する話であり、完了の必須条件として固定する形とは別である。本項目は必須条件を外すところまでを扱う。

## Scope

- `docs/development/specification-first-workflow.md`:
  - 証拠契約の表 (第 4 節) から独立検証の要求を外す。`medium` 以上には change-resistance と `spec-diff` の要求が残る。
  - 認証・認可・テナント境界・暗号・プロトコル互換・永続データ移行を理由に独立検証へ格上げする段を外す。リスクの格上げ自体は残す。
  - `reversibility: irreversible` が独立検証を追加要求する段を外す。
  - 「Independent verification is performed by ...」の段と、XP のペアレビューを本ワークフローに位置づける段を外すか、証拠契約に依存しない形へ書き直す。
  - 検証のはしご (第 5 節) の第 4 段から独立検証を外す。
- `WORK_ITEM_FORMAT.md`:
  - Completion のひな型から `Independent Verification` の項を外す。
  - 「The independent verifier must not be the person or agent that implemented the change」の記述を外す。
- `tools/check/schemas/work-item.schema.json`:
  - `medium` 以上の完了に対する `required` から `independent_verification` を外し、`change_resistance` だけを残す。
  - `reversibility: irreversible` の条件節は `independent_verification` のみを要求しているので、節ごと外す。
  - `independent_verification` のプロパティ定義そのものは残す。完了済みの記録が既にこの項を持っており、外すと過去の記録が検証を通らなくなる。
- `tools/check/src/lib.test.ts`: 上記 2 つの要求を検査していた試験を、要求が無くなったことを検査する形に置き換える。
- `.claude/skills/implement-work-item`: 手順 9 から独立検証の実施を外す。

## Out of Scope

- `independent_verification` プロパティと `tools/check/src/main.ts` の見出し解析の削除。完了済みの記録が持つ項を読めなくすると、後続の見出しまで前の項へ流れ込んで解析が壊れる。任意項目として残す。
- 完了済みの作業項目から `Independent Verification` の記載を消す作業。当時行われた検証の記録であり、事実である。
- change-resistance の要求。実装者の思い込みを検査そのもので突く仕組みであり、独立検証とは別の役割を持つ。
- Acceptance RED / Unit RED の要求。
- エージェントによる独立検証をいつ起こすかの設計。必要なら別項目にする。
- `mise run verify` や CI のジョブ構成。

## Design

削除する対象は 3 層にまたがる。方法論の文書 (`specification-first-workflow.md`)、記録の形式 (`WORK_ITEM_FORMAT.md`)、機械検査 (`work-item.schema.json` と `lib.test.ts`) である。3 つが食い違うと、文書は要求しないのに検査が落ちる、あるいはその逆が起きるので、同じ変更で揃える。

`independent_verification` は「必須でなくなる」だけで「書けなくなる」わけではない。スキーマの `properties` と `main.ts` の見出し解析を残す理由は 2 つある。完了済みの記録が既にこの項を持つこと、そして見出し解析が未知の見出しを前の項の続きとして扱うため、解析側から外すと過去の記録の `change_resistance` などに独立検証の本文が混入することである。

`reversibility` の扱いに注意する。この軸は現在、独立検証を追加要求することだけを効果として持つ。要求を外すと軸そのものが何の追加要求も持たなくなるので、`WORK_ITEM_FORMAT.md` の「`reversibility` only adds requirements」の段が宙に浮く。軸の宣言としての価値 (取り消せない決定であることを記録に残す) は残るので、フィールドは残し、証拠契約への効果を持たない軸であることが読めるように書き直す。

## Plan

- **文書と検査を同じ変更で揃える**。片方だけ動かすと、書いていないものを検査が要求する状態が生まれる。
- **スキーマの条件節は 2 か所ある**。`medium` 以上の節は `change_resistance` を残して `independent_verification` だけを外し、`irreversible` の節は要求が空になるので節ごと外す。
- **`lib.test.ts` の 2 件は削除ではなく反転させる**。「要求する」試験を「要求しない」試験に置き換える。要求を外したことを検査が押さえていないと、次に誰かが条件節を戻しても何も落ちない。
- **`check-agent-guidance` を確かめる**。リポジトリのエージェント向け案内とワークフローの整合を検査しており、手順 9 の変更がここに掛かる可能性がある。

## Tasks

- [x] T001 [Unit] `lib.test.ts` の 2 件を「要求しない」形に反転させ、現行スキーマに対して失敗することを確認する。
- [x] T002 [Schema] `work-item.schema.json` の 2 つの条件節を修正し、T001 を緑にする。
- [x] T003 [Docs] `docs/development/specification-first-workflow.md` の第 4 節と第 5 節から独立検証を外す。
- [x] T004 [Docs] `WORK_ITEM_FORMAT.md` の Completion ひな型と独立検証者の記述を外し、`reversibility` の段を書き直す。
- [x] T005 [Skill] `.claude/skills/implement-work-item` の手順 9 を修正する。
- [x] T006 [Verify] 下記 Verification を緑にする。

## Verification

- `mise run check-work-items` — 完了済みの記録がすべて通る。`Independent Verification` を持つものも、持たないものも通る。
- `mise run check-ids` / `mise run check-links` / `mise run check-agent-guidance`
- `mise run verify`
- 単体: `medium` 以上で `change_resistance` だけを持つ完了記録が通る。`change_resistance` を欠くと落ちる。
- 単体: `reversibility: irreversible` かつ `risk: low` の完了記録が、独立検証の記載なしで通る。

## Risk Notes

過去の記録を壊さないことが唯一の実質的なリスクである。`independent_verification` を任意項目として残し、`main.ts` の見出し解析に手を入れないことで避ける。`mise run check-work-items` が 434 件すべてを検証するので、壊れれば即座に分かる。

要求を外すことで、実装者以外が差分を読む機会そのものが減るわけではない。読む機会は Pull Request のレビューが持つ。本項目が外すのは、それを作業項目にもう一度書かせる要求だけである。

## Completion

- **Completed At**: 2026-08-29
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。製品の振る舞いも公開契約も動いていない。差分は証拠契約とその機械検査に閉じる。`risk: medium` 以上の完了記録に要求されるのは change-resistance と `spec-diff` を読んだ要約だけになり、`reversibility: irreversible` が追加要求を持つ条件節は消えた。`independent_verification` は任意項目として残り、完了済みの 150 件あまりが持つ記載はそのまま検証を通る。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`
  - **Requirement**: N/A: 作業記録の証拠契約であり、製品の規範シナリオを持たない。
  - **Observed Failure**: `risk: medium` かつ `reversibility: irreversible` で独立検証の記載を持たない完了記録を置くと、変更前のスキーマは `schema: /completion must have required property 'independent_verification'` で落ちた。変更後は同じ記録を含む 436 件がすべて通る。
  - **Detection Reason**: 実際の `check-work-items` 経路で観測しており、スキーマ単体ではなく作業記録の門そのものが判定を変えたことを示す。同時に既存の 435 件が通り続けることも同じ実行で確かめているので、要求を外したことと過去の記録を壊したことを区別できる。
- **Unit RED Evidence**:
  - **Test**: `bun test check/src/lib.test.ts`
  - **Requirement**: N/A: 同上。
  - **Observed Failure**: 反転させた試験を先に書いた段階で 47 pass / 4 fail。`asks an irreversible item for no evidence beyond its risk row` と `requires stronger completion evidence for medium-risk work` を含む。
  - **Detection Reason**: 「要求しない」を試験にしているので、条件節を戻すと落ちる。medium の試験は独立検証だけを足しても満たされないことを別の表明で押さえており、`change_resistance` が要求の実体であることを、単に緩めた場合と区別できる。
- **Change-Resistance Results**:
  `risk: low` のため必須ではない。実際には Acceptance RED を変更前後のスキーマで往復させ、削除した 2 つの条件節のうち `medium` 側と `irreversible` 側の両方が判定に効いていたことを確認している。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-work-items` (435 件) / `mise run check-ids` / `mise run check-links` / `mise run check-agent-guidance` - passed
