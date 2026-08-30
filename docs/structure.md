# Structure

## Directories

```text
.
├── backend/           # Go Bounded Contexts, shared, cmd/
├── frontend/          # React UI and gateway
├── docs/              # human-authored product, specification, development, and runbook documents
│   ├── contexts/<context>/
│   ├── development/
│   └── runbooks/
├── spec/              # TypeSpec, generated views, and the OpenAPI release baseline
│   └── contexts/<context>/
├── infra/             # container, local runtime, and database schema assets
├── load/k6/           # tenant-local OAuth SLO smoke
├── mise.toml          # pinned development tools and repository task map
├── tools/             # specification, boundary, compatibility, and rendering tools
└── work-items/        # units of work, decision history, and completion records
```

依存は `spec` から実装と派生成果物へ向かって流れる。`backend` のドメイン層とユースケース層のパッケージが、アダプターやランタイムへ逆向きに依存することはない。

## Development flow mapping

| 関心事 | 置き場所 | 詳細 |
| --- | --- | --- |
| 仕様と設計 | `spec/**/*.tsp`, `docs/*.md`, `docs/contexts/**/*.md` | 規範的な振る舞い、契約、現在の根拠。変更はここから始まる。 |
| 開発の進め方と手順 | `docs/development/*.md` | 仕様先行のワークフロー、環境、生成、CI、テスト、リリース。 |
| 手動の運用手順 | `docs/runbooks/*.md` | 障害時または手動作業の最中に読む手順。 |
| 変更の記録 | `work-items/*.md` | 1 つの変更についての代替案、計画、作業、完了の記録。 |
| ドメインモデル | `backend/<context>/(<feature>/)domain` | フレームワークに依存しないドメインモデル。 |
| アプリケーションロジック | `backend/<context>/(<feature>/)usecase` | フレームワークに依存しないユースケース。 |
| ポート | `backend/<context>/(<feature>/)ports` | HTTP、永続化、通知などのポート。 |
| アダプター | `backend/<context>/(<feature>/){handlers_http,db_postgres,...}` | HTTP、永続化、通知などのアダプター。 |
| ランタイム | `backend/cmd/`, `backend/cmd/internal/bootstrap` | 起動、依存注入。 |
| インフラ基盤 | `infra` | インフラ基盤の設定コード。 |
| フロントエンド | `frontend` | フロントエンドコード。 |

## Stack

- **バックエンド**：Go。
- **フロントエンド**：React/TypeScript、Bun。
- **データベース**：PostgreSQL。
- **インフラ基盤**：Docker Compose、Kubernetes、Prometheus、Grafana、Loki、Promtail、k6。
- **開発ツール管理**：mise。Go、Bun、golangci-lint、sqlc、psqldef、PostgreSQL クライアントのバージョンとリポジトリタスクを `mise.toml` に集約する。

## Context internals

Bounded Context は通常、次の 4 層で構成する。

```text
backend/<context>/
  domain/            # エンティティ、値オブジェクト、状態遷移、純粋な検証
  usecase/           # 仕様で定めた操作を行うアプリケーションロジック
  ports/             # Repository、ストア、外部サービスの抽象
  handlers_http/     # 受信 HTTP アダプター
  db_memory/         # メモリ版 Repository アダプター
  db_postgres/       # PostgreSQL 版 Repository アダプター
```

アダプターはそれが属する Context または機能の直下に置き、snake_case の `<role>_<technology>` で命名する。

`<technology>` の位置には外部技術の名前だけでなく相手の Context 名も入る。他の Context の語彙を自分のポートの語彙へ翻訳するアダプター、すなわち Anti-Corruption Layer がその形を取る。現在は `backend/provisioning/source_idmanagement`、`backend/authorization/principals_idmanagement`、`backend/oauth2/policy_tenancy`、`backend/sourcing/scim/source_idmanagement` の 4 つである。

翻訳するアダプターを置くのは、Context Map が依存を許すほうの Context であり、翻訳される側ではない。`provisioning/source_idmanagement` は下流の Provisioning に立って IdManagement の `User` を自分の `AttributeSource` へ写す。`sourcing/scim/source_idmanagement` は反対に上流の Sourcing に立ち、IdManagement が公開する取り込み元判定のポートを満たす。IdManagement が Sourcing を知ると Context Map に無い向きの依存ができるためである。どちらに立つかは Context Map が決めるので、翻訳の向きから配置を推測しない。

