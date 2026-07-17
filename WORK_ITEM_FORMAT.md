# ワークアイテム

一つの意味変更として説明・実装・検証できる作業単位。配置は `work-items/` に置く。

- ファイル名は `work-items/wi-<連番>-<kebab-title>.md`。 `<連番>` はユニークな連番。
- **未着手・進行中（`pending` / `in_progress`）は `work-items/` 直下に置く。**
  **完了・中止（`completed` / `cancelled`）は同じ `work-items/` 下の `done/` サブディレクトリに移す**
  （例 `work-items/done/<id>.md`）。これで「まだ動きのある作業」と「終わった作業」を、ファイルを開いて `status` を読まなくても配置だけで区別できる。

work-item は次のような構成となる。

```markdown
---
status: pending  # pending | in_progress | completed | cancelled
authors: [name]
risk: low        # low | medium | high | critical
created_at: 2026-01-01  # YYYY-MM-DD
depends_on: []   # この WI の完了前に完了が必要な WI ID
change_kind: feature  # feature | bugfix | operations | refactor | docs | tooling | maintenance
initial_context:
  scl: { System: [interfaces.StartTask] }
  source: [src/usecase]
  tests: [src/usecase]
  stop_before_reading: [frontend]
affected_spec:
  - { context: System, kind: interface, element: StartTask }
---

# 一文で表す意味変更

## Motivation
なぜこの変更が必要か（What ではなく Why）の背景。

## Scope
- `spec/scl.yaml` の `interfaces.StartTask`
- `src/usecase/` への実装

## Out of Scope
- 明示的にやらないこと。

## Plan
- 採る技術方針、触れる層、却下した代替案、未決定事項。

## Tasks
- [ ] T001 [SCL] 仕様を更新する。
- [ ] T002 [App] 実装する。RED: 先に落ちるテストを確認（scenario `xxx.yyy`）→ GREEN。
- [ ] T003 [Verify] 検証する。

## Verification
- 予定する検証コマンド（例：`go test ./...`）や手動手順。

## Risk Notes
リスクの根拠と軽減方法。

## Completion
- **Completed At**: 2026-01-01
- **Summary**:
  実装した変更の意味上の差分の要約。
- **Verification Results**:
  - `just verify` - passed
```

`depends_on` はこの work-item の**完了前提**だけを列挙する。参照先は同じ
work-items 名前空間（`done/` を含む）にある WI ID とし、自己参照・循環参照は許可しない。
本文中の関連リンク、範囲外への委譲、後続候補は `depends_on` に入れず、従来どおり本文で記す。
未着手・進行中の WI では `depends_on` を必ず明記し、依存がなければ `[]` とする。

`feature` / `bugfix` / `operations` は `initial_context` と `affected_spec` を必須とし、
`affected_spec` は `context`、`kind`、`element` （standard requirement だけは
`standard` + `requirement`）の direct SCL element reference を使う。仕様非影響の
`refactor` / `docs` / `tooling` / `maintenance` は、`affected_spec` の代わりに
`spec_impact: { kind: none, reason: "..." }` と具体的理由を宣言できる。新規入力で
`affected_guarantees` は使用しない（完了済み履歴は書き換えない）。

次のいずれかに該当するワークアイテムは、中規模以上として `## Plan` と `## Tasks` を書く。

- 複数のシナリオ、ユーザーストーリー、または独立した利用者価値を含む
- RA の 3 層以上にまたがる
- DB migration、認可、外部契約、破壊的変更、不可逆な移行を含む
- 予定する作業が 1 セッションで終わる確信を持てない
- 検証が複数サブシステムにまたがる

振る舞いを持つ層（Domain / Use Cases / Adapters）に触れる Task は、test-first の証跡——先に落とした
テストと参照する SCL 要素——を Task 行に self-attest として残す（ADR-119）。
