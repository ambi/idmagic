---
status: completed
---

# 複数のモジュールが使うエラーとイベントを、それを所有する場所で宣言する

SharedSignals の TypeSpec に置かれていた共通のエラー 6 件を、返す条件を要件に持つ所有者へ移した。
`InvalidRequestError`、`InvalidOriginError`、`CsrfFailedError` は System、`InsufficientScopeError` は ApiTokens、`StepUpRequiredError` と `UnlinkDeniedError` は Authentication が宣言する。
名前空間は変えないので、OpenAPI の差は `InvalidRequestError` の説明文から「OAuth2 モジュールが定める」の一文を除いたことだけである。
`EmailSent` は Go の型を `backend/shared/spec` に残し、発行する条件を REQ-AUTHENTICATION-016 と REQ-IDMANAGEMENT-053 に書いた。
語彙の検査は、共有の Go の木のイベントを、同名のモデルを TypeSpec で宣言するモジュールの要件で照合し、System の語を `docs/modules/scenarios.feature.md` の Rule でも照合する。
許容リストの `sharedsignals` と `system` は空になった。
`EmailSent` の具象の型が `backend/shared/spec` にあることは、具象のイベントをモジュールの `domain` に置く規則と食い違ったまま残る。

全文は `git log -1 -- work-items/active/wi-44063-declare-shared-vocabulary-under-its-owner.md` が示すコミットの親、またはそのコミットで読む。
