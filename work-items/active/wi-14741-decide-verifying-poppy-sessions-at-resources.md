---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-17
priority: p3
group: federation-protocols
depends_on: [wi-10861-support-poppy-authorization-server-profile, wi-61710-decide-agent-detection-signals-and-web-bot-auth]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-062 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-003 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-006 }
---

# 採否を判断したうえで、企業の資源側で Poppy の Session を検証する

## 動機

[Poppy 1.0](https://personalagentprotocol.org/docs/1.0/spec) §5.2 と §5.3 では、Personal Agent が企業の API へのすべての要求に、委譲鍵で署名した `Poppy-Session` の JWT を付ける。
企業は委譲、署名、`typ`、`aud`、`htm`、`htu`、`iat`、`ath` を検証してから、Session の主張（Personal Agent と企業ごとの User ID）を使う。
任意で、RFC 9421 と Web Bot Auth による operator の署名も付く。

[[wi-10861-support-poppy-authorization-server-profile]] は認可サーバーの役だけを扱い、資源側の検証を対象外にした。
企業が IdMagic の PEP やゲートウェイを API の手前に置く場合、Session の検証も同じ場所で行えると導入しやすい。

## 対象範囲

- 判断を最初に行う。PEP とゲートウェイで検証する案、評価 API へ渡して IdMagic で検証する案、企業側の実装に任せて手順だけを示す案を比べる。
- 採用した案の検証、`invalid_session` と `session_ended` の応答、監査での Personal Agent の識別。

## 対象外

- Poppy の Conversations、Tasks、ブラウザーの参加。
- Web Bot Auth の署名検証そのもの。[[wi-61710-decide-agent-detection-signals-and-web-bot-auth]] の判断に従う。

## 設計

判断の材料は、Poppy の採用状況、企業の API を IdMagic のゲートウェイの背後に置く構成の需要、委譲鍵の取得（Personal Agent の `jwks_uri`）による外向き通信の増加である。
外向き通信は、CIMD の取得と同じ SSRF の防御（REQ-OAUTH2-017）を使う。
Session の主張は識別にだけ使い、アクセストークンによる認可を置き換えない。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 三つの案を比べて採否を決める。
2. 採用した案だけを、仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Decision] 調査して採否と検証の場所を確定する。
- [ ] T002 [Spec] 採用した方式の規範を書く。
- [ ] T003 [Acceptance] 委譲のない Session、別の `aud`、`ath` の不一致、終了した Session の RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に、`mise run test-go-package -- ./backend/mcpgateway/...` など対象のパッケージと `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

検証前の Session の主張を監査や認可に使うと、別の利用者になりすませる。
検証の失敗では主張を一切記録しない。
