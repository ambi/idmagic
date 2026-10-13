---
status: pending
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
---

# モジュールの設計領域の索引をモジュールの README の最後の節へ統合する

## 動機

[wi-47518](../done/wi-47518-align-shared-document-placement-and-design-records.md)は、機能スライスの `design.md` を機能仕様の最後の節「実現方式」へ統合した。
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

- [ ] T001 [Docs] 着手時の文脈を記録し、参照するツールと文書を確定する。
- [ ] T002 [Tooling] 配置の宣言、索引の検査、モジュールの印、探索、生成サイトを、新しい節を読む形へ RED から変える。
- [ ] T003 [Docs] 21 モジュールの索引を README の「設計」の節へ移し、`design/README.md` を削除してリンクを張り替える。
- [ ] T004 [Docs] 文書ガイド、設計フォーマット、仕様フォーマット、フォーマットの理由を更新する。
- [ ] T005 [Verify] 生成サイトの目次とモジュールのページを確かめる。
- [ ] T006 [Verify] ツールと文書の検査、`spec-diff` を通す。

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
