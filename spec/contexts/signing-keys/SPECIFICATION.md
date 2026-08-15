---
context: signing-keys
updated_at: 2026-08-15
---

# SigningKeys Specification

## Overview

テナント単位の署名鍵素材について、ライフサイクル、ローテーション、公開の重複期間、監査をプロトコル横断で所有する。OAuth2 / OIDC は JWK / JWKS と JWT 署名器、SAML / WS-* は X.509 証明書と XML 署名アダプターを使用するが、鍵の用途、ローテーション、テナント間の分離に関する規則はここに集約する。

鍵プロバイダーの選択と保管もここに含まれる。鍵素材は公開された署名ポートと公開鍵ポートを通じてのみ提供し、各プロトコル形式への直列化は OAuth2、SAML、WS-Federation のアダプターが担う。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| SigningKeys | テナント単位の署名鍵素材について、ライフサイクルと公開を扱う境界。OAuth2 / OIDC は JWK / JWKS、SAML / WS-* は X.509 証明書を使用する。 | KeyMaterial, signing keys |
| Retire | SigningKey を Verifying から Retired に移す。 | retire |
| Archive | SigningKey を Retired から Archived に移す。 | archive |
| Verifying | 署名はしないが、過去発行トークンの検証のため JWKS に残っている状態。 | verifying |
| Retired | JWKS から除去された状態。新規検証には使われない。 | retired |
| Archived | 監査用に長期保管されている終端状態。鍵マテリアルは封印。 | archived |
| KeyProvider | 鍵素材の保管方式と署名の実行主体。Local / Database は秘密鍵をプロセス内に読み込み、アプリケーションが署名する開発・テスト用の方式である。VaultTransit は秘密鍵を Vault 内に保持し、署名を Vault API に委ねる本番用の方式である。Database は特定の製品名を表さない。 | key provider, 鍵プロバイダー |
| VaultTransit | HashiCorp Vault の Transit secrets engine を使う KeyProvider。秘密鍵マテリアルは Vault 外に出ず、署名要求ごとに Vault へ委譲する。 | Vault Transit |
| FailClosed | KeyProvider が不達のとき、新規トークン発行を停止する挙動。既発行トークン検証用の JWKS は取得可能な範囲で返す。強制点は OAuth2.Token の requires が持つ。 | fail-closed, フェイルクローズ |

## State Transitions

### SigningKeyLifecycle

署名鍵のライフサイクル (SigningKeyMinJwksOverlap)。Active から Rotate で Verifying に降り、Retire で JWKS から外し、Archive で監査保管に入る。

Initial: `Active` Terminal: `Archived`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | SigningKeyRotated | — | Verifying |  |
| Verifying | SigningKeyRetired | — | Retired |  |
| Retired | SigningKeyArchived | — | Archived |  |

## Authorization Boundary

鍵の一覧と参照は `AdminKeysRead` (`admin:keys_read`)、ローテーションは `TenantKeysRotate` (`admin:keys_rotate`)、検証用鍵の無効化は `TenantKeysDisable` (`admin:keys_disable`) を要し、いずれも `admin` または `system_admin` ロールを持つ、有効かつ認証済みのユーザーが所属テナントに対して行える。テナント横断の健全性一覧 `SystemKeyHealthRead` (`admin:keys_health_read`) だけは `system_admin` に限る。

管理 API のレスポンスに秘密鍵素材を含めることは、いずれの権限でもできない。返すのは `kid`、状態、有効期間、証明書、フィンガープリントに限る。公開鍵を配る JWKS とフェデレーションメタデータは認証を要さず、テナントごとのエンドポイントで公開する。

## Design

### Usage and scope isolation

鍵を取得するときは必ず、リクエスト元のテナント、`KeyUsage`、外部に意味を公開しないスコープ ID を指定する。用途とスコープを明示しない呼び出し元には `Signing` とデフォルトスコープを適用し、OAuth2 / OIDC 向け API を簡潔に保つ。XML プロトコルのアダプターは `XmlFederationSigning` を明示的に選び、SAML ではさらに IdP プロファイル ID をスコープとして指定する。これにより、JWT 用の鍵が XML Assertion に使われたり、ある SAML プロファイルが別のプロファイルの資格情報を使ったりすることを防ぐ。

