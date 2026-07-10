---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-11
---

# RA を SCL 単一上流方式からコンテキストローカルな複数仕様形式へ再定義する

## Motivation
現行の Regenerative Architecture (RA) は、Specification Core Language
(SCL) を単一の規範的上流ソースとし、仕様全体が機械実行可能で、外側の成果物が
SCL から派生することを要求している。しかし IdMagic での運用では、独自形式を維持する
コストに対して、実装との意味的ドリフト検出やコード・テスト・UI・監視への機械的な
派生が十分に実現できていない。

特に `invariants` はモデル制約、状態遷移規則、認可、セキュリティ方針、運用要件、
アーキテクチャ判断を一つの分類へ集め、規則の所有場所を不明瞭にしている。
`user_experience` も実際の画面・操作経路を理解する仕様というより中央台帳になり、
UI 実装との同期コストを生んでいる。Glossary と States だけを独自形式に残しても、
そのために SCL 全体を維持する利益は小さい。

RA の本質である再生成可能性、判断の保存、検証、bounded context、閉ループ開発を
維持しつつ、仕様形式を SCL に固定しない。Markdown、Mermaid、TypeSpec、Cedar、
OpenSLO、テストなど、関心ごとに適した正本を context の近傍へ置く手法へ RA 本文を
改訂する。

## Scope
- `REGENERATIVE_ARCHITECTURE.md` の中心原則を「単一上流形式」から
  「関心ごとの単一正本」へ変更する。
- 再生成を、決定的なコード生成だけでなく、保存された仕様・判断・検証から
  機能的に同等な外側の層を再構築し、適合性を確認できることとして定義する。
- Specification Core を SCL 単一文書ではなく、context-local な Markdown、
  TypeSpec 等の契約、宣言的 policy、測定可能な objective、テストからなる
  仕様ポートフォリオとして再定義する。
- 自然言語仕様を正当な規範形式として認め、機械検証は契約、型、policy、
  測定可能な目標など効果が明確な関心へ適用する。
- `invariants` を独立した必須分類から外し、規則を model constraint、state / transition、
  interface の precondition / postcondition、policy、scenario、Architecture / ADR、
  objective、テストのうち所有責務が最も明確な場所へ置く原則を定める。
- `user_experience` を独立した中央仕様分類から外し、ユーザー goal 単位の軽量な
  UI Flow、実装近傍の説明、必要に応じた Storybook / E2E を用いる方針へ変更する。
- Glossary と States は context-local な Markdown を標準とし、状態遷移には
  Mermaid state diagram と必要に応じて遷移表を使えることを例示する。
- bounded context の責務、公開言語、依存関係の正本を Architecture とし、仕様・実装・
  テストを同じ context / feature 名で発見できるコンテキストローカリティ規則を強化する。
- コンセプションから検証までの開発フローを、SCL 全セクションの更新ではなく、
  変更対象に必要な規範成果物だけを先に更新する流れへ改訂する。

## Out of Scope
- `SPECIFICATION_CORE_LANGUAGE.md` の改訂、廃止、または歴史文書化。
- `spec/scl.yaml` と `spec/contexts/*.yaml` の移行・削除。
- TypeSpec、Cedar、OpenSLO、Storybookなどの依存導入や実装。
- SCL loader、validator、renderer、OpenAPI / JSON Schema generator の削除。
- `ARCHITECTURE.md`、agent skills、work item format、AGENTS.md の新方針への同期。
- IdMagic の既存仕様、実装、テスト、実行時挙動の変更。
- 新しい独自仕様manifestまたは代替DSLの設計。

## Plan
- `Regenerative Architecture` と `Specification Core` の名称、7層、ADR、Architecture、
  bounded context、閉ループ開発は維持する。
- §2 では単一上流ソースと全仕様の機械実行必須を削除し、関心ごとの単一正本、
  適合する表現、選択的な機械検証、コンテキストローカリティ、リスクに応じた
  追跡可能性へ置換する。
- §3.1 では SCL 固有のセクションと派生物一覧を削除し、仕様ポートフォリオの
  代表形式と「正本・実装・検証証拠」の役割分担を説明する。
- §3.2〜§3.4 では Architecture の `realizes SCL element` 前提と Domain の
  SCL 機械導出前提を外し、各層が context-local な仕様を実現する関係へ変更する。
- §3.8 では context map の所有を SCL から Architecture へ移し、context 固有概念を
  `shared` へ逃がさないこと、context 間は公開契約だけを使うこと、多数 context に
  波及する変更を境界問題として扱うことを明記する。
- §4 では AI が固定セクションを網羅するのではなく、変更する関心に適した仕様形式と
  検証を選ぶよう変更する。TDD は挙動変更・不具合修正で原則利用するが、利益の小さい
  作業へ形式的に強制しない。
- TypeSpec、Cedar、OpenSLO等は選択肢の例であり、RA の必須技術にはしない。
- 独自DSLは禁止せず、複数の有用な派生物またはCI検証によって保守費用を回収できる
  場合だけ採用する。

## Tasks
- [ ] T001 [RA] §2 の再生成可能性と仕様・検証の要件を新しい原則へ改訂する。
- [ ] T002 [RA] §3.1 の Specification Core を複数形式の仕様ポートフォリオとして
  再定義し、SCL 固有の分類・派生関係を削除する。
- [ ] T003 [RA] §3.2〜§3.4 の Architecture、Domain、Use Cases から SCL 機械導出の
  前提を除き、正本・実装・証拠の関係を明記する。
- [ ] T004 [RA] §3.8 の context map、物理配置例、コンテキストローカリティ規則を
  新方針へ改訂する。
- [ ] T005 [RA] §4 のコンセプションと開発フローを、関心に適した仕様形式を選ぶ
  SCL 非依存の閉ループへ改訂する。
- [ ] T006 [RA] 参考文献と用語を整理し、SCL必須・全仕様機械実行必須・SCLからの
  機械導出という旧前提が本文に残っていないことをレビューする。
- [ ] T007 [Verify] work item の形式・IDと、改訂後のMarkdownリンク・見出し・内部整合を
  検証する。

## Verification
- `just yaml-check-work-items`
- `just check-ids`
- `rg -n "SCL|単一上流|機械実行|機械的に導出|invariants|user_experience" REGENERATIVE_ARCHITECTURE.md`
  - 旧前提が意図せず残っていないことを文脈付きでレビューする。
- `REGENERATIVE_ARCHITECTURE.md` の §2、§3.1、§3.8、§4 を通読し、次を確認する。
  - Specification Core、Architecture、Domain、開発フローの説明が矛盾しない。
  - Markdown、標準仕様形式、policy、テストの役割が正本・実装・証拠として区別される。
  - `invariants` と `user_experience` が必須の中央分類として残っていない。
  - context locality が中心原則と物理配置例の両方に反映されている。

## Risk Notes
RA の規範を変更するため、アプリケーションコードを変更しなくても後続のSCL、Architecture、
skills、agent instructions、work item workflowへ大きく波及する。今回はRA本文だけを一つの
意味変更として確定し、既存IdMagic資産の即時削除や中途半端な併存移行を行わない。

自然言語仕様を認める変更が「検証をしなくてよい」という解釈にならないよう、機械検証が
有効な契約・型・policy・測定目標には実行可能な正本を使い、テストを実装適合性の証拠として
扱う原則を明記する。また、複数形式の採用が正本競合を招かないよう、関心ごとの正本を一つに
する規則を維持する。
