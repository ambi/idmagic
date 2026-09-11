---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。
  references: []
initial_context:
  specification:
    - docs/contexts/ws-federation/standards.md
  typespec: []
  source:
    - backend/wsfederation/domain/wsfed.go
    - backend/wsfederation/usecases/signin.go
    - backend/wsfederation/handlers_http/routes.go
    - backend/wsfederation/handlers_http/wsfed_handler.go
    - backend/wsfederation/handlers_http/wstrust_handler.go
    - backend/wsfederation/handlers_http/metadata_handler.go
    - backend/wsfederation/requests_wstrust/rst.go
    - backend/wsfederation/responses_wsfederation/rstr.go
    - backend/wsfederation/tokens_saml/assertion.go
    - backend/wsfederation/metadata_wsfederation/federation_metadata.go
    - tools/check/src/normative-coverage.ts
    - tools/check/src/check-documents.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/wsfederation/handlers_http/wsfed_handler_test.go
    - backend/wsfederation/domain/wsfed_test.go
    - backend/wsfederation/domain/wsfed_fuzz_test.go
    - backend/wsfederation/requests_wstrust/rst_fuzz_test.go
    - backend/wsfederation/responses_wsfederation/rstr_test.go
    - backend/wsfederation/tokens_saml/assertion_test.go
    - backend/wsfederation/metadata_wsfederation/federation_metadata_test.go
  stop_before_reading:
    - backend/saml
    - backend/oauth2
    - frontend
---

# WsFederation が宣言する標準 6 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/ws-federation/standards.md` の 6 行を引き取る。**この文書だけが 6 行すべて名指しを持たない。** 台帳の中で全滅している唯一の文書である。

一方で `backend/wsfederation` にはテストが揃っている。`wsfed_handler_test.go`、`federation_metadata_test.go`、`rstr_test.go`、`assertion_test.go`、`wsfed_test.go`、それに `rst_fuzz_test.go` と `wsfed_fuzz_test.go` がある。つまり 6 行の全滅は、検証が無いからではなく、名指しが無いからである可能性が高い。**それでも注記だけを足す解消は認めない。** 既存のテストが行の `Statement` を区別できているかを読み、区別できていない行にはテストを足す。

## Scope

- 次の 6 行を消化する。

| ID | Adoption |
|---|---|
| `WSFed-PassiveSignIn` | required |
| `WSTrust13-IssueBearer` | required |
| `WSS-UsernameTokenPassword` | required |
| `WSAddressing-MessageIDToAction` | required |
| `WSFed-SilentSignIn` | excluded |
| `WSTrust13-WindowsTransport` | excluded |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- WS-Federation の適合範囲そのものの拡張。`excluded` の 2 行を提供へ変えるのは規範の変更であり、別の work item が扱う。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`WSFed-PassiveSignIn` の `Statement` は「登録済み `wtrealm` と許可済み `wreply` にだけトークンを返す」である。この行が固定しているのは成功経路ではなく、**許可されていない `wtrealm` や `wreply` へはトークンが出ないこと**である。したがって観測は 2 つ要る。登録済みの組で通ること、および未登録の `wtrealm` と、登録済み `wtrealm` に対する許可外の `wreply` のそれぞれでトークンが出ないこと。成功経路だけを観測すると、照合が無い実装と区別できない。

`WSAddressing-MessageIDToAction` は `MessageID`、`To`、`Action` の 3 つの検証を 1 行にまとめている。`MessageID` はリプレイ防止のための検証なので、観測は「同じ `MessageID` の 2 度目が通らないこと」であり、値が読めることではない。3 つを 1 つの入力で崩すと、手前の検証で落ちて後段が確かめられない。1 つずつ崩す。

`WSS-UsernameTokenPassword` は能動的 STS の認証である。誤ったパスワードでトークンが出ないことと、その拒否が防いだ効果を観測する。

`WSTrust13-IssueBearer` は RSTR に Bearer の SAML アサーションが載ることである。アサーションの `SubjectConfirmation` が Bearer であることまで読む。

`excluded` の 2 行は逆向きの観測になる。`WSFed-SilentSignIn` は無音認証を提供しないという行であり、`wsignin1.0` に無音を求めるパラメータを付けた要求が無音では通らないことを観測する。`WSTrust13-WindowsTransport` は WindowsTransport / Kerberos の能動的プロファイルを提供しないという行であり、その入口が存在しないことを観測する。観測の型は 1 件目で決めてからもう 1 件へ広げる。

