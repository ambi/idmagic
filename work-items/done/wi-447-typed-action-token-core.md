---
depends_on: []
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-08-30
priority: p0
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: パスワード再設定とメール変更の確定が、失敗時にトークンを消費しなくなる。運用者と利用者が観測できる差なので公開する。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-447.md }
initial_context:
  specification:
    - docs/contexts/authentication/scenarios.feature.md#REQ-AUTHENTICATION-016
    - docs/contexts/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-017
  typespec: []
  source:
    - backend/authentication/password
    - backend/idmanagement/user
    - backend/shared/security
    - backend/cmd/internal/bootstrap
    - infra/schema/postgres.sql
  tests:
    - backend/authentication/password
    - backend/idmanagement/user
    - frontend/tests/e2e/ui-scenario-actions.spec.ts
  stop_before_reading:
    - backend/oauth2
    - backend/saml
    - backend/sourcing
affected_spec:
  - { path: docs/contexts/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-016 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-017 }
primary_use_cases:
  - id: password-reset-action-token
    requirement: REQ-AUTHENTICATION-016
    observable_result: リセットリンクのトークンで新しいパスワードを設定でき、同じトークンによる二度目の要求は拒否され、パスワードは一度目の結果のまま変わらない。
    unit_test:
      path: backend/authentication/password/usecases/password_reset_test.go
      name: TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword
      task: test-go-race
    e2e_test:
      path: frontend/tests/e2e/ui-scenario-actions.spec.ts
      name: password reset succeeds through the local SMTP sink without external mail
      task: test-ui-e2e
    unit_fault_model: 検証済みトークンを使用済みにせず、同じトークンで二度目のパスワード更新が成立する。
    e2e_fault_model: 配線されたルートがリセット確定要求を共通核の検証へつなげず、メールのリンクから到達した更新が成立しない。
  - id: email-change-action-token
    requirement: REQ-IDMANAGEMENT-017
    observable_result: 確認リンクのトークンでプライマリメールアドレスが新しいアドレスへ変わり、同じトークンによる二度目の要求は拒否され、アドレスは変わらない。
    unit_test:
      path: backend/idmanagement/user/usecases/email_change_test.go
      name: TestConfirmEmailChangeAppliesEmailAndClearsVerifyAction
      task: test-go-race
    e2e_test:
      path: frontend/tests/e2e/ui-scenario-actions.spec.ts
      name: account email change confirms through the local SMTP sink
      task: test-ui-e2e
    unit_fault_model: 検証済みトークンを使用済みにせず、同じトークンで二度目の確定が成立する。
    e2e_fault_model: 配線されたルートが確認要求を共通核の検証へつなげず、確認メールのリンクからアドレスが変わらない。
---

# パスワード再設定とメール変更に型付きアクショントークンの共通核を導入する

## Motivation

パスワード再設定とメール変更は、乱数トークンの発行、ハッシュだけの保存、有効期限、単回消費、通知、監査という同じ安全性条件を、別々のストアとユースケースで実装している。現在の主要ユースケースは動作しているが、新しい招待や本人確認リンクを追加するときに、目的の束縛、原子的な単回消費、先読み安全性、監査のいずれかを実装し忘れても共通境界が検出しない。

現行実装には、その分散が生んだ欠陥がもう一つある。どちらの確定処理も、作用の前にトークンを削除する。パスワード規則違反、履歴の再利用、新アドレスの先取りで作用が失敗すると、トークンだけが失われ、利用者は正当な回復手段を失う。作用と消費を一つのトランザクションにまとめないかぎり、この失敗を検出する境界がない。

Keycloak のアクショントークンから採用するのは汎用ハンドラー SPI ではなく、用途を閉じた型で識別し、共通の検証を通過した後だけ用途別作用を実行する境界である。用途別ペイロードと業務作用は所有 Context に残し、第三者コードや実行時登録は受け入れない。

## Scope

- `REQ-AUTHENTICATION-016` と `REQ-IDMANAGEMENT-017` に、目的の一致、期限、単回使用、先読みで作用しないこと、消費と対象変更の失敗時整合性を追加する。
- `ActionTokenPurpose`、`ActionTokenEnvelope`、`ActionTokenDigest` と、発行および検証の決定的な計算を、Authentication と Identity Management が依存できる小さな共有セキュリティモジュールとして定義する。
- `Issue(purpose, subject, payload, now, ttl, random)` と `Verify(raw, expectedPurpose, stored, now)` を共有モジュールのインターフェースにし、時刻と乱数を明示的な入力として扱う。保存、通知、監査、用途別作用はこのインターフェースに露出させない。
- 各 Context の永続化アダプターは、未使用確認、使用済み化、用途別作用を同じトランザクションで確定する。作用が失敗した場合はトークンを未使用のまま残し、同じトークンで作用が二回成功する状態を作らない。
- パスワード再設定とメール変更を共通核へ移し、既存のエラー秘匿、通知のローカライズ、パスワードポリシー、メール一意性を維持する。
- ブラウザーまたはメールスキャナーによる `HEAD`、リンクプレビュー、同じ URL の先読みがトークンを消費せず、対象の状態も変更しない E2E を追加する。

