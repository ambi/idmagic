---
status: completed
authors: [claude]
risk: medium
created_at: 2026-08-08
depends_on: [wi-56-mcp-authorization-server]
---

# MCP Authorization 仕様の対象改訂を再検討し、必要なら再ピン留めする(2025-11-25 → 2026-07-28 世代)

## Motivation

[[ADR-055]] は idmagic を MCP (Model Context Protocol) の Authorization Server と位置づけ、
「MCP 認可仕様は版差が大きいため対象改訂を固定する」方針のもと **2025-11-25 改訂** に pin して
[[wi-56-mcp-authorization-server]] を実装した。ADR 自体が「改訂が進む場合は本 ADR を更新し
新たな対象改訂を pin し直す」と明記しており、再評価は想定内の運用である。

2026年8月時点の外部動向を軽く調査したところ、MCP は 2026-07-28 改訂で **stateless core 化**
(initialize/initialized handshake とプロトコルレベルの session を廃止し、client 情報・protocol
version・capabilities を毎リクエストの `_meta` に載せる自己記述型に変更)しており、さらに
**Enterprise-Managed Authorization (EMA)** 拡張が 2026年6月に安定版化し、Okta / Anthropic /
VS Code および複数の MCP サーバーで採用が始まっている。EMA は Identity Assertion JWT
Authorization Grant (ID-JAG) を実装しており、これは idmagic 側でいえば
[[wi-57-cross-app-access-identity-assertion-grant]] が対象とする
IETF `draft-ietf-oauth-identity-assertion-authz-grant`(2026年8月時点で `-04`、Standards
Track・未 RFC 化)と同一の仕組みである。

対象改訂を古いまま放置すると、(a) idmagic を認可サーバーとする MCP クライアント/サーバーが
新しい stateless core を前提に実装され始めた場合の互換性リスク、(b) EMA/ID-JAG のような
今後主流化しうる拡張との統合機会の見落とし、という2つのリスクがある。まずは差分監査を行い、
ADR-055 の再ピン留めが必要か、[[wi-57-cross-app-access-identity-assertion-grant]] との統合方針を
どう設計すべきかを判断する。

## Scope

- MCP authorization 仕様の 2025-11-25 改訂と 2026-07-28 改訂(および間の改訂があれば)の差分調査。
  特に stateless core 化が `spec/contexts/oauth2.yaml` の MCP 関連 interface
  (`/.well-known/oauth-protected-resource` 配信、`McpResourceServer` 管理、RFC 8707 audience 限定)
  の前提を崩さないかを確認する。
- Enterprise-Managed Authorization (ID-JAG) が [[wi-57-cross-app-access-identity-assertion-grant]]
  の設計(IETF `draft-ietf-oauth-identity-assertion-authz-grant`)とどう関係するかの整理。両者を
  同一実装で満たせるのか、別々に実装が必要なのかの判断。
- 調査結果に応じて [[ADR-055]] を改訂(対象改訂の pin し直し)するか、新規 ADR を起票するかの判断と実施。

## Out of Scope

- [[wi-57-cross-app-access-identity-assertion-grant]] 自体の実装(本 work item は関係整理までで、
  実装は wi-57 側で行う)。
- MCP SDK との実接続による相互運用テスト(wi-56 完了時点でも未実施であり、本 work item のスコープ外)。
- stateless core 化以外の MCP 仕様変更点で idmagic の認可サーバー実装に影響しないもの(単なる
  MCP プロトコル側の変更で OAuth 認可フローに無関係な部分)の深追い。

## Design

- 未定。まず外部の一次情報(MCP 公式仕様のチェンジログ、Enterprise-Managed Authorization
  拡張ドキュメント、IETF draft の最新版)を確認し、idmagic 側で pin している RFC 8707 / RFC 9728 /
  RFC 8414 / RFC 7591 のサブセット準拠に実質的な差分があるかを洗い出すところから始める。
- ADR-055 は改訂前提の pin 方式を既に採っているため、新しい ADR を都度作るのではなく
  「ADR-055 を更新し新たな対象改訂を pin し直す」のが基本方針(ADR-055 本文の記載どおり)。
  ただし ID-JAG/EMA 統合のような設計上の大きな分岐が生じる場合は別途 ADR を検討する。

## Plan

- 未定(調査結果次第で分岐する)。最低限のステップ:
  1. 2026-07-28 改訂と EMA 拡張の一次情報を精読し、差分を一覧化する。
  2. idmagic の現行 MCP AS 実装(`spec/contexts/oauth2.yaml` の該当 interface、
     `backend/oauth2/` の実装)に影響する差分を絞り込む。
  3. 影響がなければ ADR-055 の pin 改訂のみで完了。影響があれば SCL 変更を伴う追加 work item を
     別途起票する。

