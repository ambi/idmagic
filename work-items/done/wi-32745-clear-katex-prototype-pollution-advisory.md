---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-08
priority: p1
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 開発用の推移的依存の版だけを変え、文書が説明する手順、規則、振る舞いはどれも変わらない。
  references: []
initial_context:
  specification: []
  typespec: []
  source: [tools/package.json, osv-scanner.toml]
  tests: [tools/render-docs]
  stop_before_reading: [backend, frontend]
spec_impact:
  kind: none
  reason: >-
    tools/ の文書生成器だけが使う開発用依存の扱いを変える。
    API の応答、永続状態、ドメインイベント、外部への呼び出しはどれも変わらない。
---

# katex の脆弱性勧告 GHSA-238p-pmpm-9mq7 で失敗する依存監査を解消する

## 動機

2026-10-05 に公開された GHSA-238p-pmpm-9mq7 (KaTeX: Existing prototype pollution can bypass trust restrictions、CVSS 2.1) により、CI の `Audit dependencies for known vulnerabilities` ジョブが `mise run audit-dependencies` で失敗している。
該当するのは `tools/bun.lock` の `katex@0.16.47` で、修正版は 0.18.2 である。

`katex` は `tools/render-docs` が使う `mermaid@12.1.0` の推移的依存として入る。
`mermaid@12.1.0` は最新版であり、`katex` を `^0.16.47` で要求するので、通常の依存更新では修正版へ上がらない。

勧告の内容は、すでに `Object.prototype` が汚染されている環境で、継承した `trust` などを KaTeX が描画オプションとして読む、というものである。
KaTeX 自身は汚染を起こさない。

## 対象範囲

- 監査を合格させる。次のどちらかを設計で選ぶ。
  - `tools/package.json` の `overrides` で `katex` を修正版へ上げる。
  - `osv-scanner.toml` に `reason` と `ignoreUntil` 付きの抑止を加える。
- 選んだ手段について、`tools/render-docs` の生成結果（Mermaid 図）が壊れていないことを確かめる。

## 対象外

- `mermaid` の置き換え。
- `frontend/` の依存。`katex` は `frontend/bun.lock` に現れない。

## 設計

第一案は `overrides` による修正版への引き上げである。
0.16 から 0.18 への上げは semver 上の非互換を含みうるが、Mermaid が KaTeX を使うのは図中の数式の描画だけである。
`mise run render-docs` 相当の生成を走らせ、出力した HTML が Mermaid 図を含むことを確かめれば、影響の有無を判定できる。

引き上げで生成が壊れる場合に限り、抑止を選ぶ。
抑止を選ぶときの理由は次のとおりである。
`katex` は開発者が書いた Mermaid の図をリポジトリ内で HTML にする生成器でだけ動き、IdMagic の利用者の入力を受けない。
また、勧告の成立には別の脆弱性によるプロトタイプ汚染が前提となる。
`ignoreUntil` は既存の抑止と同じく見直し期限として置き、`mermaid` が `katex` の範囲を上げたら外す。

### 採用した手段と残る同梱物

`overrides` で `katex` を 0.18.2 に固定した。
`render-docs` は Mermaid の ESM（`mermaid.core.mjs`）を Bun から読み、全図を `mermaid.parse` する。
この経路の `katex` は `tools/node_modules` から解決されるので、override が効く。
引き上げ後も、全図の解析を含む文書生成は成功した。

一方、文書サイトがブラウザへ配る `mermaid/dist/mermaid.min.js` は、Mermaid の公開時に `katex` 0.16 系を同梱してビルドされている。
override はこの同梱物を変えない。
同梱物はロックファイルに現れないので、OSV-Scanner には見えない。
この残存は受け入れる。
同梱物はリポジトリの開発者が書いた図を描画するだけで、IdMagic の利用者の入力を受けない。
また、勧告の成立には別の脆弱性によるプロトタイプ汚染が前提となる。
`mermaid` が `katex` の範囲を上げた版へ更新すれば、override と同梱物の残存がともに解消する。

## 計画

1. `overrides` で `katex` を 0.18.2 以上へ上げ、`tools/bun.lock` を更新する。
2. 文書生成と `tools` のテストを走らせ、Mermaid 図の生成を確かめる。
3. 壊れる場合は引き上げを戻して抑止を加える。

## タスク

- [x] T001 [Deps] `katex` を修正版へ引き上げるか、理由と期限付きで抑止する。
- [x] T002 [Verify] `mise run audit-dependencies` の合格と文書生成の成功を確かめる。

## 検証

- `mise run audit-dependencies`
- `mise run verify`

## リスク

リスクは low。
`overrides` による引き上げは、Mermaid の数式描画と KaTeX の非互換を生む可能性がある。
影響は開発用の文書生成に閉じ、生成の確認で検出できる。

## 完了

- **完了日**: 2026-10-08
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  `tools/package.json` の `overrides` で `katex` を 0.18.2 に固定し、`tools/bun.lock` を更新した。
  `mise run audit-dependencies` は GHSA-238p-pmpm-9mq7 を報告しなくなった。
  文書サイトがブラウザへ配る `mermaid.min.js` には `katex` 0.16 系が同梱されたまま残る。
  この残存と受け入れる理由は設計に記録した。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run audit-dependencies`
  - **要件**: N/A: 開発用の推移的依存の版の変更であり、対応する REQ がない。
  - **観測した失敗**: 変更前は `GHSA-238p-pmpm-9mq7`（npm、`katex` 0.16.47、修正版 0.18.2、`tools/bun.lock`）を報告して終了コード 1 で失敗した。CI の `Audit dependencies for known vulnerabilities` ジョブの失敗と同じである。
  - **検出できる理由**: OSV-Scanner はロックファイルに記録された版を勧告の影響範囲と突き合わせるので、版が修正版に達しない限り失敗する。
- **単体 RED の証拠**:
  - **テスト**: N/A: 版の固定だけの変更であり、単体の境界がない。
  - **要件**: N/A: 対応する REQ がない。
  - **観測した失敗**: 該当なし。代わりに、引き上げ後に `mise run render-docs` と `mise run check-rendered-docs` を走らせ、全 Mermaid 図の `mermaid.parse` を含む生成の成功を確かめた。
  - **検出できる理由**: `render-docs` は Bun から `mermaid` の ESM を読み込み、`tools/node_modules` の `katex` を解決する。非互換があれば読み込みか解析で失敗する。
- **変更耐性の結果**:
  `rg` で `mermaid.min.js` を調べ、`katex` の描画コードが同梱されていることを確かめた。override がこの同梱物に効かないことは、ロックファイルに現れない同梱物を OSV-Scanner が見ないことと合わせて設計に記録した。
- **検証結果**:
  - `mise run audit-dependencies` - passed
  - `mise run check-rendered-docs` - passed
  - `mise run verify` - passed
