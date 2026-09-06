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

# Authentication が宣言する標準 9 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/authentication/standards.md` の 9 行を引き取る。この文書は 10 行のうち 9 行が名指しを持たない。

9 行が扱うのはパスワードの規則と保管、WebAuthn の登録と認証、TOTP、認証方式の申告（`amr`）、そして認可要求の CSRF 防護である。いずれも認証そのものの強度に直結し、外部の規範が具体的な形を指定している領域である。

## Scope

- 次の 9 行を消化する。

| ID | Adoption |
|---|---|
| `NIST63B4-NO-COMPOSITION` | required |
| `NIST63B4-PASSWORD-MINIMUM` | excluded |
| `NIST63B4-PASSWORD-STORAGE` | required |
| `OIDC-CORE-CSRF` | required |
| `OIDC-DISCOVERY-ISSUER` | required |
| `RFC6238-TOTP` | optional |
| `RFC8176-AMR-VOCABULARY` | required |
| `WEBAUTHN3-AUTHENTICATION` | required |
| `WEBAUTHN3-REGISTRATION` | required |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。OAuth2 が持つ `OIDC-CORE-*` と `OIDC-DISCOVERY-*` の他の行は [[wi-499-back-oauth2-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- 認証要素の追加や、パスワード規則そのものの変更。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`NIST63B4-PASSWORD-MINIMUM` が `excluded` であり、`NIST63B4-NO-COMPOSITION` が `required` である。この 2 行は対になっている。組成規則（大文字・記号の強制）を課さないことを宣言し、長さの下限だけを別の形で扱うという構えである。したがって観測も対にする。組成規則を満たさないが長さは足りるパスワードが受理されること、および `excluded` の側が禁じている入力の扱いが、行の書きぶりと一致していることを併せて確かめる。片方だけを観測すると、両方を課している実装と区別できない。

`NIST63B4-PASSWORD-STORAGE` は保管の形を宣言する行である。観測は「保存されたものが平文でも可逆でもないこと」であり、ハッシュ関数を呼んでいることではない。保存先を読み直して確かめる。

`WEBAUTHN3-REGISTRATION` と `WEBAUTHN3-AUTHENTICATION` は、登録と認証で別の検証を持つ。1 つのテストで両方を名指しても、片方の検証が無い実装を区別できない。行ごとに別のテストを対応付ける。

`RFC6238-TOTP` は `optional` である。提供しているならその振る舞いを観測する。提供していないなら行の `Adoption` が誤っているので、規範の変更として切り出す。

`OIDC-CORE-CSRF` の観測は、`state` の照合が無い実装で落ちることである。値が返ってくることではなく、違う値では認可が成立しないことを観測する。

## Plan

1. `NIST63B4-NO-COMPOSITION` と `NIST63B4-PASSWORD-MINIMUM` を対で消化し、`excluded` の観測の型をここで決める。
2. `NIST63B4-PASSWORD-STORAGE` を、保存先を読み直す形で消化する。
3. `WEBAUTHN3-REGISTRATION` と `WEBAUTHN3-AUTHENTICATION` を、行ごとに別のテストで消化する。
4. `OIDC-CORE-CSRF` と `OIDC-DISCOVERY-ISSUER` を消化する。
5. `RFC8176-AMR-VOCABULARY` を、実際に発行されたトークンの `amr` を読む形で消化する。
6. `RFC6238-TOTP` の実装の有無を確かめ、観測するか切り出すかを決める。
7. 解決した id を台帳から外す。

## Tasks

- [ ] T001 [Acceptance] `NIST63B4-NO-COMPOSITION` と `NIST63B4-PASSWORD-MINIMUM` を対で消化し、`excluded` の型を決める。
- [ ] T002 [Acceptance] `NIST63B4-PASSWORD-STORAGE` を、保存先を読み直して消化する。
- [ ] T003 [Acceptance] `WEBAUTHN3-REGISTRATION` と `WEBAUTHN3-AUTHENTICATION` を、行ごとに別のテストで消化する。
- [ ] T004 [Acceptance] `OIDC-CORE-CSRF` と `OIDC-DISCOVERY-ISSUER` を消化する。
- [ ] T005 [Acceptance] `RFC8176-AMR-VOCABULARY` を、発行されたトークンの `amr` から消化する。
- [ ] T006 [Inventory] `RFC6238-TOTP` の実装の有無を確かめ、観測するか切り出すかを決める。
- [ ] T007 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
- [ ] T008 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 9 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **`excluded` と `required` の対を片方だけ観測する。** `NIST63B4-PASSWORD-MINIMUM` と `NIST63B4-NO-COMPOSITION` は同じパスワード規則の裏表なので、片方のテストがもう片方を名指してしまいやすい。行ごとに別の入力と別の観測を与える。
- **保管の観測がハッシュ関数の呼び出しになる。** 呼んでいることは保管の形ではない。保存先を読み直し、平文でも可逆でもないことを観測する。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
