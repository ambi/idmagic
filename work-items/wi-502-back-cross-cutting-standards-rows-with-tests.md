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

# 製品横断の標準 7 行にテストを対応付け、所有するパッケージを決めて台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/standards.md` の 7 行を引き取る。この文書は 8 行のうち 7 行が名指しを持たない。

`docs/standards.md` は「二つ以上の Context が同じ従い方をしなければならず、Context ごとに違う従い方をすることが選択ではなく欠陥であるもの」だけを置くと自ら宣言している。**この 7 行が他の文書の行と違うのは、どのパッケージのテストが所有するかが決まっていない点である。** WCAG 22 の 4 行はフロントエンド、GDPR の 3 行は複数の Context にまたがり、行そのものが担い手を名指している（`GDPR-ERASURE` は IdManagement の Purge 遷移と Authentication の資格情報破棄）。所有を先に決めないと、名指しが 1 箇所に付いて残りの Context が素通りする。

## Scope

- 次の 7 行を消化する。いずれも `required` である。

| ID | 行が名指す担い手 |
|---|---|
| `WCAG22-KEYBOARD` | 認証操作の UI |
| `WCAG22-FOCUS` | 認証操作の UI |
| `WCAG22-LABELS-ERRORS` | 認証操作の UI |
| `WCAG22-STATUS` | 認証操作の UI |
| `GDPR-CONSENT-WITHDRAWAL` | OAuth2 Context の `Consent` と `ConsentLifecycle` |
| `GDPR-ERASURE` | IdManagement の UserLifecycle Purge 遷移と Authentication の資格情報破棄 |
| `GDPR-PROCESSING-RECORDS` | Audit Context の保持期間 |

- 行ごとに、どのパッケージのテストが所有するかを決めて本項目へ書く。担い手が複数ある行は、担い手ごとに観測を持つ。
- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行が名指す担い手が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- WCAG 2.2 の適合そのものの拡張。[[wi-292-wcag22-accessibility-conformance-and-automated-checks]] が持つ。
- 監査記録の保持期間そのものの変更。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**所有の決め方を先に決める。** 行が 1 つの Context だけを名指しているなら、その Context のテストが所有する。複数を名指しているなら、名指された Context ごとに観測を持ち、注記も Context ごとに置く。1 箇所だけに名指しを付けて台帳から外すと、検査は通るのに残りの Context は素通りしたままになる。これは他の 8 文書には現れない、この文書だけの落とし穴である。

WCAG 22 の 4 行は「すべての認証操作」を対象にしている。したがって観測の入口は認証画面であり、実装の部品ではない。`WCAG22-KEYBOARD` はキーボードだけで認証を完了できること、`WCAG22-FOCUS` はフォーカスが視認でき重要な要素が完全に隠れないこと、`WCAG22-LABELS-ERRORS` は入力にラベルが付きエラーが修正方法まで示されること、`WCAG22-STATUS` は結果がフォーカス移動なしに支援技術へ通知されることを、それぞれ別のテストで観測する。1 つの軸監査テストが 4 行すべてを名指すと、どの行が落ちたか読めない。

GDPR の 3 行は、いずれも「後から効く」性質を持つ。`GDPR-CONSENT-WITHDRAWAL` は撤回後の新規発行に使われないこと、`GDPR-ERASURE` は消去後に PII が読み出せないこと、`GDPR-PROCESSING-RECORDS` は定義済みの期間の内側で記録が残っていることを観測する。いずれも「操作が成功した」ではなく、その後の状態を読み直すことでしか区別できない。

## Plan

1. 7 行それぞれについて、所有するパッケージを決めて本項目へ書く。複数の担い手を持つ行はそのすべてを挙げる。
2. WCAG 22 の 4 行を、認証画面を入口として行ごとに別のテストで消化する。
3. `GDPR-CONSENT-WITHDRAWAL` を、撤回後の新規発行を読み直す形で消化する。
4. `GDPR-ERASURE` を、IdManagement の Purge と Authentication の資格情報破棄の両方で消化する。
5. `GDPR-PROCESSING-RECORDS` を、保持期間の内側と外側の対で消化する。
6. 解決した id を台帳から外す。

## Tasks

- [ ] T001 [Inventory] 7 行の所有パッケージを決め、複数の担い手を持つ行はそのすべてを本項目へ書く。
- [ ] T002 [Acceptance] WCAG 22 の 4 行を、行ごとに別のテストで消化する。
- [ ] T003 [Acceptance] `GDPR-CONSENT-WITHDRAWAL` を、撤回後の新規発行を読み直して消化する。
- [ ] T004 [Acceptance] `GDPR-ERASURE` を、Purge と資格情報破棄の両方で消化する。
- [ ] T005 [Acceptance] `GDPR-PROCESSING-RECORDS` を、保持期間の内側と外側の対で消化する。
- [ ] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
- [ ] T007 [Verify] `mise run verify` および `mise run test-ui-e2e`。

## Verification

- 本項目が持つ 7 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 複数の担い手を名指す行について、担い手ごとに注記と観測がある。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`
- `mise run test-ui-e2e`

## Risk Notes

- **1 箇所の名指しで台帳から外れてしまう。** この文書の行は複数の Context にまたがるため、1 つのテストが id を書けば検査は通る。担い手ごとに注記を置き、担い手を 1 つずつ崩して落ちることを確かめる。
- **WCAG の 4 行が 1 つの軸監査テストにまとめられる。** まとめると、どの行が落ちたか読めず、行を消したときに気づけない。行ごとに別のテストにする。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
