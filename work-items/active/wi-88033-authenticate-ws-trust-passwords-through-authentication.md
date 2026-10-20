---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-21
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
  - { path: docs/modules/authentication/features/sign-in.md, requirement: REQ-AUTHENTICATION-007 }
primary_use_cases:
  - id: ws-trust-throttles-by-ip
    requirement: REQ-WSFEDERATION-004
    observable_result: 同じ発信元からの WS-Trust の失敗が IP の閾値に達すると、別のアカウントの正しい UsernameToken も 401 で拒否され、トークンが出ない。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/wstrust_examples_test.go, name: TestWsTrustIssue_SharesTheLoginThrottleAndFailureAudit, task: test-go-race }
    fault_model: WS-Trust が IP 単位の試行の制限を数えない、または判定しない。
  - id: ws-trust-refuses-password-change-required
    requirement: REQ-WSFEDERATION-004
    observable_result: パスワードの変更を求められている User の正しい UsernameToken は、403 で拒否され、トークンが出ない。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/wstrust_examples_test.go, name: TestWsTrustIssue_RefusesAUserWhosePasswordMustChange, task: test-go-race }
    fault_model: WS-Trust が期限切れのパスワードと必須操作 update_password を見ずに発行する。
---

# WS-Trust の UsernameToken のパスワード認証を Authentication に委ねる

## 動機

パスワードの照合は Authentication の機能だが、WsFederation の能動的な STS（`/trust/usernamemixed`）は、それを `authenticateWsTrustUser` に自前で持つ。
User の `PasswordHash` を直接読み、アカウント単位の試行の制限、番兵のハッシュ、照合、有効性の確認を書き写している。
写しはログイン画面の認証（REQ-AUTHENTICATION-007）と比べて、次を欠く。

| 欠けているもの | 影響 |
| --- | --- |
| IP 単位の試行の制限とレート制限 | 一つの発信元から多数のアカウントを狙う credential stuffing を、ログイン画面の代わりに WS-Trust で行える |
| `AuthenticationFailed` と、閾値を超えた後の `AuthenticationEventAggregated` | WS-Trust でのパスワードの失敗が認証の監査に残らない |
| 失敗の回数のメトリクス | WS-Trust への攻撃が golden signal に現れない |

同じ規則を二箇所に持つため、ログインの防御を強めても WS-Trust は追従しない。
本人の同意の経路が OAuth2 ではなく Authentication に置かれていた件と同じく、機能を所有者でないモジュールに置いた例として、モジュール間の照合で見つけた。

## 対象範囲

- Authentication に、セッションを作らないパスワード認証の公開操作を設ける。入力はテナント、ユーザー名、パスワード、発信元の IP、経路の名前とし、結果は認証した User、または拒否の理由とする。
- ログイン画面と WS-Trust の両方を、この操作へ通す。
- REQ-WSFEDERATION-004 に、IP 単位の試行の制限と認証の失敗の監査を WS-Trust にも適用することを書く。

## 対象外

- WS-Trust の Application の関門（[wi-52958](../done/wi-52958-gate-ws-trust-issuance-by-application-assignment-and-policy.md)）。
- パスワードのハッシュの置き場所の変更。
- WS-Trust での MFA の対応。
- WS-Trust のログインの結果の指標（`RecordLoginOutcome`）。指標のラベルの値を増やすため、別に扱う。試行の制限の指標は共有の操作が記録する。
- WS-Trust のエンドポイント単位のレート制限。

## 設計

依存の向きは WsFederation → Authentication で、基準の向き（連携プロトコルが認証を利用する）に合う。
OAuth2 が Authentication の公開操作を使うのと同じ形にし、組み立て地点で結ぶ。
ログイン画面に固有の処理（セッションの作成、必須操作の判定、認可の継続、CSRF）は操作に含めない。

### 着手時の調査

ログイン画面のパスワード認証（`internal/signin/handlers_http/login.go:handleLoginAPI` と `login_throttle.go`）は、次の順で処理し、すべて echo のコンテキストに依存する。

1. エンドポイントのレート制限（HTTP の関心なので画面に残す）
2. アカウント単位、続いて IP 単位の試行の制限の取得と、その指標
3. User の検索と、番兵のハッシュを使った照合
4. 失敗の記録、閾値を超えた鍵の `LoginThrottled` と 5 分窓の集約、集約しなかった場合の `AuthenticationFailed`
5. `Active` でない User の拒否と、理由 `account_disabled` の `AuthenticationFailed`
6. パスワードのポリシーの確認（仕様に書かれていない分岐）
7. アカウントの失敗の回数の消去

2 から 7 は、セッションと画面に依存しない。

### 型と操作

公開する型は `backend/authentication/signin/ports` に置く。

```go
type PasswordRefusal string // "", throttled, invalid_credentials, account_disabled, password_policy

type PasswordAttempt struct {
    Username, Password, ClientIP, UserAgent string
}

type PasswordOutcome struct {
    User                   *userdomain.User
    Refusal                PasswordRefusal
    RetryAfterSeconds      int  // Refusal が throttled のとき
    PasswordChangeRequired bool // 期限切れか、必須操作 update_password を持つ
}

type PasswordAuthenticator interface {
    AuthenticatePassword(ctx context.Context, in PasswordAttempt) (PasswordOutcome, error)
}
```