翻訳する語彙の差が無ければ、専用のパッケージも置かない。`WorkloadIdentity` から `OAuth2` への関係では、OAuth2 が `ports.WorkloadTokenVerifier` を宣言し、実装は `backend/workloadidentity/usecases` が持ち、組み立て地点で結ぶ。越えるのが WorkloadIdentity の公開言語に含まれる戻り値の型 1 つだけであり、そこにアダプターを挟むと委譲だけの浅いモジュールが残るからである。

`backend/shared/` は、複数の Context が実際に共有する技術的な能力のための場所である。

起動時設定と実行時に選択可能な機能の定義も同じ意味で一点に集める。すべてのバックエンドプロセス (`idmagic`、`idmagic-worker`、`idmagic-batch`、`idmagic-seed`) は `backend/cmd/internal/bootstrap` が定義する単一の `Config` を通して環境を読み、`bootstrap` の外で環境変数を直接読まない。`FeatureRegistry` は実行時選択と更新影響だけを持ち、各 Context の API、標準対応、テナント設定を複製しない。読み取り点や選択規則が散らばると、あるプロセスだけが検証されない値または異なる機能集合を持つ状態が作れてしまうためである。運用者向けの設定リファレンスと機能メタデータはこれらの定義から生成し、手書きの一覧を併存させない。

