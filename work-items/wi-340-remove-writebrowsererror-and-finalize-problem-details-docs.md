---
status: pending
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-327-application-context-problem-details-migration, wi-328-audit-context-problem-details-migration, wi-329-authentication-context-problem-details-migration, wi-330-idgovernance-context-problem-details-migration, wi-331-idmanagement-context-problem-details-migration, wi-332-oauth2-context-problem-details-migration, wi-333-provisioning-context-problem-details-migration, wi-334-saml-context-problem-details-migration, wi-335-sharedsignals-context-problem-details-migration, wi-336-signingkeys-context-problem-details-migration, wi-337-tenancy-context-problem-details-migration, wi-338-workloadidentity-context-problem-details-migration, wi-339-wsfederation-context-problem-details-migration]
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

- [ ] T001 [Verify] 残存 `WriteBrowserError` 呼び出しが 0 件であることを
      確認する。
- [ ] T002 [App] `WriteBrowserError` を削除する。
- [ ] T003 [Docs] `spec/SPECIFICATION.md` の HTTP error responses 節を実装完了
      状態に更新する。
- [ ] T004 [Verify] `just verify` を通す。

## Verification

- `just verify-go`
- `just verify`

## Risk Notes

先行 13 work item の完了が前提。1 つでも未完了のまま削除すると、その
コンテキストのビルドが壊れる。