## Out of Scope

- 任意の JavaScript、Go plugin、handler を実行時に登録する SPI。
- トークンだけを根拠に任意の必須操作、管理権限、セッションを追加すること。
- OAuth2 アクセストークン、API トークン、ログインセッションの統合。
- 招待など、まだ仕様化されていない新しい用途の実装。
- 二つの用途を一つのテーブルへ統合すること。用途ごとのペイロード列と保持期間を持ったまま移行できるので、統合は新しい用途を追加するときの判断に残す。
- 使用済みトークン行の掃除ジョブ。既存の期限切れ行と同じ扱いにとどめ、保持と削除の方針は別の作業項目に残す。

## Design

### 共有モジュール

`backend/shared/security/actiontoken` は `certificates_mtls`、`passwords_argon2id`、`tokens_jose` と同じく技術上の共通機能として置く。公開する型と操作は次のとおりである。

```go
type Purpose string                    // password_reset / email_change の閉じた集合
type Digest string                     // 生トークンの SHA-256 を小文字 16 進で表した値
type Payload map[string]string         // 用途別の値。共有モジュールは中身を解釈しない

type Envelope struct {
    ID        string
    Purpose   Purpose
    Subject   string
    Payload   Payload
    IssuedAt  time.Time
    ExpiresAt time.Time
    Digest    Digest
}

func Issue(IssueInput) (Issued, error)   // IssueInput{Purpose, Subject, Payload, Now, TTL, Random}
func Verify(VerifyInput) (Envelope, error) // VerifyInput{RawToken, ExpectedPurpose, Stored, Now}
func Fingerprint(raw string) Digest
```

時刻は `Now`、乱数は `Random io.Reader` として入力に現れる。識別子生成も同じ `Random` から読むため、発行は入力だけで決まる計算になり、固定の読み取り元を渡せばテストで完全に再現できる。`Envelope` は生トークンの field を持たないので、保存経路へ生トークンが漏れる表現がそもそも作れない。`Issued` だけが `RawToken` を持ち、これは通知リンクの組み立てにしか渡らない。

`Verify` は用途不一致、ダイジェスト不一致、期限切れ、未知の用途をそれぞれ別の番兵エラーで返す。外部レスポンスはこれらを区別しない。区別するのは内部の監査だけである。ダイジェストの比較は `crypto/subtle` の定数時間比較で行う。

用途別ペイロードは所有 Context に残す。共有モジュールが持つのは束縛の仕組みだけである。

```go
type PayloadCodec[T any] interface {
    Purpose() Purpose
    Encode(T) Payload
    Decode(Payload) (T, error)
}
func EncodePayload[T any](codec PayloadCodec[T], value T) (Payload, error)
func DecodePayload[T any](codec PayloadCodec[T], env Envelope) (T, error)
```

`DecodePayload` は `codec.Purpose()` と `env.Purpose` が一致しない限り復号しない。実行時登録の表は持たない。コーデックは所有 Context の package が値として渡すので、第三者コードが用途を追加する経路はない。

### 永続化と原子性

共有モジュールのインターフェースは発行と検証の二操作に限る。Context 固有のリポジトリもトランザクションコールバックも受け取らない。各 Context の port は次の三操作を持つ。

- `Save(ctx, Envelope) error` — 自分の用途以外のエンベロープは `ErrPurposeMismatch` で拒否する。
- `Find(ctx, Digest) (*Envelope, error)` — 未使用のエンベロープを読むだけで、状態を変えない。用途では絞らない。用途の判定は `Verify` が行う。
- `ConsumeAndApply(ctx, Commit) error` — 使用済み化と用途別作用を同じトランザクションで確定する。未使用の行が無ければ `ErrAlreadyConsumed` を返し、作用が失敗すれば全体を巻き戻してトークンを未使用のまま残す。

