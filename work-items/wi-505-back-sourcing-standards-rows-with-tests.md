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

# Sourcing が宣言する標準 5 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/sourcing/standards.md` の 5 行を引き取る。この文書は 8 行のうち 5 行が名指しを持たない。

**同じ SCIM の規範を扱う Provisioning は 13 行のうち 12 行が名指しを持つ。** [[wi-238-scim-inbound-list-query-conformance]] の適合作業が id を名指すテストを書いたからである。Sourcing 側は同じ RFC 7643 / RFC 7644 を受信ではなく送信の側から採用しているが、その適合作業を経ていない。つまりこの 5 行は、規範が難しいのではなく、作業がまだ来ていないだけである。Provisioning 側の 12 行が消化の実例として読める。

## Scope

- 次の 5 行を消化する。

| ID | Adoption |
|---|---|
| `RFC7643-SERVICE-PROVIDER-CONFIG` | required |
| `RFC7644-RESOURCE-OPERATIONS` | required |
| `RFC7644-BEARER-AUTHORIZATION` | required |
| `RFC7644-ERROR-RESPONSE` | required |
| `RFC7643-ENTERPRISE-EXTENSION` | partial |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。Provisioning が持つ同じ RFC の行は、既に 12 行が名指しを持ち、残る 1 行は [[wi-507-back-provisioning-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- SCIM の適合範囲そのものの拡張。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**Provisioning 側の 12 行がどう消化されたかを先に読む。** 同じ RFC の行が既に名指しを持っているので、注記の書き方と観測の置き場所はそこに実例がある。Sourcing は送信の側なので入口は逆向きになるが、行の `Statement` を区別するという条件は同じである。

`RFC7643-ENTERPRISE-EXTENSION` だけが `partial` である。`partial` の観測は 2 つ要る。採用した範囲の振る舞いと、採用していない範囲がどう扱われるか、つまり拒否するのか単に提供しないのかである。片方だけでは、全部を採用している実装とも、何も採用していない実装とも区別できない。行の `Statement` が採用の境界をどこに引いているかを読み、その境界の両側を観測する。

`RFC7644-ERROR-RESPONSE` は誤り応答の形を宣言する行である。観測は、誤りが起きたときに SCIM の誤り応答の形（`schemas`、`status`、`scimType`、`detail`）で返ることであり、HTTP のステータスだけではない。ステータスだけを観測すると、SCIM の形になっていない応答と区別できない。

`RFC7644-BEARER-AUTHORIZATION` は拒否の行である。認可の無い、または不正な Bearer トークンでの要求が拒否されること、および**その拒否が防いだ効果**、つまり対象リソースが変わっていないことを対で観測する。ステータスだけでは、変更してから拒否を返す実装と区別できない。

## Plan

1. Provisioning の 12 行がどう名指されているかを読み、注記の書き方と置き場所を揃える。
2. `RFC7643-SERVICE-PROVIDER-CONFIG` を、生成された `ServiceProviderConfig` を読む形で消化する。
3. `RFC7644-RESOURCE-OPERATIONS` を消化する。
4. `RFC7644-BEARER-AUTHORIZATION` を、拒否が防いだ効果まで含めて消化する。
5. `RFC7644-ERROR-RESPONSE` を、応答本体の形まで読む形で消化する。
6. `RFC7643-ENTERPRISE-EXTENSION` を、採用の境界の両側で消化する。
7. 解決した id を台帳から外す。

## Tasks

- [ ] T001 [Inventory] Provisioning 側の 12 行の名指しを読み、注記の書き方と置き場所を決める。
- [ ] T002 [Acceptance] `RFC7643-SERVICE-PROVIDER-CONFIG` と `RFC7644-RESOURCE-OPERATIONS` を消化する。
- [ ] T003 [Acceptance] `RFC7644-BEARER-AUTHORIZATION` を、拒否が防いだ効果まで消化する。
- [ ] T004 [Acceptance] `RFC7644-ERROR-RESPONSE` を、応答本体の形まで消化する。
- [ ] T005 [Acceptance] `RFC7643-ENTERPRISE-EXTENSION` を、採用の境界の両側で消化する。
- [ ] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
- [ ] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 5 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **`partial` を片側だけで消化する。** `RFC7643-ENTERPRISE-EXTENSION` は採用の境界を持つ行なので、採用した側だけを観測すると全部採用している実装と区別できない。境界の両側を観測する。
- **拒否をステータスだけで観測する。** `RFC7644-BEARER-AUTHORIZATION` は、変更してから拒否を返す実装をステータスでは区別できない。対象リソースを読み直す。
- **Provisioning の名指しを流用して Sourcing の入口を観測しない。** 同じ RFC でも入口が逆向きなので、Provisioning のテストが Sourcing の行を区別することはない。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
