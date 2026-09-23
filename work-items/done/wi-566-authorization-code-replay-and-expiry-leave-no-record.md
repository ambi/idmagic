---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p2
change_kind: bugfix
spec_impact: { kind: none, reason: "EX-OAUTH2-005-07 と EX-OAUTH2-005-08 は既に宣言済みである。実装をその宣言へ合わせる作業であり、規範は動かない。" }
affected_spec:
  - { path: docs/domain/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-005 }
  - { path: docs/domain/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-006 }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例 (EX-OAUTH2-005-07、EX-OAUTH2-005-08、EX-OAUTH2-006-02) の実装をその宣言へ合わせる作業である。TokenRevoked は既存の宣言済みイベント種別であり新設しない。公開契約 (TypeSpec) も運用手順も変わらない。
  references: []
initial_context:
  specification:
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-005
    - docs/domain/oauth2/scenarios.feature.md#REQ-OAUTH2-006
  typespec: []
  source:
    - backend/oauth2/token/usecases/exchange_code.go
    - backend/oauth2/token/usecases/refresh_tokens.go
    - backend/oauth2/token/usecases/revoke_token.go
    - backend/oauth2/token/ports/refresh_token_store.go
    - backend/oauth2/token/db_memory/refresh_tokens.go
    - backend/oauth2/token/db_postgres/refresh_tokens.go
    - backend/oauth2/authorization/ports/authorization_store.go
    - backend/oauth2/authorization/db_memory/authorization_codes.go
    - backend/oauth2/db_postgres/authorization_code_store.go
    - backend/oauth2/db_postgres/refresh_tokens.sql
    - backend/oauth2/db_postgres/authorization_code_store.sql
    - backend/shared/spec/authorization_code_machine.go
    - tools/check/example-coverage-debt.json
  tests:
    - backend/oauth2/token/usecases/exchange_code_test.go
    - backend/oauth2/token/usecases/refresh_tokens_test.go
    - backend/oauth2/token/db_memory/refresh_tokens_test.go
    - backend/oauth2/authorization/db_memory/authorization_codes_test.go
    - backend/oauth2/db_postgres/repositories_test.go
    - backend/oauth2/db_postgres/flow_stores_test.go
    - backend/shared/http/server_http/token_issuance_standards_test.go
    - backend/shared/http/server_http/routes_e2e_test.go
  stop_before_reading: []
primary_use_cases:
  - id: authorization-code-replay-revokes-and-notifies
    requirement: REQ-OAUTH2-005
    observable_result: 同じ認可コードを 2 回交換すると 2 回目は invalid_grant で拒否され、発行ファミリーの refresh token がすべて失効し、失効した token ごとに TokenRevoked が発行される。
    unit_test: { path: backend/oauth2/token/usecases/exchange_code_test.go, name: TestExchangeCodeReplayRevokesRefreshFamily, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/token_issuance_standards_test.go, name: TestAuthorizationCodeReplayAtTheRealEntryPointRevokesAndNotifies, task: test-go-race }
    unit_fault_model: RevokeFamily の返り値を使わず固定の TokenID で発行する、またはそもそも TokenRevoked を組み立てない。
    e2e_fault_model: 正式な /token 入口からの再提示が revokeReplayedFamily の配線に届かない、または TokenRevoked の組み立てに届かない。
  - id: expired-authorization-code-transitions-to-expired
    requirement: REQ-OAUTH2-005
    observable_result: 発行から 60 秒を超えた認可コードの交換は invalid_grant で拒否され、記録の状態が issued から expired へ遷移する。
    unit_test: { path: backend/oauth2/token/usecases/exchange_code_test.go, name: TestExchangeCodeRejectsExpiredCode, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/token_issuance_standards_test.go, name: TestExpiredAuthorizationCodeExchangeIsRejectedAtTheRealEntryPoint, task: test-go-race }
    unit_fault_model: rec.State == Issued の判定を素通りして MarkExpired を呼ばない、または replay (既に redeemed/expired) にも誤って適用する。
    e2e_fault_model: 正式な /authorize → /token 入口から expired 判定への配線が届かない。
  - id: refresh-token-reuse-revokes-and-notifies
    requirement: REQ-OAUTH2-006
    observable_result: ローテーション済みの旧 refresh token を再提示すると invalid_grant で拒否され、family が失効し、失効した token ごとに TokenRevoked が発行される。
    unit_test: { path: backend/oauth2/token/usecases/refresh_tokens_test.go, name: TestRefreshTokensReuseRevokesFamilyAndNotifiesTokenRevoked, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/token_issuance_standards_test.go, name: TestRefreshTokenRotatesAndReuseRevokesTheWholeFamily, task: test-go-race }
    unit_fault_model: refresh_tokens.go の再利用検出分岐が revokeFamilyAndNotify を呼ばない、または RevokeFamily の返り値を捨てる。
    e2e_fault_model: 正式な /token (refresh_token grant) 入口からの再提示が再利用検出の配線に届かない。
