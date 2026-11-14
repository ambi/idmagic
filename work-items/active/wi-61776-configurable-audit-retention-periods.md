---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-14
priority: p3
group: retention-and-audit
depends_on: [wi-514-audit-retention-decision-and-sweep-disagree]
change_kind: feature
affected_spec:
  - { path: docs/modules/audit/features/event-retention.md, requirement: REQ-AUDIT-013 }
---

# 監査の記録の保持期間を設定で変えられるようにするかを決める

## 動機

[[wi-514-audit-retention-decision-and-sweep-disagree]] は、監査の記録の保持期間を種類ごとの固定の値（デフォルト 365 日）に決め、Audit が掃引を持つ形にした。
値を変える手段は持たない。

利用者の規制によっては、製品の中で 365 日より長く、または短く記録を持ちたい場合がある。
たとえば医療や金融の規制に従う利用者は数年の保管を求められることがあり、逆に個人データを最小限にしたい利用者は短い期間を望む。
現在の答えは「長期の保管は監査エクスポートで利用者の環境へ移す」であり、製品内の期間は変えられない。

優先度は p3 にする。グループの基準は p2 だが、これは新しい方式の採否の検討から始める項目で、固定の値で規範は満たしているためである。

## 対象範囲

- 保持期間を変える必要があるか、変えるなら配置ごと（起動設定）かテナントごとかを決める。
- 変えられる範囲の下限と上限を決める。下限は `GDPR-PROCESSING-RECORDS` と、平文のユーザー名を持つ失敗の記録の扱いから、上限は[キャパシティ設計](../../docs/design/performance/capacity.md)の予算から導く。
- 採る場合は、REQ-AUDIT-013 のモデルと要件、`backend/audit` の保持の方針を変える。

## 対象外

- Authentication が所有する認証イベントの集計、ログインセッション、既知の端末の掃除の期間。
- 監査の記録の外部への継続配送（SIEM、ログストリーミング）。

## 設計

未着手。
テナントごとに変える案は、テナント設定の型、管理 API、Tenancy から Audit への値の受け渡しを要する。
配置ごとに変える案は、起動設定を一つ足すだけで済むが、同じ配置の全テナントが同じ期間になる。

## 計画

着手時に、利用者の要求が実際にあるかを確かめてから案を選ぶ。

## タスク

- [ ] T001 [Spec] 採否と方式を決め、要件の差分を書く。

## 検証

- 着手時に選ぶ。

## リスク

- 期間を延ばす設定は、保存容量の見積もりと、消去の期限までにデータが消えるかの両方を変える。[保持期間を変えるときの順序](../../docs/design/data/lifecycle.md#保持期間を変えるときの順序)に従う。
