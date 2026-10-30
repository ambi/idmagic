---
status: completed
---

# 新しいキーへの同時の最初の要求を流量の制限で数える

`spec-diff` に規範の差分はなく、REQ-OAUTH2-040 と REQ-AUTHENTICATION-007 の上限を同時要求でも満たすように実装を直した。
行のないキーでは `FOR UPDATE` がロックを取らず、レート制限が上限を超えて通し、ログインの失敗が数え漏れていた。
既存の `TestRateLimiterConcurrentAllow` がまれに落ちていた原因はこれである。
確認の限界として、ほかの `FOR UPDATE` は `Lock*` の呼び出しを検索した範囲で調べただけで、網羅した監査ではない。また、書き込みの増加による性能は測っていない。