---

# 認可コードの再交換と期限切れが、宣言どおりの記録を残さない

## Motivation

[[wi-559-back-oauth2-remaining-examples-with-tests]] が REQ-OAUTH2-005 を消化する過程で、実装が具体例の `Then` を 2 つ満たしていないことを測った。

**`EX-OAUTH2-005-07`：** 同じ認可コードを 2 回交換したとき、具体例は「発行ファミリーのトークンがすべて失効する」「`RefreshTokenReuseDetected` が発行される」「`TokenRevoked` が発行される」の 3 つを言う。実測では発行されたイベントは `[AccessTokenIssued AuthorizationCodeRedeemed RefreshTokenIssued]` だけで、いずれも 1 回目の交換によるものだった。2 回目の交換は失効の記録だけを残し、通知を 1 つも出していなかった。

このうち `RefreshTokenReuseDetected` は wi-559 が直した。`refresh_tokens.go` が refresh トークン再利用の検出でまったく同じ組（`RevokeFamily` の直後に `emit`）を持っており、認可コード再提示の経路だけがその発行を欠いていた。既存の写像の欠落だったので、局所の修正として `revokeReplayedFamily` へ寄せた。

**残るのは `TokenRevoked` である。** これは局所では済まない。`RefreshTokenStore.RevokeFamily(ctx, familyID) error` は失効させた token を返さないので、token ごとの `TokenRevoked{TokenID, Reason}` を組み立てる材料が呼び出し側に無い。port の署名を変えるか、失効を通知する別の口を設けるかを決める必要がある。

**`EX-OAUTH2-006-02` も同じ欠落を持つ。** wi-559 が `/token` の入口から測った。ローテーション済みの旧 refresh トークンを再使用すると、`invalid_grant` で拒否され、記録は `Revoked` になり、family も失効し、`RefreshTokenReuseDetected` も発行される。出ないのは `TokenRevoked` だけである。原因は同じ `RevokeFamily` の署名なので、認可コード側と一緒に直る。片方だけ直すと、また片方が黙る。

**`EX-OAUTH2-005-08`：** 発行から 60 秒を超えた認可コードの交換は `invalid_grant` で拒否される（ここは正しい）。しかし具体例が言う「認可コードの状態は Expired になる」が起きない。実測では記録の状態は `issued` のままだった。

`spec.AuthorizationCodeRecordState` は `expired` を持ち、`authorization_code_machine.go` は `{issued, Expire, expired}` の遷移を宣言している。**遷移を実行する製品コードが 1 行も無い。** 宣言だけがあって、誰もその状態へ動かさない。

## Scope

- `EX-OAUTH2-005-07` と `EX-OAUTH2-006-02` の `TokenRevoked` を発行できるようにする。`RevokeFamily` が失効させた token を報告する形にするか、別の口を設けるかを決め、Design に理由を書く。`db_memory` と `db_postgres` の双方を揃える。
- `EX-OAUTH2-005-08` の `expired` 遷移を実装する。**誰がいつ遷移させるかを先に決める。** 交換の拒否時に遅延で書くのか、掃除の job が持つのかで、監査に残る時刻の意味も、期限切れのまま提示されなかったコードの扱いも変わる。決めた理由を Design に書く。
- 3 件すべてについて、`tools/check/example-coverage-debt.json` の当該行から `blocked_by` と `finding` を外し、id を名指すテストを対応付けて台帳から削除する。
- 認可コード再提示の経路と refresh トークン再利用の経路が、同じ状況で同じイベントを出すことを 1 つのテストで固定する。片方だけ直すと、また片方が黙る。

## Out of Scope