`Commit` は用途別の具体値である。Authentication は更新後の `User` と履歴へ追加する encoded hash を、Identity Management は更新後の `User` を運ぶ。トランザクションを跨ぐ書き込みは、`idgovernance` の `UserWorkflowCapture` が既に使っている、所有 Context が公開する tx 版ヘルパー (`userpostgres.SaveUserTx`) をそのまま使う。

確定処理の順序は次のとおりである。

1. `Find` で未使用のエンベロープを読む。
2. `Verify` で用途、ダイジェスト、期限を照合する。
3. 対象を読み、業務規則 (パスワード規則、履歴、メール一意性) を評価する。ここで失敗してもトークンは未使用のまま残る。
4. 更新後の値を組み立て、`ConsumeAndApply` に渡す。使用済み化と保存はここで初めて同時に確定する。
5. トランザクションが成功した後にだけイベントを発行する。

HTTP と通知のような外部作用はトランザクション内で行わない。通知は発行時だけ、監査配送はトランザクション後のイベントログ経由とする。

### 先読み安全性

`GET` と `HEAD` はトークンを消費しない。確定はどちらの用途も `POST` (`/api/auth/reset_password`、`/api/account/v1/email/verify`) であり、リンクが指すのは SPA のページである。これは現在も成り立っているが、それを述べた規範も検査も無かった。仕様の例と E2E で固定する。作用を生む `POST` は既存の CSRF 境界を通す。

### スキーマ

`password_reset_tokens` と `email_change_tokens` に `id UUID`、`purpose TEXT`、`used_at TIMESTAMPTZ` を足す。消費は `DELETE` ではなく `used_at` の設定になる。用途を行に持たせるのは、用途の束縛を「どのテーブルを引いたか」ではなく保存された値で決めるためである。テーブルを一つにまとめる案は採らなかった。`email_change_tokens.new_email` のような用途別の列と、用途ごとに異なる保持期間をそのまま残せるからである。

製品は未リリースなので、既存データの移行は考えない。`infra/schema/postgres.sql` の宣言を更新し、`mise run check-schema` が空のデータベースへ収束することだけを確かめる。

### 代替案

- **汎用ハンドラー SPI**。Keycloak と同じく用途ごとのハンドラーを実行時に登録する案。攻撃面が広く、用途の集合がコードから読めなくなるため採らない。
- **`ConsumeAndApply` にコールバックを渡す**。トランザクション境界にドメインの計算が入り込み、port が「トランザクションの中で何をしてよいか」を語らなければならなくなる。具体値を運ぶ `Commit` なら port の意味が閉じる。
- **共有モジュールにストアを持たせる**。用途別ペイロードの列と保持期間が共有モジュールへ漏れる。所有 Context に残す方針と矛盾する。

## Plan

1. 既存二用途の仕様へ共通不変条件と先読み安全性を追加する。TypeSpec の操作、リクエストボディ、状態コード、エラー契約はいずれも変わらないので更新しない。
2. 共有モジュールの型と決定的な発行および検証を Unit RED から実装し、round trip を oracle とする fuzz target を置く。
3. 各 Context の port を差し替え、memory と PostgreSQL のアダプターへ原子的な消費と作用を実装する。並行消費で一回だけ作用が成功することを検査する。
4. パスワード再設定を共通核へ移す。
5. メール変更を共通核へ移す。
6. 正式なブラウザー入口から発行、先読み、確定、再利用拒否までを通す E2E を追加する。

### 解決済みの問い

- **共有モジュールは用途別ペイロードの型を知るか**。知らない。`Payload` は文字列の対応表で、具体型への復号は所有 Context が渡すコーデックが行う。
- **用途を一つのテーブルへ統合するか**。しない。Out of Scope に置いた。
- **消費は削除か使用済み化か**。使用済み化とする。作用の失敗で巻き戻したとき、行が残っていなければ「未使用のまま残す」を表現できない。
- **失敗した確定は監査に残るか**。残す。拒否理由を持つのは内部の監査だけで、外部レスポンスは無効なトークンと区別できない。

## Tasks

