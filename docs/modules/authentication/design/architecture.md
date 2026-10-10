# Authentication のアーキテクチャ

この文書は、Authentication の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | ログインと本人の操作が User を読み、JIT が User を作る | 相手の `user` の Repository とユースケースを使う |
| `Tenancy` | パスワードのポリシーと信頼済みデバイスの有効期間の上書きを読む | 相手の `TenantRepository` を使う |
| `DataKeys` | TOTP の seed と外部 IdP の接続のシークレットを暗号化する | 相手の `FieldCipher` を使う |
| `Audit` | 相関用のソルトを得る | 相手の `TenantSaltStore` を使う |
| 共有の部品 | メールの送信、アクショントークン | `backend/shared/notification` と共通のアクショントークンの核 |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 確かめた認証の要素と認証の時刻を境界にする | 認証の完了の前の要求を、ロールでは扱えない |
| 機微な自己操作にステップアップ認証を求める | 詳細は[判断](decisions.md#機微な自己操作に-csrf-の防御に加えてステップアップ認証を求める) |
| 共有の一時的な状態は、到達できないとき失敗させる | 抑制のない試行を通さない |
| セレモニーと暗号処理を自作しない | WebAuthn は `go-webauthn/webauthn`、パスワードのハッシュは Argon2id に委ねる |

## 構成要素

コードは、機能スライスと、モジュールのルートの層に分かれる。

| 機能仕様 | 主なコード |
| --- | --- |
| [ログイン](../sign-in/README.md) | ルートの `usecases`、`handlers_http`、ログインのスロットル |
| [ログインセッション](../session/README.md) | `session` |
| [外部 IdP との連携](../federation/README.md) | `federation` |
| [パスワード](../password/README.md) | `password` |
| [多要素認証](../mfa/README.md) | `mfa` |
| [TOTP](../totp/README.md) | `totp` |
| [WebAuthn](../webauthn/README.md) | `webauthn` |
| [復旧コード](../recovery/README.md) | `recovery` |
| [信頼済みデバイス](../trusted-device/README.md) | `trusteddevice` |
| [アカウントポータル](../account-portal/README.md) | ルートの `handlers_http` と `deps_http` |
| [セキュリティ通知](../security-notification/README.md) | `securitynotification` |
| [サインイン履歴](../sign-in-activity/README.md) | ルートの `handlers_http` と `usecases` |

| ルートの層 | 責務 |
| --- | --- |
| `domain` | 機能に共通する型と認証のイベント |
| `ports` | 機能に共通する Repository と一時的な状態のストア |
| `usecases` | ログイン、ステップアップ、認証のイベント、保持期間の掃除 |
| `handlers_http`、`deps_http` | ブラウザー向けの認証 API、セルフサービス API、管理 API の組み立て |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| ログイン | ブラウザーの `/api/auth/*` への要求 | `api` が、スロットル、資格情報、第二要素の順に確かめ、セッションを作る | [ログイン](../sign-in/README.md) |
| セキュリティ通知 | 配信点を通るドメインイベント | 発行したプロセスが、配信点から切り離して送る | [セキュリティ通知の設計](../security-notification/design.md) |
| 保持期間の掃除 | 毎時、または `idmagic-batch retention-sweep` | バッチのプロセス | [認証のイベントの記録](authentication-events.md) |
