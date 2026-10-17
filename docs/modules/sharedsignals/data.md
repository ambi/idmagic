# SharedSignals のデータ

この文書は、SharedSignals の Aggregate を扱う。

## 集約

中心となるのは、`Agent` ごとの失効エポックである。
`KillAgent`、`DisableAgent`、`UnbindAgentCredential`、所有者のオフボーディング、検証した受信の Security Event Token（SET、RFC 8417）のいずれかを契機に、単調に前進する。
`OAuth2` の `Introspect` は、この値をアクセストークンの `issued_at` と比べ、即時の失効へ反映する（`LocalRevocation`）。

確定した失効は、SSF のストリームを通じて CAEP のイベントとして外部の受信側へ伝える（`EcosystemPropagation`）。
外部の送信側から受け取った検証済みのイベントも、同じ失効エポックへ反映する。

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `AgentRevocationEpoch` | `Agent` ごとの失効エポックと、最後に進めた理由 | `Agent` を参照する |
| `SsfStream` | 向き（`Transmit` または `Receive`）、状態、送信側の設定または受信側の設定 | `Tenant` を `tenant_id` で参照する |
| `SecurityEventDelivery` | 送信するストリーム、SET、状態、試行の回数、次の試行の時刻 | `SsfStream` を参照する |
| `ReceivedSecurityEvent` | 受信したストリーム、`jti`、検証の結果 | `SsfStream` を参照する |

- **判断**：失効を一つの時刻に畳む理由は、[失効を Agent ごとの一つの時刻に集約する](decisions.md#失効を-agent-ごとの一つの時刻に集約する)。
