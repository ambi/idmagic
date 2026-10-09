---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: maintenance
affected_spec:
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.InvalidRequestError }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.InsufficientScopeError }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.InvalidOriginError }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.CsrfFailedError }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.StepUpRequiredError }
  - { path: spec/contexts/sharedsignals/models.tsp, symbol: IdMagic.Contract.UnlinkDeniedError }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.EmailSent }
---

# 複数の Context が使うエラーとイベントを、それを所有する場所で宣言する

## 動機

`tools/check/unspecified-vocabulary-debt.json` には、どの要件にも書けない語が残っている。
このうち次の 7 件は、宣言の置き場所が所有者と食い違っていることが原因である。

| 語 | 宣言の置き場所 | 実際に返す、または発行する Context |
| --- | --- | --- |
| `invalid_request`、`insufficient_scope`、`invalid_origin`、`csrf_failed` | `spec/contexts/sharedsignals/models.tsp` | 管理 API とブラウザーから呼ばれるすべての Context。`support_http` の共通の防護が返す |
| `step_up_required` | 同上 | Authentication、IdManagement、OAuth2 |
| `unlink_denied` | 同上 | Authentication の連携の解除 |
| `EmailSent` | TypeSpec は `spec/contexts/authentication/models.tsp`、Go の型は `backend/shared/spec/events.go` | Authentication（パスワードのリセット）と IdManagement（メールアドレスの変更） |

SharedSignals の TypeSpec が、Shared Signals と関係のない共通のエラーを宣言している。
語彙の検査は TypeSpec の置き場所で Context を決めるので、SharedSignals の要件に書けない語として残る。
`EmailSent` は、Go の型が `backend/shared/spec` にあるため、検査が System の語として扱う。

## 対象範囲

- 共通のエラー 4 件（`invalid_request`、`insufficient_scope`、`invalid_origin`、`csrf_failed`）を、Context に属さない共通の TypeSpec の置き場所へ移す。
  語彙の検査は、共通の置き場所の語を、Context の要件ではなく横断的な要件（`docs/domain/scenarios.feature.md` または `docs/domain/standards.md`）で照合する。
- `step_up_required` と `unlink_denied` を、返す Context の TypeSpec へ移し、その Context の要件に返す条件を書く。
  複数の Context が返す `step_up_required` は、共通の置き場所へ移すか、所有する一つの Context を決めるかを着手時に決める。
- `EmailSent` を、発行する Context の要件で扱えるようにする。
  Go の型を `backend/shared/spec` に残す場合は、語彙の検査がイベントの所有者を TypeSpec の宣言元で決めるように変える。
- `tools/check/unspecified-vocabulary-debt.json` の `sharedsignals` と `system` の項目を空にする。

## 対象外

- エラーコードやイベントの意味の変更。
  移すのは宣言の置き場所であり、名前空間 `IdMagic.Contract` は変えないので、OpenAPI のスキーマ名は変わらない。

## 設計

`spec/contexts/sharedsignals/models.tsp` の冒頭の 4 件は、ほかの Context が定めるエラーを公表された言語として写したものであり、OAuth2、Authentication、Tenancy の共有のエラーも含む。
SharedSignals に置かれた経緯の調査は要らない。
移した後の置き場所が「どの Context の要件で照合するか」を決めるので、置き場所と語彙の検査の対応表（`tools/check/src/check-unspecified-vocabulary.ts` の `CONTEXTS`）を同じ変更で直す。

## タスク

- [ ] T001 [Contract] 共通のエラーを共通の置き場所へ、Context 固有のエラーを返す Context へ移す。
- [ ] T002 [Spec] 返す条件、発行する条件を要件に書く。
- [ ] T003 [Tooling] 語彙の検査の対応表と、許容リストを更新する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-unspecified-vocabulary`
- `mise run check-api-compat`
- `mise run verify`

## リスク

- TypeSpec のファイルを移すと、生成した OpenAPI の並び順が変わり、差分が大きく見える。
  `mise run check-api-compat` で、スキーマの意味が変わっていないことを確かめる。
