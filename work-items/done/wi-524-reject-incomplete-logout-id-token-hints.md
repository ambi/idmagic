---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-09
priority: p1
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: upgrade_note
  reason: '`/end_session` が、`sid`、`sub`、`aud` のいずれかを欠く `id_token_hint` と、`sid` の LoginSession の主体と食い違う `sub` を持つ `id_token_hint` を 400 で拒否するようになる。device flow と CIBA が発行した `sid` の無い ID Token を `id_token_hint` に付けている RP は、これまで Cookie のセッションが失効していたのが拒否に変わるため、読み手には互換性の変更として見える。'
  references:
    - { kind: release_note, path: docs/releases/changes/wi-524-incomplete-logout-id-token-hints.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-524-incomplete-logout-id-token-hints.md }
affected_spec:
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-024 }
  - { path: docs/contexts/oauth2/standards.md, requirement: OIDC-LOGOUT-ID-TOKEN-HINT }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.Contract.EndSession1 }
initial_context:
  specification:
    - docs/contexts/oauth2/scenarios.feature.md#REQ-OAUTH2-024
    - docs/contexts/oauth2/standards.md#OIDC-LOGOUT-ID-TOKEN-HINT
  typespec:
    - IdMagic.Contract.EndSession1
  source:
    - backend/oauth2/ports/id_token_hint_verifier.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/oauth2/token/usecases/end_session.go
    - backend/oauth2/handlers_http/end_session_handler.go
    - backend/oauth2/approval/usecases/approval_flow.go
  tests:
    - backend/shared/security/tokens_jose/jwt_signer_endsession_test.go
    - backend/oauth2/token/usecases/end_session_test.go
    - backend/oauth2/handlers_http/end_session_hint_test.go
    - backend/shared/http/server_http/logout_propagation_e2e_test.go
  stop_before_reading:
    - frontend
    - backend/saml
    - backend/wsfederation
    - backend/oauth2/db_postgres
primary_use_cases:
  - id: refuse-hint-missing-claims
    requirement: REQ-OAUTH2-024
    observable_result: '`sid` を持たない ID Token を `id_token_hint` として `/end_session` へ送った RP が 400 の `invalid_request` を受け取り、ブラウザー Cookie が示す LoginSession とその sid の RefreshTokenRecord がどちらも失効しないまま残る。'
    unit_test: { path: backend/oauth2/token/usecases/end_session_test.go, name: TestResolveEndSessionRejectsIDTokenHintWithoutSid, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/end_session_hint_e2e_test.go, name: TestEndSessionRefusesIDTokenHintWithoutSid, task: test-go-race }
    unit_fault_model: '`ResolveEndSession` が空の `sid` を素通しし、HTTP 層が Cookie のセッションへ暗黙に降格する。'
    e2e_fault_model: 拒否の応答だけを書いてローカル失効を続け、ヒントが指していない LoginSession が失効する。
  - id: refuse-hint-subject-mismatch
    requirement: REQ-OAUTH2-024
    observable_result: '`sid` が示す LoginSession の主体と違う `sub` を持つ ID Token を `id_token_hint` として `/end_session` へ送った RP が 400 の `invalid_request` を受け取り、その LoginSession と同じ sid の RefreshTokenRecord がどちらも失効しないまま残る。'
    unit_test: { path: backend/oauth2/token/usecases/end_session_test.go, name: TestResolveEndSessionRejectsIDTokenHintForAnotherSubject, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/end_session_hint_e2e_test.go, name: TestEndSessionRefusesIDTokenHintForAnotherSubject, task: test-go-race }
    unit_fault_model: '`ResolveEndSession` が `sid` の LoginSession を読まず、hint の `sub` を照合せずにログアウト対象として採用する。'
    e2e_fault_model: 主体照合が HTTP 層の失効処理より後に置かれ、拒否を返す前に LoginSession が失効する。
---

# 不完全な `id_token_hint` によるログアウト対象の誤解決を拒否する

## Motivation

`VerifyIDTokenHint` は署名と発行者を検証して `aud`、`sub`、`sid` を抽出するが、これらが空でも成功を返す。
`ResolveEndSession` も空の `sub` と `sid` を拒否せず、`sub` と `sid` が示す LoginSession の主体を照合しない。

このため、署名済みでも必須 claim が欠けた ID Token をログアウト対象の根拠として受理する。
`sid` が空なら HTTP 層はブラウザー Cookie のセッションへフォールバックするため、ヒントが示す対象とは別のローカルセッションを失効しうる。

