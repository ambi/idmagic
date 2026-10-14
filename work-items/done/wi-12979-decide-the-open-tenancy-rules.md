---
status: completed
archived_from: 7b35b02c29e6cd07c0f2d9302134a6d9b5721d46
---

# Tenancy の規則に残した要判断を決める

`mise run spec-diff` は REQ-TENANCY-026、027、031、032、037、039 と TypeSpec の `TenantQuotaUpdateRequest` の変更を示す。
REQ-TENANCY-027 では、すでに無効なテナントの再度の無効化で `disabled_at` を変えないようにした。204 と `TenantDisabled` の発行は維持する。
REQ-TENANCY-032 では、フッターリンクのラベルを UTF-8 のバイト数で制限する文を削り、ラベルと URL の長さを要件どおり文字数で数えるようにした。
REQ-TENANCY-037 では、負の上限を保存する文を、400 と `invalid_request` で拒否しどの上書きも保存しない文に置き換え、`TenantQuotaUpdateRequest` の各項目に `@minValue(0)` を付けた。
REQ-TENANCY-026、031、039 は振る舞いを維持し、理由を **判断** の欄に記した。

全文は `git show 7b35b02c29e6cd07c0f2d9302134a6d9b5721d46:<このファイルのパス>` で読む。
