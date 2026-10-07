---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-08
priority: p1
depends_on: []
change_kind: maintenance
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

## 計画

1. `overrides` で `katex` を 0.18.2 以上へ上げ、`tools/bun.lock` を更新する。
2. 文書生成と `tools` のテストを走らせ、Mermaid 図の生成を確かめる。
3. 壊れる場合は引き上げを戻して抑止を加える。

## タスク

- [ ] T001 [Deps] `katex` を修正版へ引き上げるか、理由と期限付きで抑止する。
- [ ] T002 [Verify] `mise run audit-dependencies` の合格と文書生成の成功を確かめる。

## 検証

- `mise run audit-dependencies`
- `mise run verify`

## リスク

リスクは low。
`overrides` による引き上げは、Mermaid の数式描画と KaTeX の非互換を生む可能性がある。
影響は開発用の文書生成に閉じ、生成の確認で検出できる。
