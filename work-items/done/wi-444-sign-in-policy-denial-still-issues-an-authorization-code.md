---
depends_on: []
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-08-30
priority: p0
change_kind: bugfix
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/contexts/oauth2/scenarios.md
    - docs/contexts/application/scenarios.md
  typespec: [IdMagic.OAuth2.Operations.Authorize]
  source:
    - backend/oauth2/handlers_http/authorize_login.go
    - backend/oauth2/handlers_http/authorize_handler.go
    - backend/shared/http/support_http/csrf.go
    - backend/shared/http/support_http/auth.go
  tests:
    - backend/oauth2/handlers_http/authorize_handler_test.go
  stop_before_reading: [frontend, spec, backend/sourcing]
affected_spec:
  - { path: docs/contexts/application/scenarios.md, requirement: REQ-APPLICATION-009 }
  - { path: docs/contexts/application/scenarios.md, requirement: REQ-APPLICATION-010 }
---

# サインインポリシーが拒否した認可要求に、認可コードが発行されている

## Motivation

`enforceDefaultSignInPolicy` (`backend/oauth2/handlers_http/authorize_login.go`) は、テナント既定のサインインポリシーが拒否を決めたとき 403 の Problem Details を書く。ところがその戻り値は `support.WriteProblem(...)` そのものであり、`WriteProblem` は応答を書き終えたとき `nil` を返す。呼び出し元 3 か所はいずれも `if err != nil { return err }` の形なので、**拒否は呼び出し元へ 1 ビットも伝わらない**。

`handleAuthorize` はそのまま `completeAfterAuthn` へ進み、認可コードを発行する。[[wi-397-token-endpoint-declares-a-403-it-never-returns]] の実測中に置いた検査が、これを次の形で捕まえた。

```
denied authorization issued an authorization code:
&{ClientID:auth-client-fp UserID:user_alice Scopes:[openid profile idmagic.admin] CodeChallengeMethod:S256}
```

`AuthorizationCodeIssued` が実際に発行され、コードが保存される。応答本体は先に書かれた 403 が勝つので、**利用者から見ると拒否されたように見えながら、認可コードは存在している**。echo は `response already written to client` を ERROR で記録するが、これは拒否が漏れた合図としては読まれていない。

これは [[wi-390-security-control-test-standard-and-gate]] の R1 が落とすはずの形 —— 「応答を書いて、書いた結果 (`nil`) を返す」—— そのものである。**R1 が見逃すのは、この番人が `error` 単独ではなく `(bool, error)` を返すからである。** [[wi-398-guard-rules-follow-one-level-of-indirection]] は writer の間接を追えるようにしたが、戻り値の形が違う番人は依然として判定の外にいる。

同じ関数の 401 `authentication_required` 分岐 (`SessionManager.RequireFactor` が `nil` を返した場合) も同じ形で、こちらは期限切れセッションが同様に素通りする。

## Scope

- `enforceDefaultSignInPolicy` の 3 つの拒否分岐が、応答を書いたうえで `ErrResponseWritten` を包むエラーを返すようにする。呼び出し元 3 か所は既に `if err != nil { return err }` を持つので、番人側だけで閉じる。
- 拒否が「何も通していない」ことを、状態を読み戻して固定する。認可コードが発行されないことを否定検査として置く。

## Out of Scope

- R1 を `(bool, error)` を返す番人まで広げること。検査そのものの強化は [[wi-390-security-control-test-standard-and-gate]] の系列が持つ話で、この work item は実害のある 1 件を閉じる。別途 work item を立てる。
- `Authorize` の 403 の契約宣言。[[wi-397-token-endpoint-declares-a-403-it-never-returns]] が持つ。
- Application 個別のサインインポリシー経路 (`authorize_completion.go`)。こちらは `authorizationErrorURL` で `access_denied` を RP へ返しており、応答を書いてから続行する形ではない。
- `enforceDefaultSignInPolicy` の拒否が `AppAccessDeniedByPolicy` を発行していないこと。`REQ-APPLICATION-009` の ALT は拒否時にこの event を求めており、SAML (`saml/usecases/signin.go`)、WS-Fed (`wsfederation/usecases/signin.go`)、Application 個別ポリシー (`authorize_completion.go`) の 3 経路は発行しているが、この分岐だけが発行しない。監査記録の欠落であって認可の迂回ではないので、素通り自体を閉じる本 work item とは分けて別途立てる。

## Design

