---
status: pending
authors: [tn]
risk: high
created_at: 2026-11-17
priority: p3
group: federation-protocols
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-SUBJECT-TOKEN }
  - { path: docs/modules/oauth2/standards.md, requirement: RFC8693-TARGET }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-008 }
---

# 採否を判断したうえで、Transaction Tokens を発行する

## 動機

[OAuth Transaction Tokens](https://datatracker.ietf.org/doc/draft-ietf-oauth-transaction-tokens/) は WG の合意に達し、RFC 化の手前にある。
外部から来た呼び出しの利用者、ワークロード、認可の文脈を、信頼ドメイン内のサービス連鎖へ短命で改変できないトークンとして伝える。

IdMagic のゲートウェイは許可した呼び出しごとに下流宛のトークンへ差し替える（REQ-MCPGATEWAY-008）。
しかし MCP サーバーの背後の内部サービスは、利用者、Agent、承認の文脈を受け取れず、元のトークンを転送するか文脈を失う。

## 対象範囲

- 判断を最初に行う。IdMagic を Transaction Token Service にする案、ゲートウェイの差し替えの延長で済ませる案、見送る案を比べる。
- 採用した場合の発行の入口、内容（`sub`、`act`、目的、要求の文脈）、署名鍵、検証の手順。

## 対象外

- 信頼ドメインをまたぐ連鎖（Identity Chaining）。
- 内部サービス側の検証ライブラリの各言語での提供。

## 設計

判断の材料は、RFC 化の時期、内部の文脈伝播を求める利用者の要望、既存の `act` と承認の束縛を Transaction Token へ写せるかである。
発行の入口はトークンエンドポイントでの RFC 8693 の交換が draft の形なので、既存の交換の検証を再利用できるかを先に確かめる。
署名鍵は外部向けのアクセストークンと分ける案を第一候補にする。
見送る場合は、再検討の条件を `## 要約` に書いて中止する。

## 計画

1. 三つの案を比べて採否を決める。
2. 採用した案だけを、仕様、受け入れテスト、実装の順に進める。

## タスク

- [ ] T001 [Decision] 調査して採否と方式を確定する。
- [ ] T002 [Spec] 採用した方式の標準表と規範を書く。
- [ ] T003 [Acceptance] 外部のトークンの直接の利用、文脈の改変、期限切れの RED を作る。
- [ ] T004 [App] 実装する。

## 検証

- 採用した場合に、対象のパッケージと `mise run verify-changed -- <着手時の基準リビジョン>` を実行する。

## リスク

Transaction Token を外部のアクセストークンとして受け入れる資源があると、短命の内部トークンで外部の権限を得られる。
`aud` と `typ` で区別し、外部の資源での拒否を試験する。
