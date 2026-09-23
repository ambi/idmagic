---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 表示言語を明示選択済みの利用者に対して、RP が `ui_locales` で指定した言語が効かなくなるため、RP の開発者とリリースの読者へ知らせる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-575-explicit-display-language-outranks-the-ui-locales-hint.md }
initial_context:
  specification: [docs/domain/system/scenarios.feature.md#REQ-SYSTEM-008, docs/domain/system/glossary.md]
  typespec: []
  source:
    - frontend/src/lib/i18n/resolveLocale.ts
    - frontend/src/lib/i18n/context.tsx
  tests:
    - frontend/src/lib/i18n/resolveLocale.test.ts
    - frontend/src/lib/i18n/context.test.tsx
    - frontend/src/components/LanguageSwitcher.test.tsx
  stop_before_reading: [backend, frontend/tests/e2e]
primary_use_cases:
  - id: explicit-choice-survives-ui-locales-hint
    requirement: REQ-SYSTEM-008
    observable_result: 表示言語 `ja` を明示選択した利用者が `ui_locales=en` 付きの認可リクエストで新しいページを開いても、画面は `ja` 辞書で表示される。
    unit_test: { path: frontend/src/lib/i18n/resolveLocale.test.ts, name: prefers the saved explicit choice over a supported ui_locales hint, task: test-ui-unit }
    e2e_test: { path: frontend/src/components/LanguageSwitcher.test.tsx, name: keeps an explicit Japanese choice when an authorization request hints English, task: test-ui-unit }
    unit_fault_model: 解決順で `ui_locales` ヒントを保存済み設定より先に評価する。
    e2e_fault_model: 明示選択を保存しない、または新しいページの寿命で保存済み設定を読まずに URL のヒントだけで決める。
affected_spec:
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-008 }
---

# 明示選択した表示言語が `ui_locales` ヒントに負ける

## Motivation

[[wi-541-back-system-examples-with-tests]] が `EX-SYSTEM-008-02` を消化しようとして、シナリオと実装が食い違っていることを測った。

`EX-SYSTEM-008-02` は「表示言語を `ja` と明示選択済みの EndUser へ `ui_locales=en` の認可リクエストが来ても、ログイン画面は `ja` 辞書で表示される」と述べる。[System の用語集](../../docs/domain/system/glossary.md)の `DisplayLanguage` も「選択はブラウザーに保存し、以後のアクセスでは保存済みの設定を優先する」と述べている。

実装の解決順は `frontend/src/lib/i18n/resolveLocale.ts` にあり、**`ui_locales` ヒント > 保存済み設定 > ブラウザー言語 > 起動時の既定**である。明示選択を保存する先は `localStorage` の `idmagic.displayLocale` だけで (`context.tsx` の `writeSavedLocale`)、その値は「保存済み設定」として読み直される。したがって:

- 明示選択が同じページの寿命の中にあるあいだは、再解決が起きないので `ja` のままである。
- 認可リクエストはブラウザーの遷移なので、`/authorize?...&ui_locales=en` で新しいページの寿命が始まる。`readInitialLocale` はそこで `ui_locales` を読み、`en` を返す。

つまり具体例の言う場面では `en` が出る。既存テスト `resolveLocale.test.ts` の "uses the first supported ui_locales hint before a saved or browser locale" は、この順序を明示的に固定している。

## Scope

- 次のどちらが正かを決める。**決めるまでテストを書かない。** どちらを選んでも、いま台帳に残っている `EX-SYSTEM-008-02` を消化できる。
  - 解決順を「保存済みの明示選択 > `ui_locales` ヒント」に変える。用語集の記述と `EX-SYSTEM-008-02` はそのまま通る。OP が `ui_locales` で指定した言語を RP が上書きできなくなる影響を評価する。
  - シナリオを変える。`ui_locales` は RP の意図として明示選択より優先する、と規範側を書き直す。用語集の `DisplayLanguage` の記述も併せて直す。