具象のドメインイベントの構造体は、それが属する Context の `domain/events.go` に置く。`backend/shared/spec/events.go` はイベントのエンベロープとなるインターフェースと、そのワイヤ表現への変換だけを持つ。イベントが Context の境界を越えるときに何が契約になるかは [Cross-context events](#cross-context-events) が持つ。

2 つ以上の独立した機能を持つ Context は、4 層の構成に機能ごとの垂直分割を追加してよい：`backend/<context>/<feature>/{domain,ports,usecase,<role>_<technology>}/`。機能が 1 つしかない Context は分割しない。

```text
backend/idmanagement/
  module.go                 # Context ごとに 1 つ置く DI の組み立て
  domain/                   # 機能間で共有する型と計算だけ（列挙、DomainEvent、CSV 基盤）
  ports/                    # 機能間で共有するポートだけ（CSV 成果物ストア）
  usecase/                  # 機能をまたぐユースケース補助とエラー値だけ
  db_memory/  db_postgres/  # 機能間で共有するポートのアダプター
  deps_http/                # Deps 型を定義する末端パッケージ
  handlers_http/            # ルート登録と機能をまたぐ統合テスト
  user/  group/  agent/     # 各機能の domain、ports、usecase、アダプター
```

機能をまたぐ層に置いてよいのは、複数の機能が同じ意味で使う語彙と機構に限る。CSV の転送ポリシー、解析器、可逆なセル変換、不変な成果物ストアは `User` と `Group` が同じ意味で共有するためここに置き、列の語彙と計画器は Aggregate ごとの不変条件なので機能側に残す。

## Cross-context events

[README.md](README.md#context-map) の Context Map が `Events` と型付けする関係は、性格の異なる 2 つの機構で実現している。どちらもイベントバスではなく、メッセージ基盤も介さない。

**ライフサイクルの通知は、ドメインイベントを 1 件も運ばない。** IdManagement から IdGovernance と Provisioning への通知は、上流の IdManagement が語彙とポートを宣言し、下流の Context がそれを実装する同期の呼び出しである。IdGovernance は `idmanagement/user/ports` の `UserMutationCommitter` を実装し、User の保存と、そこから導かれる LifecycleWorkflow の実行の生成を 1 つのトランザクションで確定する。Provisioning は同じ package の `ProvisioningNotifier` を実装し、呼び出し元のコミットが済んだ後に自分のトランザクションで配信対象を捕捉する。上流が公開言語を持ち下流が従う形なので、これらは公開イベントによる関係ではなく Open Host Service である。

**監査の事実は、組み立て地点に 1 つだけある配信点を通る。** `backend/cmd/internal/bootstrap` が組み立てる発行の閉包が、`EventSink` への出力、アカウントのセキュリティ通知のディスパッチ、監査記録の追記を順に行う。ドメインイベントを発行する Context はこの閉包だけを関数として受け取り、監査にも通知にも依存しない。逆に消費する側も発行元の Go の型を知らず、後述のワイヤ表現の上だけで動く。したがってこの関係に import は存在せず、依存の向きはどちらの側にも生じない。

### The published language of events

イベントが境界を越えるときの契約は、payload の全体ではない。`AdminAuditEventResponse.payload` は意図して不透明な JSON であり、Context の内部でしか読まれない項目はその Context のものである。契約になるのは次の 2 つに限る。

- **エンベロープ**：`spec.MarshalDomainEvent` が必ず載せるイベント種別名と発生時刻。監査の記録、管理 API の応答、セキュリティ通知のディスパッチがすべてこの形の上で動く。
- **公開項目の語彙**：他の Context が名前で読む payload の項目。監査の検索属性の抽出器がこれを検索軸へ写し、セキュリティ通知が宛先と送信条件をここから解決する。

どちらも `spec/contexts/system/models.tsp` の `DomainEventEnvelope` と `DomainEventPayload` が正本である。配信点を所有する System が持ち、消費者である Audit は持たない。供給側が下流の契約に従う倒立を避けるためである。

宣言を置くだけでは、読み取り側と静かに食い違う。項目名を変えてもコンパイルは通り、監査の絞り込みが空を返すようになるだけだからである。`mise run check-event-contract` が、宣言された語彙と Go の読み取り点の集合が一致することを確かめる。

### Compatibility of published events

公開したイベント種別名と公開項目の名前は、削除も改名もしない。監査記録は追記のみで 7 年保持するので、名前を変えても既存の行は書き換えられず、古い行だけが新しい軸から見えなくなる。項目の追加と、まだ誰も読んでいない内部項目の変更は、この規則の対象ではない。

リリース済みベースラインとの互換性判定は持たない。公開イベントの消費者はこのリポジトリの中にしかおらず、外部の消費者がいない契約にベースラインを敷いても守る相手がいないためである。この判断は、外部に配信する Security Event Token には及ばない。あちらは RFC 8417 が別の契約を定めている。

## Frontend component structure

仕様上の機能とそろえた UI の境界は `frontend/src/features/<feature>/` に置く。その機能のビュー、ローカルコンポーネント、ヘルパー、テスト、ローカライズ辞書（`*.i18n.ts`）は必ずそのディレクトリに置く。特定の機能境界にひも付かない、横断的で再利用可能なコンポーネントは `frontend/src/components/` に置く。

## HTTP routing

HTTP ルーティングは `backend/shared/http/server_http/routes.go` で組み立てる。ここがテナント単位のルートを既定のテナントと `/realms/:tenant_id` の両方に登録し、制御面のテナント管理だけを `/realms/default/admin/tenants` に分離する。

各 Context のルーティングは `backend/<context>/handlers_http/routes.go` にある。正確なエンドポイントの一覧はそのファイルを参照する。新しい HTTP API は、それが属する Context の `routes.go` に、同じ `handlers_http` 配下のハンドラーとともに登録する。Context 固有の Repository とルーティングの接続は `backend/<context>/module.go` に集約し、中央のルーターは Module を呼ぶだけにする。

## Architecture style

単一の Go モジュール内で Bounded Context の境界を保ちつつ、複数の実行単位が実装を共有する現在のアーキテクチャを **Modular Monolith** とする。Context 間は公開された言語とポートで接続する。

通常は複数の Context を 1 つの API プロセスに組み合わせ、リソースやレイテンシーの特性が異なるジョブと横断的なバッチ処理だけを別の実行単位にする。独立したデータ所有権、担当チーム、SLO が必要になるまではサービスを分割しない。この記述は現在の設計を示すものであり、将来も同じ構成を義務付けるものではない。

`backend/cmd/internal/bootstrap/deps.go` の `Dependencies` は HTTP 層へ渡す依存を集約し、メモリ、PostgreSQL、コンソール、OpenTelemetry など実行時の実装選択を吸収する。Context 固有の Repository は各 `Module` にまとめ、中央の `Dependencies` とサーバーの `Deps` はその Module を受け取る。ポートを追加した場合は、その Context の `ports/`、メモリと PostgreSQL の各アダプター、スキーマ変更の要否、`bootstrap.Dependencies`、`assembleMemory`、`assemblePostgres`、`support.Deps`、関連する HTTP ハンドラーまたはユースケースの構築処理を確認する。
