# SharedSignals

Shared Signals Framework (SSF) と Continuous Access Evaluation Profile (CAEP) による、エージェントのほぼリアルタイムな失効を担う。IdMagic は SSF の送信側と受信側の両方として振る舞う。

中心となるのは `Agent` ごとの失効エポックである。`KillAgent`、`DisableAgent`、`UnbindAgentCredential`、所有者のオフボーディング、検証済みの受信 Security Event Token (SET、RFC 8417) のいずれかを契機に単調に前進する。OAuth2 の `Introspect` はこの値をアクセストークンの `issued_at` と比較し、即時失効へ反映する (`LocalRevocation`)。

確定した失効は、SSF ストリームを通じて CAEP イベントとして外部の受信側へ伝える (`EcosystemPropagation`)。伝播はローカル失効の後に行うため、受信側の障害や遅延がローカル失効を妨げることはない。外部の送信側から受け取った検証済みイベントも、同じ失効エポックへ反映する。

| 文書 | 内容 |
|---|---|
| [SharedSignals の用語集](glossary.md) | この Context での語義 |
| [SharedSignals の採用規範](standards.md) | 準拠する外部規範 |
| [SharedSignals の状態遷移](states.md) | 状態と遷移 |
| [SharedSignals の設計判断](decisions.md) | 設計判断 |
| [SharedSignals の内部設計](internals.md) | 機構の説明 |
| [SharedSignals Scenarios](scenarios.feature.md) | 受け入れシナリオ |
