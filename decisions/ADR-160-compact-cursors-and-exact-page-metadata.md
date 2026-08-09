---
status: accepted
authors: [tn]
created_at: 2026-08-09
supersedes: [ADR-159]
---

# ADR-160: 管理一覧は compact cursor と正確なページ metadata を返す

## コンテキスト
ADR-159 の無期限・双方向 keyset cursor は reload、browser history、共有 URL を再現できる一方、署名 payload が長く、現在位置や全件数を UI に示せない。quota counter は既存データと乖離し得るため一覧の正確な件数にも使えない。

深いページでも一定の query cost を保ち、server state や client の取得履歴に依存しない URL という ADR-159 の性質を維持したまま、短い位置 token と先頭・末尾への直接移動が必要になった。

## 決定
新規 cursor は compact binary payload と短縮 HMAC tag を使う stateless version とし、legacy cursor の decode を維持する。管理 UI を持つ一覧は filter と同じ predicate の exact count、ページ位置 metadata、`first` / `prev` / `next` / `last` Link を返す。

v0/v2 cursor は page number を保持しないため、その cursor を直接受理した1応答だけは keyset 境界の互換性を優先し、`Pagination-Current-Page` と新規 v3 Link の page number を forward は2、backward は1から再開する。以後の v3 cursor、cursor なしの先頭、end anchor の末尾では正確な現在ページを返す。旧 token の正確な page rank を復元するための追加走査は行わない。

規範契約は IdManagement `models.AdminUserListResponse`、IdManagement / Application / Audit の対象 list interfaces・objectives・scenarios・flows に置く。現在の context/module 構造は [ARCHITECTURE.md](../ARCHITECTURE.md) から変更しない。

## 却下した代替案
- offset または page-number pagination: 任意ページへ移動できるが、深いページほど走査 cost が増え、同時更新時の位置ずれを再導入する。
- server-side cursor table / cache: token は短くできるが、共有 URL の寿命が server state、cleanup、障害復旧に依存する。
- browser session の page-to-cursor map: server 変更は小さいが、reload 後の共有 URL や別 browser で同じ位置を再現できない。
- quota counter または推定 count: 安価だが filter と一致せず、backfill 前の既存データでは管理 UI に誤った値を表示する。

## 影響
- SCL: IdManagement `models.AdminUserListResponse`、`interfaces.ListAdminUsers` / `ListGroups` / `ListAgents`、Application `interfaces.ListAdminApplications`、Audit `interfaces.ListAdminAuditEvents` と関連 objectives / scenarios / flows。
- ADR-159 の cursor wire format と Link relation を部分的に上書きする。無期限 token、双方向 keyset、通常の認証・tenant authorization は維持する。
- exact count による追加 DB work は既存 latency objectives の対象に含め、page query と独立して測定・最適化する。