### T001 の棚卸し

既存の 6 テストファイルを読み、行ごとに「どこまで区別できているか」を書き出した。

| ID | 既存テストが届いている範囲 | 届いていない範囲 |
|---|---|---|
| `WSFed-PassiveSignIn` | 成功経路（`TestWsFedSignIn_AuthenticatedIssuesPassiveForm` が RP と同じ手順で wresult の署名まで検証）。未登録 `wtrealm` と許可外 `wreply` はどちらも 400。domain 側に `TestValidateSignIn_Rejections` と `FuzzParseSignInRequest` がある | **拒否側が「トークンが出ていない」ことを観測していない。** どちらも状態符号しか見ておらず、400 を返しつつ wresult を本文に載せる実装と区別できない。`TestWsFedSignIn_DisallowedWreplyRejected` は event すら見ていない。成功側も許可済み `wreply` を明示した組を HTTP 境界で通していない（既定の先頭 URL だけ） |
| `WSTrust13-IssueBearer` | `TestWsTrustUsernameMixed_IssuesRSTR` が RSTR の外形（`RequestSecurityTokenResponseCollection`、`RequestedSecurityToken`、TokenType、AppliesTo）を観測。`TestWsTrustUsernameMixed_RejectsNonBearerKeyType` が Bearer 以外の KeyType を拒否 | **Bearer であることを観測していない。** RSTR 本文にも `assertion_test.go` にも `SubjectConfirmation` の確認が 1 箇所も無い。`bearerConfirmation11` / `bearerConfirmation20` を定数ごと消しても、既存テストは 1 件も落ちない |
| `WSS-UsernameTokenPassword` | 正しいパスワードの成功経路だけ | **誤ったパスワードのテストが存在しない。** `authenticateWsTrustUser` の検証結果を無視して常に成功させても、既存テストは全部通る。拒否が防いだ効果（トークンが出ない）も当然観測が無い |
| `WSAddressing-MessageIDToAction` | `MessageID` のリプレイ（`TestWsTrustUsernameMixed_RejectsExpiredTimestampAndReplay`）と `To` の不一致（`TestWsTrustUsernameMixed_RejectsMismatchedTo`） | **`Action` の検証にテストが無い。** `rst.go` には `rst_test.go` が無く、fuzz target の oracle は「拒否時に値を持ち出さない」ことなので、`Action != Issue` を通す実装を検出しない。3 つとも拒否側の観測が状態符号止まりでもある |
| `WSFed-SilentSignIn` | `TestWsFedSignIn_UnsupportedWauthRejected` が統合 Windows 認証の `wauth` を 400 で拒否 | **`excluded` としての観測になっていない。** 無音で発行されないことも、無音の失敗応答に化けないことも見ていない。未認証で無音を求める入力（`prompt=none` 相当）を置いたテストが無い |
| `WSTrust13-WindowsTransport` | 無し。`TestTrustMEX_Published` は usernamemixed の広告を観測するだけで、WindowsTransport の不在は見ていない | **入口が存在しないことの観測が無い。** `/trust/13/windowstransport` を足しても、metadata に Kerberos の binding を足しても、既存テストは 1 件も落ちない |

つまり 6 行すべてが、名指しが無いだけでなく**区別できていない**。注記だけで消化できる行はゼロである。

### `excluded` の観測の型（T005 が決めた）

`WSFed-SilentSignIn` で決め、`WSTrust13-WindowsTransport` へ広げた。`excluded` の行は 2 つの形のどちらかを
取り、観測はその形で決まる。

| 形 | 観測 |
|---|---|
| 入口はあるが、その機能の要求を拒否する | 拒否そのものと、拒否が防いだ効果（トークンが出ていないこと）。加えて、**その拒否が無音の失敗応答に化けないこと**。「無音では発行しないが無音で静かに失敗する」実装は、RP から見れば無音の失敗が成立しているので、行を満たしていない |
| 入口そのものを持たない | 要求の宛先が存在しないこと（404）と、**metadata がその binding を広告しないこと**。入口が無くても広告が残っていれば、RP はその機能が提供されていると読んで要求を組み立てる |

