---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-19
change_kind: bugfix
priority: p2
depends_on: []
evidence_policy: risk-based-v2
initial_context:
  specification: [docs/api-rules.md]
  typespec:
    - IdMagic.IdManagement.Operations.UpdateUserProfile
    - IdMagic.Contract.ActiveJobQuotaExceededError
  source:
    - backend/idmanagement/user/handlers_http/account_handler.go
    - backend/idmanagement/handlers_http/admin_data_export_handler.go
    - backend/shared/http/support_http/error_handler.go
  tests:
    - backend/idmanagement/user/handlers_http
    - backend/idmanagement/handlers_http
  stop_before_reading: [backend/oauth2, backend/sourcing]
affected_spec:
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateUserProfile }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.ActiveJobQuotaExceededError }
---

# 同じ error code が接点によって別の意味を持つ 2 件を実装側で解消する

## Motivation

`wi-382` は契約を実装に合わせる作業だったが、T009 で「同じ code を別 status で返している 3 件」の判断を求められ、そのうち 2 件は実装側の欠陥だと結論した。契約は正しい側を書いてあるので、いま実装と契約がずれているのはこの 2 点である。

**`invalid_attribute` の status。** `PATCH /api/account/v1/profile` は属性スキーマ違反を 400 で返す (`backend/idmanagement/user/handlers_http/account_handler.go`)。同じ違反を `UpdateAdminUser` は 422 で返す。`docs/api-rules.md` の HTTP error responses は 400 を「リクエストを解析できないこと」、422 を「解析できた内容が業務規則に違反すること」と定める。テナントの属性スキーマへの適合は後者なので、422 が正しい。`wi-382` は契約側に 422 を書いた。

**`quota_exceeded` の code。** export の開始 (`handleStartExport`) は「実行中ジョブ数の上限」に達したとき 429 で `quota_exceeded` を返し、共通のエラーハンドラー (`backend/shared/http/support_http/error_handler.go`) は「テナント資源クォータ」に達したとき 422 で同じ `quota_exceeded` を返す。status は両方とも正しい — 前者は待てば通る一時的な状態 (RFC 6585)、後者は解析できた内容の業務規則違反である。欠陥は 1 つの code が 2 つの別概念を指していることで、クライアントは code だけでは再試行すべきかを判断できない。`wi-382` は契約側に `ActiveJobQuotaExceededError` を分けて置き、`type` がいまは同じ URN であることを `@doc` に明記した。

## Scope

- `writeAccountError` の `ErrInvalidAttribute` 分岐を 400 から 422 へ変える。frontend が 400 に依存していないことを確認する。
- 実行中ジョブ数の上限に固有の error code を与え (`active_job_quota_exceeded` 等)、`ActiveJobQuotaExceededError` の `@doc` から重複の注記を外す。
- 変更後の code と status を、対応する `REQ-` シナリオと Go のテストで固定する。

## Out of Scope

- `mfa_enrollment_not_allowed` の 403 と 422。`wi-382` はこれを「別の条件が同じ code を共有している」と判断し、status は双方正しいと結論した。ブラウザー経路の 403 が `ErrMfaAlreadyEnrolled` まで畳んでいる点だけが別の欠陥で、それは `wi-384` が扱う。
- 契約 (`spec/**/*.tsp`) の構造の変更。`wi-382` で既に正しい側を書いてある。`ActiveJobQuotaExceededError` の `@doc` から重複の注記を外すのは Scope に含む (`type` の URN を述べる文なので、記述の訂正であって宣言の変更ではない)。
- `writeDataExportError` が `active_jobs` 以外の資源のクォータを扱うこと。この接点のクォータ検査は `Enqueue` 経由の `active_jobs` だけで、他の資源はここからは起きない。返らない応答を契約へ足すのは [[wi-397-token-endpoint-declares-a-403-it-never-returns]] が直した誤りの繰り返しになるので、他の資源は共通のエラーハンドラー (422 `quota_exceeded` と指標の記録を持つ) へ委ねるに留める。

## Design