- [x] T001 [Spec] `REQ-AUTHENTICATION-016` と `REQ-IDMANAGEMENT-017` に目的束縛、単回使用、先読み安全性、作用との整合性を定める。EX-AUTHENTICATION-016-03..06、EX-IDMANAGEMENT-017-02..06 を追加した。`mise run check-spec` が「no test names it」で落ちることを Acceptance RED として観測した。
- [x] T002 [Domain] `Purpose`、`Envelope`、`Digest`、決定的な `Issue` と `Verify`、ペイロードコーデックを Unit RED から実装した。`backend/shared/security/actiontoken`。round trip を oracle とする `FuzzVerify` を置き、探索を 20 秒走らせた (execs 4.7M、新規の interesting 8、失敗なし)。
- [x] T003 [Persistence] 各 Context の port を `Save` / `Find` / `ConsumeAndApply` に差し替え、memory と PostgreSQL のアダプターに原子的な消費と作用を実装した。並行消費 (`Test*AppliesOnceUnderConcurrency`) と巻き戻し (`Test*RollsBackWhenTheEffectFails`、`Test*LeavesTheTokenUnusedWhenTheEffectFails`) を両アダプターで検証した。
- [x] T004 [Authentication] パスワード再設定を共通核へ移行した。秘匿 (拒否理由を外へ出さない)、パスワード規則、履歴、通知のローカライズは維持した。EX-AUTHENTICATION-016-01..06 を `password_reset_test.go` と `password_reset_handler_test.go` が名指しする。
- [x] T005 [Identity Management] メール変更を共通核へ移行した。メール一意性の再検査と `verify_email` required action の解除は維持した。EX-IDMANAGEMENT-017-01..06 を `email_change_test.go` が名指しする。
- [x] T006 [E2E] `ui-scenario-actions.spec.ts` の二つの流れに、リンクの二重 `GET` の後でも確定できること、確定済みリンクの再送が拒否されることを加えた。18 pass / 0 fail。
- [x] T007 [Verify] 主要ユースケースの Unit/E2E RED と故障注入を記録した。変更した純粋ロジックとその判断を運ぶ分岐に 39 個の変異を当て、全数が殺されることを確認した。等価変異と検査限界は Completion に記録した。

## Verification

- `mise run check-spec`
- `mise run test-go-package ./backend/shared/security/actiontoken/...`
- `mise run test-go-package ./backend/authentication/password/...`
- `mise run test-go-package ./backend/idmanagement/user/...`
- `mise run test-ui-e2e-file -- tests/e2e/ui-scenario-actions.spec.ts`
- `mise run verify`
- 並行する二つの確定要求のうち一つだけが作用を記録し、もう一つが状態を変えずに拒否される。
- `HEAD`、期限切れ、用途違い、再利用、用途別作用の失敗で、token と対象状態が仕様どおり維持または拒否される。

## Risk Notes

リスクは high。トークンの目的確認、トランザクション、エラー秘匿を誤ると、アカウント乗っ取り、正当な回復手段の喪失、token の再利用につながる。既存二用途を一度に置き換えず、共通核の検証後に一用途ずつ移す。

- **セキュリティ**。生トークンは保存経路の型に現れない。ダイジェスト比較は定数時間で行う。用途不一致と無効トークンの外部レスポンスは区別しない。
- **互換性**。HTTP の経路、リクエストボディ、状態コード、エラーコードは変わらない。UI の変更も要らない。
- **移行**。未リリースの製品なので、移行すべき既存データは無い。宣言的スキーマの更新だけで済む。
- **巻き戻し**。スキーマの追加列は後方互換で、旧コードは新しい列を読まない。列を残したままコードだけ戻せる。

## Completion

- **Completed At**: 2026-09-06
- **Summary**:
  `mise run spec-diff` が報告する規範の差分は `REQ-AUTHENTICATION-016` と `REQ-IDMANAGEMENT-017` の 2 件である。
  どちらも既存の例はそのまま残し、拒否が何を変えないかを述べる例を足した。パスワード再設定には先読みで消費しないこと
  (EX-AUTHENTICATION-016-03)、確定済みトークンの再利用拒否 (016-04)、用途違いの拒否 (016-05)、パスワード規則で拒否された
  ときトークンが未使用のまま残ること (016-06) を加えた。メールアドレス変更には先読み (EX-IDMANAGEMENT-017-02)、再利用拒否
  (017-03)、用途違い (017-04)、期限切れ (017-05)、起票後にアドレスを取られた場合にトークンが残ること (017-06) を加えた。
  TypeSpec の操作、リクエストボディ、状態コード、エラー契約はいずれも変わっていない。
