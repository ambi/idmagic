---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-14
priority: p1
depends_on: []
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "未公開の間のリポジトリ内ツールと開発文書の変更で、製品の公開契約、既存データ、成熟度は変えない。"
  references: []
initial_context:
  specification:
    - docs/development/task-context.md
    - docs/development/specification-first-workflow.md
    - docs/formats/work-item-format.md
    - AGENTS.md
  typespec: []
  source: [tools/brief/src/context.ts, tools/brief/src/main.ts]
  tests: [tools/brief/src/context.test.ts, tools/brief/src/context-measurement.test.ts]
  stop_before_reading: [backend, frontend, spec]
spec_impact:
  kind: none
  reason: "エージェントが着手時に読む開発文書の範囲と量を変える。製品の HTTP 応答、永続状態、イベント、TypeSpec と規範要件は維持する。"
---

# brief が読ませる開発文書の量を測り、変更に適用しない節を候補から外す

## 動機

`docs/development/` と `docs/formats/` は合わせて 313 KB ある。
2026-10-14 に、呼び出し元のない関数を一つ消すだけの [wi-66580](../active/wi-66580-remove-the-unused-resolve-policy-for-tenant.md) へ `mise run brief -- wi-66580 --target backend/authentication` を実行すると、明示の候補だけで約 48 KB の節を読むよう示した。

| 候補 | バイト数 |
| --- | --- |
| `specification-first-workflow.md#3-開発サイクル` | 15,511 |
| `specification-first-workflow.md#4-証拠の要件` | 10,843 |
| `specification-first-workflow.md#仕様影響の宣言` | 4,334 |
| `testing.md#主要ユースケース` と `#テストダブル` | 8,476 |
| `work-item-format.md#着手時の文脈と設計` | 3,758 |
| そのほかの明示の候補 | 約 5,700 |

推定の候補には `spec/modules/authentication/main.tsp` と `models.tsp` の全体（150 KB）も含まれる。
`spec_impact: none` の refactor では、主要ユースケース、TypeSpec、証拠の大半の子節は適用しない。
候補の出力にサイズがないため、読み手は量を知らずに開き、作者は文書が増えたことに気付けない。

## 対象範囲

- `brief` が候補ごとのバイト数と合計を表示する。
- AGENTS.md の参照表の条件を見直し、`spec_impact: none` の変更や refactor に適用しない行を候補から外す。
- `specification-first-workflow.md` の 3 節と 4 節を、変更の種類ごとに読む子節へ分け、参照表から子節を指す。
- `spec_impact: none` の変更では、TypeSpec のファイル全体を推定の候補にしない。
- 分割の過程で見つけた重複を削り、`docs/development/` と `docs/formats/` の合計を純減させる。

## 対象外

- 規則そのものの追加と変更。読む範囲と書き方だけを変える。
- `format-rationale.md` と `documentation-guide.md` の全面的な書き直し。前者は既に通常の参照経路の外にある。
- 合計のバイト数を検査で強制すること。上限の数値は計測を重ねてから決める。

## 設計

