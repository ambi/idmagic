# IdMagic

**プロダクションレディなエンタープライズ向け ID プロバイダー。**

IdMagic は Go で実装したマルチテナントの IdP / IdM である。OAuth 2.0 / OpenID Connect、SAML 2.0、WS-Federation による認証と、SCIM 2.0 による双方向のアイデンティティ連携、ユーザーと AI エージェントのライフサイクル管理、監査を 1 つの製品として提供する。

## 主な機能

- **ID プロトコル**: OAuth 2.0 / OIDC の認可サーバー、SAML 2.0 IdP、WS-Federation Passive Profile、WS-Trust STS。FAPI 2.0 Security Profile と RFC 9700 (BCP 240) に沿って運用する。対応する RFC と対応範囲は [docs/contexts/oauth2/standards.md](docs/contexts/oauth2/standards.md) をはじめとする各 Context の `standards.md` が持つ。
- **マルチテナント**: テナントごとの正規ロケーションと発行者、自動ローテーションする署名鍵、ブランドを差し替えられるログイン画面とポータル。
- **エージェントを第一級の主体として扱う**: 自律型・監督型の AI エージェントを `Agent` として登録し、所有者、目的、ライフサイクルを管理する。Kubernetes や SPIFFE の JWT-SVID を長期シークレットなしでトークンへ交換でき、代行はなりすましではなく委譲を既定とする。人間の承認が要る操作は CIBA を通し、キルスイッチは即時に効く。
- **ID ガバナンス**: 入社・異動・退職をライフサイクルワークフローとして自動化し、実行前に結果を試算できる。
- **双方向のデータ連携**: 上流の IdP から SCIM 2.0 サーバーとして受け取り、下流の SaaS へは SCIM 2.0 クライアントとして反映する。管理者向けには CSV の一括インポートとエクスポートを提供する。
- **宣言的なクレーム開示**: テナント定義の属性スキーマと、属性からクレームへの宣言的な対応付け。OIDC、SAML、WS-Federation が同じフェイルクローズの発行経路を共有する。
- **継続的アクセス評価 (SSF / CAEP)**: OpenID Shared Signals Framework の送信側と受信側の両方として動く。失効はまず自身の中で即時に確定させ、そのうえで署名済みの Security Event Token として外部へ伝える。
- **資格情報ポリシー**: NIST SP 800-63B-4 に沿ったパスワード規則を既定とし、TOTP、パスキー (WebAuthn)、復旧コードによる多要素認証に対応する。
- **単一のデータストア**: 状態は PostgreSQL に集約し、第 2 のデータストアを運用しない。永続ジョブはレーンごとの `worker` が処理し、API とは独立して台数を増やせる。

## 対象外

対象範囲の宣言は [docs/README.md](docs/README.md) の Context Map の索引である。Bounded Context が増減したときだけ変わるので、機能の箇条書きより長く正しくいられる。

次は IdMagic が担わない。境界を接する相手が具体的なものだけを挙げる。

| 対象外 | 代わりに担うもの |
| --- | --- |
| 人事情報の正本 | 人事システムなどの上流の権威。IdMagic は `Sourcing` として取り込む側であり、在籍情報の発生源にはならない。 |
| メールと SMS の配信経路の運用 | 外部の SMTP サーバーと配信事業者。IdMagic は送信を依頼するだけで、到達性、送信者評判、キャリア接続は持たない。 |
| 汎用の API ゲートウェイと WAF | 前段のゲートウェイまたはリバースプロキシ。TLS の終端、同一オリジン境界の成立、slowloris のような過負荷への対処はそちらが担う（[docs/deployment.md](docs/deployment.md)）。 |

## クイックスタート

組み込み PostgreSQL、バックエンド API、非同期処理の `worker`、フロントエンドを含む Docker 不要のローカルスタックを起動する。

```bash
mise run dev
```

初回実行時に、約 190 MB の組み込み PostgreSQL バイナリをダウンロードしてキャッシュする。開発データは一時的なもので、スタックの停止時に削除される。PostgreSQL のローカルエンドポイントは `127.0.0.1:55432` である。

<http://localhost:5173/> を開き、ローカルデモ認証を選択する。使用できるアカウントは `alice`（テナント管理者）と `root`（テナント管理者とシステム管理者を兼ねる）で、どちらもパスワードは `demo-password-1234` である。

```bash
mise run dev-memory     # 永続ジョブと worker を使わない最小構成
mise run dev-compose    # PostgreSQL、OTel Collector、Prometheus、API、UI ゲートウェイ (http://localhost:8080/)
mise run demo-ciba      # CIBA のポーリング方式による承認フローを試す
```

