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

# ApiTokens が宣言する標準 11 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/api-tokens/standards.md` の 11 行を引き取る。この文書は 12 行のうち 11 行が名指しを持たない。

11 行が扱うのは API アクセストークンの提示、内省、失効、そして送信者拘束である。この防護は他の Context のスコープ検査より手前にあり、ここが素通りすれば以降の検査は攻撃者の名乗った主体に対して働く。件数の少なさは重要度の低さを意味しない。

## Scope

- 次の 11 行を消化する。

| ID | Adoption |
|---|---|
| `RFC6750-API-TOKEN-HEADER` | required |
| `RFC6750-API-TOKEN-QUERY` | excluded |
| `RFC7009-API-TOKEN-REVOKE` | required |
| `RFC7009-API-TOKEN-UNKNOWN` | required |
| `RFC7662-API-TOKEN-INACTIVE` | required |
| `RFC7662-API-TOKEN-INTROSPECT` | required |
| `RFC9068-API-TOKEN-CLAIMS` | required |
| `RFC9068-API-TOKEN-SIGNATURE` | required |
| `RFC9449-API-TOKEN-DPOP` | optional |
| `RFC9700-API-TOKEN-AUDIENCE` | required |
| `RFC9700-API-TOKEN-SENDER-CONSTRAINT` | optional |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。OAuth2 の同名に見える行（`RFC6750-AUTHORIZATION-HEADER` など）は別の入口を指す別の行であり、[[wi-499-back-oauth2-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`required` の 8 行は、宣言した振る舞いが製品の正式な入口から到達できることを観測する。API トークンの検証はミドルウェアの位置にあるため、入口は保護されたエンドポイントであり、トークン検証関数の単体テストでは代わりにならない。関数が正しくても配線されていなければ素通りする。

`RFC6750-API-TOKEN-QUERY` は `excluded` である。クエリ文字列でのトークン送信を提供しないという行なので、観測は「提供していないこと」になる。クエリに正しいトークンを載せた要求が認証されないこと、および状態が変わっていないことを対で観測する。ヘッダーに載せた同じトークンが通ることを先に確かめないと、そのトークンが最初から無効だった実装と区別できない。

`optional` の 2 行（`RFC9449-API-TOKEN-DPOP`、`RFC9700-API-TOKEN-SENDER-CONSTRAINT`）は、提供しているならその振る舞いを観測する。提供していないなら、その行の `Adoption` が誤っているということなので規範の変更として切り出す。この 2 行は同じ送信者拘束の機構を別の角度から書いているので、実装の有無は 1 度読めば両方に答えが出る。

## Plan

1. `RFC6750-API-TOKEN-HEADER` から着手し、保護されたエンドポイントを 1 つ選んで、`required` の観測の型を決める。
2. `RFC9068-API-TOKEN-SIGNATURE` と `RFC9068-API-TOKEN-CLAIMS` を進める。
3. `RFC9700-API-TOKEN-AUDIENCE` を進める。値だけが違う正しい形式のトークンで確かめる。壊れた文字列では形式の検証で落ち、audience の照合が無くても成立する。
4. `RFC7662-*` と `RFC7009-*` の内省と失効を進める。失効は失効前の成功と対で観測する。
5. `RFC6750-API-TOKEN-QUERY` の `excluded` を進める。
6. `RFC9449-API-TOKEN-DPOP` と `RFC9700-API-TOKEN-SENDER-CONSTRAINT` の実装の有無を確かめ、`optional` として観測するか規範の変更として切り出すかを決める。
7. 解決した id を台帳から外す。

## Tasks

- [ ] T001 [Acceptance] `RFC6750-API-TOKEN-HEADER` を保護されたエンドポイントから観測し、`required` の型を決める。
- [ ] T002 [Acceptance] `RFC9068-API-TOKEN-SIGNATURE` / `RFC9068-API-TOKEN-CLAIMS` / `RFC9700-API-TOKEN-AUDIENCE` を消化する。
- [ ] T003 [Acceptance] `RFC7662-API-TOKEN-INTROSPECT` / `RFC7662-API-TOKEN-INACTIVE` / `RFC7009-API-TOKEN-REVOKE` / `RFC7009-API-TOKEN-UNKNOWN` を消化する。
- [ ] T004 [Acceptance] `RFC6750-API-TOKEN-QUERY` を、ヘッダーでの成功と対にして消化する。
- [ ] T005 [Inventory] `RFC9449-API-TOKEN-DPOP` と `RFC9700-API-TOKEN-SENDER-CONSTRAINT` の実装の有無を確かめ、観測するか切り出すかを決める。
- [ ] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
- [ ] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 11 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **手前の検証で落ちて後段の照合が確かめられない。** 理由ごとに 1 要素だけを崩したトークンを作る。崩した要素以外が有効であることは、正しいトークンが通ることで先に確認する。
- **`optional` の 2 行が「未実装だから観測できない」で止まる。** 止めない。実装が無いなら行の `Adoption` が誤っているので、規範の変更として切り出す。台帳へ残す場合は、理由を `present when the check was introduced` から具体的な理由へ書き換える。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