量を減らす手段は、文書の短縮より先に、適用しない節を候補から外すことに置く。
本文を短くしても、適用しない節を読ませる限り費用は残るためである。
条件の書き方は[作業文脈の導出](../../docs/development/task-context.md#一次情報からの導出)の `<!-- context: -->` を使い、新しい条件のキーは足さない。
`spec_impact` による絞り込みが既存のキーで表せない場合に限り、`kinds` の値として扱う方法を先に検討する。

採らない案:

- `brief` が節の本文を要約して返す案。本文を連結しない現在の方針に反し、要約と一次情報の食い違いを生む。
- バイト数の上限を検査で強制する案。適切な上限がまだ分からず、数値のために規則を削る圧力になる。

## 計画

1. 代表的な 3 件（本番コードの refactor、feature、tools の変更）で、現在の候補と合計のバイト数を記録する。
2. `brief` にバイト数の表示を加える。
3. 参照表の条件と、3 節と 4 節の分割を変え、同じ 3 件で再計測する。
4. 推定の TypeSpec 候補の条件を変える。

## タスク

- [x] T001 [Tooling] `brief` が候補ごとのバイト数と合計を表示する。RED を先に確かめる。
- [x] T002 [Docs] 3 節と 4 節を変更の種類ごとの子節へ分け、重複を削る。
- [x] T003 [Docs] AGENTS.md の参照表の条件を、分けた子節へ向ける。
- [x] T004 [Tooling] `spec_impact: none` では TypeSpec 全体を推定の候補にしない。
- [x] T005 [Verify] 3 件の前後の合計と、開発文書の合計バイト数の純減を記録する。

## 検証

- `mise run test-tools-file` で `brief` の表示と条件の判定を確かめる。
- 3 件の代表で、候補の合計が減り、各変更に適用する節が候補に残ることを確かめる。
- `mise run check-links` と `mise run check-work-items` を通す。

## リスク

適用する節まで候補から外し、エージェントが規則を読まずに作業する。
代表の 3 件で、外した節がその変更に適用しないことを一つずつ確かめ、判断を完了節に残す。

## 完了
- **完了日**: 2026-10-14
- **要約**:
  `mise run spec-diff` に規範の差分はない。
  `brief` は、各候補について最初の子見出しまでの本体の量と、子節の名前と量を表示し、本体の合計を示す。
  `spec_impact: none` の変更では、`--concern contract` がない限り TypeSpec のファイルを候補にしない。
  計画では 3 節と 4 節を子節へ分ける予定だったが、`brief` が子節を量とともに並べるので、文書の構造は変えなかった。
  AGENTS.md の着手の行は、証拠に限らず子節を当てはまるものだけ読むと改め、作業文脈の導出へ表示と TypeSpec の条件を記した。
  形式とワークフローの文書から、ほかの節と重なる 2 文を削った。`docs/development/` と `docs/formats/` は 313,546 バイトから 313,502 バイトになった。
  wi-66580 の `brief` は、変更前は全体で約 48 KB の節と 150 KB の TypeSpec を示していたが、変更後は本体の合計が 23 KB で、TypeSpec を含まない。
- **受け入れ RED の証拠**:
  - **テスト**: `tools/brief/src/context.test.ts` の `reading size > leaves TypeSpec out when the change declares no specification impact`
  - **要件**: N/A: リポジトリ内のツールの変更で、製品要件はない。
  - **観測した失敗**: 実装前は `measureCandidates` を export していないため、テストファイルの読み込みが `SyntaxError: Export named 'measureCandidates' not found` で失敗した。
  - **検出できる理由**: TypeSpec の候補の条件を常に真へ戻す故障を注入すると、このテストだけが失敗した。
- **単体 RED の証拠**:
  - **テスト**: `tools/brief/src/context.test.ts` の `reading size > measures the lead of a section and lists its children with their sizes`
  - **要件**: N/A: リポジトリ内のツールの変更で、製品要件はない。
  - **観測した失敗**: 受け入れの RED と同じ読み込みの失敗を観測した。
  - **検出できる理由**: 本体の量を節の全体で数える故障を注入すると、本体と子節の和が節の全体を超えるため、このテストが失敗した。最初の表明（本体が節の全体より小さい）はこの故障を見逃したので、和の表明へ強めた。
- **変更耐性の結果**:
  手で 2 つの故障を注入した。TypeSpec の条件を常に真にする故障と、本体を節の全体で数える故障は、どちらも検出された。2 つ目は最初の表明では生存したので、テストを強めて検出を確かめた。
- **検証結果**:
  - `mise run test-tools-file -- brief`: 35 件成功。
  - `mise run typecheck-tools`、`mise run lint-tools`: 成功。
  - `mise run check-links`、`mise run check-agent-guidance`、`mise run check-work-items`: 成功。
