---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-09
priority: p2
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの標準行にテストを対応付けるだけで、利用者が読むリリース情報に変化は無い。
  references: []
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/contexts/oauth2/standards.md
    - docs/contexts/oauth2/scenarios.feature.md
  typespec:
    - IdMagic.Contract.DeviceAuthorizationRequest
    - IdMagic.Contract.DeviceAuthorizationResponse
    - IdMagic.Contract.BackchannelAuthenticationRequest
    - IdMagic.Contract.BackchannelAuthenticationResponse
    - IdMagic.Contract.AccountApprovalRequest
  source:
    - backend/shared/spec/discovery.go
    - backend/oauth2/handlers_http/routes.go
    - backend/oauth2/handlers_http/device_handler.go
    - backend/oauth2/handlers_http/approval_handler.go
    - backend/oauth2/handlers_http/register_handler.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/device/usecases/device_flow.go
    - backend/oauth2/approval/usecases/approval_flow.go
    - backend/oauth2/device/ports/device_code_store.go
    - backend/oauth2/approval/ports/approval_request_store.go
    - backend/oauth2/device/db_memory/device_codes.go
    - backend/oauth2/approval/db_memory/approval_requests.go
    - frontend/src/features/account/AccountApprovalsPage.tsx
    - tools/check/standards-coverage-debt.json
    - tools/check/src/check-specifications.ts
  tests:
    - backend/oauth2/handlers_http/device_handler_test.go
    - backend/oauth2/handlers_http/approval_handler_test.go
    - backend/oauth2/handlers_http/endpoint_refusal_effects_test.go
    - backend/shared/http/server_http/authorization_request_standards_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
    - backend/shared/http/server_http/trusted_device_e2e_test.go
    - frontend/src/features/account/AccountApprovalsPage.test.tsx
  stop_before_reading:
    - backend/oauth2/db_postgres
    - backend/jobs
    - frontend/src/routes
---

# 非対話グラント（Device / CIBA）が宣言する標準 8 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-499-back-oauth2-standards-rows-with-tests]] は `docs/contexts/oauth2/standards.md` の 80 行を引き取り、最初の節（`OAuth Client ID Metadata Document` の 7 行）を消化したうえで、残る 73 行を**行が共有する製品の入口**を単位に 7 件へ割った。本項目はそのうち 8 行を持つ。

8 行は、ブラウザーのリダイレクトを使わずに承認を取る 2 つのグラントを定める。`authorization_pending`、`slow_down`、`expired_token` のポーリングの意味づけが両方に現れ、CIBA の 3 行は提供しない配信モードと補助パラメーターを言う。7 行が `optional` または `excluded` であり、8 行のほとんどが「提供していること」以外を観測する行である。

## Scope

- 次の 8 行を消化する。