## 主なコマンド

このリポジトリはツールのバージョン管理とコマンド一覧を `mise` に統合している。最初に `mise install` で固定バージョンを導入し、基本的なコマンドは下地のツール（`bun`、`go`、`docker` など）を直接呼ばず `mise run` のタスクから実行する。

```bash
mise install
mise tasks
mise run setup
mise run verify
mise run verify-go
mise run verify-ui
mise run test-ui-e2e
mise run spec-render     # 仕様の HTML を spec/generated/docs/index.html へ生成する
```

ビルド時にバージョンのメタデータを埋め込むには `VERSION=1.0.0 mise run build-go` とする。指定しない場合は `0.0.0-dev` になる。Docker ビルドではビルド引数 (`VERSION`、`GIT_COMMIT`、`BUILD_DATE`) で渡す。

## 設定

ローカルの既定はメモリ永続化とコンソールへのメール出力である。すべての起動時環境変数について、型、デフォルト、条件付き要件、所有するプロセス、シークレット区分を確認するには、生成済みの [設定リファレンス](CONFIGURATION.md) を参照する。不正な起動設定は、リスナー、依存先への接続、seed の適用を始める前にすべてまとめて報告し、プロセスは起動しない。

WebAuthn はパスキーをブラウザーのオリジンと RP ID に束縛する。ローカル以外の環境では HTTPS を使い、`WEBAUTHN_RP_ID` にはユーザーが訪れる登録可能なドメインを、`WEBAUTHN_RP_ORIGINS` には UI が使用するすべての公開オリジンを設定する。

開発中の SMTP テストには [Mailpit](https://mailpit.axllent.org/) が使える。

```bash
mailpit --smtp 127.0.0.1:1025 --listen 127.0.0.1:8025
EMAIL_SENDER=smtp SMTP_HOST=127.0.0.1 SMTP_PORT=1025 SMTP_TLS=none SMTP_FROM=noreply@idmagic.test mise run dev
```

### 運用上の注意

- **データ暗号鍵のマスターキーを失うと復旧できない。** `DATA_KEY_PROVIDER` が保持するマスターキーは PostgreSQL のバックアップに含まれない。手順と危険性は [バックアップ・復元・災害復旧のランブック](docs/runbooks/backup-restore-dr.md) を参照する。`DATA_KEY_PROVIDER` を設定しない場合はプロセス内の平文鍵セットを使うため、本番では決して選択してはならない。
- **テナントの正規ロケーションの変更は ID 移行である。** `path` 形式と `subdomain` 形式を切り替えると発行者とプロトコルメタデータの URL が変わり、RP の再設定、既存パスキーの再登録、進行中セッションの終了が必要になる。`subdomain` 形式にはワイルドカード DNS と対応する TLS 証明書が要る（[infra/README.md](infra/README.md)）。

## 文書

| 読みたいこと | 場所 |
| --- | --- |
| 貢献の作法と Pull Request に求めること | [CONTRIBUTING.md](CONTRIBUTING.md) |
| 脆弱性の報告 | [SECURITY.md](SECURITY.md) |
| 利用条件 | [LICENSE](LICENSE)（MIT License） |
| 仕様と設計の入口、Context Map | [docs/README.md](docs/README.md) |
| API とモデルの契約 | [spec/main.tsp](spec/main.tsp)、`mise run spec-render` で生成する HTML |
| 開発の進め方（仕様先行のループと検証） | [DEVELOPMENT.md](DEVELOPMENT.md) |
| 仕様文書と work item の書式 | [SPECIFICATION_FORMAT.md](SPECIFICATION_FORMAT.md)、[WORK_ITEM_FORMAT.md](WORK_ITEM_FORMAT.md) |
| 文書の種類、責務、配置の考え方 | [DOCUMENTATION_GUIDE.md](DOCUMENTATION_GUIDE.md) |
| 進行中と完了済みの変更の記録 | [work-items/](work-items/) |
| 仕様、境界、互換性、生成の各ツール | [tools/README.md](tools/README.md) |
| 起動時設定の一覧 | [CONFIGURATION.md](CONFIGURATION.md) |
| Kubernetes、監視、負荷スモーク | [infra/README.md](infra/README.md) |
| PostgreSQL のスキーマ運用 | [infra/schema/README.md](infra/schema/README.md) |
| 障害時の手順 | [docs/runbooks/](docs/runbooks/) |
| UI の設計指針とローカライズ | [frontend/README.md](frontend/README.md) |
| seed プロファイル | [seed/README.md](seed/README.md) |
