---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p1
change_kind: tooling
spec_impact: { kind: none, reason: "検査器 tools/check の突き合わせ規則の欠陥であり、製品の振る舞いも規範文書も変えない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 変更は work item を検査する側にあり、製品にも規範文書にも届かない。リリースの読み手に見える差分が無い。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/documentation-impact.ts
    - tools/check/src/spec-diff.ts
    - tools/check/src/check-work-items.ts
  tests:
    - tools/check/src/documentation-impact.test.ts
  stop_before_reading:
    - backend
    - frontend
    - docs
---

# `standards.md` の行を足すと、無関係な in_progress 記録が documentation_impact で落ちる

## Motivation

`docs/contexts/<context>/standards.md` に行を 1 つ足すと、その行を足した記録ではなく、無関係な in_progress 記録が `mise run check-work-items` で落ちる。

```
work-items/wi-495-burn-down-the-standards-coverage-debt.md: documentation_impact none is weaker than inferred release_note
```

`tools/check/src/documentation-impact.ts` の `ownsSpecificationAdditions` は、仕様への追加を `affected_spec` で名指した記録だけに帰属させる仕組みである。
長期間 in_progress のままの親記録が子記録の追加を被らないよう、[[wi-509-documentation-impact-attributes-the-diff-to-every-open-record]] が入れた。
`standards.md` の行に対しては、この仕組みが働かない。

原因は突き合わせる文字列の綴りである。
`spec-diff.ts` の `standardRows` は標準行を `<path>#<ID>` を鍵として集める（`docs/contexts/sourcing/standards.md#RFC7644-DELETE-SEMANTICS`）。
一方 `referenceNames` は `affected_spec` の `requirement` をその鍵と丸ごと比較し、一致しなければ `<path>:<name>` 形式の TypeSpec 宣言として解釈しようとする。
記録が書くのは規範 id そのもの（`RFC7644-DELETE-SEMANTICS`）なので、どちらの経路でも一致しない。
`#` を含む鍵を扱う分岐が無い。

`referenceNames` の doc comment は「A scenario or standard is added under its own id, and the record writes that id verbatim, so those compare directly」と書いており、実装がその doc comment と食い違っている。
scenario は `REQ-SOURCING-002` のように id そのものが鍵なので直接比較で足りるが、standard の鍵は `<path>#<ID>` である。

`claimsSpecificationAddition` を 2 つの綴りで直接呼び、次を観測した。

```
bare id form: false   # requirement: RFC7644-DELETE-SEMANTICS
path#id form: true    # requirement: docs/.../standards.md#RFC7644-DELETE-SEMANTICS
```

後者の綴りは `work-item-references.ts` が解決できないので、記録の側で回避することはできない。

この欠陥は [[wi-534-deleted-scim-users-stay-visible-to-the-scim-client]] が `RFC7644-DELETE-SEMANTICS` を足す過程で見つかった。
同項目は `mise run check-work-items` を通せず、したがって `mise run verify` も通せないので、本項目を `depends_on` に置いて待つ。

## Scope

- `standards.md` への行追加を、その行を `affected_spec` で名指した記録に帰属させる。
- 規範 id を `requirement` に書いた記録が追加を主張できることを、テストで固定する。
- 同じ綴りの食い違いが `addedScenarios` と `addedDeclarations` に無いことを確かめる。

## Out of Scope

- `documentation_impact` の水準の導出規則そのもの。
- `spec-diff` が標準行を `<path>#<ID>` で鍵付けすること。同じ id が複数の文書に現れうるので、この鍵は正しい。
- 記録の側に `<path>#<ID>` を書かせる回避策。`work-item-references.ts` が解決できない綴りを持ち込むことになる。
- `wi-495` の `documentation_impact` の変更。同記録は追加を 1 つも所有していないので `none` が正しい。

## Plan

1. `documentation-impact.test.ts` に、規範 id を `requirement` に書いた記録が `<path>#<ID>` の追加を主張できることを期待する検査を足し、RED を確認する。
2. `referenceNames` に `#` 区切りの分岐を足す。`path` と id の両方が一致することを要求し、`<path>:<name>` の分岐と同じく、別文書の同名 id を主張できないようにする。
3. `mise run test-tools` と `mise run check-work-items` を通す。

## Tasks

- [ ] T001 [Acceptance] 規範 id での主張が通らないことを `check-work-items` の水準で観測する。
- [ ] T002 [Unit] `referenceNames` の Unit RED を確認する。
- [ ] T003 [App] `#` 区切りの分岐を足して GREEN にする。
- [ ] T004 [Verify] `mise run verify`。

## Verification

- `standards.md` に行を 1 つ足した作業木で `mise run check-work-items` が、その行を名指していない in_progress 記録を報告しない。
- 別文書の同名 id を主張できない。
- `mise run verify`

## Risk Notes

- **主張の判定を緩めすぎる。** id だけの一致で通すと、別の context の同名行を主張できてしまう。`<path>:<name>` の分岐と同様に、path と id の両方を要求する。
- **`wi-495` を落ちないようにすることを目的にする。** 目的は追加の帰属であって、特定の記録を通すことではない。テストは帰属の規則を固定する。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `tools/check/src/documentation-impact.ts` の `referenceNames` 1 箇所と、それを固定するテスト 2 件である。
  `standards.md` へ足した行を、その行を `affected_spec` で名指した記録に帰属させるようにした。
  同関数は追加要素の綴りを 3 通り扱うが、`<path>#<ID>` を扱う分岐だけが無く、標準行の追加は
  どの記録も主張できなかった。
  **帰属が無いのではなく、逆向きだった。** 追加を主張できない以上、行を足した当人は
  `ownsSpecificationAdditions` で「追加を所有しない」と判定されて `documentation_impact` を問われず、
  代わりに無関係な in_progress の記録がその追加を被って落ちていた。単体 RED の
  `Received: []` はこの逆転を示している。
  **同じ欠陥は 2 度目である。** [[wi-523-introspection-token-type-is-not-the-rfc6749-token-type]] の作業で `<path>:<name>` の TypeSpec 宣言に対して
  同じ修正が入っている（テスト `gives an added TypeSpec declaration to the record that declares it`
  の注記が記録している）。そのとき標準行は見られていない。今回の注記には、3 通りの綴りと、
  どれがどの分岐に対応するかを書いた。
  `addedScenarios` と `addedDeclarations` に同じ食い違いが無いことを確かめた。scenario は
  `spec-diff.ts:178` で id そのものを鍵にするので逐語比較で足り、declaration は
  `spec-diff.ts:79,157` で `<path>:<name>` を鍵にし、対応する分岐が既にある。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-work-items`（`docs/contexts/sourcing/standards.md` に行を 1 つ足し、
    その id を [[wi-534-deleted-scim-users-stay-visible-to-the-scim-client]] の `affected_spec` に
    書いた作業木で）
  - **Requirement**: N/A: 検査器の突き合わせ規則の欠陥であり、製品の規範要求には対応しない。
  - **Observed Failure**: exit 1。行を名指していない 2 件の記録が報告された。
    `work-items/wi-495-burn-down-the-standards-coverage-debt.md: documentation_impact none is weaker
    than inferred release_note` と、同じ指摘が本項目自身に対して 1 件。行を足した wi-534 は
    報告されなかった。修正後の同じ作業木では `ok  533 work-item dependency record(s)` になる。
  - **Detection Reason**: この検査は記録の集合と作業木の仕様差分を突き合わせる唯一の入口であり、
    帰属が壊れていれば、行を足していない記録が必ず名指しで現れる。応答の組み立てではなく
    報告される記録の集合そのものを読むので、誤った帰属と正しい帰属を区別できる。
- **Unit RED Evidence**:
  - **Test**: `tools/check/src/documentation-impact.test.ts` の
    `gives an added standards row to the record that declares it` と
    `matches an added standards row by document and requirement id together`
  - **Requirement**: N/A: 上と同じ理由による。
  - **Observed Failure**: 2 件とも失敗した。前者は行を足した当人について
    `Expected to contain: "documentation_impact none is weaker than inferred release_note" /
    Received: []`、後者は `claimsSpecificationAddition` が `Expected: true / Received: false`。
    修正後は 13 件すべてが通る。
  - **Detection Reason**: 前者は帰属の向き（当人が問われ、兄弟が問われない）を両側から読み、
    後者は突き合わせ規則そのものを読む。前者だけでは規則の緩めすぎを検出できないので、
    後者が別文書の同名 id と id 違いの 2 つを否定側として持つ。
- **Change-Resistance Results**:
  risk は `low` なので契約上は不要だが、Risk Notes が挙げた「主張の判定を緩めすぎる」を 1 件注入した。
  `#` の分岐から `path` の一致検査を落とし、id だけで主張できるようにした
  （`git diff --stat` で `tools/check/src/documentation-impact.ts | 24 +++++---` を確認）。
  `matches an added standards row by document and requirement id together` が
  `docs/contexts/other/standards.md` の参照について `Expected: false / Received: true` で落ちた。
  注入は観測後に元へ戻し、13 件が通ることを確認している。
- **Verification Results**:
  - `mise run verify` - passed（2026-09-12 に取得、exit 0、13.87s）
  - `mise run check-work-items` - passed（`ok  533 work-item dependency record(s)`）
  - `mise run test-tools` - passed（467 件、38 ファイル）
  - `mise run lint-tools` / `mise run typecheck-tools` / `mise run format-tools` - passed
  - `mise run spec-diff` - `no normative specification change against main`
  - `mise run test-ui-e2e` - N/A: 変更は `tools/check` の TypeScript 2 ファイルだけで、製品コードも
    フロントエンドも動いていない。ブラウザーへ届く経路が無い。