- REQ-OAUTH2-005 のほかの具体例。[[wi-559-back-oauth2-remaining-examples-with-tests]] が持つ。
- `RefreshTokenReuseDetected` の発行。wi-559 が済ませた。
- 認可コードの TTL そのものの変更。60 秒は SCL の不変条件である。
- 監査イベントの一覧や保持期間の変更。
- 一度も提示されなかった期限切れ認可コードを `Expired` へ遷移させること。`DeleteExpiredBatch` の housekeeping が cutoff を超えた行を状態遷移なしで物理削除する既存の挙動は変えない。
- `RevokeFamily` を呼ぶが再提示検出ではない経路 (client 不一致、user 不在/無効化、同時実行によるローテーション喪失、`/revoke` エンドポイントの所有者チェック) のテスト網羅とエラー伝播方針の見直し。

## Design

**`RevokeFamily` の署名。** `RefreshTokenStore.RevokeFamily(ctx, familyID) error` を
`RevokeFamily(ctx, familyID) ([]string, error)` に変える。返り値はこの呼び出しで新たに
`Revoked` へ遷移した `TokenID` の一覧で、呼び出し時点で既に `Revoked` だった行は含めない
(繰り返し呼んでも同じ id を返さない、idempotent な形)。`db_memory` はループで判定し、
`db_postgres` は `UPDATE ... WHERE family_id = $1 AND revoked = FALSE RETURNING id::text`
(`:many`) にして `sort.Strings` で順序を揃える (`RETURNING` の行順は未規定なので)。

失効そのもの (`revoked = TRUE` への一括更新) と、失効した token を報告することは別の
関心であり、別々に観測する: `db_memory`/`db_postgres` それぞれに、新たに失効した id が
返ること、別 family には波及しないこと、再呼び出しでは空を返すこと (idempotency) を
固定する単体テストを置いた。

**通知の組み立てを 1 か所へ集める。** 認可コード再提示 (`exchange_code.go` の
`revokeReplayedFamily`) と refresh トークン再利用 (`refresh_tokens.go` の
`IsRefreshTokenReplay` 分岐) は同じ状況 (発行ファミリーの盗用検知) を表すので、
`RevokeFamily` の返り値から `TokenRevoked` を組み立てる部分を `revokeFamilyAndNotify`
(`refresh_tokens.go`、パッケージ内共有) へまとめ、両方の経路がこの一つの実装を呼ぶ。
`RefreshTokenReuseDetected` は経路固有の検出通知なので、各経路がそれぞれ発行する。
`TestReplayDetectionEmitsTheSameEventTypesAcrossAuthorizationCodeAndRefreshTokenPaths`
が、2 経路の再提示検出イベント種別の集合が一致することを固定する (Scope の 4 点目)。

**`/revoke` (RFC 7009) も同じ返り値を使う。** `revoke_token.go` の `RevokeToken` は
signature 変更に合わせて調整するだけでなく、返ってきた id ごとに `TokenRevoked` を
発行するよう直した。1 token だけを対象にした従来の呼び出しでは挙動は変わらないが、
family に複数の既存行がある場合により正確になる。署名変更で無料に手に入る修正であり、
別の観測境界を要する新機能ではないため、この wi の範囲に含めた。

**`RevokeFamily` を呼ぶがどれもここに挙げた 2 経路ではない箇所**
(`refresh_tokens.go` の client 不一致・user 不在・user 無効化・同時ローテーション喪失の
4 箇所) は、返り値を `_, _ =` で無視するだけに留めた。いずれも再提示検出ではなく別の
拒否理由であり、Scope が名指す 2 つの `EX-*` のどちらにも該当しない。ここへ
`TokenRevoked` を足すことは新しい規範の主張になるので、この bugfix の範囲外とする。
エラーの握り潰し (`_ = ...RevokeFamily(...)`) も、この 4 箇所と `revokeFamilyAndNotify`
の内部では変更前と同じ best-effort のまま維持した。エラー伝播方針の見直しは、失効の
成否がクライアントへ返す OAuth エラーの種別に波及する設計判断であり、この wi の対象
(通知の欠落) とは別の問題である。

**`expired` 遷移の実行者とタイミング。** `AuthorizationCodeStore` に
`MarkExpired(ctx, code) (*domain.AuthorizationCodeRecord, error)` を追加する。
`Redeem` と同じ CAS の形 (`state = 'issued'` の行だけを対象に 1 行更新、対象がなければ
`nil`) を取る。`ExchangeCodeForToken` は、コードが `issued` のまま `ExpiresAt` を過ぎて
「いま提示された」場合だけこれを呼ぶ。既に `redeemed`/`expired` の行 (genuine replay) は
対象にしない — 状態はもう動かないので、`revokeReplayedFamily` だけが処理する。