## Tasks

- [x] T001 [Research] MCP authorization 仕様 2026-07-28 改訂のチェンジログを精読し、2025-11-25 からの差分を一覧化する。
- [x] T002 [Research] Enterprise-Managed Authorization (ID-JAG) の仕様と [[wi-57-cross-app-access-identity-assertion-grant]] の設計の関係を整理する。
- [x] T003 [Decision] ADR-055 の pin を更新するか、追加の SCL 変更が必要かを判断する。
- [x] T004 [ADR] 判断に応じて [[ADR-055]] を改訂する(対象改訂を pin し直す、または統合方針を追記する)。
- [x] T005 [Verify] 変更があれば `just check-scl` / `just verify-spec` を通す。 → SCL 変更なし(結論: 差分は ADR-055 の pin 文言更新のみで吸収可能)。`just check` で work-item / ADR 記録の整合を検証した。

## Verification

- **差分調査の結論(2025-11-25 → 2026-07-28、一次情報: MCP 公式 changelog、Enterprise-Managed
  Authorization extension 仕様、IETF datatracker の draft-ietf-oauth-identity-assertion-authz-grant)**:
  - **stateless core 化**(session / `initialize` ハンドシェイク廃止、`server/discover` 追加等)は
    MCP プロトコル・トランスポート層の変更で、idmagic の認可サーバー実装(session 非依存)に
    影響しない。**影響なし**。
  - **認可まわりのハードニング**(RFC 9207 `iss` 必須化、DCR の `application_type` 要求、client
    credential のイシュアー束縛、RFC 7591 DCR の非推奨化 → Client ID Metadata Documents への移行方針)
    のうち、RFC 9207 `iss` は idmagic 側で既に `adoption: required` として実装済みで差分なし。
    DCR 非推奨化は「後方互換で RFC 7591 を使い続けてよい」とされており今は変更不要(将来の
    pin 更新時に CIMD の普及状況を再確認する)。`application_type` は MCP client 側の要求であり
    idmagic 側の即時対応は不要。**破壊的差分なし**。
  - **Enterprise-Managed Authorization (ID-JAG)** は [[wi-57-cross-app-access-identity-assertion-grant]]
    が対象とする IETF `draft-ietf-oauth-identity-assertion-authz-grant`(2026-08 時点で `-04`、
    2026-05-21 リビジョン、Standards Track・未 RFC 化)と同一の grant であることを一次仕様で確認した。
    wi-57 は既にこれを見据えていたため**新規 work item の起票は不要**。ただし wi-57 の Plan にあった
    「token exchange の subject token profile として追加する」という設計前提は不正確で、実際は
    2 ホップ(IdP への RFC 8693 Token Exchange → 宛先 AS への **RFC 7523 JWT-Bearer grant**)である
    ことが判明したため、wi-57 の Plan に訂正を追記した(idmagic は現状 RFC 7523 を client 認証にのみ
    使っており、authorization grant としての JWT-Bearer は wi-57/ADR-056 側で新設が必要)。
  - **結論**: ADR-055 の pin を 2025-11-25 → 2026-07-28 に更新し、監査結果を ADR-055 本文に記録した。
    追加の SCL 変更は不要(既存の RFC 8707/9728/8414/7591 サブセットで対象改訂を満たす)。
- SCL に変更を加えた場合は `just check-scl` / `just verify-spec` を通す。→ 本 work item では SCL
  (`spec/scl.yaml` / `spec/contexts/*.yaml`)への変更なし。`just check` で work-item・ADR 記録の
  整合を検証した。

## Risk Notes

- リスクは medium。MCP 側の仕様変更(特に stateless core 化)が idmagic の認可サーバー実装の
  前提(session 管理まわり)に影響する可能性があり、影響範囲の見極めを誤ると後続の
  wi-57 実装や既存 MCP クライアントとの互換性に波及しうる。まず調査を先行させ、実装判断を
  急がないことでリスクを抑える。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
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
- **Verification Results**:
  - `just check` - passed (work-item 339 件・change-record id 488 件・architecture ledger 23 件・
    traceability manifest/evidence を含め all green)。
  - SCL (`spec/scl.yaml` / `spec/contexts/*.yaml`) への変更なし。差分は ADR-055 の pin 文言更新
    (2025-11-25 → 2026-07-28)と、wi-57 Plan への訂正追記のみで吸収した。
