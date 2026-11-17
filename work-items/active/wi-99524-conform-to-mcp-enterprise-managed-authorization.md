---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-17
priority: p2
group: federation-protocols
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: IDJAG-ISSUANCE }
  - { path: docs/modules/oauth2/standards.md, requirement: IDJAG-REDEMPTION }
  - { path: docs/modules/oauth2/standards.md, requirement: IDJAG-METADATA }
  - { path: docs/modules/oauth2/standards.md, requirement: IDJAG-SENDER-CONSTRAINT }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-066 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-069 }
  - { path: docs/modules/oauth2/features/token.md, requirement: REQ-OAUTH2-070 }
---

# MCP の Enterprise-Managed Authorization への適合を IdP と認可サーバーの両方で確かめる

## 動機

MCP 2026-07-28 の公式拡張 [Enterprise-Managed Authorization](https://blog.modelcontextprotocol.io/posts/enterprise-managed-auth/)（`io.modelcontextprotocol/enterprise-managed-authorization`、SEP-990）は stable になった。
企業の IdP へ一度ログインすれば、管理者が承認した MCP サーバーへ ID-JAG で接続できる。
Anthropic、Microsoft、Okta が採用を表明し、企業内の Agent が MCP へ接続する標準の経路になりつつある。

IdMagic は ID-JAG の発行と償還を `optional` で実装しているが、EMA 拡張の要求との対応は確かめていない。
MCP クライアントが EMA の手順どおりに IdMagic へ接続できないと、ゲートウェイと評価の優位を使う前に導入の候補から外れる。
新しい機能領域ではなく既存の ID-JAG の補完なので、群の基準の p3 より上げて p2 とする。

## 対象範囲

- [EMA 仕様](https://github.com/modelcontextprotocol/ext-auth/blob/main/specification/stable/enterprise-managed-authorization/2026-07-28/enterprise-managed-authorization.mdx)の要求を、IdP の役と MCP の認可サーバーの役ごとに、必須、推奨、任意へ分けた適合表。
- 適合表の不足のうち、IdP の役（ID Token から ID-JAG への交換、発行の許可）と認可サーバーの役（償還、資源の束縛、メタデータ）の実装。
- 拡張の識別子の広告など、EMA が要求するメタデータ。
- EMA の手順を模した MCP クライアントと MCP サーバーを使い、IdMagic を IdP とする構成と認可サーバーとする構成の両方を通す受け入れ試験。

## 対象外

- 実在の外部 IdP（Okta、Entra）との接続。[[wi-38172-verify-id-jag-interoperability-with-an-external-idp]] が扱う。
- ID-JAG での Agent の行為者の扱い。[[wi-79144-decide-actor-chain-in-id-jag]] が扱う。
- ID-JAG の draft のバージョンの更新。適合表で差が見つかれば別項目にする。

## 設計

既存の REQ-OAUTH2-066〜072 を正とし、EMA の要求を標準表へ追加する。
規則を複製せず、EMA 固有の要求だけを新しい標準 ID にする。
ID-JAG の draft と EMA が参照するバージョンに差がある場合は、IdMagic が固定したバージョンを維持し、差を標準表の `partial` として明示する。

IdMagic が IdP と認可サーバーの両方を兼ねる構成（同じテナント内）と、別テナントの IdP から受ける構成を区別して試験する。
同じテナントで完結する経路だけを試すと、発行者と受け入れの許可の取り違えを見逃すためである。

## 計画

1. 基準リビジョンを固定し、EMA の適合表を作る。
2. 不足を仕様と TypeSpec へ反映する。
3. 受け入れ試験で RED を確認してから実装する。

## タスク

- [ ] T001 [Spec] EMA の適合表を作り、標準表と TypeSpec を更新する。
- [ ] T002 [Acceptance] 二つの役と二つの構成で、EMA の手順を正式な HTTP 入口から通す RED を作る。
- [ ] T003 [App] 不足を実装する。
- [ ] T004 [Docs] EMA で接続する手順を開発文書へ追加する。
- [ ] T005 [Verify] 検証する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/handlers_http`：メタデータと交換、償還の契約。
- `mise run test-go-package -- ./backend/shared/http/server_http`：二つの構成の全経路。
- `mise run check-spec`、`mise run check-schema`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>`。
- 模擬相手による適合と、実製品との接続結果を区別して報告する。

## リスク

償還の許可を緩めると、別の IdP の ID-JAG で資源へ到達できる。
受け入れの許可がない発行者、別テナント、別の資源の拒否を独立に観測する。
