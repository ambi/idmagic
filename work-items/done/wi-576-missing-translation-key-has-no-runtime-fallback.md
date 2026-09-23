---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 型検査を通った辞書には欠落も空の値もないので、利用者が観測する文言は変わらない。変わるのは型検査を迂回した辞書に対する実行時の振る舞いと開発時の検査だけである。
  references: []
initial_context:
  specification: [docs/domain/system/scenarios.feature.md#REQ-SYSTEM-010, docs/domain/system/glossary.md]
  typespec: []
  source:
    - frontend/src/lib/i18n/dictionary.ts
    - frontend/src/lib/i18n/context.tsx
    - frontend/src/lib/i18n/locale.ts
    - frontend/src/lib/i18n/errorMessage.ts
  tests:
    - frontend/src/lib/i18n/context.test.tsx
  stop_before_reading: [backend, frontend/tests/e2e, frontend/src/features]
primary_use_cases:
  - id: missing-translation-falls-back-to-en
    requirement: REQ-SYSTEM-010
    observable_result: "`ja` 辞書で値が欠けたキーを、`ja` を選んだ画面が `en` 辞書の同じキーの文言で表示する。"
    unit_test: { path: frontend/src/lib/i18n/dictionary.test.ts, name: fills a missing or empty ja entry from the en entry, task: test-ui-unit }
    e2e_test: { path: frontend/src/lib/i18n/context.test.tsx, name: shows the en text for a key the ja dictionary lacks, task: test-ui-unit }
    unit_fault_model: "欠落だけを見て空文字列を訳として通す、または `en` 側の値ではなくキー名や空文字列で埋める。"
    e2e_fault_model: "画面へ渡る経路 (`useDictionary`) がフォールバックを解決した辞書ではなく生の `ja` を返す。"
affected_spec:
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-010 }
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
- 決めた側に応じて `docs/domain/system/scenarios.feature.md` または `frontend/src/lib/i18n` を変える。
- `EX-SYSTEM-010-03` を名指すテストを書き、`tools/check/example-coverage-debt.json` から外す。

## Out of Scope

- `EX-SYSTEM-010-01` と `010-02`。[[wi-541-back-system-examples-with-tests]] が消化済みである。
- 翻訳そのものの追加と修正。
- `en` 辞書の空の値。`en` は `FallbackLocale` そのものなので落ちる先がなく、空文字列も訳として扱う。`AdminSettingsPage.i18n.ts` の `countSuffix` は、英語では単位を付けないので意図して空である。

## Design

**実装をシナリオに合わせる。** `defineDictionary` が定義の時点で `ja` の欠けた値を `en` の同じキーの値で埋める。

決めた理由は次のとおりである。

- [System の用語集](../../docs/domain/system/glossary.md)の `FallbackLocale` は「対応する辞書に翻訳キーがない場合に使うロケール」と定め、`EX-SYSTEM-010-03` も同じ振る舞いを述べる。規範は二か所で一致しており、書き換える理由は「実装がまだない」ことしかない。
- 型検査は `as` や `any` を経た辞書、空文字列の値を止めない。型で防いでいる範囲の外で、画面に空の文言が出るか `en` が出るかを決めるのが具体例である。

フォールバックを `useDictionary` ではなく `defineDictionary` に置くのは、`commonDictionary[getCurrentLocale()]` や `shellDictionary[locale]` のように、フックを通らず辞書を直接引く箇所が 5 か所あるためである。定義の時点で解決すれば、どの経路でも同じ辞書を読む。

型と操作は次のとおりである。すべて純粋で、作用を持たない。

| 名前 | 形 | 役割 |
| --- | --- | --- |
| `defineDictionary` | `<T>(ja: T, en: { [K in keyof T]: string }) => Record<Locale, T>` | `ja` の値が未定義または空文字列のキーを `en` の値で埋めた辞書を返し、埋めたキーを記録する |
| `untranslatedKeys` | `(dictionary: Record<Locale, Record<string, string>>) => string[]` | `defineDictionary` が `en` の値で埋めた `ja` のキーを返す |

実行時のフォールバックは、翻訳漏れを型エラーで気付く圧力を弱める。そこで `*.i18n.ts` をすべて読み込み、どの辞書にも `untranslatedKeys` が空であることを確かめるテスト (`dictionaries.test.ts`) を置く。いまある `commonDictionary` だけの確かめを、52 個の辞書全体へ広げる。

採用しない案は次のとおりである。

| 案 | 採用しない理由 |
| --- | --- |
| シナリオと用語集を書き換え、欠落は型で起きないと定める | 型検査を迂回した辞書での表示が規範から消え、用語集の `FallbackLocale` の定義も変えることになる |
| `useDictionary` で描画のたびに解決する | 辞書を直接引く 5 か所に効かない。描画のたびに新しいオブジェクトを作るので、`t` を依存配列に入れる呼び出し側も揺れる |
| 欠落を `console.warn` で知らせる | 開発者がログを見る保証がない。すべての辞書を読む検査のほうが、漏れを CI で止められる |

## Tasks

