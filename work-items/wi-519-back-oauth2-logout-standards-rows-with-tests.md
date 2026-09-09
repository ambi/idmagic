---
depends_on:
  - wi-257-oidc-front-back-channel-logout-notifications
  - wi-524-reject-incomplete-logout-id-token-hints
status: in_progress
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
    - IdMagic.Contract.EndSessionParameters
    - IdMagic.Contract.EndSession1
    - IdMagic.Contract.CheckSessionIframe
  source:
    - backend/shared/spec/discovery.go
    - backend/oauth2/handlers_http/routes.go
    - backend/oauth2/handlers_http/end_session_handler.go
    - backend/oauth2/token/usecases/end_session.go
    - backend/oauth2/ports/id_token_hint_verifier.go
    - backend/shared/security/tokens_jose/jwt_signer.go
    - backend/oauth2/handlers_http/check_session_iframe_handler.go
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/shared/spec/discovery_test.go
    - backend/oauth2/handlers_http/discovery_handler_test.go
    - backend/oauth2/handlers_http/end_session_handler_test.go
    - backend/oauth2/handlers_http/end_session_hint_test.go
    - backend/oauth2/token/usecases/end_session_test.go
    - backend/shared/security/tokens_jose/jwt_signer_endsession_test.go
    - backend/oauth2/handlers_http/check_session_iframe_handler_test.go
  stop_before_reading:
    - backend/jobs
    - backend/oauth2/db_postgres
    - frontend
---

# ログアウトとセッション管理が宣言する標準 9 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-499-back-oauth2-standards-rows-with-tests]] は `docs/contexts/oauth2/standards.md` の 80 行を引き取り、最初の節（`OAuth Client ID Metadata Document` の 7 行）を消化したうえで、残る 73 行を**行が共有する製品の入口**を単位に 7 件へ割った。本項目はそのうち 9 行を持つ。

9 行は、セッションを終わらせる経路を定める。`id_token_hint` の検証、`post_logout_redirect_uri` の登録値限定、front-channel の `iframe`、back-channel のログアウトトークンと再試行がここに集まる。8 行が `required` であり、この文書の中でもっとも `required` に偏った 9 行である。

## Scope

- 次の 9 行を消化する。

| ID | Adoption | 節 |
|---|---|---|
| `OIDC-LOGOUT-ENDPOINT` | required | OpenID Connect RP-Initiated Logout 1.0 |
| `OIDC-LOGOUT-REDIRECT` | required | OpenID Connect RP-Initiated Logout 1.0 |
| `OIDC-LOGOUT-ID-TOKEN-HINT` | required | OpenID Connect RP-Initiated Logout 1.0 |
| `OIDC-FRONTCHANNEL-IFRAME` | required | OpenID Connect Front-Channel Logout 1.0 |
| `OIDC-FRONTCHANNEL-BEST-EFFORT` | required | OpenID Connect Front-Channel Logout 1.0 |
| `OIDC-BACKCHANNEL-LOGOUT-TOKEN` | required | OpenID Connect Back-Channel Logout 1.0 |
| `OIDC-BACKCHANNEL-DELIVERY-RETRY` | required | OpenID Connect Back-Channel Logout 1.0 |
| `OIDC-BACKCHANNEL-REPLAY` | required | OpenID Connect Back-Channel Logout 1.0 |
| `OIDC-SESSION-MGMT-CHECK-IFRAME` | optional | OpenID Connect Session Management 1.0 |

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

**観測の入口は `end_session` エンドポイントと、front-channel / back-channel の配信経路 である。** [[wi-499-back-oauth2-standards-rows-with-tests]] が 7 行を測って出した結論は、消化の費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点にある。本項目の 9 行はこの入口を共有するので、ハーネスは 1 度組めば足りる。

この入口の観測は、ログアウトを起こしてから、ローカルセッションの状態と外部への配信の両方を読む形になる。`OIDC-FRONTCHANNEL-BEST-EFFORT` と `OIDC-BACKCHANNEL-DELIVERY-RETRY` はどちらも「配信結果に依存させない」ことを言っているので、配信を失敗させたうえでローカルの失効が成立していることを読む。配信先は、宛先・メソッド・本文を記録する代役で足りる。

`OIDC-BACKCHANNEL-LOGOUT-TOKEN` は `Statement` が claim の列挙と署名の 2 つを言っているので、観測も 2 つ要る。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

### 着手時に確認した実装状況

9 行を製品の入口と既存テストへ照合した結果、現状のまま消化できるのは 3 行である。

