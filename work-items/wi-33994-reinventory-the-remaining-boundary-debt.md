---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: [wi-17193-move-context-specific-code-out-of-support-http, wi-39119-publish-the-resolved-tenant-as-tenancy-public-language, wi-65906-inject-time-randomness-and-network-into-domain, wi-92970-adopt-information-hiding-modular-design]
change_kind: tooling
spec_impact: { kind: none, reason: "境界の負債の台帳の理由を書き直し、残りの解消を work item として起票するだけである。製品のコード、HTTP の応答、ドメインイベント、永続状態は変えない。" }
---

# 残った境界の負債を Context の対ごとに棚卸しし、理由を書き直す

## 動機

`tools/check/boundary-debt.json` の全項目の理由は、「`<A> -> <B> remains coupled across N observed path(s) until its owning Context exposes or consumes the required published language.`」という同じ定型文である。
境界の検査を導入したとき、台帳の各項目には、何が設計上の負債で何が意図した例外かが読み取れる理由を持たせると決めていた。
定型文の理由からは、どの依存をどう直すのかが読み取れず、解消の work item も起票されていない。

wi-17193、wi-39119、wi-65906 で、原因が一か所に集まった負債（`support_http` を経由した依存、Tenancy への依存、`domain` の作用）が消える。
残るのは、Context の対ごとに原因の異なる依存である。
現時点で多いのは次の対である（括弧内は `undeclared-context-edge` の件数）。

- OAuth2 から IdManagement（18）、IdManagement から Authentication（17）、IdManagement から Jobs（16）、Tenancy から IdManagement（9）、Sourcing から IdManagement（9）
- `private-import` の参照先として多いのは `backend/claimmapping/usecases`（11）と `backend/jobs/usecases`（9）

## 対象範囲

- 前提の 3 項目が完了した後の `mise run check-boundaries` の結果から、残った違反を Context の対ごとにまとめる。
- 対ごとに、直す方法を次のどれかに決める。
  - Context Map に関係が欠けているだけなら、Context Map に書き足す。
  - 公開されていないパッケージを使っているなら、Supplier 側の `ports` へ公開するか、組み立て地点で結ぶ。
  - 依存の向きが逆なら、依存を逆転する。
- 直す方法と、それが必要な理由を、台帳の各項目の理由として書き直す。
- 直すための work item を、まとめて直せる単位で起票する。
- `backend/shared/security/tokens_jose` と `backend/shared/storage/db_memory` を経由した依存も同じ方法で扱う。

## 対象外

- 個々の依存の解消そのもの。起票した work item で行う。

## タスク

- [ ] T001 [Inventory] 残った違反を Context の対ごとにまとめ、直す方法を決める。
- [ ] T002 [Tooling] 台帳の理由を書き直す。
- [ ] T003 [Plan] 解消の work item を起票する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-work-items`

## リスク

- 理由を書き直すだけでは負債は減らない。
  この項目の完了の条件を、すべての残りの項目に解消の work item が対応していることにする。