## Scope

- `id_token_hint` の署名、発行者、audience、subject、`sid` を一組の検証結果として扱う。
- `aud`、`sub`、`sid` のいずれかが空なら `invalid_request` として fail-closed で拒否する。
- `sid` が示す LoginSession の User ID と `sub` が一致する場合だけローカル失効へ進む。
- 拒否時に LoginSession と RefreshTokenRecord が失効しないことを HTTP 経路から観測する。

## Out of Scope

- 期限切れの ID Token を `id_token_hint` として許容する既存方針の変更。
- front-channel と back-channel の通知。
- CIBA が使う `id_token_hint` の有効期限規則。
- CIBA と device flow が `sid` の無い ID Token を発行していること自体。
  本項目はログアウトが `sid` を要求する側だけを直す。
- `sid` がどの LoginSession も指さないヒントの扱い。
  失効させる対象が無いので誤失効は起きず、現行どおり Cookie を消してリダイレクトする。

## Design

### 検証を置く境界

`IDTokenHintClaims` は検証済み claim の値を運ぶだけであり、空値を有効とする型ではない。
`VerifyIDTokenHint` は署名と発行者の検証に加えて `aud` と `sub` の非空を検証し、この 2 つを欠く token を use case へ渡さない。
IdMagic が署名する ID Token は必ず `aud` と `sub` を持つため、この厳格化は自分が発行した token を 1 つも落とさない。

`sid` の非空は共通の verifier ではなく `ResolveEndSession` が要求する。
起票時の Design は `sid` も verifier に置くとしていたが、同じ verifier は CIBA の `resolveApprovalUser` も使う。
`SignIDToken` は `Sid` が空なら `sid` claim を落とし、`sid` を渡すのは認可コード交換だけなので、device flow と CIBA が発行した ID Token には `sid` が無い。
共通層で `sid` を必須にすると、CIBA が正当に受け取っている hint を巻き添えで拒否する。
`sid` を要求するのはログアウトの側の条件なので、境界もログアウト側に置く。

### 主体の照合

`ResolveEndSession` が `sid` の LoginSession を読み、その主体と hint の `sub` を照合する。
照合は「ログアウト対象を解決する」ことそのものであり、対象が決まったあとの後段検査ではない。
そのため HTTP 層ではなく、対象を解決する use case が持つ。

セッションの読み取りは port として入力側へ出す。

```go
// backend/oauth2/ports/login_session_owner.go
type LoginSessionOwnerLookup interface {
    LoginSessionOwner(ctx context.Context, sid string) (userID string, found bool, err error)
}

// backend/oauth2/token/usecases/end_session.go
type EndSessionDeps struct {
    ClientRepo   ports.OAuth2ClientRepository
    HintVerifier ports.IDTokenHintVerifier
    SessionOwner ports.LoginSessionOwnerLookup
}

func ResolveEndSession(ctx context.Context, deps EndSessionDeps, in EndSessionInput) (*EndSessionTarget, error)
```

LoginSession は Authentication の Aggregate なので、実装は HTTP 層のアダプターが `SessionManager.Store` から作る。
`found` が false のときは失効させる対象が無いので拒否しない。
時刻、識別子生成、永続化はいずれも `ResolveEndSession` に入らず、判定は claim と主体の 2 つの値の比較だけになる。

### 拒否が効果を残さないこと

拒否は `ResolveEndSession` が返すエラーであり、HTTP 層はそれを受けた時点で `writeOAuthError` して終わる。
ローカル失効も Cookie の削除もその後段にあるため、拒否と失効が同時に起きる形にならない。
テストは応答だけでなく、LoginSession と同じ sid の RefreshTokenRecord が残っていることも読む。

## Plan

1. claim 欠落と主体不一致を `/end_session` から送る E2E テストを先に追加し、誤った失効または受理を観測する。
2. `VerifyIDTokenHint` で `aud` と `sub` の非空を検証する。
3. `ResolveEndSession` で `sid` の非空と主体の照合を行い、port と HTTP 層のアダプターを配線する。
4. 狭いパッケージテストから変更パッケージ、最終検証へ広げる。

## Tasks

