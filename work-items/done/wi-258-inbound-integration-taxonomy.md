---
status: completed
archived_from: 4cf541f366143f339db1d073b6f41e5944156a4a
---

# inbound 統合 (identity import) の taxonomy と context 構造を確定する

inbound の分類軸を「方向 / runtime 形状」から「上流の外部権威 + durable な source
binding の有無」へ引き直し、条件を満たすものだけを単一 bounded context `Sourcing` (source 別 feature
slice、thin root) に束ねる目標構造を [[ADR-141-inbound-identity-sourcing-taxonomy]] に確定した。
Plan の候補では (A)+(C) の統一 context 案を採用し、命名は `Provisioning` と対称の capability 名
`Sourcing` とした。調査の結果、当初 inbound 3 shape とされていたもののうち 2 つは別族と判定し帰属を
明記した: 管理者 CSV import は IdManagement に残す (権威は idmagic 自身、source binding 無し、
ADR-140 の CSV export と対称・wi-284 が対を拡張)、login-time federation は Authentication
(wi-30 Plan / ADR-064)、downstream target の台帳照合は Application / Provisioning 側 (wi-156)。
ADR-128 §コンテキスト (2)(b) / §影響 の「CSV import は適所でない」という申し送りはこれに伴い覆し、
partial supersede として双方向リンクと一文注記を張った。SCL は context_map に `Sourcing` を
scaffold し、`spec/contexts/sourcing.yaml` に taxonomy の語彙 (IdentitySource / SourceCorrelation /
Ingestion / IngestionRun / SourceCursor / SourceDrift) と混同封じを置いた。models / interfaces の
実体は wi-259 / wi-95 が入れる。

全文は `git show 4cf541f366143f339db1d073b6f41e5944156a4a:<このファイルのパス>` で読む。
