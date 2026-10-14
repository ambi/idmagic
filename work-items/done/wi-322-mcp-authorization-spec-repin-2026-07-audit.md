---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# MCP Authorization 仕様の対象改訂を再検討し、必要なら再ピン留めする(2025-11-25 → 2026-07-28 世代)

MCP authorization 仕様 2025-11-25 → 2026-07-28 の差分を一次情報(MCP 公式 changelog、
Enterprise-Managed Authorization extension 仕様、IETF datatracker の
`draft-ietf-oauth-identity-assertion-authz-grant`)で監査した。結論は「破壊的差分なし」:
stateless core 化は MCP プロトコル・トランスポート層の変更で idmagic の認可サーバー実装
(session 非依存)に影響しない。認可ハードニング項目(RFC 9207 `iss` 必須化・DCR の
`application_type`・client credential のイシュアー束縛・RFC 7591 DCR 非推奨化)のうち
RFC 9207 は既に `adoption: required` で満たしており、DCR 非推奨化も後方互換維持のため
即時対応不要。[[ADR-055]] の対象改訂を 2025-11-25 → 2026-07-28 に pin し直し、監査結論を
ADR 本文に記録した(新規 ADR は起票せず、ADR-055 自身の「改訂が進む場合は本 ADR を更新」
方針に従った)。
Enterprise-Managed Authorization (ID-JAG) は [[wi-57-cross-app-access-identity-assertion-grant]]
が対象とする IETF draft (`-04`, 2026-05-21) と同一 grant であることを確認し、新規 work item は
起票不要と判断した。ただし wi-57 の Plan にあった「token exchange の subject token profile」と
いう設計前提は不正確で、実際は 2 ホップ(IdP への RFC 8693 Token Exchange → 宛先 AS への
**RFC 7523 JWT-Bearer grant**、idmagic は現状 RFC 7523 を client 認証にのみ用いており
authorization grant としての JWT-Bearer は未対応)であることが判明したため、wi-57 の Plan に
訂正を追記した。この JWT-Bearer grant の新設自体は [[wi-57-cross-app-access-identity-assertion-grant]]
/ [[ADR-056]] 側のスコープであり、本 work item では実施していない(Out of Scope どおり)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
