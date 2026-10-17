---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: maintenance
affected_spec:
  - { path: spec/modules/system/models.tsp, symbol: IdMagic.Contract.InvalidRequestError }
  - { path: spec/modules/api-tokens/models.tsp, symbol: IdMagic.Contract.InsufficientScopeError }
  - { path: spec/modules/system/models.tsp, symbol: IdMagic.Contract.InvalidOriginError }
  - { path: spec/modules/system/models.tsp, symbol: IdMagic.Contract.CsrfFailedError }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.StepUpRequiredError }
  - { path: spec/modules/authentication/models.tsp, symbol: IdMagic.Contract.UnlinkDeniedError }
  - { path: docs/modules/authentication/password/README.md, requirement: REQ-AUTHENTICATION-016 }
  - { path: docs/modules/identity-management/account/README.md, requirement: REQ-IDMANAGEMENT-053 }
---

# 複数のモジュールが使うエラーとイベントを、それを所有する場所で宣言する

## 動機

`tools/check/unspecified-vocabulary-debt.json` には、どの要件にも書けない語が残っている。
このうち次の 7 件は、宣言の置き場所が所有者と食い違っていることが原因である。

| 語 | 宣言の置き場所 | 実際に返す、または発行するモジュール |
| --- | --- | --- |
| `invalid_request`、`insufficient_scope`、`invalid_origin`、`csrf_failed` | `spec/modules/sharedsignals/models.tsp` | 管理 API とブラウザーから呼ばれるすべてのモジュール。`support_http` の共通の防護が返す |
| `step_up_required` | 同上 | Authentication、IdManagement、OAuth2 |
| `unlink_denied` | 同上 | Authentication の連携の解除 |
| `EmailSent` | TypeSpec は `spec/modules/authentication/models.tsp`、Go の型は `backend/shared/spec/events.go` | Authentication（パスワードのリセット）と IdManagement（メールアドレスの変更） |

SharedSignals の TypeSpec が、Shared Signals と関係のない共通のエラーを宣言している。
語彙の検査は TypeSpec の置き場所でモジュールを決めるので、SharedSignals の要件に書けない語として残る。
`EmailSent` は、Go の型が `backend/shared/spec` にあるため、検査が System の語として扱う。

## 対象範囲

- 共通のエラー 4 件（`invalid_request`、`insufficient_scope`、`invalid_origin`、`csrf_failed`）を、モジュールに属さない共通の TypeSpec の置き場所へ移す。
  語彙の検査は、共通の置き場所の語を、モジュールの要件ではなく横断的な要件（`docs/modules/scenarios.feature.md` または `docs/requirements/standards.md`）で照合する。
- `step_up_required` と `unlink_denied` を、返すモジュールの TypeSpec へ移し、そのモジュールの要件に返す条件を書く。
  複数のモジュールが返す `step_up_required` は、共通の置き場所へ移すか、所有する一つのモジュールを決めるかを着手時に決める。
- `EmailSent` を、発行するモジュールの要件で扱えるようにする。
  Go の型を `backend/shared/spec` に残す場合は、語彙の検査がイベントの所有者を TypeSpec の宣言元で決めるように変える。
- `tools/check/unspecified-vocabulary-debt.json` の `sharedsignals` と `system` の項目を空にする。

## 対象外

- エラーコードやイベントの意味の変更。
  移すのは宣言の置き場所であり、名前空間 `IdMagic.Contract` は変えないので、OpenAPI のスキーマ名は変わらない。

## 設計

`spec/modules/sharedsignals/models.tsp` の冒頭の 4 件は、ほかのモジュールが定めるエラーを公開契約として写したものであり、OAuth2、Authentication、Tenancy の共有のエラーも含む。
SharedSignals に置かれた経緯の調査は要らない。
移した後の置き場所が「どのモジュールの要件で照合するか」を決めるので、置き場所と語彙の検査の対応表（`tools/check/src/check-unspecified-vocabulary.ts` の `MODULES`）を同じ変更で直す。

着手時に決めた置き場所は次のとおりである。