| ID | 状況 | 扱い |
|---|---|---|
| `OIDC-LOGOUT-ENDPOINT` | Discovery の広告と `/end_session` の経路が存在する。 | 本項目でテストを対応付ける。 |
| `OIDC-LOGOUT-REDIRECT` | 登録値との完全一致と未登録値の拒否が存在する。 | 本項目でテストを対応付ける。 |
| `OIDC-LOGOUT-ID-TOKEN-HINT` | 署名、発行者、audience は検証するが、空の `sub` と `sid` を受理し、`sub` と対象 LoginSession の主体も照合しない。 | 欠陥を [[wi-524-reject-incomplete-logout-id-token-hints]] へ切り出す。 |
| `OIDC-FRONTCHANNEL-IFRAME` | 配信先を算出して応答へ埋め込む実装が無い。 | 既存の [[wi-257-oidc-front-back-channel-logout-notifications]] が実装する。 |
| `OIDC-FRONTCHANNEL-BEST-EFFORT` | front-channel の実装が無い。 | 既存の [[wi-257-oidc-front-back-channel-logout-notifications]] が実装する。 |
| `OIDC-BACKCHANNEL-LOGOUT-TOKEN` | logout token の生成と署名が未実装である。 | 既存の [[wi-257-oidc-front-back-channel-logout-notifications]] が実装する。 |
| `OIDC-BACKCHANNEL-DELIVERY-RETRY` | 永続ジョブによる配信と再試行が未実装である。 | 既存の [[wi-257-oidc-front-back-channel-logout-notifications]] が実装する。 |
| `OIDC-BACKCHANNEL-REPLAY` | logout token の `jti` 発行が未実装である。 | 既存の [[wi-257-oidc-front-back-channel-logout-notifications]] が実装する。 |
| `OIDC-SESSION-MGMT-CHECK-IFRAME` | `/session/check` が `postMessage` に応答し、セッション状態を返す。 | 本項目でテストを対応付ける。 |

したがって、本項目は 2 件の実装 work item を前提条件とし、先に現存する 3 行だけを消化する。
残る 6 行は前提条件の完了後に同じ入口で観測し、9 行が揃うまで本項目を完了にしない。

### 着手前に名指しした RED 検査

本項目は製品の振る舞いを変えないため、製品要求に対する Acceptance RED と Unit RED は持たない。
Acceptance RED の代替は、9 件を台帳から一時的に外した状態での `mise run check-spec` である。
同検査が各 ID を `is declared, but no test names it` として報告すれば、台帳だけを縮める誤りを検出できる。

Unit RED の代替は、対応付けたテストごとの故障注入である。
`OIDC-LOGOUT-ENDPOINT` は Discovery の広告または経路を外し、`OIDC-LOGOUT-REDIRECT` は登録値の完全一致を崩し、`OIDC-SESSION-MGMT-CHECK-IFRAME` は `message` の受信または応答を外して、対象テストが落ちることを観測する。

## Plan

1. 9 行を台帳から一時的に外し、被覆検査の RED を確認する。
2. 現存する 3 行を正式な HTTP 経路から観測し、テストと注記を対応付ける。
3. `optional` の 1 件目で、`postMessage` の受信と OP セッション状態の応答を観測する型を決める。
4. 3 行の production 判断を崩し、対応するテストが落ちることを確認してから台帳から外す。
5. [[wi-257-oidc-front-back-channel-logout-notifications]] と [[wi-524-reject-incomplete-logout-id-token-hints]] の完了後に残る 6 行を同じ手順で消化する。

## Tasks

- [x] T001 [Acceptance] 9 件を台帳から先に外し、`mise run check-spec` が当該 ID ごとに `is declared, but no test names it` を報告することを観測する。
  9 件すべてが個別に報告された。
  未実装の 6 件は台帳へ戻し、実装済みの 3 件だけを RED のまま残してからテストを対応付けた。
  recipe: `mise run check-spec`
- [x] T002 [Harness] Discovery と `/end_session` の正式な HTTP 経路で `OIDC-LOGOUT-ENDPOINT` と `OIDC-LOGOUT-REDIRECT` を観測する。
  Discovery がテナントの issuer 配下の `end_session_endpoint` を広告すること、GET 経路が RP の要求を受け付けること、登録済み URI だけへリダイレクトすることを観測した。
  recipe: `mise run test-go-package -- ./backend/shared/spec`
  recipe: `mise run test-go-package -- ./backend/oauth2/handlers_http`
- [x] T003 [Type] `OIDC-SESSION-MGMT-CHECK-IFRAME` について、`postMessage` の受信と OP セッション状態の応答を観測する `optional` の型を決める。
  `/session/check` の HTML が `message` を受信し、送信元 origin へ `changed` または `unchanged` を返すことを対にして観測する型とした。
  本項目に `excluded` の行は無い。
  recipe: `mise run test-go-package -- ./backend/oauth2/handlers_http`
