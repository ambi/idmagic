---
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-19
change_kind: bugfix
priority: p2
depends_on: [wi-385-typespec-go-struct-drift-check]
affected_spec:
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.RegisterClient }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.Authorize }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Sourcing.Operations.CreateScimUser }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CompleteFederatedLogin1 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminUser }
---

# operation が宣言するステータスコード集合と handler が実際に返すものを総なめで突き合わせる

## Motivation

`wi-382` は本体の形を直したが、ステータスコード集合は 5 件だけを閉じ、全面的な突き合わせを Out of Scope に置いた。その 5 件を直す過程で、宣言と実装がずれている operation が他にもあることが確かめられている。

- `RegisterClient` は `Success_200` を宣言するが handler は 201 を返す。SCIM の `CreateScimUser` / `CreateScimGroup` も同じく 200 宣言で 201 実装である。
- `/userinfo` は `invalid_token` を 401 として宣言するが、`writeOAuthError` は `invalid_client` と `server_error` 以外をすべて 400 にする。
- `AuthorizeError403` は宣言されているが、`/authorize` の access_denied は redirect_uri へのリダイレクトで返るため到達しない。
- `CompleteFederatedLogin` は 200 を宣言するが handler は 303 リダイレクトしか返さない。`StartFederatedLogin` も同じ。
- `GetAdminUser` は 400 / 403 を宣言するが handler は 404 を返す。
- テナント解決 middleware は routing の手前で 404 `{"error": "tenant_not_found"}` を返す。これは `application/json` の第 4 のエラー本文であり、どの operation も宣言していない。

いずれも 1 件ずつは小さいが、327 operation を総なめしないと全体が読めない。

## Scope

- 全 operation について、宣言するステータスコード集合と handler が到達しうるコードを突き合わせた表を作る。
- 差分を「契約が足りない」「契約が余っている」「実装が仕様どおりでない」に分類する。
- 契約側を直せるものは直す。実装の変更を要するものは分類だけ残し、別 work item に切り出す。
- middleware が operation の手前で返す応答をどう契約に書くか (共通の応答として宣言するか、書かないと決めるか) を決め、根拠を `spec/SPECIFICATION.md` に残す。

## Out of Scope

- 要求・応答の本体の形。`wi-382` が閉じており、再発防止は `wi-385` が担う。

## Verification

- `just check-spec`
- `just check-api-compat`
- `just verify`
- 手動確認: 監査表に 327 operation すべてが載り、突き合わせられなかった行が残っていない。

## Risk Notes

宣言に無いステータスを契約へ足すのは非破壊だが、宣言にあって実装が返さないステータスを消すのは、生成クライアントの分岐を壊しうる。消す側は 1 件ずつ、到達不能であることを実装から示してから行う。