- **Primary Use Case Evidence**:
  - id: password-reset-action-token
    unit_red: 実装前は共通核が存在せず `mise run test-go-package -- ./backend/shared/security/actiontoken/...` が "no non-test Go files" でビルド失敗した。移行後の TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword も、旧 port の Consume を参照する形では "tokenStore.Consume undefined" でコンパイルできなかった。
    e2e_red: mise run check-spec が EX-AUTHENTICATION-016-03..06 について "is declared, but no test names it" で失敗した。正式なブラウザー入口の検査がまだ無いことを示す観測である。
    unit_fault_injection: memory アダプターで使用済み化 (record.used = true) を無効にすると TestResetPasswordWithTokenConsumesTokenAndUpdatesPassword が "reused token error=<nil>, want ErrInvalidResetToken" で失敗した。
    e2e_fault_injection: POST /api/auth/reset_password のルート登録を別パスへ変えて配線を切ると "password reset succeeds through the local SMTP sink without external mail" が失敗した (16 pass / 2 fail)。
  - id: email-change-action-token
    unit_red: 移行後の TestConfirmEmailChangeAppliesEmailAndClearsVerifyAction は、旧 port を参照する形では "undefined - userports.EmailChangeTokenRecord" でコンパイルできなかった。
    e2e_red: mise run check-spec が EX-IDMANAGEMENT-017-02..06 について "is declared, but no test names it" で失敗した。
    unit_fault_injection: memory アダプターで使用済み化 (record.used = true) を無効にすると TestConfirmEmailChangeAppliesEmailAndClearsVerifyAction が "reused token error=<nil>, want ErrInvalidEmailChangeToken" で失敗した。
    e2e_fault_injection: POST /api/account/v1/email/verify のルート登録を別パスへ変えて配線を切ると "account email change confirms through the local SMTP sink" が失敗した (16 pass / 2 fail)。
- **Change-Resistance Results**:
  risk が high なので、代表的な誤実装 1 つでは足りない。変更した純粋ロジック (`actiontoken` の用途判定、入力検証、発行の計算、
  検証の 4 分岐、ペイロードコーデック) と、その判断を運ぶアダプターおよびユースケースの分岐へ、39 個の変異を 1 つずつ当てた。
  手順は `ast-grep` による構造的な削除と `sd` による 1 行の値の入れ替えで、各変異ごとに
  `./backend/shared/security/actiontoken/... ./backend/authentication/password/... ./backend/idmanagement/user/...` を走らせた。
  **結果は 39/39 が殺された。** 内訳は共通核 19、memory と PostgreSQL のアダプター 17、ユースケース 3 である。
  初回の走査では 3 つが生き残り、いずれも検査の欠落だったので検査を足した。
  (1) `Random == nil` の防護を外しても誰も気づかなかった → `TestIssueRefusesInvalidInput` に "no randomness" と
  "short randomness" を追加した。
  (2) トークン素材を 32 バイトから 1 バイトへ縮めても往復は成立した → 素材長そのものを
  `TestIssueIsDeterministicInItsInputs` で固定した。推測への耐性は往復の成立とは別の性質である。
  (3) `clonePayload` が写しではなく呼び出し側の対応表をそのまま返しても通った →
  `TestIssueDoesNotAliasTheCallersPayload` を追加した。
  さらに `DecodePayload` を素の map 読み出しへ置き換える変異が生き残ったので、新アドレスを持たないエンベロープを拒否する
  `TestConfirmEmailChangeRejectsAnEnvelopeWithoutTheNewAddress` を追加した。
  **等価変異と検査の限界**:
  - `Purpose.Valid` の `return true` 化は、`ParsePurpose` と `Issue` の両方から観測できるので独立した変異として数えていない。
    M01 と M02 は入口が違うだけで、殺すのは同じ検査である。
  - 定数時間比較そのもの (`subtle.ConstantTimeCompare` を `==` に置き換える) は変異させていない。タイミングは
    テストの oracle にできず、時間を oracle にすると恒久的に不安定な検査になる。この一点は検査ではなく読みで担保している。
  - PostgreSQL アダプターの変異は embedded-postgres 上で走る。宣言スキーマの収束は別の検査であり、`mise run check-schema` を
    通して空のデータベースへ適用 → 差分なし → 再適用 → 差分なしを確認した。`psqldef` は
    `authorization_detail_types` と `provisioning_connections` で汎用パーサーに失敗して代替パーサーへ落ちる警告を出すが、
    これはこの変更の前からある 2 つの表の話で、収束の判定には影響していない。
  - 変異はコンパイルが通るものだけを数えている。最初に用意した 2 つ (ダイジェスト検査ブロックの削除、`clonePayload` の
    本体置換) は未使用の識別子でコンパイルに失敗したため、条件を常偽にする形へ書き直してから数えた。
    コンパイルで落ちる変異を「殺した」と数えると、検査の力を過大に見積もる。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-go-fuzz -- ./backend/shared/security/actiontoken FuzzVerify 20s` - passed (execs 4.7M)
  - `mise run check-schema` - passed (空のデータベースで適用と dry-run を 2 往復し、いずれも差分なし)
