---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-09
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
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

`excluded` の 3 行（`CIBA-CORE-PING-PUSH`、`CIBA-CORE-USER-CODE`、`CIBA-CORE-SIGNED-REQUEST`）はいずれも標準側の機能を書いているので、その機能を使うリクエストが通らないことと、通らなかった結果として `auth_req_id` が発行されていないことを対で読む。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

## Plan

1. 8 行が共有する入口にハーネスを組み、`required` の 1 件目を通しで消化して型を決める。
2. `optional` があれば、その 1 件目で「提供している」と言える根拠の形を決める。提供していなければ規範の変更として切り出す。
3. `excluded` があれば、その 1 件目で観測の型を決める。
4. 残りを消化し、解決した id を台帳から外す。
5. 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。

## Tasks

- [ ] T001 [Acceptance] 消化する id を台帳から先に外し、`mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。
- [ ] T002 [Harness] デバイス認可エンドポイントと CIBA のバックチャネル認証エンドポイント にハーネスを組み、`required` の 1 件目で型を決める。
- [ ] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
- [ ] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
- [ ] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
- [ ] T006 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
- [ ] T007 [Verify] `mise run verify`。

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
