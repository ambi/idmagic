# 制約

この文書は、IdMagic の設計の選択肢を外から狭める技術上の制約を扱う。
制約の内側で選んだ方式は[解決戦略](strategy.md)と[アーキテクチャ上の判断](decisions.md)が扱う。

| 制約 | 内容 | 出典 |
| --- | --- | --- |
| 標準仕様への準拠 | 連携の実装者は、OAuth 2.0、OpenID Connect、SAML 2.0、WS-Federation、SCIM 2.0 で接続する。プロトコルの形と意味は標準が決め、製品が変えられない | [プロダクト概要](../../requirements/product-overview.md#利用者)、各モジュールの `standards.md` |
| 前段のゲートウェイ | TLS の終端、同一オリジンの境界、低速接続への防御は、前段のゲートウェイまたはリバースプロキシが担う。製品はその内側で動く | [プロダクト概要](../../requirements/product-overview.md#対象としない責務)、[ネットワーク設計](../infrastructure/network.md) |
| 外部の配信経路 | メールと SMS の到達性、送信者の評価、通信事業者との接続は、外部の配信事業者が担う。製品は送信を依頼するだけである | [プロダクト概要](../../requirements/product-overview.md#対象としない責務) |
| 上流の権威 | 人事情報の正本は上流の権威にある。製品は取り込む側であり、在籍情報の発生源にならない | [プロダクト概要](../../requirements/product-overview.md#対象としない責務) |
| 実行基盤 | 製品は、ローカルの Docker Compose と汎用の Kubernetes の二つのデプロイプロファイルで動かせなければならない | [デプロイメントアーキテクチャ](deployment.md#デプロイプロファイル) |
| 言語とツールチェーン | バックエンドは Go、フロントエンドとリポジトリの道具は Bun で動かす。バージョンと実行環境は `mise.toml` の一か所で固定する | `mise.toml`、[開発文書](../../development/README.md) |
| API の契約の一次情報 | HTTP の操作、モデル、エラーの形は TypeSpec で宣言し、OpenAPI はそこから生成する | [仕様フォーマット](../../../SPECIFICATION_FORMAT.md) |