- [x] T001 [Spec] 現行仕様が必須 claim と対象主体の照合を十分に表しているか確認し、不足があれば仕様を先に更新する。
  `OIDC-LOGOUT-ID-TOKEN-HINT` は「署名、発行者、audience、subject、`sid` を検証してログアウト対象のセッションとクライアントを解決する」と既に述べており、行の更新は要らない。
  一方 REQ-OAUTH2-024 は拒否の具体例を `aud` 不一致と署名不能の 2 件しか持たず、claim 欠落と主体不一致を宣言していなかった。
  EX-OAUTH2-024-05 と EX-OAUTH2-024-06 を追加した。
- [x] T002 [E2E] `/end_session` へ不完全または主体が矛盾する `id_token_hint` を送り、拒否と効果の不在について E2E RED を確認する。
  `TestEndSessionRefusesIDTokenHintWithoutSid`、`TestEndSessionRefusesIDTokenHintForAnotherSubject`
  (`backend/shared/http/server_http/end_session_hint_e2e_test.go`、EX-OAUTH2-024-05 / EX-OAUTH2-024-06)。
- [x] T003 [Adapter] `VerifyIDTokenHint` が空の `aud` と `sub` を拒否する Unit RED を GREEN にする。
  `TestVerifyIDTokenHintRejectsMissingSubjectOrAudience`
  (`backend/shared/security/tokens_jose/jwt_signer_endsession_test.go`)。
- [x] T004 [Use Case] `sid` の非空を要求し、hint の `sub` と LoginSession の User ID を照合して不一致を fail-closed で拒否する。
  `TestResolveEndSessionRejectsIDTokenHintWithoutSid`、`TestResolveEndSessionRejectsIDTokenHintForAnotherSubject`
  (`backend/oauth2/token/usecases/end_session_test.go`、REQ-OAUTH2-024)。
- [x] T005 [Resistance] 必須 claim の検査と主体照合をそれぞれ外し、対応するテストが落ちることを確認する。
- [x] T006 [Verify] `mise run verify`。

## Verification

