---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
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

## Plan

1. 既存の 6 つのテストファイルを読み、どの行がどこまで区別されているかを行ごとに書き出す。
2. `WSFed-PassiveSignIn` を、許可外の `wtrealm` と `wreply` でトークンが出ないことまで含めて消化する。
3. `WSTrust13-IssueBearer` と `WSS-UsernameTokenPassword` を消化する。
4. `WSAddressing-MessageIDToAction` を、3 つの検証を 1 つずつ崩す形で消化する。
5. `WSFed-SilentSignIn` で `excluded` の観測の型を決め、`WSTrust13-WindowsTransport` へ広げる。
6. 解決した id を台帳から外す。

## Tasks

- [ ] T001 [Inventory] 既存テストが 6 行それぞれをどこまで区別しているかを書き出す。
- [ ] T002 [Acceptance] `WSFed-PassiveSignIn` を、許可外の `wtrealm` / `wreply` でトークンが出ないことまで消化する。
- [ ] T003 [Acceptance] `WSTrust13-IssueBearer` と `WSS-UsernameTokenPassword` を消化する。
- [ ] T004 [Acceptance] `WSAddressing-MessageIDToAction` を、3 つの検証を 1 つずつ崩して消化する。
- [ ] T005 [Type] `WSFed-SilentSignIn` で `excluded` の型を決め、`WSTrust13-WindowsTransport` へ広げる。
- [ ] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
- [ ] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 6 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **テストが揃っているので注記だけで終わる。** この文書はテストが 6 ファイルあり、名指しを貼れば 6 件が一度に減る。減った件数は何も意味しない。T001 で「どこまで区別されているか」を先に書き出し、区別できていない行にはテストを足す。
- **`WSAddressing-MessageIDToAction` の 3 つの検証を 1 つの入力で崩す。** 手前で落ちて後段が確かめられない。1 つずつ崩し、崩した以外が有効であることは正しい要求が通ることで先に確認する。
- **`excluded` に書けるテストが無い。** 決められない行は、台帳へ残す理由を `present when the check was introduced` から具体的な理由へ書き換えたうえで残す。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
