---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-20
priority: p3
group: mcp-gateway
depends_on: [wi-38469-fetch-public-mcp-server-metadata-and-tools]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-094 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-095 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-096 }
---

# 公開された MCP サーバーの一覧を定期的に取得し、認証の結果を接続確認に示す

## 動機

[wi-38469](../done/wi-38469-fetch-public-mcp-server-metadata-and-tools.md) は、公開された MCP サーバーから、管理者の操作でだけ PRM と `tools/list` を取得する（REQ-OAUTH2-095、REQ-OAUTH2-096）。
取得の結果は応答として返すだけで保存しない。
そのため、上流でツールの説明や引数が変わっても、管理者が取得し直すまでカタログは変わらない。
Agent の権限の材料（REQ-OAUTH2-094）も、カタログの有無だけを見て、MCP サーバーが IdMagic のトークンを受け付けるかを示せない。

## 対象範囲

- 登録ごとに、管理者が一覧の定期取得を有効にでき、有効化を保存する。無効にしたら次の回から取得しない。
- 有効な登録について、REQ-OAUTH2-095 と同じ制限と同じ主体 `urn:idmagic:mcp-tool-discovery` で、定期的に取得する。
- 最後の取得の結果（メタデータの状態、`authorization_servers` が IdMagic を含むか、一覧の状態と時刻）を保存する。
- REQ-OAUTH2-094 の接続確認に、公開された MCP サーバーについて認証の項目（PRM が IdMagic を宣言し、トークンを提示した呼び出しが 401 以外を返したこと）を加える。
- MCP サーバーが introspection で一覧取得用のトークンを検査する構成への対応を比べ、採否を決める。

## 対象外

- 閉域の MCP サーバーからの取得。Gateway の報告（REQ-MCPGATEWAY-011、REQ-MCPGATEWAY-012）が担う。
- マネージド型の登録の上流からの取得。
- 取得した値の自動公開と、説明文からの権限の推測。

## 設計

着手時に次を決める。

- 定期取得の間隔、実行する場所（`worker` のジョブか）、テナントとプロセス全体の流量と同時実行の上限。
- 有効化を保存する場所（資源登録の列か、別の表か）と、管理画面での見せ方。
- 取得の結果を保存する形と、接続確認の項目の状態（確認済み、未確認、失敗、取得していない）。
- introspection への対応。wi-38469 は IdMagic のどの入口でもトークンを受け付けないことを優先し、introspection も `active: false` を返す。対応するなら、登録の評価用クライアントが自分の `resource` 宛てのトークンを問い合わせた場合だけ active にする案と比べる。
- THREAT-117（`aud` だけを検査する MCP サーバー）の受容の再検討。定期取得はトークンを繰り返し発行するので、受容の再検討の条件に当たる。

### 既存製品との比較

wi-38469 の調査（2026-11-19、各社の公式文書の本文）を引き継ぐ。

- AWS Bedrock AgentCore Gateway は、接続先の作成、更新、明示的な同期（`SynchronizeGatewayTargets`）のときだけ `tools/list` を呼び、結果をキャッシュする。自動の定期取得は文書になかった。
- IBM の Agent 基盤は、取り込んだツールを自動で更新せず、取り込み直しを求める。
- Microsoft Agent 365 は、開発者が登録時にツールを申告し、プレビューの間は再公開できない。

定期取得は比べた製品にない。採るなら、明示の同期と同じ制限を保ち、有効化を登録ごとの明示の選択にする。

## 計画

設計の問いを解いてから、仕様、保存、取得の実行、接続確認の画面の順に進める。

## タスク

- [ ] T001 [Design] 間隔、実行場所、保存の形、introspection への対応を決める。
- [ ] T002 [Spec] 要件と契約を更新する。
- [ ] T003 [App] 有効化、定期取得、結果の保存、接続確認の項目を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

着手時に主要ユースケースと検証タスクを決める。
定期取得は、無効化した登録へ接続しないことと、流量の上限で外へ接続しないことを観測する。

## リスク

定期取得は、管理者の操作がなくても外へ接続し、一覧取得用のトークンを繰り返し発行する。
有効化を明示の選択にし、取得ごとに監査を残す。
