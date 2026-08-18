---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-327-application-context-problem-details-migration, wi-328-audit-context-problem-details-migration, wi-329-authentication-context-problem-details-migration, wi-330-idgovernance-context-problem-details-migration, wi-331-idmanagement-context-problem-details-migration, wi-332-oauth2-context-problem-details-migration, wi-333-provisioning-context-problem-details-migration, wi-334-saml-context-problem-details-migration, wi-335-sharedsignals-context-problem-details-migration, wi-336-signingkeys-context-problem-details-migration, wi-337-tenancy-context-problem-details-migration, wi-338-workloadidentity-context-problem-details-migration, wi-339-wsfederation-context-problem-details-migration]
initial_context:
  specification: [spec/SPECIFICATION.md]
  source: [backend/shared/http/support_http/response.go, backend/shared/http/support_http/problem.go]
  tests: [backend/shared/http/support_http]
  stop_before_reading: [frontend]
---

# `WriteBrowserError` を削除し Problem Details 移行完了を spec/SPECIFICATION.md に記録する

## Motivation

`wi-326` とそのコンテキスト単位の後続 work item (`wi-327`〜`wi-339`) が全
コンテキストの `WriteBrowserError` 呼び出しを `WriteProblem` (RFC 9457
Problem Details、ADR-154) へ移行し終えたら、レガシー `{error, message}`
envelope を返す `WriteBrowserError` 自体を削除し、`spec/SPECIFICATION.md` の
HTTP error responses 節にある「未実装」の記載を実装完了状態に更新する
(`wi-326` Plan 手順 5)。

## Scope

- `backend/shared/http/support_http/response.go` の `WriteBrowserError`
  関数本体の削除。
- `spec/SPECIFICATION.md` の HTTP error responses 節
  (「この規約は未実装」段落) の更新。

## Out of Scope

- 新しい実装の追加 (このコンテキスト単位の移行はすべて先行 work item で完了
  済みが前提)。

## Design

- `backend/shared/http/support_http/api_error_language_test.go` の
  `errorTextArgument` マップから `"WriteBrowserError"` エントリを削除する。
- `grep -rn "WriteBrowserError" backend --include="*.go"` が 0 件になっている
  ことを削除前に確認する。

## Plan

1. `grep -rn "WriteBrowserError" backend --include="*.go"` で残存呼び出しが
   ないことを確認する (0 件でなければ該当 work item が未完了なので先に完了
   させる)。
2. `WriteBrowserError` を `response.go` から削除する。
3. `api_error_language_test.go` の tracking map から該当エントリを削除する。
4. `spec/SPECIFICATION.md` の HTTP error responses 節を更新する。
5. `just verify` を通す。

## Tasks

- [x] T001 [Verify] 残存 `WriteBrowserError` 呼び出しが 0 件であることを
      確認する (`grep -rn "WriteBrowserError" backend --include="*.go"` の
      残りは定義本体・コメント・tracking map・テストコメントのみ)。
- [x] T002 [App] `WriteBrowserError` を削除する。
      `api_error_language_test.go` の tracking map、`WriteRateLimited` と
      apitoken テストのコメント参照もあわせて落とした。`wi-329` で
      過渡的に両 envelope を読ませていた `account_step_up_handler_test.go` の
      `errorCode` も Problem Details だけを読む形に戻した。
- [x] T003 [Spec] `spec/SPECIFICATION.md` の HTTP error responses 節を実装完了
      状態に更新する。
- [x] T004 [Verify] `just verify` を通す。

## Verification

- `just verify-go`
- `just verify`

## Risk Notes

先行 13 work item の完了が前提。1 つでも未完了のまま削除すると、その
コンテキストのビルドが壊れる。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  レガシー `{error, message}` envelope を返す `WriteBrowserError` を
  `backend/shared/http/support_http/response.go` から削除した。
  `wi-327`〜`wi-339` で 14 コンテキスト・約 400 箇所の呼び出しがすべて
  `WriteProblem` へ移行済みで、削除時点の残存参照は定義本体とコメントだけ
  だった。`api_error_language_test.go` の tracking map からもエントリを外し、
  英語リテラル検査の対象は `WriteProblem` 側で引き継がれる。

  `spec/SPECIFICATION.md` の HTTP error responses 節は、起票時に想定していた
  「未実装」段落が既に存在しなかった (`ARCHITECTURE.md` から
  `SPECIFICATION.md` へ移る際に現在形へ書き直されていた) ため、代わりに
  実装完了で確定した内容を反映した: Problem Details を適用しない例外の一覧に
  SharedSignals の受信エンドポイント (RFC 8935、`/ssf/streams/{stream_id}/events`)
  を加え、境界がパッケージ単位ではなく接点単位で引かれることを明記した。
  これは `wi-335` で receiver 用のエラー経路を分離した結果、コード上も
  はっきりした事実である。

  これで汎用 API のエラーレスポンスは全コンテキストで
  `application/problem+json` (`type`/`title`/`status`/`detail`/`instance`) に
  統一され、標準がエラー形式を定める接点 (OAuth2 / SCIM / DCR /
  SharedSignals 受信エンドポイント) だけが例外として残る。

  **対応していないこと**:
  - `WriteRateLimited` (429) は `{error, retry_after_seconds, message}` の
    まま。`retry_after_seconds` を運ぶ独自形状で、Problem Details 化するなら
    拡張メンバーの扱いを決める必要があるため、この work item の
    「レガシーヘルパーの削除」とは別種の変更として残す。
  - SharedSignals 受信エンドポイントの body が RFC 8935 §2.3 の
    `{err, description}` と一致していない既知の乖離 (`wi-335` Risk Notes) は
    未対応のまま。
  - 移行中に見つかった契約の欠落 (自己サービスの属性更新エンドポイントと
    `/api/auth/mfa/enrollment/totp/*` が TypeSpec に未宣言、
    export の `quota_exceeded` が `QuotaExceededError` と別概念) は
    `wi-331`/`wi-332` の Completion に記録した。`wi-382` の領域。
- **Verification Results**:
  - `just test-go` - passed (全パッケージ)
  - `just lint-go` - 0 issues
  - `just check-spec` - passed (25 document / 327 operation / 797 TypeSpec symbol)
  - `just check-api-compat` - no breaking changes vs baseline
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
    (変更は Design 節の散文で、REQ / TypeSpec symbol の増減はない)
