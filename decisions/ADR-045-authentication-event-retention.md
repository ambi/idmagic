---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-045: 認証イベントの保持期間と sweep による削除

## コンテキスト

認証イベントは単調増加する。partition 化なしの単一テーブルでは、保持期間を定めず溜め続けると
検索 index が効かなくなり、PII を必要以上に長く抱えるコンプライアンス上の問題も生む。一方、
成功ログイン履歴は「いつもと違うログイン」の調査のため一定期間は残す必要があり、失敗詳細や
セッション記録は短期で十分という非対称がある。種類ごとに根拠ある保持期間を決め、確実に削除する。

## 決定

種類別に既定保持期間を定める: 成功イベント 365 日 (過去 1 年の正規ログイン傾向を調査可能にする)、
失敗詳細の個別行 30 日 (短期調査に足り、平文粒度の PII を長く持たない)、bucket 集約・セッション
記録・MFA チャレンジイベントは 90 日。各保持期間は `TenantSettings` で短縮・延長できるが
`max_retention_days` の global cap を上限とする。削除は `internal/bootstrap` の周期 job が
`occurred_at < now - retention` の行を消す idempotent な batched sweep で行う。impersonation
イベントは本人保護のため tenant override による短縮の対象外とし、global cap までは保持する。
partition / cold storage は将来の最適化として見送り、retention sweep による行数抑制 +
`(tenant_id, occurred_at)` index + query `limit` で検索性能を担保する。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Authentication event logging セクションにある。

## 影響

- retention sweep が確実に動くこと・index が当たること・admin 検索が期間絞り込みを要求すること
  をテストで確認する (境界 29 / 31 / 91 日)。
- [[ADR-036]] の user 削除 (anonymize cascade) と独立に動く。user 削除は sub に紐づく行を
  匿名化し、retention sweep は時間でまとめて消す。両者の二重適用で不整合が出ないよう、sweep は
  匿名化済み行もそのまま時間条件で削除する。

## 却下した代替案

- **全種類一律の保持期間**: 成功履歴は長く・失敗詳細は短くという非対称を表現できず、PII を
  必要以上に持つか、調査窓が足りなくなる。
- **削除しない (無限保持)**: 検索 index の劣化と PII コンプライアンスの両面で不可。
- **アプリ起動時のみの一括削除**: 長時間稼働で溜まり続ける。時間単位 cron で継続的に削る。