**`invalid_attribute` は 422。** `docs/api-rules.md` は 400 を「リクエストを解析できないこと」、422 を「解析できた内容が業務規則に違反すること」と定める。テナントの属性スキーマへの適合は後者で、同じ違反を `UpdateAdminUser` は既に 422 で返している。契約 (`UpdateUserProfileError422` / `InvalidUserAttributeError`) も 422 を書いているので、実装だけが取り残されていた。`writeAccountError` の 1 分岐を変える。

frontend の確認: `response.status === 400` を見ている分岐は無い。HTTP の状態コードで分岐している箇所は frontend 全体に存在せず、`invalid_attribute` の唯一の参照は Group CSV 取り込みの行単位エラー code の写像で、こちらは code だけを見ていてこの接点とは無関係である。Risk Notes が先に確かめるよう求めた点は満たされている。

**`quota_exceeded` は資源で分ける。** 実行中ジョブ数の上限と、テナント資源クォータは、どちらも `*tenancydomain.QuotaExceededError` として上がってくる。両者を隔てているのは `Resource` フィールドだけである。status はどちらも正しい —— 前者は待てば通る一時的な状態 (RFC 6585 の 429)、後者は解析できた内容の業務規則違反 (422) —— ので、欠陥は 1 つの code が 2 つの概念を指していることにある。クライアントは code だけでは再試行すべきかを判断できない。

`Resource == ResourceActiveJobs` のときに `active_job_quota_exceeded` を返す。他の資源はこの接点からは起きないので、`return err` で共通のエラーハンドラーへ委ねる。そちらが 422 `quota_exceeded` と `metrics.RecordQuotaExceeded` を持っている。

検討した代替案:

- **`writeDataExportError` で他の資源も 422 `quota_exceeded` として直接書く**: 共通のエラーハンドラーが持つ指標の記録を取りこぼす。また、この接点から起きない応答を宣言する動機が生まれる。採用しない。
- **上限の側の status も 422 に揃える**: 待てば通る状態を「解析できた内容の業務規則違反」と呼ぶことになり、`Retry-After` の意味も失う。`wi-382` の判断どおり status は双方正しい。採用しない。

## Verification

- `mise run verify`
- 手動確認: `PATCH /api/account/v1/profile` に属性スキーマ違反を送ると 422 と `urn:idmagic:error:invalid_attribute` が返る。
- 手動確認: 実行中ジョブ数の上限に達した export 開始が、テナント資源クォータとは別の code を 429 で返す。

## Risk Notes

400 から 422 への変更は、`response.status === 400` を見ている frontend の分岐を壊しうる。i18n 辞書は code で引いているので文面は影響を受けないが、分岐の有無を先に確認する。code の分割は、いまの `quota_exceeded` を見ているクライアントがあれば影響する。実在するのは自リポジトリの frontend だけなので、同一コミットで揃える。

## Tasks

- [x] T001 [Adapters] `writeAccountError` の `ErrInvalidAttribute` 分岐を 400 から 422 へ変えた。
- [x] T002 [Adapters] `writeDataExportError` で `active_jobs` の上限に `active_job_quota_exceeded` を与え、
  他の資源は共通のエラーハンドラーへ委ねるようにした。