交換の拒否時に遅延で書く方式を、既存の掃除 job
(`AuthorizationCodeStore.DeleteExpiredBatch`) に持たせる方式より採った。理由は 3 つ:
(1) `EX-OAUTH2-005-08` が求める観測は「提示されたときに `Expired` になる」であり、遅延
書き込みはこの観測をそのまま満たす。(2) `DeleteExpiredBatch` は状態遷移を経ない純粋な
物理削除で、リクエスト文脈も持たない全体 housekeeping であり、これへ状態遷移を足すのは
本 bugfix より大きい変更になる。(3) 一度も提示されなかった期限切れコードは、この方式
では `issued` のまま `DeleteExpiredBatch` の cutoff を超えるまで残り、`Expired` へは
遷移せずに消える。この帰結は Risk Notes と Out of Scope に明記し、削除自体の挙動
(データが残り続けない) は変えていない。

エラー処理は `MarkExpired` の呼び出し側でも `_, _ =` の best-effort とした。他の
「認可コード拒否経路の副作用」(`revokeReplayedFamily` を含む) と同じ扱いに揃えるためで
あり、ここだけ強い失敗を返すと拒否応答の一貫性が崩れる。

## Plan

設計判断はすべて実装前に決め切ったため、実装順序だけを残す。

1. `RevokeFamily` の port/db_memory/db_postgres (sqlc 再生成含む) を先に直し、
   `mise run lint-go` で配線を通す (構造変更)。
2. `revokeFamilyAndNotify` を追加し、`exchange_code.go` と `refresh_tokens.go` の
   再提示検出分岐をそこへ寄せる。`revoke_token.go` も新しい返り値を使うよう直す
   (振る舞いの変更)。
3. `AuthorizationCodeStore.MarkExpired` を port/db_memory/db_postgres (sqlc 再生成含む)
   に追加し、`ExchangeCodeForToken` の拒否分岐から呼ぶ (振る舞いの変更)。
4. 単体テスト (`db_memory`、`db_postgres`、`usecases`) と E2E テスト
   (`server_http` の実サーバー経由) を、宣言済み `EX-*` id を `//spec:covers` で claim
   しながら追加する。
5. `tools/check/example-coverage-debt.json` から 3 行を外し、`mise run check-spec` を
   通す。
6. `mise run test-go-mutation` で変更した usecases/db_memory パッケージの生存変異を
   読み、`mise run verify` を通す。

## Tasks

- [x] T001 [Port] `RefreshTokenStore.RevokeFamily` の署名を `([]string, error)` に変え、
      `db_memory`・`db_postgres` (sqlc 再生成) を揃える。新たに失効した id の報告と
      idempotency を単体テストで固定する。
- [x] T002 [Use Cases] `revokeFamilyAndNotify` を追加し、`exchange_code.go` の
      `revokeReplayedFamily` と `refresh_tokens.go` の再利用検出分岐から呼ぶ。
      単体テストと実サーバー経由のテストで `TokenRevoked` の発行を固定する
      (EX-OAUTH2-005-07, EX-OAUTH2-006-02)。
- [x] T003 [Use Cases] `revoke_token.go` の `RevokeToken` を新しい返り値へ揃える。
- [x] T004 [Port] `AuthorizationCodeStore.MarkExpired` を追加し (`db_memory`・
      `db_postgres` sqlc 再生成)、`ExchangeCodeForToken` の拒否分岐から呼ぶ。
      単体テストと実サーバー経由のテストで `Expired` への遷移を固定する
      (EX-OAUTH2-005-08)。
- [x] T005 [Acceptance] 認可コード再提示と refresh トークン再利用が同じ状況で同じ
      イベント種別集合を出すことを
      `TestReplayDetectionEmitsTheSameEventTypesAcrossAuthorizationCodeAndRefreshTokenPaths`
      で固定する。
- [x] T006 [Spec] `tools/check/example-coverage-debt.json` から対象 3 行を外し、
      `mise run check-spec` を通す。
- [x] T007 [Verify] `mise run test-go-mutation` で変更パッケージの生存変異を読み、
      `mise run verify` を通す。

## Verification

