---
status: completed
---

# Soft Quota の文書と実装の食い違いを解消する

Soft Quota を維持し、REQ-TENANCY-047 と REQ-TENANCY-048 に、超過しても拒否しない計数と保持期限バッチによる照合と警告を追加した。
REQ-TENANCY-036 の未知リソースの扱いから Soft を除き、TypeSpec の TenantQuota、TenantUsage、TenantQuotaUpdateRequest に意味と鮮度を明記した。
監査の既定値は 1,000,000 件、成果物は 1 GiB とし、0 を含む上書きを使う。
メモリと PostgreSQL の共有契約をそろえ、Soft の列を BIGINT に広げ、Hard の使用量を維持してテナント別の実データを測り、超過を QuotaExceeded(hardLimit=false) と警告ログへ送る。
測定失敗は以前の使用量を保持し、警告は実行ごとに再発行するため、即時性と単回配送は保証しない。
旧バイナリの範囲を超えた値を保存した後の後退には前進修正を要する。
故障注入は上限比較とバッチ配線に限定し、全変異の調査と長時間 fuzz は行わない。
監査エクスポートの非同期化と新しい成果物ストアは対象外とした。