| 語 | 移す先 | 照合する要件 | 理由 |
| --- | --- | --- | --- |
| `invalid_request` | `spec/modules/system/models.tsp` | `docs/modules/system/` の要件（REQ-SYSTEM-024、REQ-SYSTEM-025） | System は共通の `ProblemDetails` と `AuthenticationRequiredResponse` を既に宣言しており、どのモジュールにも属さない共通の置き場所である。新しい置き場所と照合先を作るより、既存のモジュールの対応で足りる |
| `invalid_origin`、`csrf_failed` | 同上 | `docs/modules/scenarios.feature.md` の REQ-PLATFORM-004 | 各モジュールの経路へ同じ形で適用する保証として、横断的なシナリオが一度だけ定める。System へ同じ要件を写すと二重になるので、語彙の検査が System の語を横断的なシナリオの Rule でも照合する |
| `insufficient_scope` | `spec/modules/api-tokens/models.tsp` | ApiTokens の要件（REQ-APITOKENS-004） | API アクセストークンのスコープの語彙と、不足したときの拒否を ApiTokens が定める |
| `step_up_required` | `spec/modules/authentication/models.tsp` | Authentication の要件 | 再認証の期限の判定（`mfadomain.StepUpSatisfied`、`ErrStepUpRequired`）を Authentication が所有し、IdManagement と OAuth2 はそれを使う |
| `unlink_denied` | 同上 | Authentication の連携の解除の要件 | 返すのは Authentication だけである |
| `EmailSent` | Go の型は `backend/shared/spec` に残す | Authentication の要件 | Go の型を Authentication へ移すと、IdManagement からの依存と共有の通知の型の所有者が変わる。語彙の検査が、`backend/shared/spec` のイベントを、同名のモデルを宣言する TypeSpec のモジュールに帰属させれば、境界を変えずに済む |

`AccessDeniedError` と `QuotaExceededError` も SharedSignals に置かれているが、SharedSignals の要件に現れるので負債ではなく、今回は移さない。

着手時の証拠は、語彙の検査の RED（移した後に要件がまだない語の `fresh`、許容リストに残った語の `stale`）と、イベントの帰属の単体テストの RED とする。

## タスク

- [x] T001 [Contract] 共通のエラーを共通の置き場所へ、モジュール固有のエラーを返すモジュールへ移す。
  `invalid_request`、`invalid_origin`、`csrf_failed` は System、`insufficient_scope` は ApiTokens、`step_up_required` と `unlink_denied` は Authentication へ移した。
- [x] T002 [Spec] 返す条件、発行する条件を要件に書く。
  エラーは移した先の既存の要件が返す条件を述べている。`EmailSent` を発行する条件を REQ-AUTHENTICATION-016 と REQ-IDMANAGEMENT-053 に足した。
- [x] T003 [Tooling] 語彙の検査の対応表と、許容リストを更新する。
  単体テストは `ruleTexts`、`collectModelNames`、`sharedEventOwners` を対象とし、`mise run test-tools-file -- check/src/unspecified-vocabulary.test.ts` で実行する。
- [x] T004 [Verify] 変更を検証する。

## 検証

Go と UI のコードを変えないので、`verify` ではなく、触れた契約、要件、ツールを担当する検査を選ぶ。

- `mise run check-unspecified-vocabulary`：移した語と許容リストの照合。
- `mise run check-spec`、`mise run check-generated-contract`：TypeSpec の移動と要件の形式。
- `mise run check-api-compat`：基準ファイルとの比較。今回の変更より前からある差分で失敗するので、HEAD と作業ツリーの TypeSpec から生成した OpenAPI をキーで整列して比べ、差が `InvalidRequestError` の説明文だけであることを確かめる。
- `mise run check-published-api-compat`：公開済みの契約がないことの確認。
- `mise run test-tools`、`mise run typecheck-tools`、`mise run lint-tools`：語彙の検査のコード。
- `mise run check -- main`、`mise run check-work-items`、`mise run check-links`：記録、仕様影響、リンク。

## リスク

- TypeSpec のファイルを移すと、生成した OpenAPI の並び順が変わり、差分が大きく見える。
  `mise run check-api-compat` で、スキーマの意味が変わっていないことを確かめる。

## 要約

SharedSignals の TypeSpec に置かれていた共通のエラー 6 件を、返す条件を要件に持つ所有者へ移した。
`InvalidRequestError`、`InvalidOriginError`、`CsrfFailedError` は System、`InsufficientScopeError` は ApiTokens、`StepUpRequiredError` と `UnlinkDeniedError` は Authentication が宣言する。
名前空間は変えないので、OpenAPI の差は `InvalidRequestError` の説明文から「OAuth2 モジュールが定める」の一文を除いたことだけである。
`EmailSent` は Go の型を `backend/shared/spec` に残し、発行する条件を REQ-AUTHENTICATION-016 と REQ-IDMANAGEMENT-053 に書いた。
語彙の検査は、共有の Go の木のイベントを、同名のモデルを TypeSpec で宣言するモジュールの要件で照合し、System の語を `docs/modules/scenarios.feature.md` の Rule でも照合する。
許容リストの `sharedsignals` と `system` は空になった。
`EmailSent` の具象の型が `backend/shared/spec` にあることは、具象のイベントをモジュールの `domain` に置く規則と食い違ったまま残る。