ローカル、PostgreSQL、Vault の各アダプターは、テナント、用途、スコープの組み合わせごとに有効な鍵を 1 つだけ保持する。PostgreSQL では部分一意インデックスによって同じ不変条件を保証し、Vault では鍵集合の識別情報にスコープを含める。これは、1 つの SAML プロファイルの鍵をローテーションしたときに、別のプロファイルやすべての JWT 検証鍵まで変更されることを防ぐためである。

### XML federation credentials

XML フェデレーション用の鍵には、その公開鍵を含む自己署名 X.509 証明書を対応付ける。証明書は公開メタデータとして扱うが、秘密鍵は設定されたプロバイダーで保管し、管理 API のレスポンスには決して含めない。新しいメッセージには現在有効な鍵で署名する一方、有効期限内の検証用証明書は、ローテーションの重複期間中も SAML と WS-Federation のメタデータから参照できるようにする。

Local と Database の `KeyProvider` は、署名の間 RSA 秘密鍵をプロセス内に保持する。VaultTransit は秘密鍵を渡さないまま `crypto.Signer` を実装する。パディングは用途で使い分け、JWT の署名には RSA-PSS を、XML Signature と X.509 の操作には PKCS#1 v1.5 を使う。後者のワイヤー形式が RSA-PSS ではなく RSA-SHA256 を広告するからである。

### Lifecycle

鍵は、使用するテナント、用途、スコープが確定した時点で初めて作成する。デフォルトテナントにだけ事前に初期鍵を作ることはしない。これにより、すべてのテナントを同じ手順で作成でき、リクエスト単位のライフサイクルでは説明できない特別な状態を避けられる。

テナントの有効な署名鍵は少なくとも 90 日ごとにローテーションする。これは、手動で即時実行する `RotateTenantSigningKey` とは別に、定期実行する運用ジョブが行う。ローテーションでは、古い有効鍵を同じ処理内で検証用に降格させ、少なくとも 7 日間の重複期間を設ける。これにより、JWKS の利用者と RP は、ローテーション直前に発行されたメッセージも検証できる。終端状態の `Archived` に達した鍵素材は、退役した鍵で署名された監査トークンを検証できるように 7 年間保持する。個別に完全削除するインターフェースは提供しない。

公開鍵と証明書の一覧には、有効な鍵と、期限切れでない検証用の鍵を含める。`Archived` へ移った鍵は一覧から外れ、以後は公開しない。

鍵プロバイダーに到達できない場合のフェイルクローズ動作は、`SigningKeys` 自身では強制しない。この Context は署名や発行を行うインターフェースを持たないためである。ここでは、`TenantSigningKey.provider_healthy` と `ListTenantKeyHealth` を通じて、観測可能な `provider_healthy` シグナルだけを提供する。実際にフェイルクローズを強制するのは OAuth2 の `Token` 発行インターフェースであり、プロバイダーに到達できない場合はそこで新しい署名を停止する。

### Design Decisions

- 署名鍵は、差し替え可能な `KeyProvider` の背後でテナントごとに分離する。システム全体で鍵を共有したり、各プロトコルのアダプターにプロバイダーを埋め込んだりしない。
- 鍵のローテーション間隔（最短 90 日）、公開の重複期間（最短 7 日）、保管期間（7 年）は固定の規範的なポリシー値であり、文書化されない設定にはせず本設計に記録する。

## Scenarios

### REQ-SIGNINGKEYS-001: 署名鍵をローテーションしても以前の kid は JWKS に残る
- ACTOR TenantAdministrator
- GIVEN `admin` ロールを持つ "operator" が認証済みである
- GIVEN 現在の署名鍵は `kid` "kid-old" を持つ
- WHEN "operator" が管理画面で現在の署名鍵をローテーションする
- THEN ローテーションによって `kid` "kid-new" が新しい有効鍵になる
- WHEN クライアントが JWKS を取得する
- THEN レスポンスに `kid` "kid-old" と "kid-new" の両方が含まれる

