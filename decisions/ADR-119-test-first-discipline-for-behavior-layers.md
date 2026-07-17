---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-119: 振る舞いを持つ層で test-first を必須にし、self-attest で運用する

## コンテキスト

RA は理念として test-first を掲げている（`REGENERATIVE_ARCHITECTURE.md`「振る舞いをテストとして
先に書き」、テストは `scenarios` / `constraints` / `state guards` から導く）。しかし実装手続きである
`implement-work-item` skill §1 は各層で「単体テストを同時に書く」までしか要求しておらず、失敗テストを
先に見る red-green を課していない。理念は test-first、運用は test-alongside という乖離がある。

AI エージェントによる実装では、この乖離が品質劣化として顕在化する。実装を先に書いてからそれを追認する
だけのテスト、あるいは常にパスして実質何も検証しないテストが混入し、テストが再生の合否判定器として
機能しなくなる。失敗を先に目視する規律は、この二つの劣化を構造的に防ぐ。

一方、Superpowers 型の strict TDD をそのまま全層・機械強制で移植すると、RA 固有の摩擦が出る。テストが
SCL と独立に振る舞いを規定すれば正本が二重化する。配線主体の Infrastructure / Deploy 層に「失敗する
単体テスト」を強いれば儀式化して回避を招く。層境界ごとの subagent 検証は実効的だが現段階では過剰である。

## 決定

振る舞いを持つ層——Domain / Use Cases / Adapters——で **test-first を必須**とする。手順は
RED → GREEN → REFACTOR。RED ゲートは「テストを実行し、意図した理由で落ちることを確認し、その失敗が
対応する SCL 要素（`scenario` / `constraint` / `state guard`）を参照している」ことを満たす。

テストは SCL から導く。テスト側で新しい振る舞いを創作しない（正本の二重化を避ける）。SCL に存在しない
振る舞いが必要になったら、テストを書く前に `scl-change` に戻って SCL を更新する。テストは SCL の受け入れ
条件を実行可能にした従属物であり、SCL と競合する第二のオラクルではない。

Infrastructure / Deploy Pipeline 層は単体 red-green を免除し、contract / E2E テストで代替する。ここでは
「失敗する単体テストを先に書く」ことは意味を持たず、境界の契約とデプロイ経路の検証が本質だからである。

強制方式は **self-attest ＋ レビュー抜き取り**（機械強制はしない）。work-item の `# Tasks` チェックリストに、
層ごとの test-first 証跡——先に落としたテストと、それが参照する SCL 要素——を自己申告として残す。
レビューはこの証跡を抜き取りで検証する。証跡は進捗正本である Tasks に同居させ、追加のツールを要さない。

## 却下した代替案

- 全層一律の strict TDD: 配線主体の境界層で儀式化し、回避と形骸化を招く。
- テストを独立オラクルとする一般的 TDD: SCL と競合する第二の正本を生み、再生アーキテクチャの前提を崩す。
- subagent 検証による機械強制（option A）: 実効性は最も高いが層境界ごとのコストが重く、現段階では過剰。
  将来、劣化が self-attest で止まらないと判明した時点で再検討する余地を残す。
- hook / commit 証跡による機械強制（option C）: 証跡形式の固定が先行し、軽量な立ち上げを妨げる。
- 現状維持（test-alongside）: AI 実装で生じる追認テスト・空テストの劣化を止められない。

## 影響

- `.agents/skills/implement-work-item/SKILL.md` §1 の「単体テストを同時に書く」を、層別の test-first
  ループ（Domain / UseCase / Adapters は必須、Infra / Deploy は contract / E2E 代替）へ改訂する。
- `REGENERATIVE_ARCHITECTURE.md` の test-first 理念と実装手続きが整合する（理念と運用の乖離を解消）。
- work-item の `# Tasks` に test-first 証跡欄が加わる。`WORK_ITEM_FORMAT.md` / `new-work-item` skill の
  Tasks 粒度が self-attest の証跡を含む。
- レビュー運用に test-first 証跡の抜き取り確認が加わる。機械強制は将来 option A / C として再検討可能。
