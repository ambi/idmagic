---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-14
priority: p2
depends_on: []
change_kind: tooling
spec_impact:
  kind: none
  reason: "モジュールの設計領域の索引を置くファイルと、その検査、探索、生成サイトを変える。REQ、例、状態遷移、標準の行、TypeSpec の宣言と各設計文書の内容は維持する。"
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "未公開の製品について、開発者が読むモジュール設計文書の配置と、その検査、探索、生成サイトだけを変える。HTTP 契約、既存データ、運用手順に利用者が観測する差分はない。"
  references: []
initial_context:
  specification:
    - docs/formats/design-format.md
    - docs/formats/documentation-guide.md
    - docs/formats/specification-format.md
    - docs/development/format-rationale.md
    - docs/modules/README.md
    - docs/modules/tenancy/README.md
  typespec: []
  source:
    - tools/workspace/src/document-layout.ts
    - tools/check/src/specification-doc.ts
    - tools/check/src/check-documents.ts
    - tools/check/src/feature-slices.ts
    - tools/brief/src/context.ts
    - tools/render-docs/src/render.ts
  tests:
    - tools/check/src/specification-doc.test.ts
    - tools/check/src/canonical-document-set.test.ts
    - tools/check/src/feature-layout.acceptance.test.ts
    - tools/brief/src/context.test.ts
    - tools/render-docs/src/render.test.ts
  stop_before_reading: [backend, frontend, spec, work-items/done]
---

# モジュールの設計領域の索引をモジュールの README の最後の節へ統合する

## 動機

[wi-47518](wi-47518-align-shared-document-placement-and-design-records.md)は、機能スライスの `design.md` を機能仕様の最後の節「実現方式」へ統合した。
理由は、機能について知るべき規則と仕組みを一つのファイルで読めるようにし、`README.md` と `design.md` のように名前から違いが読めないファイルを並べないためだった。

モジュールの階層には同じ問題が残っている。
`docs/modules/<module>/README.md` は仕様（責務と境界、モデル、公開する契約、機能の索引）、`docs/modules/<module>/design/README.md` は設計領域の索引である。
どちらも README であり、名前から違いを読み取れない。

21 個の `design/README.md` は 18 から 29 行で、中身は次の三つに限られる。

| 内容 | 実態 |
| --- | --- |
| 導入の段落 | 「仕様は README、機能に閉じた設計は実現方式、記載のない領域は共通設計」という全モジュール共通の文 |
| 設計領域の索引 | 固有設計、委譲、対象外の 1 から 6 行 |
| 設計の記録の表 | `decisions.md` と `risks.md` へのリンクで、全モジュールでほぼ同じ |

固有の内容は索引の数行だけであり、一つのファイルとして入口を分ける理由が弱い。
機能の階層と違い、モジュールの設計は `architecture.md`、`data.md`、`decisions.md` など平均 4 本の文書にまたがるため、`design/` ディレクトリは残す必要がある。
統合するのは索引だけである。

## 対象範囲