- [x] T003 [Spec] `ActiveJobQuotaExceededError` の `@doc` から重複の注記を外し、`type` を新しい URN にした。
- [x] T004 [Verify] `mise run verify`。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範は動いておらず、契約 (`wi-382` が既に正しい側を書いていた) に実装が追いついた変更である。`PATCH /api/account/v1/profile` の属性スキーマ違反が 400 から 422 になり、`UpdateAdminUser` と `UpdateUserProfileError422` の宣言に揃った。export 開始が実行中ジョブ数の上限で拒否されるとき、テナント資源クォータと共有していた `quota_exceeded` をやめて `active_job_quota_exceeded` を返すようになった。status (429) は変えていない。この 2 つは「待てば通る一時的な状態」と「解析できた内容の業務規則違反」で再試行の判断が逆になるので、code が分かれて初めてクライアントが区別できる。`ActiveJobQuotaExceededError` の `@doc` から、共有が server 側の欠陥であるという注記を外した。
- **Acceptance RED Evidence**:
  - **Test**: `TestAccountProfilePatchRejectsSchemaViolationAsUnprocessable` (`backend/idmanagement/user/handlers_http/account_handler_test.go`) と `TestDataExportHTTP_ActiveJobCeilingHasItsOwnCode` (`backend/idmanagement/handlers_http/admin_data_export_handler_test.go`)
  - **Requirement**: N/A: どちらも `REQ-` シナリオではなく `docs/api-rules.md` の状態コードの規約と、`wi-382` が書いた TypeSpec の宣言が規範である。
  - **Observed Failure**: 前者が `status=400 body={"type":"urn:idmagic:error:invalid_attribute",...,"status":400}, want 422`。後者が `type="urn:idmagic:error:quota_exceeded", want urn:idmagic:error:active_job_quota_exceeded`。
  - **Detection Reason**: どちらも HTTP の境界で、状態コードと `type` の両方を見る。属性の側は `type` が変わらず status だけが変わるので、status を主張しなければ何も落ちない。クォータの側は逆に status が変わらず code だけが変わるので、code を主張しなければ落ちない。加えて両方とも拒否の効果を状態から読み戻す —— 属性はプロフィールを読み直して `zone` が保存されていないこと、export は実行可能なジョブが 1 件も残っていないことを確かめる。応答だけを見る主張は、拒否を書いてから操作を続行する実装にも通ってしまう。クォータの側はさらに、応答本文に `urn:idmagic:error:quota_exceeded` が現れないことを主張するので、新しい code を足しつつ古い code も併記するような実装は落ちる。
- **Unit RED Evidence**:
  - **Test**: `TestAccountProfileHTTPExtra` (`backend/idmanagement/handlers_http/extra_identity_test.go`)
  - **Requirement**: N/A: 上と同じ理由で、対応する `REQ-` シナリオを持たない。
  - **Observed Failure**: `invalid attr status=422 body={"type":"urn:idmagic:error:invalid_attribute",...}`。この検査は欠陥のあった 400 を固定していたので、修正によって落ちた。400 を期待する主張を 422 へ書き換えた。
  - **Detection Reason**: 同じ検査が、解析そのものに失敗する `invalid-json` を 400 のまま隣で固定している。属性スキーマ違反 (422) と解析失敗 (400) が別の条件であることが 1 つの検査の中で対になっているので、両方を 422 に倒すような実装は落ちる。既存の検査が欠陥を固定していた事実を残すため、書き換えでは理由をコメントに置いた。
- **Change-Resistance Results**:
  代表的な誤実装を 3 つ実測した。
  M1 `invalid_attribute` を 400 へ戻す → `TestAccountProfilePatchRejectsSchemaViolationAsUnprocessable` と `TestAccountProfileHTTPExtra` の 2 パッケージが落ちる。
  M2 `active_job_quota_exceeded` を `quota_exceeded` へ戻す → `TestDataExportHTTP_ActiveJobCeilingHasItsOwnCode` が落ちる。
  M3 `qErr.Resource == tenancydomain.ResourceActiveJobs` の条件を外し、あらゆるクォータ超過を 429 `active_job_quota_exceeded` にする → **検出されなかった**。
  **等価変異と方法の限界**: M3 が生存するのは、この接点のクォータ検査が `Enqueue` 経由の `active_jobs` 1 種しか無く、他の資源のクォータ超過が `writeDataExportError` へ到達しないためである。現在の到達可能性のもとでは M3 は等価変異であり、落とすには実装が生成しえないエラーを検査が組み立てる必要がある。資源による分岐は将来この接点に別のクォータが加わったときに 500 へ落ちるのを防ぐ防御であって、いま観測できる振る舞いではない。検査を足す代わりにこの制約を記録する。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-spec` - ok (148 document(s), 333 operation(s), 845 TypeSpec symbol(s))
  - `mise run check-api-compat` - `no breaking changes` (`type` の URN は schema の description にのみ現れるため非破壊)
  - `mise run lint-go` - 0 issues
  - `mise run spec-diff` - `no normative specification change against main`
