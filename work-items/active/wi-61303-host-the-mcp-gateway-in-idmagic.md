---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-11-11
priority: p1
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-005 }
---

# IdMagic がホストする MCP Gateway を加え、配置の作業なしに接続先を保護できるようにする

## 動機

[wi-66764](../done/wi-66764-decide-where-the-mcp-gateway-runs.md) で、Gateway の運用場所を段階的に増やすと決めた。
自前設置（MCP サーバーの近くと専用ホスト）は現行の Gateway で支え、IdMagic がホストする構成は、この項目で別に設計して加える。
判断と比べた候補は [McpGateway の設計判断](../../docs/modules/mcp-gateway/decisions.md#運用場所を段階的に増やす)にある。

Okta Agent Gateway は配置の作業がなく、登録の直後に Agent が接続できる。
IdMagic の自前設置は、配置、TLS、更新、監視が導入側の作業になり、初期構築の比較で重い。
ホスト型を加えれば、導入側は上流の場所と、データと資格情報を手元に置くかどうかで運用場所を選べる。

## 対象範囲

- テナントと接続先ごとの公開 URL を定め、保護リソースのメタデータ（REQ-MCPGATEWAY-005 の相当）を IdMagic が返す。
- 評価と転送を IdMagic のプロセスで行う実行単位、組み立て地点、所有するテーブルを決める。
- 上流への外向きの接続を、登録した宛先だけに限る。公開アドレスの検査、リダイレクトの拒否、期限、本文と応答の上限、流量を要件にする。
- 上流の資格情報を注入する。IdMagic の発行したトークンを受け付ける上流には下流のトークン交換（REQ-MCPGATEWAY-008 の相当）を、独自の OAuth を持つ社外の上流には TokenVault の利用者ごとのトークンを使う。
- 管理画面の接続先の登録で、運用場所（ホスト型、自前設置）を選べるようにし、選んだ場所で必要な入力だけを示す。
- 採用前にセキュリティレビューを行う。

## 対象外

- API キー型の上流の資格情報の保管。[wi-62382](wi-62382-decide-non-oauth-downstream-credentials-in-the-vault.md) の判断に従う。
- 閉じたネットワークの上流へ、導入側の中継を介して届ける構成。ホスト型の利用を見てから判断する。
- 競合製品の実テナントでの計測。

## 設計

着手時に次を決める。

- 実行単位。`api` のプロセスの中で動かすか、別の実行単位にするかを、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)で比べる。長く開く `text/event-stream` の接続数、障害の影響範囲、テナントの分離を制約にする。
- 判定の再利用。現行の Gateway の `Inspect` と `Decide` は作用のない計算なので、サーバー側から再利用できるかと、配布済みの Gateway との版のずれを比べる。
- 公開 URL。Agent の設定に配った後で変えにくいので、テナントのホスト名か経路のどちらで分けるか、接続先の識別子を URL に含めるかを決める。
- 外向きの防御。TokenVault の REQ-TOKENVAULT-009 から 011 と `backend/shared/security/safehttp` を共有できるかを確かめる。[wi-38469](wi-38469-fetch-public-mcp-server-metadata-and-tools.md) の直接取得と同じ防御を使う。
- 資格情報の注入。TokenVault の代理実行は本文の上限を持ち、ストリーミングを扱わない（TokenVault の判断の再検討の条件）。MCP の応答の逐次転送とどう両立させるかを決める。
- 迂回の防止。上流の資格情報を IdMagic だけが持つか、上流が IdMagic の下流鍵に束縛したトークンだけを受け付けることを、利用者へ示す手引きにする。

## 計画

着手時に設計の問いを解決し、主要ユースケース、仕様、TypeSpec を先に更新する。
セキュリティレビューは仕様と設計の確定後、実装の前に行う。

## タスク

- [ ] T001 [Design] 実行単位、公開 URL、資格情報の注入を比べて決める。
- [ ] T002 [Spec] ホスト型の要件、契約、設定を一次仕様へ反映する。
- [ ] T003 [Review] セキュリティレビューを行う。
- [ ] T004 [App] ホスト型の評価と転送を実装する。
- [ ] T005 [UI] 接続先の登録で運用場所を選べるようにする。
- [ ] T006 [Verify] 登録から許可、拒否、取消までを検証し、競合の初期構築と比べて比較表を更新する。

## 検証

着手時に主要ユースケースを決める。
上流への外向き接続が登録した宛先だけに届き、内部のアドレスとリダイレクトで迂回できないことを確認する。
あるテナントの公開 URL で、別のテナントの接続先と資格情報へ届かないことを確認する。

## リスク

ホスト型は IdMagic が上流の資格情報と業務の通信を扱うので、侵害の影響範囲が広がる。
テナントの分離、資格情報の保管、ログへの非出力を先に設計する。
公開 URL とテナントの対応は、Agent の設定に配った後で変えにくい。
