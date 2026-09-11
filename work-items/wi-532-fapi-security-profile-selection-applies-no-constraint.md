---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-PROFILE-SELECTION }
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-PAR-PKCE }
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-CLIENT-AUTH }
  - { path: docs/contexts/oauth2/standards.md, requirement: FAPI2-SENDER-CONSTRAINT }
---

# `Fapi2SecurityProfile` を選んでも追加の制約が 1 つも掛からない

## Motivation

`docs/contexts/oauth2/standards.md` の FAPI 2.0 Security Profile 節は 4 行を宣言する。

| ID | Adoption | Statement |
|---|---|---|
| `FAPI2-PROFILE-SELECTION` | optional | `Fapi2SecurityProfile` を選択したクライアントだけに本プロファイルの追加制約を適用する。 |
| `FAPI2-PAR-PKCE` | optional | FAPI クライアントは PAR と S256 PKCE を使用する。 |
| `FAPI2-CLIENT-AUTH` | optional | FAPI クライアントは `private_key_jwt` または mTLS で認証する。 |
| `FAPI2-SENDER-CONSTRAINT` | optional | FAPI アクセストークンに DPoP または mTLS による送信者制約を付ける。 |

`docs/contexts/oauth2/decisions.md` も同じ向きの判断を 2 つ持つ。PKCE は「公開クライアントと FAPI 2.0 クライアントではデフォルトで必須」であり、PAR は「FAPI 2.0 クライアントで必須」である。

実装は `fapi_profile` を保存し、列挙として検証し、admin API と `/register` の応答へ書き戻し、管理 UI へ表示する。**しかしこの値を読んで制約を掛ける箇所が 1 つも無い。** 非テストの Go 全体で `fapi` を探すと、当たるのは認可規則の名前 `par_required_if_fapi` と `authorize.go` の予定を書いたコメントだけである。

規則の名前は FAPI を名乗るが、実装は `backend/shared/spec/policy.go:403` のとおり `Subject.Properties.RequirePAR` を読む。

```go
"par_required_if_fapi": func(r AuthZRequest) bool { return !r.Subject.Properties.RequirePAR || r.Context.ParUsed },
```

`RequirePAR` は `RequirePushedAuthorizationRequests` に由来し、`backend/oauth2/client/usecases/register_client.go:148` が入力の同名フラグをそのまま写す。`FapiProfile` からは導かれない。送信者制約の `DpopBoundAccessTokens` も、クライアント認証方式 `TokenEndpointAuthMethod` も同じで、いずれも `FapiProfile` と無関係に決まる。

したがって `fapi_2_security_profile` を選んだクライアントは、PAR 無しの `/authorize` を通し、`plain` PKCE を使い、`client_secret_basic` で認証し、送信者制約の無いアクセストークンを受け取れる。**プロファイルの選択は現状、表示専用のラベルである。**

これは宣言と実装の不一致であり、[[wi-508-amr-vocabulary-declaration-and-implementation-disagree]] と同じ形の欠陥である。宣言した採用 `optional` は「提供しているならその振る舞いを観測できる」ことを意味するが、提供されていない。

## Scope

- `Fapi2SecurityProfile` を選んだクライアントに、4 行が言う追加制約を適用する。
  - PAR を必須とし、PKCE を `S256` に限る。
  - クライアント認証を `private_key_jwt` または mTLS に限る。
  - アクセストークンに DPoP または mTLS による送信者制約を付ける。
- プロファイルを選んでいないクライアントが同じリクエストで通り続けることを、対にして観測する。制約が全クライアントへ漏れると、既存のクライアントが一斉に壊れる。
- 制約を掛ける層を決める。`par_required_if_fapi` が既にある認可規則の層に寄せるのか、クライアントの登録時点で派生フラグを立てるのかは着手時に決める。後者は保存された値と宣言の整合を後から崩しうる。
- 名前と実装が食い違っている `par_required_if_fapi` を、実装に合わせるか名前に合わせるかを決める。

## Out of Scope

- `standards.md` の 4 行の `Adoption` および `Strength` の変更。実装を宣言へ合わせるのが本項目であり、逆ではない。
- FAPI 2.0 Message Signing。`standards.md` は宣言していない。
- [[wi-293-request-object-jar-and-jarm-signed-authorization-messages]] が持つ JAR と JARM。
- 標準行へのテストの対応付けと台帳からの削除。[[wi-522-back-oauth2-client-profile-standards-rows-with-tests]] が持ち、本項目の完了を待つ。

## Design

未着手。着手時に記す。判断が要る点は Scope に挙げた 2 つ、すなわち制約を掛ける層と `par_required_if_fapi` の扱いである。

## Tasks

- [ ] T001 着手時に記す。

## Verification

- `fapi_2_security_profile` を選んだクライアントが、PAR 無しの `/authorize`、`plain` PKCE、共有シークレットによるクライアント認証、送信者制約の無いアクセストークンのいずれでも進めない。
- 同じリクエストを `none` のクライアントが送ると、これまでどおり通る。
- `mise run verify`

## Risk Notes

- **制約が全クライアントへ漏れる。** プロファイルを選んでいないクライアントが同じリクエストで通ることを対にして観測しないと、PAR と非対称クライアント認証が既定になった実装と区別できない。既存のクライアントが一斉に壊れる向きの失敗なので、選んでいない側の観測を先に置く。
- **保存された値と宣言がずれる。** 登録時に `FapiProfile` から `RequirePAR` などの派生フラグを立てる設計を採ると、保存後に admin API がその派生フラグだけを落とせてしまい、`fapi_profile` は `fapi_2_security_profile` のまま制約が消える。判断の時点を要求の評価側へ寄せるか、派生フラグを更新契約の不変項目にするかを決める。
- **`par_required_if_fapi` の名前が先に嘘をついている。** 規則の名前を信じて読むと、FAPI の制約が既にあると誤認する。本項目はこの名前を実装に合わせるか、名前どおりの実装にするかを決めるまで終わらない。
- **既存のシナリオが用語を混同している。** `EX-OAUTH2-009-02` は「PAR 必須の FAPI クライアント」と書くが、固定しているのは `RequirePAR` であってプロファイルではない。実装を宣言へ合わせるとき、このシナリオが何を指すかを併せて決める。
