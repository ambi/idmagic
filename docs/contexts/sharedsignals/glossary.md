# SharedSignals Glossary

| Term | Definition | Aliases |
|---|---|---|
| SSF | OpenID Shared Signals Framework。Security Event Token (SET) を搬送形式として、セキュリティイベントをプッシュまたは受信する標準。IdMagic は送信側と受信側の両方として振る舞う。 |  |
| CAEP | OpenID Continuous Access Evaluation Profile。SSF 上で `session-revoked`、`token-claims-change`、`credential-change`、`assurance-level-change` を表現するイベント種別の規約。 |  |
| RevocationEpoch | `Agent` ごとに保持する単調増加のタイムスタンプ。これより前に発行されたアクセストークンやセッションは、フェイルクローズで無効とみなす。`KillAgent`、`DisableAgent`、`UnbindAgentCredential`、所有者 (`owner_user_id`) の無効化または削除、受信した `SecurityEvent` のいずれでも前進する。所有者のオフボーディングでは、配下の全 `Agent` のエポックを一括して前進させる。 |  |
| LocalRevocation | IdMagic 自身の `Introspect` または保護 API が、失効エポックとアクセストークンの `issued_at` を比較して行う、当該 IdP 内で完結する失効判定。CAEP / SSF イベントの配送より常に先に確定する。 |  |
| EcosystemPropagation | `LocalRevocation` で確定した失効を、SSF ストリームを通じて外部の受信側 (別の IdP、リソースサーバー、ガバナンス基盤) へ CAEP イベントとして伝える層。受信側の障害や遅延によって `LocalRevocation` を遅らせない。 |  |
| FailClosed | SET の署名検証失敗、未知の鍵、改ざんの検知、未登録の発行者、重複の検知、失効エポックを判定できない場合のいずれでも、常に「反映しない」または「無効とみなす」側に倒す方針。 |  |
| SubjectIdentifier | SET が指す主体の表現。RFC 9493 は `format` メンバーで種別を区別する標準形式を定める。IdMagic は自身の送信側が使う独自形式に加えて、外部の送信側との相互運用のために RFC 9493 の `iss_sub` と `opaque` を解釈する。 | Subject Identifier, サブジェクト識別子 |
| System | `SharedSignals` の失効反映と SET 送受信のユースケースそのものを指す、人間の操作者を伴わない技術的な主体。`KillAgent` などの Domain Event を契機に失効エポックを進め、CAEP イベントを生成する。 |  |
