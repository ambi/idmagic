---
status: completed
---

# 送信者制約を保ったままゲートウェイが資源宛のトークンへ差し替える

評価が元の鍵の証明を確かめた結果として一回限りの交換の許可を発行し、登録したゲートウェイがそれを根拠に、Agent のトークンを自身の鍵へ束縛した資源宛の短命なトークンへ交換できるようにした。
`idmagic-mcp-gateway` は `downstream` の設定があれば呼び出しごとに差し替えて転送する。

仕様は REQ-OAUTH2-079 と REQ-MCPGATEWAY-008 を追加し、REQ-OAUTH2-048、056、060、REQ-MCPGATEWAY-001、004 を変更した。
契約は、規則の `token_exchange_resources`、評価の `token_exchange_resource` と `token_exchange_grant`、`/token` の `token_exchange_grant`、`TokenExchanged` の `senderConstraintRebound`、不許可の理由 `token_exchange_not_permitted` を加えた。
テーブル `oauth2_token_exchange_grants` と THREAT-098 を追加した。

判断は次のとおりである。
登録だけを根拠にする案は盗んだトークンを持ち込めるので、Agent の改修が要る事前の委譲の案は無改修の価値を損なうので採らなかった。
Bearer への書き換えは許可があっても受け付けず、承認の操作には許可を出さない。
Agent が主体のトークンは再束縛後も主体を保ち、`act` がゲートウェイだけのトークンの評価では、登録したゲートウェイを行為者に数えない。

検証は、正式な `/token` と評価の受け入れテスト、実際の評価と交換を通して下流の評価の許可を観測するゲートウェイの受け入れテスト、PostgreSQL の並行消費、単体テスト、`verify-changed`、`check-schema` である。
手動の故障注入として、許可の照合の無効化、宣言のない宛先の許可、Bearer の許容、元のトークンの転送、交換失敗時の転送を試し、すべて検出された。

確認の限界と残るリスクは次のとおりである。

- ゲートウェイの受け入れテストは、試験の URL 宛の証明を同じ鍵で発行者の URL へ署名し直す前段を置いている。`/token` の `htu` の値は単体テストだけが確かめ、実際の発行者の URL での結合は確かめていない。
- 実在の Okta、Ping、別の言語の MCP SDK との相互運用は確かめていない。
- mTLS は交換のユースケースだけを確かめ、ゲートウェイは DPoP の鍵だけを持つ。
- ゲートウェイを乗っ取られると、受けた要求の範囲で再束縛できる。
- Envoy の構成での差し替えと管理画面の専用の入力は未実施である。
