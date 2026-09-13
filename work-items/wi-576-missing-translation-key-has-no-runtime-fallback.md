---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/system/scenarios.feature.md, requirement: REQ-SYSTEM-010 }
---

# 翻訳キーの欠落に実行時のフォールバックが無い

## Motivation

[[wi-541-back-system-examples-with-tests]] が `EX-SYSTEM-010-03` を消化しようとして、シナリオと実装が食い違っていることを測った。

`EX-SYSTEM-010-03` は「翻訳キーが欠落している → `FallbackLocale`（`en`）の対応するキーを表示する」と述べる。実装にその経路は無い。

- `useDictionary(dictionary)` は `dictionary[locale]` を返すだけで、キー単位のフォールバックを持たない (`frontend/src/lib/i18n/context.tsx`)。
- 欠落を防いでいるのは型である。`defineDictionary(ja, en)` が `ja` をキー集合の正とし、`en` に同じキー集合を強制する (`dictionary.ts`)。欠落や余剰は呼び出し箇所の型エラーになる。
- `FALLBACK_LOCALE` が使われるのは未対応ロケール (`configuredDefaultLocale`) と Provider を持たない木の既定値だけで、キーの欠落には効かない。

したがって具体例の言う実行時の振る舞いは存在しない。「欠落しない」という保証で置き換えられている。

## Scope

- 次のどちらが正かを決める。**決めるまでテストを書かない。**
  - シナリオを実装に合わせる。`TranslationKeyIntegrity` が欠落を型で防ぐことを規範として書き、`EX-SYSTEM-010-03` をその形に書き直す。この場合、52 個の `*.i18n.ts` 全体に対してキー集合の一致を機械で確かめる検査が要る (いまあるのは `commonDictionary` の 1 件だけ)。型検査を通らない書き方が将来できたときに落ちる場所を持たないと、規範に検査が対応しない。
  - 実装をシナリオに合わせる。`useDictionary` にキー単位のフォールバックを足す。値が空または未定義のときに `en` の同じキーを返す経路を作る。
- 決めた側に応じて `docs/contexts/system/scenarios.feature.md` または `frontend/src/lib/i18n` を変える。
- `EX-SYSTEM-010-03` を名指すテストを書き、`tools/check/example-coverage-debt.json` から外す。

## Out of Scope

- `EX-SYSTEM-010-01` と `010-02`。[[wi-541-back-system-examples-with-tests]] が消化済みである。
- 翻訳そのものの追加と修正。

## Verification

- `mise run check-spec` が `EX-SYSTEM-010-03` を台帳から外した状態で通る。
- `mise run test-ui-unit-file -- src/lib/i18n/context.test.tsx`
- `mise run verify`

## Risk Notes

- **型で防いでいることを、具体例を観測した証拠として数える。** キー集合の一致を確かめる検査は「欠落しない」ことしか言わず、「欠落したら `en` を出す」は言わない。どちらを規範とするか決めたうえで、その規範に対応する検査を置く。
- **フォールバックを足して、翻訳漏れが静かに英語で出るようになる。** いまは型検査が落ちて気付く。実行時フォールバックはその圧力を消すので、足す場合は漏れを別の場所 (検査またはログ) で見えるようにする。
