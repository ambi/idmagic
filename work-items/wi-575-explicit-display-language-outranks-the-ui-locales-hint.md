---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/system/scenarios.feature.md, requirement: REQ-SYSTEM-008 }
---

# 明示選択した表示言語が `ui_locales` ヒントに負ける

## Motivation

[[wi-541-back-system-examples-with-tests]] が `EX-SYSTEM-008-02` を消化しようとして、シナリオと実装が食い違っていることを測った。

`EX-SYSTEM-008-02` は「表示言語を `ja` と明示選択済みの EndUser へ `ui_locales=en` の認可リクエストが来ても、ログイン画面は `ja` 辞書で表示される」と述べる。[System の用語集](../docs/domain/system/glossary.md)の `DisplayLanguage` も「選択はブラウザーに保存し、以後のアクセスでは保存済みの設定を優先する」と述べている。

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

## Verification

- `mise run check-spec` が `EX-SYSTEM-008-02` を台帳から外した状態で通る。
- `mise run test-ui-unit-file -- src/lib/i18n/resolveLocale.test.ts`
- `mise run verify`

## Risk Notes

- **実装に合わせてシナリオを弱める。** どちらを正とするかは利用者に見える振る舞いの選択であり、「テストが書きやすいほう」で決めてはならない。決めた理由を記録へ書く。
- **解決順を変えて、`ui_locales` を指定する RP の意図を壊す。** OP が RP の指定を無視する方向の変更なので、`ui_locales` を送っている経路 (認可リクエスト、CIBA、SAML) への影響を先に数える。