実装は `backend/authentication/internal/usecases/password_authentication.go` の `PasswordAuthentication` に置き、2 から 7 を echo なしで行う。
テナントは要求のコンテキストから読む。
`PasswordChangeRequired` は User を保存せずに判定する。
ログイン画面は応答の形と結果の指標（`RecordLoginOutcome`）を、WS-Trust は 401 と 403 の `text/plain` を、拒否の理由から決める。
組み立ては `authentication.Module` の `PasswordAuthentication(RouteDeps)` が行い、WsFederation の `Register` は試行の制限、ハッシュ、番兵のハッシュの 3 つの引数の代わりにこれを受け取る。

### 利用者に確認した判断

| 問い | 決定 | 理由 |
| --- | --- | --- |
| ポリシーを満たさない古いパスワードを WS-Trust でも拒否するか | 拒否する。ログイン画面の分岐を REQ-AUTHENTICATION-007 に書く | 同じ認証を一つの操作で行い、経路ごとの差を残さない |
| 期限切れのパスワードと必須操作 `update_password` を持つ User に WS-Trust で発行するか | 403 で拒否する | 変更を求めている資格情報で新しいトークンを作らない。WS-Trust には変更の画面がない |

期限切れのときにログイン画面と同じく `update_password` を付ける案は採らない。
付けるには User を丸ごと保存する書き手を増やすことになり、[同時の更新と排他](../../docs/design/data/database.md#同時の更新と排他)に反する。
利用者が次にブラウザーでログインしたとき、ログイン画面が付ける。

### 構造の変更と振る舞いの変更

先に、ログイン画面の振る舞いを変えずに操作を切り出す。
移したロジックの単体テスト（試行の制限の指標、失敗の集約、相関のハッシュ）は、同じ表明のまま usecase のパッケージへ移す。
その後に、WS-Trust を操作へ通す振る舞いの変更を行う。

## 計画

1. 着手時に、ログイン画面のパスワード認証のうち、セッションと画面に依存しない部分を列挙し、操作のシグネチャを設計へ記す。
2. WS-Trust で IP 単位の制限と `AuthenticationFailed` が働かない故障を RED にする。
3. 操作を実装し、ログイン画面の既存のテストを保ったまま WS-Trust を置き換える。

## タスク

- [x] T001 [Refactor] パスワード認証を `PasswordAuthentication` に切り出し、ログイン画面をそれに通す。移したロジックの単体テストを usecase のパッケージへ移す。ログイン画面の既存のテストは変えない。
- [x] T002 [Spec] REQ-AUTHENTICATION-007 にポリシーの確認を、REQ-WSFEDERATION-004 に試行の制限、監査、ポリシー、パスワードの変更の要求による拒否を書く。
- [x] T003 [Acceptance] WS-Trust の IP 単位の制限、失敗の監査、パスワードの変更の要求の欠落を RED にする。
- [x] T004 [App] WS-Trust を `PasswordAuthenticator` へ通し、写しを消す。
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`、`mise run lint-go`、`mise run check-boundaries`、`mise run check-spec`。
- ログイン画面の既存のテストが変更なしで成功することを確かめる。

## リスク

ログイン画面の失敗の応答（401、429 と `Retry-After`）と WS-Trust の応答（401 と `text/plain`）は異なる。
操作は拒否の理由だけを返し、応答の形は各経路が決める。

## 要約

REQ-WSFEDERATION-004 に、UsernameToken の照合がログインの API と同じ試行の制限、失敗の数え方と集約、認証の失敗のイベントを使うこと、パスワードのポリシーを満たさない資格情報と IP の制限を 401 に含めること、パスワードの変更を求められている User を 403 で拒否することを書いた。
REQ-AUTHENTICATION-007 には、仕様に書かれていなかったログインの API のポリシーの確認を書いた。
二つの判断（WS-Trust でもポリシーを確認する、パスワードの変更を求められている User に発行しない）は利用者に確認した。

まず、ログインの API のパスワード認証を echo に依存しない `PasswordAuthentication` に切り出し、ログインの API をそれに通した。
ログインの API の既存のテストは変えずに成功し、移したロジックの単体テストは同じ表明のまま usecase のパッケージへ移した。
次に、WS-Trust の能動的な STS を `PasswordAuthenticator` へ通し、写しの照合を消した。
受け入れテストで、WS-Trust の失敗が `AuthenticationFailed` を出さないことと、`update_password` を持つ User に発行することを RED にしてから直した。

期限切れのときに `update_password` を付ける案は、User を丸ごと保存する書き手を増やすので採らず、拒否だけにした。
ミューテーションで見つかった、失敗を記録する鍵の種類と IP の制限の分岐の観測の不足は、単体テストを足して塞いだ。
成功の記録のストアが失敗する経路は、切り出す前と同じくテストがない。
WS-Trust のログインの結果の指標とエンドポイント単位のレート制限は対象外に残した。
