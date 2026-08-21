# SharedSignals

Shared Signals Framework (SSF) と Continuous Access Evaluation Profile (CAEP) による、エージェントのほぼリアルタイムな失効を所有する。IdMagic は SSF の送信側と受信側の両方として振る舞う。

中心となるのは `Agent` ごとの失効エポックである。`KillAgent`、`DisableAgent`、`UnbindAgentCredential`、所有者のオフボーディング、検証済みの受信 Security Event Token (SET、RFC 8417) のいずれかを契機に単調に前進する。OAuth2 の `Introspect` はこの値をアクセストークンの `issued_at` と比較し、即時失効へ反映する (`LocalRevocation`)。

確定した失効は、SSF ストリームを通じて CAEP イベントとして外部の受信側へ伝える (`EcosystemPropagation`)。伝播はローカル失効の後に行うため、受信側の障害や遅延がローカル失効を妨げることはない。外部の送信側から受け取った検証済みイベントも、同じ失効エポックへ反映する。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [standards.md](standards.md) | 準拠する外部規範 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