拒否を伝える手段はリポジトリに既にある。`support.ErrResponseWritten` (`backend/shared/http/support_http/csrf.go`) は「拒否の応答を書き終えたので呼び出し元は処理を止めよ」を表す番兵で、`error_handler.go` はこれを未処理エラーとして記録し直さない。`WriteAdminAccessError` は `ErrAdminAccessRefused` としてこれを包む先例である。

同じ形に揃える。`ErrSignInPolicyRefused` を `ErrResponseWritten` を包む形で定義し、3 つの拒否分岐が応答を書いたあとにそれを返す。

```go
var ErrSignInPolicyRefused = fmt.Errorf("%w: sign-in policy refused", support.ErrResponseWritten)
```

戻り値の `redirected bool` は「遷移させたので呼び出し元は次の画面を返せ」の意味のまま変えない。拒否は遷移ではないので `redirected` には載せない —— 載せると呼び出し元が 403 のあとに 200 の `browserFlowResponse` を重ねて書く。

検討した代替案:

- **`redirected` を `true` にして拒否を表す**: 呼び出し元 3 か所はいずれも `redirected` のとき別の応答を書く (`pendingAuthPath` への遷移、`browserFlowResponse`)。403 の上に 200 を重ねることになり、いま起きている二重書き込みを別の形で残す。採用しない。
- **呼び出し元で `c.Response().Committed` を見る**: 応答が書かれたかどうかは echo が知っているので判定はできるが、番人の契約が戻り値でなく副作用の観測になる。R1 が落とそうとしているのはまさにその形なので、逆行する。採用しない。

書き込み結果を番兵へ変える箇所は `refused` (`support_http/auth.go`) と同じ形の小さなヘルパーに寄せる。最初は分岐 3 か所で `if err := support.WriteProblem(...); err != nil` と書いたが、これは `WriteProblem` を番人の位置に置く形であり、`mise run check-security-controls` の R1 が「`WriteProblem` は書き込み結果 (成功時 nil) を返すので番人にしてはならない」として正しく落とした。書き込みを引数として渡す `refusedBySignInPolicy(support.WriteProblem(...))` に改め、`WriteAdminAccessError` の先例と同じ形に揃えた。

```go
func refusedBySignInPolicy(writeErr error) error {
    if writeErr != nil {
        return writeErr
    }
    return ErrSignInPolicyRefused
}
```

## Plan

1. 拒否が認可コードを発行しないことを RED で置く (`wi-397` の実測で既に落ちている検査を使う)。
2. 番兵を定義し、3 分岐を書き換える。
3. 呼び出し元 3 か所が変更不要であることを確かめる。

## Tasks

- [x] T001 [Acceptance] 拒否された認可要求が認可コードを発行しないことを RED で置いた。
  `TestAuthorizeDeniedBySignInPolicyReturnsProblemDetails403`。シナリオ: `REQ-APPLICATION-009` / `REQ-APPLICATION-010`。
- [x] T002 [Adapters] `ErrSignInPolicyRefused` と `refusedBySignInPolicy` を定義し、3 つの拒否分岐で
  応答を書いたあとに返すようにした (`authorize_login.go`)。
- [x] T003 [Acceptance] 変異試験で無防備と判明した残り 2 分岐にも効果検査を足した。
  `TestAuthorizeMfaRequiredWithoutSecondFactorIssuesNoCode` と
  `TestAuthorizeStepUpWithExpiredSessionIssuesNoCode`。
- [x] T004 [Verify] `mise run verify` と `mise run check-security-controls` を通した。

## Verification

- `mise run verify`
- `mise run check-security-controls`

## Risk Notes