- [x] T001 [Acceptance] `context.test.tsx` の `shows the en text for a key the ja dictionary lacks` の RED を `mise run test-ui-unit-file -- src/lib/i18n/context.test.tsx` で確認する (`EX-SYSTEM-010-03`)。
- [x] T002 [Domain] `dictionary.test.ts` の `fills a missing or empty ja entry from the en entry` と `reports the ja entries it had to fill` の RED を `mise run test-ui-unit-file -- src/lib/i18n/dictionary.test.ts` で確認し、`defineDictionary` と `untranslatedKeys` を GREEN にする。
- [x] T003 [Check] すべての `*.i18n.ts` に `untranslatedKeys` が空であることを確かめるテストを置き、空の値を一つ入れて失敗することを確かめる。
- [x] T004 [Verify] 台帳から `EX-SYSTEM-010-03` を外し、フォールト注入、`mise run check-spec`、`mise run verify`、`mise run test-ui-e2e` を通す。

## Verification

- `mise run check-spec` が `EX-SYSTEM-010-03` を台帳から外した状態で通る。
- `mise run test-ui-unit-file -- src/lib/i18n/context.test.tsx`
- `mise run verify`

## Risk Notes

- **型で防いでいることを、具体例を観測した証拠として数える。** キー集合の一致を確かめる検査は「欠落しない」ことしか言わず、「欠落したら `en` を出す」は言わない。どちらを規範とするか決めたうえで、その規範に対応する検査を置く。
- **フォールバックを足して、翻訳漏れが静かに英語で出るようになる。** いまは型検査が落ちて気付く。実行時フォールバックはその圧力を消すので、足す場合は漏れを別の場所 (検査またはログ) で見えるようにする。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の差分なしを報告した。規範は変えず、`EX-SYSTEM-010-03` と用語集の `FallbackLocale` が述べる振る舞いに実装を揃えた。
  `defineDictionary` が定義の時点で、`ja` の値が未定義または空文字列のキーを `en` の同じキーの値で埋める。フックを通らず辞書を直接引く 5 か所にも効かせるため、`useDictionary` ではなく定義の側で解決した。`en` は落ちる先がないので、空文字列も訳として扱う。
  フォールバックが翻訳漏れを画面から隠さないよう、`defineDictionary` が埋めたキーを `untranslatedKeys` で取り出し、`dictionaries.test.ts` がすべての `*.i18n.ts` (52 個) でそれが空であることを確かめる。最初の版は `en` の空の値も漏れとして数えたが、`AdminSettingsPage.i18n.ts` の `countSuffix` が英語で単位を付けないため意図して空であることが分かったので、数えないことにした。
  `tools/check/example-coverage-debt.json` から `EX-SYSTEM-010-03` を外した。
- **Primary Use Case Evidence**:
  - id: missing-translation-falls-back-to-en
    unit_red: 実装前、`fills a missing or empty ja entry from the en entry` が `empty` に空文字列、`missing` に `undefined` を受け取って失敗した。`reports the ja entries it had to fill` は空配列を受け取って失敗した。
    e2e_red: 実装前、`shows the en text for a key the ja dictionary lacks` が `ja` を選んだ画面で `Empty in ja` を期待して空文字列を受け取り、失敗した。
    unit_fault_injection: 欠落の判定を `ja[key] !== undefined` に変えて空文字列を訳として通すと、単体テスト 2 件と受け入れテストが失敗した。
    e2e_fault_injection: 埋めた `ja` を返す辞書へ配線せず生の `ja` を返すと、受け入れテストが失敗した。
- **Acceptance RED Evidence**:
  - **Test**: `frontend/src/lib/i18n/context.test.tsx` の `shows the en text for a key the ja dictionary lacks`
  - **Requirement**: REQ-SYSTEM-010
  - **Observed Failure**: `LocaleProvider` で `ja` を選び、`ja` の値が空文字列のキーを `useDictionary` 経由で描くと、`en` の文言ではなく空文字列が出た。
  - **Detection Reason**: 空文字列と値の欠落の両方を与え、画面が受け取る経路 (`useDictionary`) の出力を見るので、どちらか一方だけを埋める実装も、辞書へ配線しない実装も区別できる。
- **Change-Resistance Results**:
  変更は TypeScript だけで、`mise run test-go-mutation` は Go を対象とするので適用しない。欠落判定の弱化と配線の除去は上の fault injection で確かめた。全辞書の検査は `common.i18n.ts` の `languageSwitcherLabel` の `ja` を空にすると `lib/i18n/common.i18n.ts#commonDictionary.languageSwitcherLabel` を報告して失敗した。
- **Verification Results**:
  - `mise run check-work-items` - 成功
  - `mise run check-spec` - 成功
  - `mise run test-ui-unit-file -- src/lib/i18n/dictionary.test.ts` - 成功 (3 件)
  - `mise run test-ui-unit-file -- src/lib/i18n/context.test.tsx` - 成功 (3 件)
  - `mise run test-ui-unit-file -- src/lib/i18n/dictionaries.test.ts` - 成功 (1 件)
  - `mise run verify` - 成功 (初回は `dictionaries.test.ts` の整形で失敗し、`mise run format-ui` の後に成功)
  - `mise run test-ui-e2e` - 成功 (38 件)