`WSFed-SilentSignIn` は前者である。無音を求める入力は 2 種類あり、`prompt=none` は未認証のまま対話的な
ログインへ誘導され、無音の統合 Windows 認証を求める `wauth` は 400 で拒否される。後者は「実施済みの
パスワード認証で黙って代用しない」ことまで見る。代用は RP から見れば無音認証が成立したのと同じ意味になる。

`WSTrust13-WindowsTransport` は後者である。AD FS が能動的 Windows / Kerberos プロファイルに用いる 3 つの
入口がいずれも 404 であること、および MEX と federation metadata のどちらにも `WindowsTransport` と
`Kerberos` が現れないことを観測する。

### 意図した RED 検査

- **Acceptance RED**: `mise run check-spec`。6 件を `tools/check/standards-coverage-debt.json` から先に外し、
  テストを書く前の状態で走らせる。6 行それぞれについて `<ID> is declared, but no test names it.` が出ることを
  観測する。製品の規範要求に対応する受入境界はこの作業に無いので、Requirement は N/A である。
- **Unit RED**: 各行に対応付けたテストへの故障注入。行が言っていることを production 側で崩し、対応するテストが
  落ちることを 1 件ずつ観測する。`checkNormativeCoverage` は文字列の一致しか見ないので、名指しが実在の検証に
  付いていることの担保はこの観測しかない。recipe は
  `mise run test-go-test -- ./backend/wsfederation/handlers_http <Test>` と
  `mise run test-go-package -- ./backend/wsfederation/...`。

## Plan

1. 既存の 6 つのテストファイルを読み、どの行がどこまで区別されているかを行ごとに書き出す。
2. `WSFed-PassiveSignIn` を、許可外の `wtrealm` と `wreply` でトークンが出ないことまで含めて消化する。
3. `WSTrust13-IssueBearer` と `WSS-UsernameTokenPassword` を消化する。
4. `WSAddressing-MessageIDToAction` を、3 つの検証を 1 つずつ崩す形で消化する。
5. `WSFed-SilentSignIn` で `excluded` の観測の型を決め、`WSTrust13-WindowsTransport` へ広げる。
6. 解決した id を台帳から外す。

## Tasks

- [x] T001 [Inventory] 既存テストが 6 行それぞれをどこまで区別しているかを書き出す。
  結果は Design の「T001 の棚卸し」節。**6 行のうち注記だけで済む行は 1 つも無かった。**
- [x] T002 [Acceptance] `WSFed-PassiveSignIn` を、許可外の `wtrealm` / `wreply` でトークンが出ないことまで消化する。
  `TestWsFedPassiveSignIn_IssuesOnlyToTheRegisteredRealmAndAllowedReply`（`wsfed_handler_test.go`）。
  共有フィクスチャの RP に 2 つ目の返信先を足し、指定した許可済み `wreply` へ届いたのか既定の先頭が
  使われただけなのかを区別できるようにした。拒否側は `assertNoPassiveTokenIssued` で応答本文と event の
  双方を見る。再実行の recipe は
  `mise run test-go-test -- ./backend/wsfederation/handlers_http TestWsFedPassiveSignIn_IssuesOnlyToTheRegisteredRealmAndAllowedReply`。
- [x] T003 [Acceptance] `WSTrust13-IssueBearer` と `WSS-UsernameTokenPassword` を消化する。
  `TestWsTrustIssueBearer_ReturnsABearerSAMLAssertionInTheRSTR` と
  `TestBuildAssertion_ConfirmsTheSubjectAsBearerInBothVersions`（`assertion_test.go`）、および
  `TestWsTrustUsernameTokenPassword_AuthenticatesTheSuppliedCredential`。**Bearer は 2 箇所に名指しを
  置いた。** HTTP 境界は RP の既定 token type である SAML 1.1 しか通らないので、SAML 2.0 の枝を保持者証明へ
  変えても境界側では気づけない（故障 F6 がそれを示す）。
- [x] T004 [Acceptance] `WSAddressing-MessageIDToAction` を、3 つの検証を 1 つずつ崩して消化する。
  `TestWsTrustAddressing_ValidatesMessageIDToAndAction`。崩していない要求が通ることを最初の部分試験で
  先に確認し、`MessageID`・`To`・`Action` を 1 つずつ崩す。`Action` を崩すときは `RequestType` を Issue の
  まま残し、崩した検証だけが理由になるようにしている。
