---
status: completed
archived_from: 98489398288dd7f28be141625b113876573254ce
---

# Go バックエンドの残り低カバレッジ領域をまとめて底上げする

対象 10 package すべてに、拒否・期限切れ・tenant 不一致・CSRF 境界などの安全側分岐を中心とした
ユニットテストを追加した。既存の振る舞いへの変更は無く (`spec_impact: none` の通り)、契約・規範シナリオへの
影響も無い (`mise run spec-diff` で無変更を確認済み)。
| Package | Before | After | 閾値 |
|---|---|---|---|
| `backend/shared/security/tokens_jose` | 62.0% | 88.9% | >=75% |
| `backend/signingkeys/keys_memory` | 75.7% | 91.6% | >=75% |
| `backend/shared/spec` | 58.4% | 84.2% | >=75% |
| `backend/authentication/handlers_http` | 29.7% | 74.8% | >=70% |
| `backend/shared/http/support_http` | 62.3% | 85.7% | >=70% |
| `backend/sourcing/scim/usecases` | 29.4% | 79.9% | >=70% |
| `backend/shared/resilience` | 0.0% | 95.4% | >=70% |
| `backend/shared/events/sinks_console` | 0.0% | 100.0% | >=70% |
| `backend/shared/observability/telemetry_otlp` | 0.0% | 94.0% | >=70% |
| `backend/application/domain` | 82.9% | 96.1% | >=80% |
全 package が閾値を超過した。全体平均 (`mise run test-go-cover` の `total`) は 60.5% → 63.3%。
実装中に副次的な発見が 1 件あった: `tokens_jose.jwkThumbprint` (RFC 7638 canonical form) が RSA 専用の
メンバー集合 (`e`/`kty`/`n`) を前提にしており、DPoP が受理を宣言している ES256 (EC 鍵) の proof は
署名検証成功後もサムプリント計算で必ず拒否される。本 WI はテストの追加のみを scope とするため、この
制約は現状の挙動としてテストに固定しただけで、修正はしていない。修正は [[wi-389-dpop-ec-key-thumbprint-support]]
として起票済み。

全文は `git show 98489398288dd7f28be141625b113876573254ce:<このファイルのパス>` で読む。