- 決めた側に応じて、`resolveLocale.ts` または `docs/domain/system/scenarios.feature.md` と用語集を変える。
- `EX-SYSTEM-008-02` を名指すテストを書き、`tools/check/example-coverage-debt.json` から外す。

## Out of Scope

- `EX-SYSTEM-003-01`、`004-01`、`005-01`、`005-02`、`008-01`。[[wi-541-back-system-examples-with-tests]] が消化済みである。
- 明示選択の保存先を `localStorage` から変えること。保存先はこの食い違いに関係しない。

## Design

**保存済みの明示選択を `ui_locales` ヒントより優先する。** 解決順は「保存済み設定 > `ui_locales` ヒント > ブラウザー言語 > 起動時の既定」になる。

決めた理由は次のとおりである。

- OpenID Connect Core 1.0 は `ui_locales` を、利用者が希望する言語についての任意のヒントと定め、OP が従わなくてもよいとしている。同じ利用者が OP の画面で自分で選んだ言語は、RP による推定より強い根拠である。
- `idmagic.displayLocale` へ書くのは言語切り替え UI の `setLocale` だけなので、保存済み設定は常に明示選択である。ブラウザー言語のような推定値が紛れ込んで RP の指定を覆すことはない。
- 用語集の `DisplayLanguage` は「以後のアクセスでは保存済みの設定を優先する」と述べ、`EX-SYSTEM-008-02` もこれに従う。規範を書き換える必要がない。

失うものは、言語を一度でも明示選択した利用者に対して RP が表示言語を強制する手段である。`ui_locales` を読むのは frontend の `readInitialLocale` だけで、backend は discovery の `ui_locales_supported` を広告するほかは値を扱わない。CIBA と SAML の経路は画面を開かないか `ui_locales` を運ばないので、影響を受けるのは認可リクエストからログイン画面を開く経路だけである。

変更するのは純粋関数 `resolveLocale(input: LocaleResolutionInput, startupDefault?: Locale): Locale` の候補の並びだけであり、入力型と作用の境界 (`readInitialLocale` が URL、`localStorage`、`navigator.languages` を読む) は変わらない。

採用しない案は次のとおりである。

| 案 | 採用しない理由 |
| --- | --- |
| シナリオと用語集を書き換え、`ui_locales` を優先する | 利用者自身の明示選択が RP の推定に負け、認可リクエストのたびに言語を選び直すことになる |
| 明示選択と保存済み設定を別の保存先へ分ける | 保存済み設定はすでに明示選択だけから作られており、区別する値がない |

## Tasks

- [x] T001 [Acceptance] `LanguageSwitcher.test.tsx` の `keeps an explicit Japanese choice when an authorization request hints English` の RED を `mise run test-ui-unit-file -- src/components/LanguageSwitcher.test.tsx` で確認する (`EX-SYSTEM-008-02`)。
- [x] T002 [Domain] `resolveLocale.test.ts` の `prefers the saved explicit choice over a supported ui_locales hint` の RED を `mise run test-ui-unit-file -- src/lib/i18n/resolveLocale.test.ts` で確認し、解決順を変えて GREEN にする。`EX-SYSTEM-008-01` のテストから、前提に反する保存済み設定を外す。
- [x] T003 [Verify] 台帳から `EX-SYSTEM-008-02` を外し、フォールト注入、`mise run check-spec`、`mise run verify`、`mise run test-ui-e2e` を通す。

## Verification

- `mise run check-spec` が `EX-SYSTEM-008-02` を台帳から外した状態で通る。
- `mise run test-ui-unit-file -- src/lib/i18n/resolveLocale.test.ts`
- `mise run verify`

## Risk Notes