- `mise run test-go-test -- ./backend/oauth2/token/usecases ResolveEndSession`
- `mise run test-go-package -- ./backend/shared/security/tokens_jose`
- `mise run test-go-package -- ./backend/oauth2/token/usecases`
- `mise run test-go-package -- ./backend/oauth2/handlers_http`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run test-go-changed`
- `mise run lint-go`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

- 署名検証だけでは token 内の claim 同士と保存済みセッションの整合性を保証しない。
- 拒否応答だけを検証すると、拒否後にローカル失効を続ける欠陥を見逃す。
- CIBA とログアウトは同じ verifier を使う。`sid` の必須化を共通層へ置くと CIBA の既存入力を巻き添えにするため、Design のとおり境界を分けたうえで `backend/oauth2/approval/usecases` の狭いテストで確認する。

## Completion

- **Completed At**: 2026-09-11
- **Summary**:
  `mise run spec-diff` は `changed scenarios: REQ-OAUTH2-024` を返す。規範の差分はこの Rule 一つで、
  拒否の具体例が 2 件 (EX-OAUTH2-024-05、EX-OAUTH2-024-06) 増えた。標準行 OIDC-LOGOUT-ID-TOKEN-HINT の
  文言は変えていない。行が既に「署名、発行者、audience、subject、`sid` を検証して対象を解決する」と
  述べており、足りなかったのは宣言ではなく、その条件を破る入力を名指しする具体例だった。
  製品の側では、`/end_session` が `id_token_hint` から対象を決める条件が 3 つ増えた。
  `sub` または `aud` を欠くヒントは `VerifyIDTokenHint` が claim を運ぶ前に落とす。
  `sid` を欠くヒントと、`sid` の LoginSession の主体と `sub` が食い違うヒントは `ResolveEndSession` が
  `invalid_request` で拒否する。`sid` を欠くヒントがブラウザー Cookie のセッションへ降格する経路は無くなった。
  `sid` の必須化は共通の verifier ではなくログアウトの use case に置いた。同じ verifier を使う CIBA は
  `sid` を持たない ID Token をヒントに受け取るためである。
  検証の材料としては、Authentication の LoginSession の主体を読む `ports.LoginSessionOwnerLookup` が増え、
  HTTP 層の `loginSessionOwner` が `SessionManager` からそれを満たす。
- **Primary Use Case Evidence**:
  - id: refuse-hint-missing-claims
    unit_red: '`TestResolveEndSessionRejectsIDTokenHintWithoutSid` は `sid を欠くヒントで対象が解決された: &usecases.EndSessionTarget{Sid:"", Subject:"alice", ...}` で落ちた。空の sid がそのまま解決結果として返っていた。'
    e2e_red: '`TestEndSessionRefusesIDTokenHintWithoutSid` は `status=303 body=, want 400` で落ちた。効果側を先に読ませ直すと `拒否されたヒントで LoginSession が失効した` で落ち、Cookie が示す alice のセッションが実際に失効していることを確認した。'
    unit_fault_injection: '`ResolveEndSession` から `claims.Sid == ""` の検査を外すと `TestResolveEndSessionRejectsIDTokenHintWithoutSid` が同じ形で落ちた。'
    e2e_fault_injection: 'HTTP 層を「`writeOAuthError` を書いたうえでローカル失効を続ける」形へ変えると、`TestEndSessionRefusesIDTokenHintWithoutSid` はステータス 400 の検査を通したまま `拒否されたヒントで LoginSession が失効した` で落ちた。応答だけを読むテストなら見逃す欠陥である。'
  - id: refuse-hint-subject-mismatch
    unit_red: '`TestResolveEndSessionRejectsIDTokenHintForAnotherSubject` は `主体の食い違うヒントで対象が解決された: &usecases.EndSessionTarget{Sid:"session-1", Subject:"mallory", ...}` で落ちた。alice のセッションが mallory のヒントで対象になっていた。'
    e2e_red: '`TestEndSessionRefusesIDTokenHintForAnotherSubject` は `status=303 body=, want 400` で落ちた。効果側を先に読ませ直すと `拒否されたヒントで LoginSession が失効した` で落ち、mallory のヒントで alice のセッションが失効していることを確認した。'
    unit_fault_injection: '`ResolveEndSession` から `checkSessionSubject` の呼び出しを外すと `TestResolveEndSessionRejectsIDTokenHintForAnotherSubject` が同じ形で落ちた。'
    e2e_fault_injection: '主体照合を use case から外し、HTTP 層の `endLocalSession` の後ろへ移すと、`TestEndSessionRefusesIDTokenHintForAnotherSubject` が `status=303, want 400` で落ちた。失効後には照合相手の LoginSession がもう無いため、後ろに置いた検査は成立しない。'
- **Change-Resistance Results**:
  上記 4 件の障害注入はいずれも、対応するテストだけを落とした。注入のたびに元へ戻し、`mise run test-go-package`
  で緑に復帰することを確かめている。
  4 件のうち 2 件は、応答だけを読むテストでは検出できない形だった。
  「拒否を書いたうえでローカル失効を続ける」注入では、ステータス 400 の検査が通ったまま効果側の検査だけが落ちた。
  「主体照合を失効処理の後ろへ置く」注入では、照合相手が既に失効しているために検査自体が成立せず、
  303 が返った。どちらも [[wi-390-security-control-test-standard-and-gate]] が見つけた欠陥と同じ形である。
  `TestEndSessionAcceptsCompleteIDTokenHint` を対照として同じ fixture に置いた。完全なヒントは 303 を返し、
  LoginSession と同じ sid の RefreshTokenRecord を失効させる。拒否のテストが
  「この配線では何を送っても 400 になる」ことを見ているのではないと、これで区別できる。
  CIBA への波及は `mise run test-go-package -- ./backend/oauth2/approval/usecases` で確認した。
  `sid` の必須化を共通層へ置かなかったため、CIBA の `id_token_hint` 経路は変わっていない。
- **Verification Results**:
  - `mise run test-go-package -- ./backend/shared/security/tokens_jose` - passed
  - `mise run test-go-package -- ./backend/oauth2/token/usecases` - passed
  - `mise run test-go-package -- ./backend/oauth2/handlers_http` - passed
  - `mise run test-go-package -- ./backend/shared/http/server_http` - passed
  - `mise run test-go-package -- ./backend/oauth2/approval/usecases` - passed
  - `mise run test-go-changed` - passed
  - `mise run lint-go` - passed (0 issues)
  - `mise run check-spec` - passed (156 standard(s), 311 rule(s), 746 example(s), 244 id(s) named by a test)
  - `mise run check-links` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - N/A: 変更はブラウザーへ届かない。UI のログアウトは `id_token_hint` を付けずに
    end_session エンドポイントへ遷移するだけで (`frontend/src/api/oidc.ts`)、`id_token_hint` を送る経路を
    フロントエンドは持たない。