- [x] T005 [Type] `WSFed-SilentSignIn` で `excluded` の型を決め、`WSTrust13-WindowsTransport` へ広げる。
  `TestWsFedSilentSignIn_NotProvided` と `TestWsTrustWindowsTransport_NotProvided`。型は Design の
  「`excluded` の観測の型」節。
- [x] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
  6 件を削除し、台帳は 17 件から 11 件になった。コメント配列と他の id には触れていない。
- [x] T007 [Verify] `mise run verify`。
  exit 0。詳細は Completion の Verification Results。

## Verification

- 本項目が持つ 6 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `docs/contexts/ws-federation/standards.md` の 6 行に対する被覆の状態である。6 行すべてが
  その行の `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json`
  から消え、台帳は 17 件から 11 件になった。名指しを持つ id は 270 件から 276 件へ増えた。
  新設したテストは Go 6 件（部分試験を数えると 14 件）で、**製品コードは 1 行も変わっていない**。
  **この文書は「テストが揃っているので注記だけで終わる」という Risk Notes の予想が外れた。**
  6 ファイル 14 テストが既にあったが、T001 の棚卸しでは 6 行すべてが区別できていなかった。
  `WSS-UsernameTokenPassword` には誤ったパスワードのテストが存在せず、`WSTrust13-IssueBearer` は
  `SubjectConfirmation` を 1 箇所も読んでおらず、`WSAddressing-MessageIDToAction` の 3 つのうち
  `Action` には例示テストが無く、`excluded` の 2 行は観測が逆向きになっていなかった。
  注記だけで消化できた行はゼロである。
  **欠陥は 1 件も見つからなかった。** 6 行はいずれも宣言した採用を満たしており、切り出した work item は無い。
  既存テストを 2 件整理した。`TestWsTrustUsernameMixed_RejectsMismatchedTo` は
  `TestWsTrustAddressing_ValidatesMessageIDToAndAction` の `To` の部分試験に完全に含まれるので削除し、
  `TestWsTrustUsernameMixed_RejectsExpiredTimestampAndReplay` はリプレイの半分を同じ試験へ移して
  `TestWsTrustUsernameMixed_RejectsExpiredTimestamp` に絞った。同じ判断を強さの違う 2 つのテストが
  観測する状態を残さないためである。
  **測ったことが 2 つある。** 1 つは `sd` の複数行置換で、パターンが複数行にわたると一致せず、
  終了状態も 0 のまま何も置換しない。故障 F4 と F11 はこれで「検出されず」と出たが、実際には注入自体が
  起きていなかった。`perl -0pi -e` で入れ直すと 2 件とも検出された。**注入した故障が本当に入ったことを
  `git diff --stat` で確かめない限り、「検出されず」は読めない。** もう 1 つはコンパイルの通らない故障で、
  F7 の最初の形（条件から `!ok` を落とす）は `declared and not used: ok` でビルドが落ちた。ビルド失敗は
  `FAIL` を出すので、粗い判定では「検出された」と読めてしまう。`(!ok && false)` に変えて入れ直した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（6 件を台帳から外し、テストを書く前の状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/ws-federation/standards.md` の 6 行それぞれについて
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（9 行目 `WSFed-PassiveSignIn`、10 行目
    `WSFed-SilentSignIn`、18 行目 `WSTrust13-IssueBearer`、19 行目 `WSTrust13-WindowsTransport`、
    27 行目 `WSS-UsernameTokenPassword`、35 行目 `WSAddressing-MessageIDToAction`）
  - **Detection Reason**: 検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。消化後の同じコマンドは
    `ok normative coverage (156 standard(s), 311 rule(s), 746 example(s), 276 id(s) named by a test)`
    を返す（270 → 276）。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 13 件の故障すべてを、対応するテストが検出した。等価変異が 2 件あり、下表に残している。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
- **Change-Resistance Results**:
  6 行 13 観測すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを
  観測した。故障は注入のたびに元へ戻している。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `WSFed-PassiveSignIn` | `resolveReplyURL` が許可集合を照合せず `wreply` をそのまま返す | `…/wreply outside the allowed set receives no token`: `https://evil.example/steal` に 200 が返る |
  | 〃 | `FindByWtrealm` が `wtrealm` を無視し、かつ `ValidateSignIn` の `wtrealm` 一致検査を外す | `…/unregistered wtrealm receives no token`: `urn:not-registered` に 200 が返る |
  | `WSFed-SilentSignIn` | `ResolveAuthnMethod` が統合 Windows の要求を実施済みの方式で黙って代用する | `…/the integrated Windows wauth …`: 400 のはずが 200 でトークンが出る |
  | 〃 | 未認証を `SignInNeedLogin` ではなく本文の無い 204 の拒否にする | `…/a silent-auth hint on an unauthenticated request …`: 303 のはずが 204 になる |
  | `WSTrust13-IssueBearer` | `addSAML11Subject` の確認方法を holder-of-key にする | `TestWsTrustIssueBearer_ReturnsABearerSAMLAssertionInTheRSTR`: RSTR の assertion が保持者証明 |
  | 〃 | `buildSAML20` の `Method` だけを holder-of-key にする | `TestBuildAssertion_ConfirmsTheSubjectAsBearerInBothVersions`: SAML 2.0 側が保持者証明。**HTTP 境界のテストはこれを検出しない** |
  | `WSS-UsernameTokenPassword` | 拒否条件から `!ok`（パスワード検証の結果）を外す | `…/a wrong password receives no token`: `wrong-password` に 200 が返る |
  | 〃 | `FindByUsername` が提示された username ではなく固定の `alice` を引く | `…/an unknown username receives no token`: `mallory` に 200 が返る |
  | `WSAddressing-MessageIDToAction` | `recordWsTrustMessageID` が常に `true` を返す | `…/a replayed MessageID receives no second token`: 同じ `MessageID` の 2 度目に 200 が返る |
  | 〃 | `rst.To != expectedTo` の拒否を無効化する | `…/a To outside the active STS endpoint …`: `https://evil.example/…` に 200 が返る |
  | 〃 | `Validate` の `r.Action != RequestIssue` を無効化する | `…/an Action other than Issue receives no token`: `…/Renew` に 200 が返る |
  | `WSTrust13-WindowsTransport` | `/trust/13/windowstransport` を能動的 STS へ結線する | `TestWsTrustWindowsTransport_NotProvided`: 404 のはずが 200 |
  | 〃 | MEX の policy の `wsu:Id` を `WindowsTransportBinding` にする | 同テスト: MEX が `WindowsTransport` を広告する |

  **等価変異が 2 件ある。** どちらも `WSFed-PassiveSignIn` の `wtrealm` 照合で、この判断は
  `FindByWtrealm` の検索鍵と `ValidateSignIn` の一致検査の 2 箇所が独立に持っている。片方だけを崩すと
  もう片方が拒否するので、テストは落ちない。行が言う「登録済み `wtrealm` にだけ」を崩すには 2 箇所とも
  崩す必要があり、上表の観測はその形で取っている。これは二重の防御であって、テストが弱いのではない。
