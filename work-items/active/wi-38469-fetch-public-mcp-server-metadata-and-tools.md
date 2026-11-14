---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-04
priority: p2
group: mcp-gateway
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-086 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-087 }
---

# 公開された MCP サーバーから、IdMagic がメタデータとツールの一覧を取得して候補に示す

## 動機

[wi-58793](../done/wi-58793-fill-mcp-connection-candidates-from-metadata-and-change-notices.md) から分けた。
Gateway を使わず、MCP サーバー自身がトークンを検査する構成では、ツールの一覧を IdMagic へ報告する主体がない。
インターネットから届く MCP サーバーであれば、IdMagic が直接取得すれば、管理者がツール名とスコープを手で入力せずに済む。

利用者の判断により、公開された MCP サーバーについて IdMagic が直接取得する経路を採る。
閉域の MCP サーバーへは IdMagic から接続できないため、この経路の対象外とする。
閉域のサーバーには、Gateway の起動時の報告（REQ-MCPGATEWAY-011）を使う。

## 対象範囲

- 登録の URL から Protected Resource Metadata（RFC 9728）を取得し、`scopes_supported` と `authorization_servers` を、出所を付けてスコープの候補に示す。
- 公開された MCP サーバーの `tools/list` を取得し、既存の報告と同じ候補として示す。
- REQ-OAUTH2-086 の判断「IdMagic から MCP サーバーへ取得に行かない」を改める。

## 対象外

- 閉域の MCP サーバーからの取得と、閉域側で動かす報告用のコマンド。
- 取得した説明文から権限や承認の要否を推測すること。
- 取得した値の自動公開。公開と割当は管理者の選択に限る。

## 設計

着手時に次を決める。

- 取得の制限。宛先は既存の `backend/shared/security/safehttp` で公開アドレスに限り、リダイレクト、取得量、期限、流量の上限を要件にする。
- `tools/list` の認証に使うトークンの持ち方。PRM は認証なしで取得できる。トークンについては、IdMagic が自分宛てに発行する方式と、管理者が資格情報を渡して保管する方式を比べる。前者は主体の種類の追加として認可を設計し、後者は保管の方式を設計する。
- PRM の `resource` と登録の URL の一致の検査（RFC 9728 §3.3）。
- 取得を管理者の操作で起こすか、登録時に自動で行うか。

外向きの防御は、[[wi-61303-host-the-mcp-gateway-in-idmagic]] の上流への転送と、TokenVault の REQ-TOKENVAULT-009 から 011 と同じものを使う。
先に着手した項目が共通の部品を決め、後の項目はそれを再利用する。
[[wi-85145-check-mcp-connection-without-calling-tools]] の接続確認は、MCP サーバーへの到達をこの項目の経路で行う。

## 計画

今回は起票だけを行う。

## タスク

- [ ] T001 [Design] 取得の制限、トークンの持ち方、取得の契機を決める。
- [ ] T002 [Spec] 要件と契約を更新する。
- [ ] T003 [App] 取得と候補の表示を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

着手時に、主要ユースケースと検証タスクを決める。
宛先の制限は、非公開アドレス、リダイレクト、上限の超過の各境界で拒否を観測する。

## リスク

テナントの管理者が宛先を指定するため、取得の制限を誤ると SSRF になる。
取得した値は MCP サーバーの申告として候補に示すだけにし、公開と割当には使わない。