- [ ] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
  `OIDC-LOGOUT-ENDPOINT`、`OIDC-LOGOUT-REDIRECT`、`OIDC-SESSION-MGMT-CHECK-IFRAME` の 3 件を外した。
  残る 6 件は 2 件の前提 work item の完了を待つ。
  recipe: `mise run check-spec`
- [ ] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
  現存する 3 行について、Discovery 広告、GET 経路、登録値照合、`message` 受信、`postMessage` 応答の 5 件の故障を注入し、対応するテストが落ちることを観測した。
  残る 6 行の故障注入は前提 work item の完了後に行う。
  recipe: `mise run test-go-package -- ./backend/oauth2/handlers_http`
- [x] T006 [Defect] 宣言した採用を満たしていない行を実装 work item へ対応付ける。
  front-channel と back-channel の 5 行は既存の [[wi-257-oidc-front-back-channel-logout-notifications]] が持つ。
  `id_token_hint` の必須 claim とセッション主体の照合は [[wi-524-reject-incomplete-logout-id-token-hints]] へ切り出した。
  recipe: `mise run check-work-items`
- [ ] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 9 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## 作業時間

時刻区間はエージェントの読解と判断を含む経過時間であり、コマンド欄は `/usr/bin/time -p` の `real` である。
キャッシュの状態と sandbox の制約を分けて分析できるように、失敗した実行も残す。

| 細目 | 所要 | 結果 |
|---|---:|---|
| 着手から `initial_context` の読解範囲確定まで | 6 分 1 秒 | `brief`、規約、親項目、標準 9 行、実装入口を読み、未実装 6 行を特定した。 |
| `mise run brief -- wi-519` | 0.87 秒 | 読解候補は空だった。 |
| 仕様変更不要の確認としての `mise run check-spec` | 2.47 秒 | 成功。 |
| `mise run check-api-compat` | 0.98 秒 | 成功。 |
| frontmatter、Design、Plan、Tasks と欠陥項目の編集 | 約 3 分 | `wi-257` と `wi-524` を前提条件として確定した。 |
| `mise run check-work-items` | 2.87 秒 | 成功。 |
| `mise run check-ids` | 0.11 秒 | 成功。 |
| 台帳から 9 件を外した Acceptance RED | 1.17 秒 | 9 件すべてを個別に報告した。 |
| `mise run format-go` | 0.99 秒 | 成功。 |
| `backend/shared/spec` の初回パッケージテスト | 0.43 秒 | sandbox 外の既定 Go キャッシュへ書けず失敗した。 |
| `/tmp` の Go キャッシュによる `backend/shared/spec` のパッケージテスト | 7.10 秒 | 成功。 |
| `backend/oauth2/handlers_http` の sandbox 内パッケージテスト | 16.83 秒 | `httptest` の待受を作れず失敗した。 |
| 同パッケージの sandbox 外再実行 | 6.98 秒 | 成功。承認待ちを含むツール全体の経過は 20.8 秒だった。 |
| Discovery 広告を外す故障注入 | 4.46 秒 | 対象テストが失敗した。 |
| GET `/end_session` の経路を外す故障注入 | 9.82 秒 | 対象テストが失敗した。 |
| 登録済み URI の照合を外す故障注入 | 2.02 秒 | 対象テストが失敗した。 |
| `postMessage` 応答を外す故障注入 | 1.94 秒 | 対象テストが失敗した。 |
| `message` 受信を外す故障注入 | 1.90 秒 | 対象テストが失敗した。 |
| 最終の狭い 2 パッケージ GREEN | 0.29 秒、0.37 秒 | Go のテストキャッシュから成功した。 |
| 3 件消化後の `mise run check-spec` | 2.37 秒 | 成功し、テストが名指す ID は 238 件から 241 件へ増えた。 |

着手から 3 行の消化と故障注入を終えた時点までの経過時間は 15 分 37 秒である。

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。注記に「何を固定しているか」を書かせ、崩して落ちることを T005 で確かめる。
- **ハーネスが本番とずれる。** 入口を通すという方針は、その入口が製品と同じ部品でできているときだけ意味を持つ。[[wi-500-back-api-tokens-standards-rows-with-tests]] はここを一度間違え、製品の欠陥でないものを欠陥として起票した。組み立ては `cmd/internal/bootstrap` が作る形に合わせる。
- **1 行が 2 つのことを言っている。** `Statement` に動詞が 2 つあれば観測も 2 つ要る。
- **配信の成功だけを読んで、失敗時の挙動を読まない。** 2 行が「配信結果に依存させない」ことを言っている。配信を失敗させる事例が要る。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、並行しても衝突はエントリー単位に収まる。自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