- **実装に合わせてシナリオを弱める。** どちらを正とするかは利用者に見える振る舞いの選択であり、「テストが書きやすいほう」で決めてはならない。決めた理由を記録へ書く。
- **解決順を変えて、`ui_locales` を指定する RP の意図を壊す。** OP が RP の指定を無視する方向の変更なので、`ui_locales` を送っている経路 (認可リクエスト、CIBA、SAML) への影響を先に数える。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の差分なしを報告した。規範は動かさず、`EX-SYSTEM-008-02` と用語集の `DisplayLanguage` が述べる優先順位に実装を揃えた。
  `resolveLocale` の解決順を「保存済みの明示選択 > `ui_locales` ヒント > ブラウザー言語 > 起動時の既定」に変えた。`ui_locales` は OP が従わなくてもよい RP 側の推定であり、利用者が OP の画面で自分で選んだ言語より弱いと判断した。保存済み設定へ書くのは言語切り替え UI だけなので、推定値が RP の指定を覆すことはない。
  `EX-SYSTEM-008-01` のテストは前提に反する保存済み設定を与えていたので外した。
  `tools/check/example-coverage-debt.json` から `EX-SYSTEM-008-02` を外した。
- **Primary Use Case Evidence**:
  - id: explicit-choice-survives-ui-locales-hint
    unit_red: 解決順を変える前、`prefers the saved explicit choice over a supported ui_locales hint` が 期待値 `ja` に対して `en` を受け取って失敗した。
    e2e_red: 解決順を変える前、`keeps an explicit Japanese choice when an authorization request hints English` が言語切り替え UI の `aria-label` で ja 辞書の文言を期待して en 辞書の文言を受け取り、失敗した。
    unit_fault_injection: 候補の並びで `ui_locales` ヒントを保存済み設定の前へ戻すと、単体テストが同じ差分で失敗する (上の RED と同じ実装)。
    e2e_fault_injection: 言語切り替えの `setLocale` から `writeSavedLocale` を外すと、受け入れテストと既存の 2 件が失敗した。`readInitialLocale` が保存済み設定を読まず `saved` へ `null` を渡すと、受け入れテストだけが失敗した。
- **Acceptance RED Evidence**:
  - **Test**: `frontend/src/components/LanguageSwitcher.test.tsx` の `keeps an explicit Japanese choice when an authorization request hints English`
  - **Requirement**: REQ-SYSTEM-008
  - **Observed Failure**: `EX-SYSTEM-008-02` の場面として、言語切り替え UI で `ja` を選んだ後、`?client_id=web-app&ui_locales=en` の URL で `LocaleProvider` を描き直すと `en` 辞書の文言が出た。
  - **Detection Reason**: 最初の木を捨ててから URL を差し替えて描き直すので、同じページの寿命の中で再解決が起きないことに頼る実装とは区別できる。テスト環境の `window.location` は静的なスナップショットなので、`history.replaceState` ではなく `stubGlobal('location', …)` で URL を与えた。`history.replaceState` を使った最初の版は URL が届かず、修正前でも通ってしまった。
- **Change-Resistance Results**:
  変更は TypeScript だけで、`mise run test-go-mutation` は Go を対象とするので適用しない。解決順の入れ替え、保存の除去、読み直しの除去は上の fault injection で手作業により確かめた。
- **Verification Results**:
  - `mise run check-work-items` - 成功
  - `mise run check-spec` - 成功
  - `mise run test-ui-unit-file -- src/lib/i18n/resolveLocale.test.ts` - 成功 (7 件)
  - `mise run test-ui-unit-file -- src/components/LanguageSwitcher.test.tsx` - 成功 (3 件)
  - `mise run test-ui-unit-file -- src/lib/i18n/context.test.tsx` - 成功 (2 件)
  - `mise run verify` - 成功 (初回は UI の整形、release note の表記、既存の `wi-573` の相対リンク切れで失敗した。`done/` へ移した後はこの記録の相対リンクと完了欄の書式で失敗した。それぞれを直した後に成功)
  - `mise run test-ui-e2e` - 成功 (38 件)
