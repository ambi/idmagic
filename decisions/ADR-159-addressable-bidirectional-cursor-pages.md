---
status: accepted
authors: [tn]
created_at: 2026-08-09
supersedes: [ADR-158]
---

# ADR-159: 管理一覧の cursor URL は無期限かつ双方向にする

## コンテキスト
ADR-158 は cursor を 1 時間で失効させ、`Link` は `rel="next"` だけを返すと決めた。この契約は逐次読み込みには十分だったが、管理画面の現在位置を URL として共有し、reload・browser history・前ページ移動で再現する要件とは両立しない。

cursor は通常の認証・tenant authorization を置き換える capability ではない。tenant・query・sort・keyset を署名した位置情報であり、長寿命化によって保護対象への到達権を付与しない。この性質と管理者向け URL の再現性を比較し、expiry と next-only の判断を見直す。

## 決定
新規 cursor は expiry を持たない versioned token とし、一覧 API は存在する方向の `rel="prev"` / `rel="next"` を返す。既存の expiry 付き token は移行期間中も従来どおり検証し、署名鍵の rotation を新 token の一括無効化手段とする。

規範契約は対象 list interface、scenario、flow、objective に置く。cursor の署名対象と通常の認証・tenant authorization は維持する。ADR-158 の RFC 8288 `Link` header 採用、domain-only response body、offset/ページ番号を採用しない判断は上書きしない。

## 却下した代替案
- 1 時間 expiry を維持する: 長時間の調査、共有 URL、bookmark が同じ一覧位置を再現できず、要件を満たさない。
- `rel="next"` だけを維持して browser history だけで戻る: 共有 URL から開始した利用者や直接の前ページ操作を表現できず、client ごとに取得履歴を保持させる。
- offset またはページ番号を URL に載せる: 同時更新時の位置ずれと深いページの query cost を再導入し、keyset pagination の目的に反する。

## 影響
- SCL: IdManagement `interfaces.ListAdminUsers` / `ListGroups` / `ListAgents`、Application `interfaces.ListAdminApplications` / `ListApplicationAssignments`、Audit `interfaces.ListAdminAuditEvents`、Authentication `interfaces.ListAuthenticationEventBuckets`、OAuth2 `interfaces.ListAdminOAuth2Clients` / `ListAdminConsents`、Provisioning `interfaces.ListProvisioningDeliveries` と関連 scenario / flow / objective。
- 設計正本: [ARCHITECTURE.md](../ARCHITECTURE.md) の既存 context/module 構造は変更しない。