リスクは high。認可コードの発行を止める変更なので、いま通っている正当な要求まで止めると、サインインポリシーを設定したテナントのログインが全面的に壊れる。`PolicyAllow` と `PolicyStepUpRequired` の経路が従来どおり通ることを、変更後に既存のテストで確かめる。逆に、拒否の側を直したことで初めて 403 が実効になるので、これまで「拒否されているのに通っていた」利用者は通らなくなる。それが本来の意図された振る舞いである。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `enforceDefaultSignInPolicy` の 3 つの拒否分岐が、応答を書いたうえで `support.ErrResponseWritten` を包む `ErrSignInPolicyRefused` を返すようになった。それまでは `support.WriteProblem` の戻り値 (書き終えたとき `nil`) をそのまま返しており、呼び出し元 3 か所の `if err != nil { return err }` が素通りしていた。結果として、サインインポリシーが拒否した認可要求に対して 403 / 401 を書いたあとも `completeAfterAuthn` が走り、`AuthorizationCodeIssued` が発行され認可コードが保存されていた。利用者には拒否に見えて、認可コードは存在している状態だった。`mise run spec-diff` は `no normative specification change against main` を返す。規範は動かしておらず、`REQ-APPLICATION-009` が既に求めている「トークン発行前にポリシーを評価し、拒否する」に実装が追いついた変更である。
- **Acceptance RED Evidence**:
  - **Test**: `TestAuthorizeDeniedBySignInPolicyReturnsProblemDetails403` (`backend/oauth2/handlers_http/authorize_handler_test.go`)
  - **Requirement**: REQ-APPLICATION-009
  - **Observed Failure**: `denied authorization issued an authorization code: &{TenantID:00000000-0000-4000-8000-000000000000 ClientID:auth-client-fp UserID:user_alice Scopes:[openid profile idmagic.admin] CodeChallengeMethod:S256}`。echo も同じ要求で `response already written to client` を ERROR として記録していた。
  - **Detection Reason**: 拒否を 2 つの主張で見る。応答の側 (403 と `application/problem+json` と `urn:idmagic:error:access_denied`) は、これだけなら「拒否を書いてから操作を続行する」実装にも通ってしまう。効いているのは 2 つめ、すなわち拒否が何も通していないことを状態から読み戻す主張で、`AuthorizationCodeIssued` が 1 件も発行されていないことを確かめる。実際にこの 2 つめだけが落ちた。応答の 3 つの主張はすべて当初から通っており、欠陥が応答の文言ではなく効果の側にあることを分けて示している。
- **Unit RED Evidence**:
  - **Test**: `TestAuthorizeMfaRequiredWithoutSecondFactorIssuesNoCode` と `TestAuthorizeStepUpWithExpiredSessionIssuesNoCode` (同ファイル)
  - **Requirement**: REQ-APPLICATION-009
  - **Observed Failure**: 両方とも `refused authorization issued an authorization code: &{... ClientID:auth-client-fp UserID:user_alice ...}`。番人を元の形へ戻した変異 (M3 / M4) に対して観測した。
  - **Detection Reason**: 番人の 3 分岐は別々の条件から出るので、1 つの検査でまとめて固定すると残りは無防備になる。実際、最初は `PolicyDeny` の 1 件しか置いておらず、変異試験 M3 / M4 がその 2 分岐を素通りした (下記)。分岐ごとに、その分岐だけが選ばれる条件 (MFA 必須 + 第二要素なし / MFA 必須 + TOTP 登録済み + セッション不在) を組み立て、いずれも認可コードが発行されないことを個別に確かめる。
- **Change-Resistance Results**:
  変更した番人の分岐を系統的に変異させ、5 件すべてが検出されることを実測した。
  M1 `PolicyDeny` 分岐を元の「書き込み結果を返す」形へ戻す → `TestAuthorizeDeniedBySignInPolicyReturnsProblemDetails403` が認可コードの発行で落ちる。
  M2 拒否を `redirected = true` で伝える → 同検査が `Location="/realms/default/totp"` で落ちる。403 の上に遷移を重ねる実装を分離できている。
  M3 「MFA 必須だが第二要素なし」分岐を元へ戻す → `TestAuthorizeMfaRequiredWithoutSecondFactorIssuesNoCode` が落ちる。
  M4 「セッション期限切れ」分岐を元へ戻す → `TestAuthorizeStepUpWithExpiredSessionIssuesNoCode` が落ちる。
  M5 `refusedBySignInPolicy` が常に書き込み結果を返す (番兵を返さない) → 3 検査すべてが落ちる。
  **方法の限界と、それが見つけたもの**: 最初の版では M3 と M4 が生存した。`PolicyDeny` の 1 件しか検査が無く、他の 2 分岐は変更されていながら 1 つも主張に触れられていなかった。生存を隠さず検査を 2 件追加してから再測している。なお `PolicyAllow` と、ステップアップへ正常に遷移する `PolicyStepUpRequired` の経路は本変更で触れておらず、既存の `TestAuthorizeFirstPartyClientSkipsConsent` ほか同パッケージの検査と `backend/shared/http/server_http` の e2e が通ることで退行が無いことを確かめた。
- **Verification Results**:
  - `mise run verify` - passed (exit 0)
  - `mise run check-security-controls` - ok (183 declared refusal(s), 18 promised by a 403 on a state change, 128 awaiting a test)
  - `mise run test-go-package -- ./backend/oauth2/handlers_http/...` - ok
  - `mise run test-go-package -- ./backend/shared/http/server_http/...` - ok
  - `mise run spec-diff` - `no normative specification change against main`