| ID | Adoption | 節 |
|---|---|---|
| `RFC8628-DEVICE-AUTHORIZATION` | optional | OAuth 2.0 Device Authorization Grant |
| `RFC8628-POLLING` | required | OAuth 2.0 Device Authorization Grant |
| `CIBA-CORE-BACKCHANNEL-REQUEST` | optional | OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 |
| `CIBA-CORE-POLL-MODE` | optional | OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 |
| `CIBA-CORE-BINDING-MESSAGE` | optional | OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 |
| `CIBA-CORE-PING-PUSH` | excluded | OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 |
| `CIBA-CORE-USER-CODE` | excluded | OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 |
| `CIBA-CORE-SIGNED-REQUEST` | excluded | OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の work item が持つ標準行。[[wi-499-back-oauth2-standards-rows-with-tests]] の Design にある分割表が所属を定める。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。[[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**観測の入口は デバイス認可エンドポイントと CIBA のバックチャネル認証エンドポイント である。** [[wi-499-back-oauth2-standards-rows-with-tests]] が 7 行を測って出した結論は、消化の費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点にある。本項目の 8 行はこの入口を共有するので、ハーネスは 1 度組めば足りる。

この入口の観測は、承認が成立する前と後でポーリングの応答が変わることを、時間を進めながら読む形になる。時間は入力として渡せる位置に無ければ観測できないので、ハーネスを組む前にそこを確かめる。

`excluded` の 3 行（`CIBA-CORE-PING-PUSH`、`CIBA-CORE-USER-CODE`、`CIBA-CORE-SIGNED-REQUEST`）はいずれも標準側の機能を書いている。この 3 行をどう読むかは後述の「`excluded` の観測の型」が定める。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

### 着手時に確認した実装状況

8 行を製品の入口と既存テストへ照合した結果、8 行とも現状のまま消化できる。前提となる実装 work item は無い。

| ID | 状況 | 扱い |
|---|---|---|
| `RFC8628-DEVICE-AUTHORIZATION` | `/device_authorization` が 3 つの値を発行し、`/api/auth/device` が承認と拒否を受け付ける。 | 本項目でテストを対応付ける。 |
| `RFC8628-POLLING` | トークンエンドポイントの `device_code` グラントが 3 つのエラーを出し分ける。 | 本項目でテストを対応付ける。 |
| `CIBA-CORE-BACKCHANNEL-REQUEST` | `/bc-authorize` がクライアント認証、`scope`、ちょうど一方のヒント、`unknown_user_id` を実装している。 | 本項目でテストを対応付ける。 |
| `CIBA-CORE-POLL-MODE` | トークンエンドポイントの CIBA グラントが 4 つのエラーと一度きりの消費を実装している。 | 本項目でテストを対応付ける。 |
| `CIBA-CORE-BINDING-MESSAGE` | 承認一覧 API と承認画面が 4 つを併せて示す。 | 本項目でテストを対応付ける。 |
| `CIBA-CORE-PING-PUSH` | 配信モードを持つクライアントメタデータが存在せず、Discovery は `poll` だけを広告する。 | 本項目でテストを対応付ける。 |
| `CIBA-CORE-USER-CODE` | `user_code` を読む経路が無く、Discovery は非対応を広告する。 | 本項目でテストを対応付ける。 |
| `CIBA-CORE-SIGNED-REQUEST` | 署名済み JWT を読む経路が無い。 | 本項目でテストを対応付ける。 |

### `excluded` の観測の型

3 行はいずれも標準側の機能を書いている。着手前の想定は「その機能を使うリクエストが通らないこと」と「`auth_req_id` が発行されていないこと」の対だったが、実測はそうならなかった。`/bc-authorize` へ `user_code` または `client_notification_token` を足しても 200 で `auth_req_id` が返り、`backchannel_token_delivery_mode` を名指した動的クライアント登録も 201 で成功する。

拒否しないことは、この 3 行に限れば宣言した採用の未達ではない。RFC 6749 §3.1 は認可サーバーに未知のリクエストパラメーターを無視することを求めており、行が言っているのは「その機能を提供する」ことであって「その機能を名指したリクエストを拒む」ことではないからである。したがって観測の型は次のように決める。

**`excluded` かつ標準側の機能を書いている行は、機能の効果が 1 つも現れないことを、能力の広告と、その機能を名指したリクエストの結果の対で読む。** 拒否が起きる行では拒否とその効果を読み、拒否が起きない行では「名指しても何も変わらない」ことを読む。後者では、対照として同じリクエストを機能なしで送り、結果が一致することを見る。一致していれば、その機能はどこにも効いていない。

この型で 3 行は次のように分かれる。`CIBA-CORE-PING-PUSH` と `CIBA-CORE-USER-CODE` は拒否が起きないので、Discovery の広告と、登録済みクライアントに配信モードも通知先も残らないこと、および `user_code` の有無で結果が変わらないことを読む。`CIBA-CORE-SIGNED-REQUEST` は署名済み JWT だけを送るリクエストが実際に拒否され承認要求を作らないので拒否の型で読み、加えて平文パラメーターと矛盾する `request` を送ったとき平文側が効くことで、JWT が読まれていないことを区別する。

### 時間を進められることの確認

ポーリングの意味づけは時間の経過でしか観測できない。HTTP ハンドラーは `time.Now().UTC()` を直接渡すので、入口の外から時計を差し替える口は無い。一方でどちらのグラントも期限と最終ポーリング時刻をレコードに持ち、判定はその値と現在時刻の比較でしかない。そこで**時計ではなくレコードを遡らせる**。`DeviceCodeStore` と `ApprovalRequestStore` の正式なポートで読み出し、`IssuedAt` / `RequestedAt`、`ExpiresAt`、`LastPolledAt` を同じ幅だけ過去へずらして書き戻す。有効期間の幅は変えないので、製品が作らないレコードを作ることにはならない。

### 着手前に名指しした RED 検査

本項目は製品の振る舞いを変えないため、製品要求に対する Acceptance RED と Unit RED は持たない。
Acceptance RED の代替は、8 件を台帳から一時的に外した状態での `mise run check-spec` である。
同検査が各 ID を `is declared, but no test names it` として報告すれば、台帳だけを縮める誤りを検出できる。

Unit RED の代替は、対応付けたテストごとの故障注入である。`RFC8628-POLLING` は `slow_down` の間隔判定を外し、`CIBA-CORE-POLL-MODE` は消費の一度きりを外し、`CIBA-CORE-BACKCHANNEL-REQUEST` はヒントがちょうど一方であることの検証を外し、`CIBA-CORE-PING-PUSH` と `CIBA-CORE-USER-CODE` は Discovery の広告を裏返して、対象テストが落ちることを観測する。

## Plan

1. 8 行が共有する入口にハーネスを組み、`required` の 1 件目を通しで消化して型を決める。
2. `optional` があれば、その 1 件目で「提供している」と言える根拠の形を決める。提供していなければ規範の変更として切り出す。
3. `excluded` があれば、その 1 件目で観測の型を決める。
4. 残りを消化し、解決した id を台帳から外す。
5. 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。

## Tasks

- [x] T001 [Acceptance] 消化する id を台帳から先に外し、`mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。
  8 件すべてが `docs/contexts/oauth2/standards.md` の行番号つきで個別に報告された。
  recipe: `mise run check-spec`
- [x] T002 [Harness] デバイス認可エンドポイントと CIBA のバックチャネル認証エンドポイント にハーネスを組み、`required` の 1 件目で型を決める。
  `Register` が組み立てたスタックへ HTTP で入る `newNonInteractiveGrantFixture` を
  `backend/shared/http/server_http/non_interactive_grant_standards_test.go` に組んだ。
  `required` の 1 件目である `RFC8628-POLLING` を、1 本の `device_code` に対する
  4 回のポーリングで通しで消化し、型を決めた。
  時計は差し替えられないので、レコードを幅ごと過去へずらす `advanceDeviceClock` /
  `advanceApprovalClock` を置いた。
  recipe: `mise run test-go-test -- ./backend/shared/http/server_http <test>`
- [x] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
  `optional` は `RFC8628-DEVICE-AUTHORIZATION` で「まず提供していることを確かめ、
  Statement の動詞の数だけ観測を置く」型とした。
  `excluded` は実測に基づいて型を決め直した。詳細は Design の「`excluded` の観測の型」に書いた。
  recipe: `mise run test-go-test -- ./backend/shared/http/server_http <test>`
- [x] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
  8 件を外し、台帳の `untested` は 38 件から 30 件へ減った。
  テストが名指す ID は 249 件から 257 件へ増えた。
  recipe: `mise run check-spec`
- [x] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
  8 行に対して 13 件の故障を注入し、いずれも対象テストが落ちた。内訳は Completion の表に書いた。
  recipe: `mise run test-go-test -- ./backend/shared/http/server_http <test>`
  recipe: `mise run test-ui-unit-file -- src/features/account/AccountApprovalsPage.test.tsx`
- [x] T006 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
  8 行とも宣言した採用を満たしており、切り出す欠陥は無かった。
  `excluded` の 3 行が機能を名指したリクエストを拒否しないことは、Design に書いたとおり
  未達ではない。
  recipe: `mise run check-work-items`
- [x] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 8 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。注記に「何を固定しているか」を書かせ、崩して落ちることを T005 で確かめる。
- **ハーネスが本番とずれる。** 入口を通すという方針は、その入口が製品と同じ部品でできているときだけ意味を持つ。[[wi-500-back-api-tokens-standards-rows-with-tests]] はここを一度間違え、製品の欠陥でないものを欠陥として起票した。組み立ては `cmd/internal/bootstrap` が作る形に合わせる。
- **1 行が 2 つのことを言っている。** `Statement` に動詞が 2 つあれば観測も 2 つ要る。
- **時間を進められないハーネスを組む。** ポーリングの意味づけは時間の経過でしか観測できない。時間が入力として渡せることを T002 の前に確かめる。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、並行しても衝突はエントリー単位に収まる。自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-11
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の差分は無い。
  変わったのは、非対話グラントが宣言する 8 行に対する観測の有無である。
  `tools/check/standards-coverage-debt.json` の `untested` は 38 件から 30 件へ減り、本項目が持つ
  8 件は 1 件残らず消えた。テストが名指す ID は 257 件になった。製品コードは変えていない。
  増えたのは backend のテスト 7 本と frontend の注記 1 件である。
  着手前の想定と実測が食い違ったのは `excluded` の 3 行である。想定は「その機能を使う
  リクエストが通らないこと」と「`auth_req_id` が発行されていないこと」の対だったが、
  `/bc-authorize` は `user_code` も `client_notification_token` も無視して 200 を返し、
  `backchannel_token_delivery_mode` を名指した動的クライアント登録も 201 で成功する。
  RFC 6749 §3.1 が未知のリクエストパラメーターを無視することを求めている以上、これは
  宣言した採用の未達ではないので、観測の型を「能力を広告していないこと」と「機能を
  名指しても結果が名指さない場合と一致すること」の対へ決め直した。
  `CIBA-CORE-SIGNED-REQUEST` だけは拒否が実際に起きるので拒否の型で読み、平文と矛盾する
  `request` を添えたとき平文側が効くことで、JWT を読んで採用しない実装と、そもそも
  読んでいない実装とを区別できるようにした。
  ポーリングの 2 行は時計を差し替えられない入口を相手にする。有効期間の幅を保ったまま
  レコード全体を過去へずらす形にしたので、製品が作らないレコードを作ることなく、
  `slow_down` が終端でないことと期限切れの両方を 1 本のコードで読めている。
  `CIBA-CORE-BINDING-MESSAGE` は「運ぶこと」と「見せること」の 2 つを言っているので、
  承認一覧 API と承認画面の 2 か所に同じ id を対応付けた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`。
  - **Requirement**: N/A: 製品の振る舞いを変えないため、対応する製品要求を持たない。
  - **Observed Failure**: 8 件を台帳から先に外した状態で、`RFC8628-DEVICE-AUTHORIZATION`、
    `RFC8628-POLLING`、`CIBA-CORE-BACKCHANNEL-REQUEST`、`CIBA-CORE-POLL-MODE`、
    `CIBA-CORE-BINDING-MESSAGE`、`CIBA-CORE-PING-PUSH`、`CIBA-CORE-USER-CODE`、
    `CIBA-CORE-SIGNED-REQUEST` を `docs/contexts/oauth2/standards.md` の行番号つきで
    `is declared, but no test names it` と 1 件ずつ報告した。
  - **Detection Reason**: 検査は台帳の縮小とテストによる名指しを別々に読む。台帳だけを縮めた状態で
    RED になることを先に見ておけば、注記を足さずに件数を減らす誤りをこの検査が捕まえると確かめられる。
- **Unit RED Evidence**:
  - **Test**: `TestDeviceCodePollingSemantics`
    (`backend/shared/http/server_http/non_interactive_grant_standards_test.go`)。
  - **Requirement**: N/A: 既存の振る舞いに観測を対応付ける作業であり、新しい製品要求を持たない。
  - **Observed Failure**: `ExchangeDeviceCode` から最終ポーリング時刻と間隔の比較を外すと
    `間隔を空けない 2 回目: error=authorization_pending, want slow_down` で落ちた。
    有効期限の比較を外すと `期限切れの device_code: 拒否されるべき要求が 200 で通った` で落ちた。
  - **Detection Reason**: 3 つのエラーは 3 つとも「まだ出せない」ことを言うが、機械が次に取る
    行動が違う。同じ待機状態から間隔だけを変えて 3 つに割れることを 1 本の `device_code` で読む。
    別々のコードで読むと、コードが最初から違っていた場合と区別できない。
- **Change-Resistance Results**:
  行ごとに、その行が言っている判断を production 側で崩し、対応するテストが落ちることを観測した。
  いずれも観測後に元へ戻している。

  | 行 | 注入した故障 | 落ちたテスト |
  |---|---|---|
  | `RFC8628-DEVICE-AUTHORIZATION` | `verification_uri` を応答に載せない | `TestDeviceAuthorizationIssuesCodesAndTakesTheResourceOwnerDecision` |
  | `RFC8628-DEVICE-AUTHORIZATION` | 承認画面の拒否を承認として扱う | `TestDeviceAuthorizationIssuesCodesAndTakesTheResourceOwnerDecision` |
  | `RFC8628-POLLING` | 最終ポーリング時刻と間隔の比較を外す | `TestDeviceCodePollingSemantics` |
  | `RFC8628-POLLING` | `device_code` の有効期限の比較を外す | `TestDeviceCodePollingSemantics` |
  | `CIBA-CORE-BACKCHANNEL-REQUEST` | ヒントがちょうど一方であることの検証を外す | `TestBackchannelAuthenticationRequestResolvesExactlyOneHintForAnAuthenticatedClient` |
  | `CIBA-CORE-BACKCHANNEL-REQUEST` | `scope` が `openid` を含むことの検証を外す | `TestBackchannelAuthenticationRequestResolvesExactlyOneHintForAnAuthenticatedClient` |
  | `CIBA-CORE-POLL-MODE` | ポーリング間隔の判定を常に偽にする | `TestCibaPollModeSemanticsAndSingleUseExchange` |
  | `CIBA-CORE-POLL-MODE` | 消費で状態を進めない | `TestCibaPollModeSemanticsAndSingleUseExchange` |
  | `CIBA-CORE-BINDING-MESSAGE` | 承認一覧の応答から `binding_message` を落とす | `TestApprovalListCarriesBindingMessageWithTheRequestItBinds` |
  | `CIBA-CORE-BINDING-MESSAGE` | 承認画面が `binding_message` を描かない | `AccountApprovalsPresentation > shows all information needed to identify the requested action` |
  | `CIBA-CORE-PING-PUSH` | Discovery で `ping` と `push` を広告する | `TestPingPushDeliveryAndUserCodeAreNotOffered` |
  | `CIBA-CORE-USER-CODE` | Discovery で `user_code` 対応を広告する | `TestPingPushDeliveryAndUserCodeAreNotOffered` |
  | `CIBA-CORE-SIGNED-REQUEST` | `/bc-authorize` が `request` の payload を読んで採用する | `TestSignedBackchannelRequestObjectIsNotAccepted` |

  この方法の限界は、注入が手書きの代表例である点にある。行が言っていない振る舞いの退行は、
  ここで選んだ 13 件の注入では捕まえられない。
- **Verification Results**:
  - `mise run verify` - passed
