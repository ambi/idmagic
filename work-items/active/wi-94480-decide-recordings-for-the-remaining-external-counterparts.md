---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-26
priority: p3
group: development-process
depends_on: [wi-27706-verify-upstream-oauth-token-endpoints-with-recordings]
change_kind: tooling
spec_impact: { kind: none, reason: "残りの外部の相手ごとに記録の再生と模倣の採否を決め、採るものに検査を足す。規範シナリオ、標準の宣言、TypeSpec シンボルを追加も変更もしない。検査が規範に反する実装を見つけたら、その時点で affected_spec を宣言し直す。" }
---

# 残りの外部の相手について、記録の再生と模倣を入れるかを決める

## 動機

外向きの HTTP を送る本番コードを 2026-11-26 に洗い出すと、SCIM（[wi-422](../done/wi-422-external-contract-and-failure-verification.md)）、上流のトークンエンドポイント、上流の OIDC IdP、Vault と OpenBao の Transit、上流の MCP サーバーのほかに、次の相手があった。

| 相手 | 送る側 | 壊れたときの影響 | 記録の出どころ |
| --- | --- | --- | --- |
| Microsoft のレルム探索 | WS-Federation | ドメインの送り先を誤る | 非公開の API で、転記できる一次資料がない |
| HIBP の Range API | パスワードの方針 | 漏洩したパスワードの検査が効かなくなる | 公開の API 文書 |
| OAuth Client ID Metadata Document | OAuth2 | クライアントを解決できない | 仕様の例 |
| 外部の JWKS | 共有のトークン検証、Workload Identity | 署名を検証できない | RFC 7517 の例 |
| Workload の issuer の Discovery | Workload Identity | 信頼設定の候補を出せない | OIDC Discovery の例 |
| SET の受信者 | SharedSignals | 受信者に届かない | RFC 8935 の例 |

どれも前提が単純か、既存の単体テストやファズが形を確かめているので、記録と模倣が見つけるものは SCIM より少ないと見込む。
採否は、記録と模倣の費用と、見つけられる故障を相手ごとに比べてから決める。
レルム探索だけは一次資料がなく、記録は実物からしか取れない。

Directory Connector から Sourcing への呼び出しと、MCP Gateway から評価 API への呼び出しは、自分で提供する契約（TypeSpec）なので、消費する側の検証の対象にしない。

## 対象範囲

- 上の相手ごとに、前提にしている振る舞い、吸収している層、既存の検査を書き出し、記録と模倣の採否と理由を決める。
- 採ったものに、[wi-27706](../done/wi-27706-verify-upstream-oauth-token-endpoints-with-recordings.md) が決めた共有の部品で記録の再生と模倣を置く。
- 採らなかったものは、理由をこの記録の要約に残す。

## 対象外

- 自分で提供する契約の検証。提供する側の検査（TypeSpec と互換性の検査）が扱う。
- 実在の相手への常時の接続。

## 設計

比べる観点は、壊れたときの影響、相手が変わる頻度、既存の検査が同じ故障を見つけるか、記録を取る費用である。
レルム探索は、実物から記録を一度取って再生できるかと、利用規約の上で記録を残してよいかを確かめてから採否を決める。

## 計画

1. 相手ごとに比べて、採否を決める。
2. 採ったものに記録と再生、必要なら模倣を置く。

## タスク

- [ ] T001 [Research] 相手ごとに比べて採否を決める。
- [ ] T002 [Adapter] 採ったものに記録と再生、模倣を置く。
- [ ] T003 [Verify] 実際の差分から選んだ検査を実行する。

## 検証

- 採ったものについて、記録の応答を仕様に反する形へ変えると再生の検査が落ち、模倣を記録から外すと照合が食い違いを報告する。
- `mise run verify-changed -- main`。

## リスク

採らなかった相手は、相手の変化を検出しないまま残る。
採らなかった理由に、見直す契機（障害の発生、相手の仕様の改訂）を書く。