- **Verification Results**:
  - `mise run verify` - passed（2026-09-12 に取得、exit 0）
  - `mise run check-spec` - passed
    (`ok normative coverage (156 standard(s), 311 rule(s), 746 example(s), 276 id(s) named by a test)`)
  - `mise run lint-go` - passed（0 issues）
  - `mise run test-go-package -- ./backend/wsfederation/...` - passed
  - `mise run spec-diff` - `no normative specification change against main`
  - `mise run test-ui-e2e` - N/A: 変更は Go のテスト 2 ファイルと台帳だけで、製品コードもフロントエンドも
    1 行も動いていない。ブラウザーへ届く経路が無い。

## Risk Notes

- **テストが揃っているので注記だけで終わる。** この文書はテストが 6 ファイルあり、名指しを貼れば 6 件が一度に減る。減った件数は何も意味しない。T001 で「どこまで区別されているか」を先に書き出し、区別できていない行にはテストを足す。
- **`WSAddressing-MessageIDToAction` の 3 つの検証を 1 つの入力で崩す。** 手前で落ちて後段が確かめられない。1 つずつ崩し、崩した以外が有効であることは正しい要求が通ることで先に確認する。
- **`excluded` に書けるテストが無い。** 決められない行は、台帳へ残す理由を `present when the check was introduced` から具体的な理由へ書き換えたうえで残す。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