- `mise run check-spec` — `EX-OAUTH2-005-07`、`EX-OAUTH2-005-08`、`EX-OAUTH2-006-02` を
  台帳から外した状態で通ることを確認した。
- `mise run test-go-package -- ./backend/oauth2/token/usecases`
- `mise run test-go-package -- ./backend/oauth2/token/db_memory`
- `mise run test-go-package -- ./backend/oauth2/authorization/db_memory`
- `mise run test-go-package -- ./backend/oauth2/db_postgres`
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run test-go-changed`
- `mise run test-go-mutation -- backend/oauth2/token/usecases`
- `mise run test-go-mutation -- backend/oauth2/token/db_memory`
- `mise run test-go-mutation -- backend/oauth2/authorization/db_memory`
- `mise run lint-go`
- `mise run verify`

## Risk Notes

- **失効の記録と通知を、片方だけで満足する。** この欠陥がそもそも「失効はするが黙っている」形だった。テストは保存先の状態とイベントの双方を読む。
- **`expired` 遷移を交換の拒否時にだけ書いて、済んだことにする。** それでは一度も提示されなかった期限切れコードは永久に `issued` のまま残る。決めた方式がその場合に何を残すかを、Design に書いて観測する。
- **`RevokeFamily` の署名変更が、失効そのものを弱める。** 返り値を足す変更は `db_postgres` の実装にも及ぶ。失効したことと、失効した token を報告することを、別々に観測する。
- **`_ = deps.RefreshStore.RevokeFamily(...)` がエラーを捨てている。** 現状は失効に失敗しても拒否だけ返る。署名を触るときに、この握り潰しを続けるかどうかも決める。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は「no normative specification change against main」を返す。規範は
  動いていない。動いたのは実装で、`RefreshTokenStore.RevokeFamily` の返り値を
  `error` から `([]string, error)` に変え、新たに `Revoked` へ遷移した `TokenID` を
  報告するようにした (`db_memory`・`db_postgres` 双方、sqlc 再生成込み)。認可コード
  再提示 (`exchange_code.go`) と refresh トークン再利用 (`refresh_tokens.go`) の両経路は
  この返り値を使って `revokeFamilyAndNotify` (パッケージ内共有) から `TokenRevoked` を
  token ごとに発行するようになった。`/revoke` (`revoke_token.go`) も同じ返り値を使うよう
  揃えた。`AuthorizationCodeStore` に `MarkExpired` の CAS 遷移 (`issued -> expired`) を
  追加し、`ExchangeCodeForToken` は期限切れのまま提示された認可コードだけをこれで
  `Expired` に遷移させる。`tools/check/example-coverage-debt.json` から
  `EX-OAUTH2-005-07`、`EX-OAUTH2-005-08`、`EX-OAUTH2-006-02` の 3 行を外した。
  併せて、認可コード再提示と refresh トークン再利用の 2 経路が同じ状況で同じイベント
  種別集合を出すことを固定するテストを追加した (Scope の 4 点目)。この検査は「片方
  だけ直してもう片方が黙る」将来の退行を防ぐものであり、両経路が対称に `TokenRevoked`
  を欠いていた変更前のコードでは (どちらの集合も同じだったため) 偶然 GREEN だった。
  今回の欠落そのものの検出は、下記の Primary Use Case Evidence が個別に担った。
- **Primary Use Case Evidence**:
  - id: authorization-code-replay-revokes-and-notifies
    unit_red: 変更前のコードで TestExchangeCodeReplayRevokesRefreshFamily を実行すると、再交換の検出で TokenRevoked が発行されず RefreshTokenReuseDetected だけになって失敗した。
    e2e_red: 変更前のコードで TestAuthorizationCodeReplayAtTheRealEntryPointRevokesAndNotifies (実サーバー経由) を実行すると、同じく TokenRevoked が発行されず失敗した。
    unit_fault_injection: git diff と git checkout で本番コード (port・db_memory・db_postgres・usecases) だけを変更前へ戻し、新しいテストコードのまま実行して同じ失敗を再現した。戻した本番コードを元に戻すと GREEN に戻った。
    e2e_fault_injection: 同じ本番コードの巻き戻しを実サーバー経由のテストでも再現し、同じ TokenRevoked の欠落で失敗することを確認した。
  - id: expired-authorization-code-transitions-to-expired
    unit_red: 変更前のコードで TestExchangeCodeRejectsExpiredCode を実行すると、記録の状態が issued のまま変わらず expired へ遷移しないため失敗した。MarkExpired が存在せず拒否分岐が状態を書き換えなかったことが原因である。
    e2e_red: 変更前のコードで TestExpiredAuthorizationCodeExchangeIsRejectedAtTheRealEntryPoint は成功した。invalid_grant の応答自体は元から正しく、E2E 層は応答だけを読むためこの具体例の欠落 (内部状態が Expired にならないこと) を検出できない。E2E の fault model は配線の断絶であり、expired へ遷移しない欠陥そのものは Unit 層だけが検出できる。
    unit_fault_injection: 本番コードを変更前へ戻して TestExchangeCodeRejectsExpiredCode を再実行し、同じ失敗を再現した。戻すと GREEN に戻った。
    e2e_fault_injection: 未実施。E2E 層はこの欠陥を判別できないため (e2e_red 参照)、正式な入口からの配線そのものが壊れていないことは、変更後に TestExpiredAuthorizationCodeExchangeIsRejectedAtTheRealEntryPoint が GREEN であることで確認した。
  - id: refresh-token-reuse-revokes-and-notifies
    unit_red: 変更前のコードで TestRefreshTokensReuseRevokesFamilyAndNotifiesTokenRevoked を実行すると、再利用の検出で TokenRevoked が発行されず失敗した。
    e2e_red: 変更前のコードで TestRefreshTokenRotatesAndReuseRevokesTheWholeFamily (実サーバー経由) を実行すると、同じく TokenRevoked が発行されず失敗した。wi-566 の Motivation が引用した実測と同じ結果である。
    unit_fault_injection: 本番コードを変更前へ戻して同テストを再実行し、同じ失敗を再現した。戻すと GREEN に戻った。
    e2e_fault_injection: 同じ本番コードの巻き戻しを実サーバー経由のテストでも再現し、同じ TokenRevoked の欠落で失敗することを確認した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/oauth2/token/usecases`: Killed 188 / Lived 21 /
  Not covered 5 / Not viable 2 (efficacy 89.95%)。この wi で新設・変更したコード
  (`revokeFamilyAndNotify`、`ExchangeCodeForToken` の `MarkExpired` 分岐、
  `RevokeToken` の返り値ループ) に属する変異はすべて Killed だった。生存した 21 件は
  いずれもこの wi が触れていない行 (`exchange_code.go` の DPoP/mTLS 分岐、
  `refresh_tokens.go` の resource indicator 計算、`revoke_token.go` の
  `revokeAccessToken` と所有者チェック) にあり、変更前から存在する別のギャップである。
  `mise run test-go-mutation -- backend/oauth2/token/db_memory`: Killed 4 / Lived 0 /
  Not covered 2。`RevokeFamily` に属する変異はすべて Killed。Not covered の 2 件は
  この wi が触れていない `RevokeBySid` にある。
  `mise run test-go-mutation -- backend/oauth2/authorization/db_memory`: Killed 7 /
  Lived 0 / Not covered 0 (efficacy 100%)。`MarkExpired` を含む本パッケージ全体が
  完全に被覆された。
  `db_postgres` は `mise run test-go-mutation` のカバレッジ収集が 0% を返し (`pgtest`
  が使う組み込み PostgreSQL がカバレッジ計測の実行経路に乗らないためと見られる、
  この wi が触れていないパッケージ全体の既存の制約)、変異ではなく手作業の CAS/
  idempotency 観測 (`go test` ベースの `TestAuthorizationCodeStore`、
  `TestRefreshTokenStoreRevokeFamily_ReportsNewlyRevokedIDs`) で代替した。
- **Verification Results**:
  - `mise run check-spec` - 成功 (`EX-OAUTH2-005-07`、`EX-OAUTH2-005-08`、
    `EX-OAUTH2-006-02` を台帳から外した状態で通った)
  - `mise run test-go-package -- ./backend/oauth2/token/usecases` - 成功
  - `mise run test-go-package -- ./backend/oauth2/token/db_memory` - 成功
  - `mise run test-go-package -- ./backend/oauth2/authorization/db_memory` - 成功
  - `mise run test-go-package -- ./backend/oauth2/db_postgres` - 成功
  - `mise run test-go-package -- ./backend/shared/http/server_http` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功 (0 issues)
  - `mise run verify` - 成功