- 21 モジュールの `design/README.md` の設計領域の索引と設計の記録へのリンクを、モジュールの `README.md` の最後の節「設計」へ移し、`design/README.md` を削除する。
- 全モジュール共通の導入の文は、各モジュールへ写さず、[設計フォーマット](../../docs/formats/design-format.md#1-設計領域)の規則に一度だけ置く。
- 索引の検査（列、設計領域の語彙と順序、参照先の存在、固有の領域文書への参照）を、新しい節に対して行うよう変える。
- モジュールの印を `design/README.md` の有無から、`docs/modules/<module>/README.md` の位置と「設計」の節へ変え、検査、探索、`brief`、生成サイトの目次を揃える。
- 文書ガイド、設計フォーマット、仕様フォーマット、フォーマットの理由の該当する記述を更新する。
- `spec-diff` が「設計」の節を規範の差分として扱わないことを確かめる。

## 対象外

- `design/` の中の設計文書（`architecture.md`、`data.md`、`decisions.md`、`risks.md`、`<concept>.md`）の統合や内容の変更。
- システムの `docs/design/README.md` の変更。システムは九領域すべてを索引し、共通設計の適用範囲を定める独立した入口である。
- 機能スライスの「実現方式」の節の変更。
- 完了した作業項目の中のリンクの書き換え。

## 設計

### 候補の比較

| 候補 | 利点 | 欠点 | 採否 |
| --- | --- | --- | --- |
| 索引をモジュールの README の最後の節「設計」へ移し、`design/README.md` を削除する | 機能の階層と同じく、モジュールについて知るべき仕様と設計の入口が一つになる。README が二つ並ばない | モジュールの印と索引の検査の入力を変える。仕様の文書に設計の索引が入る | 採用。機能仕様も最後の節に実現方式を置いており、規範の節と区別できる |
| `design/README.md` を `design/index.md` などへ改名する | 検査の変更が小さい | 入口が二つのまま残り、どちらから読むかは変わらない | 不採用 |
| `design/` の文書をすべてモジュールの README へ統合する | ファイル数が最も減る | 平均 4 本、長いものでは数百行の設計がモジュールの仕様に混ざり、仕様が読めなくなる | 不採用 |
| 現状を維持する | 変更がない | 名前から違いが読めない入口が 21 組残る | 不採用 |

### 節の形

モジュールの `README.md` の最後に `## 設計` を置き、既存の索引の表（`設計領域 | 区分 | 参照先 | 内容`）と、設計の記録（`design/decisions.md`、置いていれば `design/risks.md`）へのリンクを書く。
差分のない領域は書かず、記載のない領域は共通設計に従う、という規則は設計フォーマットが定める。
参照先のパスは `design/` からの相対へ書き換える。
機能仕様の「実現方式」と同じく、この節に `REQ-*` を置かない。

### モジュールの印

現在は `design/README.md` の有無でモジュールを判定し、履歴のリビジョンや一つのファイルだけを読む道具も同じ判定ができるようにしている。
新しい印は、`docs/modules/` の直下のディレクトリにある `README.md` とする。
`docs/modules/` の直下の `README.md`、`glossary.md`、`scenarios.feature.md` はディレクトリではないため区別できる。
着手時に、`tools/workspace/src/document-layout.ts`、`tools/check/src/specification-doc.ts`、`tools/check/src/feature-slices.ts`、`tools/brief/src/context.ts`、`tools/render-docs/src/render.ts` の判定を確かめ、同じ印へ揃える。
`spec-diff` は main の履歴の旧配置を読むため、旧配置の `design/README.md` を設計の索引として読める状態を保つかを着手時に判断する。

## 計画

1. 着手時に、`design/README.md` を参照するツール、文書、リンクを数え、印と索引の検査の変更点を確定する。
2. 検査と配置の宣言を、新しい節と印を受理し、`design/README.md` を拒否する形へ、テストを先に失敗させてから変える。
3. 21 モジュールの索引を移し、`design/README.md` を削除し、リンクを張り替える。
4. 文書ガイド、設計フォーマット、仕様フォーマット、フォーマットの理由を更新する。
5. 生成サイトの目次とモジュールのページを確かめ、検証して完了する。

## タスク

- [x] T001 [Docs] 着手時の文脈を記録し、参照するツールと文書を確定する。
- [x] T002 [Tooling] 配置の宣言、索引の検査、モジュールの印、探索、生成サイトを、新しい節を読む形へ RED から変える。
- [x] T003 [Docs] 21 モジュールの索引を README の「設計」の節へ移し、`design/README.md` を削除してリンクを張り替える。
- [x] T004 [Docs] 文書ガイド、設計フォーマット、仕様フォーマット、フォーマットの理由を更新する。
- [x] T005 [Verify] 生成サイトの目次とモジュールのページを確かめる。
- [x] T006 [Verify] ツールと文書の検査、`spec-diff` を通す。

## 検証

- `mise run test-tools`、`mise run lint-tools`、`mise run typecheck-tools`：配置の宣言、検査、探索、描画の変更を対象に選ぶ。
- `mise run check-spec`、`mise run verify-spec`、`mise run check-rendered-docs`：モジュールの文書の形式、索引の検査、生成サイトを対象に選ぶ。
- `mise run check-links`、`mise run check-work-items`、`mise run check-work-item-references`、`mise run check-terminology`：張り替えた参照と用語を対象に選ぶ。
- `mise run spec-diff`：「設計」の節が規範の差分にならないことを確かめる。
- `mise run render-docs` で、モジュールのページに「設計」の節があり、目次に `design/README.md` のページが残らないことを確かめる。

## リスク

モジュールの印は検査、探索、生成サイトの多くの入口で使われており、一つでも旧い印を読み続けると、モジュールを見落とす。
配置の宣言を一か所に保ち、新旧の配置を入力にしたテストで各入口を確かめる。
仕様の文書に設計の索引が入ることで、規範と設計の区別が曖昧になる恐れがある。
最後の節に限り `REQ-*` を置かない規則を、機能仕様の「実現方式」と同じく検査する。

## 完了

- **完了日**: 2026-10-14
- **要約**:
  `mise run spec-diff` は main に対して「no normative specification change against main」を報告した。REQ、例、状態遷移、標準の行、TypeSpec の宣言に意味上の差分はない。
  21 モジュールの `design/README.md` を削除し、設計領域の索引、設計の記録へのリンク、横断的概念の一覧をモジュールの `README.md` の最後の節「設計」へ移した。全モジュール共通の導入文は写さず設計フォーマットの規則に置き、固有の文（SLO の参照）だけを節に残した。リンクは `design/` からの相対をモジュールからの相対へ書き換えた。
  モジュールの印を `docs/modules/<module>/README.md` の有無へ変え、文書の種別、配置の宣言、索引の網羅検査、規範規則の検査、`brief`、生成サイトの目次を揃えた。検査は `design/README.md` を拒否し、索引を最後の節「設計」の外に置くことを拒否する。生成サイトは `design/` の文書をモジュールの「設計」の段に並べる。`brief` は「設計」の節の行だけを作用の索引として読む。
  文書ガイド、設計フォーマット、仕様フォーマット、モジュール設計文書の入口、フォーマットの理由を更新した。
  移行は一度だけのスクリプトで行い、リポジトリには残していない。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/check/src/feature-layout.acceptance.test.ts` の「requires every module to have README.md」「rejects a separate design index in the design directory」「rejects a design area index outside the last section 設計」と、索引をモジュールの README に置いた既存のフィクスチャ。
  - **要件**: N/A: 製品の振る舞いを変えない、文書の配置とその検査ツールの変更である。
  - **観測した失敗**: 実装前は、モジュールの印を `design/README.md` で判定していたため、README に索引を置いたモジュールのすべての文書が拒否され、26 件が失敗した。
  - **検出できる理由**: 検査をリポジトリと同じ入口（`runner.ts`）から仮の作業ツリーへ実行するため、旧い印、旧い索引の位置、節の外の索引を受理する実装を区別できる。
- **単体 RED の証拠**:
  - **テスト**: `tools/brief/src/context.test.ts` の 3 件と、`tools/render-docs/src/render.test.ts` の「renders a module in the feature layout with generated rule indexes」。
  - **要件**: N/A: 製品の振る舞いを変えない、文書の配置とその探索と描画の変更である。
  - **観測した失敗**: `brief` はモジュールの README の「設計」の節を読まず、機能の表を作用の索引に含め得た。生成サイトは索引の文書がないと `design/` の文書を目次から到達できないページとして拒否した。
  - **検出できる理由**: `brief` は設計の節の外に作用の語を含む行を置いた入力で、生成サイトは索引のない `design/` の文書を含む入力で、それぞれの欠落を区別できる。
- **変更耐性の結果**:
  手作業で次の故障を注入し、それぞれ局所テストが検出した。
  - 索引が最後の節「設計」にあるかの検査を無効にする：`feature-layout.acceptance.test.ts` の 1 件が失敗した。
  - 目次の「設計」の段を外す：`render.test.ts` の複数件が到達できないページとして失敗した。
  変更は TypeScript のツールと Markdown なので、Go のミューテーションテストは適用しない。
- **検証結果**:
  - `mise run test-tools`：配置の宣言が検査、探索、描画で共有されるため、ツール全体を選んだ。1019 件が成功した。
  - `mise run lint-tools`、`mise run typecheck-tools`：変更したツールのコードを対象に選んだ。成功した。
  - `mise run verify-spec`：移した索引を含むモジュールの文書の形式、索引の検査、規範参照、生成サイトの検査として選んだ。成功した。
  - `mise run check-links`、`mise run check-work-items`、`mise run check-work-item-references`、`mise run check-terminology`：張り替えた参照、作業記録、用語を対象に選んだ。成功した。
  - `mise run spec-diff`：規範の差分がないことの確認として選んだ。差分なしだった。
  - `mise run render-docs` で再生成し、Tenancy の設計判断のページで、目次のモジュールの下に「設計」の段があり、`design/` の 4 文書が並ぶことを確かめた。