### REQ-SIGNINGKEYS-002: 猶予期間終了後の署名鍵は JWKS から除去してアーカイブする
- ACTOR SystemAdministrator
- GIVEN `kid` "kid-old" の `Verifying` 鍵は `expires_at` を経過している
- WHEN スケジューラーがアーカイブ処理を実行する
- WHEN クライアントが JWKS を取得する
- THEN レスポンスに `kid` "kid-old" は含まれない
- THEN SigningKeyArchived イベントに `kid`、`retiredAt`、`expiresAt`、`disposedAt` が記録される

### REQ-SIGNINGKEYS-003: ライフサイクル設定が不正なバッチは起動しない
- ACTOR SystemAdministrator
- GIVEN `grace_days` が `cadence_days` 以上である
- WHEN `system_admin` が `idmagic-batch signing-key-lifecycle` を起動する
- THEN 設定エラーで終了し、鍵を回転しない

### REQ-SIGNINGKEYS-004: テナントごとの JWKS は互いに分離される
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" とテナント "tenant-b" がそれぞれ署名鍵を持つ
- WHEN テナント "tenant-a" の管理者が署名鍵を回転する
- WHEN クライアントがテナント "tenant-a" の JWKS を取得する
- THEN レスポンスにはテナント "tenant-a" の `kid` だけが含まれ、テナント "tenant-b" の `kid` は含まれない

### REQ-SIGNINGKEYS-005: XML フェデレーション署名資格情報はテナントと用途で分離される
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" とテナント "tenant-b" が存在する
- GIVEN 両テナントが JWT Signing 鍵と XmlFederationSigning 鍵を持つ
- WHEN テナント "tenant-a" が SAML Assertion を発行する
- THEN Assertion はテナント "tenant-a" の有効な `XmlFederationSigning` 鍵で署名される
- THEN テナント "tenant-b" の証明書でも、テナント "tenant-a" の JWT Signing 公開鍵でも署名を検証できない

### REQ-SIGNINGKEYS-006: XML フェデレーション鍵のローテーション中も既存の信頼関係を検証できる
- ACTOR TenantAdministrator
- GIVEN `XmlFederationSigning` の現在の鍵 K1 がメタデータに掲載されている
- WHEN 管理者が XmlFederationSigning 鍵を K2 へ回転する
- THEN 新しい XML メッセージは K2 で署名される
- THEN 猶予期間中の SAML / WS-Fed メタデータには K1 と K2 の証明書が掲載される
- THEN 猶予期間終了後は K1 がメタデータから除去される

### REQ-SIGNINGKEYS-007: XML フェデレーション署名資格情報は再起動後も同一である
- ACTOR SystemAdministrator
- GIVEN PostgreSQL または Vault プロバイダーでテナントの `XmlFederationSigning` 鍵が作成済みである
- WHEN API プロセスを再起動する
- WHEN クライアントが同じテナントのメタデータを取得する
- THEN 有効な証明書のフィンガープリントは再起動前と一致する

### REQ-SIGNINGKEYS-008: KeyProvider の障害時は健全性を観測でき、JWKS は取得可能な範囲で返る
- ACTOR SystemAdministrator
- GIVEN テナント "tenant-a" の KeyProvider が到達不能である
- WHEN `system_admin` が署名鍵の健全性一覧を取得する
- THEN テナント "tenant-a" の `provider_healthy` は `false` として返る
- THEN テナント "tenant-a" の JWKS は取得可能な範囲でキャッシュされた鍵を返す

### REQ-SIGNINGKEYS-009: 通常のテナント管理者はシステムコンソールの署名鍵ヘルスにアクセスできない
- ACTOR TenantAdministrator
- GIVEN "operator" は `admin` ロールだけを持ち、`system_admin` ロールを持たない
- WHEN "operator" が署名鍵ヘルス一覧を呼び出す
- THEN AccessDeniedError で拒否される

### REQ-SIGNINGKEYS-010: 管理者は回転後の検証用鍵だけを即時無効化できる
- ACTOR TenantAdministrator
- GIVEN 現在の署名鍵 K2 と、回転後に JWKS へ残る検証用鍵 K1 がある
- WHEN 管理者が K1 を無効化する
  - ALT 管理者が現在の署名鍵 K2 を無効化しようとする → エラー "InvalidRequestError"
- THEN K1 は JWKS から除去される
- THEN K2 は現在の署名鍵のまま残る
