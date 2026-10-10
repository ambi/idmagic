# 署名鍵のライフサイクル

## 概要

この文書は、署名鍵のローテーション、期限切れの鍵のアーカイブ、検証用の鍵の無効化の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 管理者による即時のローテーション、定期実行のバッチによるローテーションとアーカイブ、検証用の鍵の即時の無効化 |
| 行為者 | テナント管理者（ローテーションと無効化）、System 管理者（バッチの起動）、スケジューラー（定期実行） |
| 扱わないもの | 鍵の分離の規則は[テナントと用途による鍵の分離](../separation/README.md)が、提供元の選択は[鍵の提供元と健全性](../provider/README.md)が扱う |

## モデル

鍵は、使うテナント、用途、スコープが確定した時点で作る。
デフォルトテナントだけに初期の鍵を事前に作ることはしない。

ローテーションは、古い有効な鍵を同じ処理の中で検証用に降格させ、重複期間を設ける。
JWKS の利用者と RP は、ローテーションの直前に発行されたメッセージも検証できる。

| 値 | デフォルト | 設定 |
| --- | --- | --- |
| ローテーションの間隔 | 90 日 | `idmagic-batch signing-key-lifecycle` の `-cadence-days` |
| 公開の重複期間 | 7 日 | `-grace-days`。間隔より短くなければならない |
| アーカイブした鍵素材の保管 | 少なくとも 7 年 | 設定はない。個別に完全に削除するインターフェースも、アーカイブした鍵を消す処理もない |

`Archived` に達した鍵素材は、退役した鍵で署名された監査のトークンを検証できるように保管する。
公開鍵と証明書の一覧には、有効な鍵と、期限の切れていない検証用の鍵を含め、`Archived` の鍵は含めない。

## 状態遷移

### SigningKeyLifecycle

署名鍵は `Active` からローテーションによって `Verifying` へ遷移し、公開の重複期間を過ぎると JWKS から外れて `Retired` になり、アーカイブの実行で `Archived` に移る。管理者が検証用の鍵を無効化すると、その鍵は直ちに `Archived` になる。遷移の表の `SigningKeyRetired` と無効化は遷移の契機の名前であり、ドメインイベントとしては発行しない。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 新しいメッセージへの署名に使う。テナント、用途、スコープの組ごとに 1 本だけ存在する |
| Verifying | — | 署名には使わないが、JWKS とフェデレーションメタデータに載せて検証に使う |
| Retired | — | 公開の重複期間（`expires_at`）を過ぎ、JWKS から除外した。公開しない |
| Archived | terminal | 監査保管。退役した鍵で署名された監査トークンを検証できるよう鍵素材を保持する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | SigningKeyRotated | — | Verifying |  |
| Verifying | SigningKeyRetired | `expires_at` を過ぎた | Retired |  |
| Retired | SigningKeyArchived | — | Archived |  |
| Verifying | 管理者による無効化 | — | Archived |  |
| Retired | 管理者による無効化 | — | Archived |  |

| State | ローテーション | 検証用の鍵の無効化 | 重複期間の経過 | アーカイブの実行 |
|---|---|---|---|---|
| Active | → Verifying | 拒否：400 invalid_request | 何もしない | 何もしない |
| Verifying | 何もしない | → Archived | → Retired | 何もしない |
| Retired | 何もしない | → Archived | 何もしない | → Archived |
| Archived | 何もしない | 何もしない | 何もしない | 何もしない |

## 操作

### 管理者による署名鍵のローテーション

#### REQ-SIGNINGKEYS-001 署名鍵をローテーションしても以前の kid は JWKS に残る

- テナント、用途、スコープの組に有効な鍵がない間、その組の鍵で初めて署名するとき、SigningKeys は、その組の `Active` の鍵を作る。
- 管理者が署名鍵をローテーションしたとき、SigningKeys は、指定した用途（省略時は `Signing`、または `XmlFederationSigning`）の新しい鍵を `Active` にし、それまでの `Active` の鍵を同じ処理の中で `Verifying` にして `expires_at` を重複期間（7 日）の後にし、200 と新旧の鍵を返し、`SigningKeyRotated` を発行する。
- クライアントが JWKS を取得したとき、SigningKeys は、`Active` の鍵と、`expires_at` を過ぎていない `Verifying` の鍵の公開鍵を返し、`Retired` と `Archived` の鍵を返さない。
- 管理者が署名鍵を一覧または取得したとき、SigningKeys は、`kid`、状態、有効期間、証明書、フィンガープリントを返し、秘密鍵の素材を返さない。
- `Signing` と `XmlFederationSigning` 以外の用途のローテーションを要求された場合、SigningKeys は、400 と `invalid_request` で拒否し、鍵を変えない。
- 存在しない `kid` の取得を要求された場合、SigningKeys は、404 と `key_not_found` で拒否する。
- 鍵の保管先を構成していない場合、SigningKeys は、ローテーションと無効化を 503 と `key_store_unavailable` で拒否する。
- **例**：EX-SIGNINGKEYS-001-01

#### REQ-SIGNINGKEYS-011 署名鍵の回転と無効化は自テナントの管理者だけが要求できる

- `admin` も `system_admin` も持たない利用者が署名鍵の一覧、取得、ローテーション、無効化を要求した場合、SigningKeys は、403 と `access_denied` で拒否し、鍵を変えず、`SigningKeyRotated` を発行しない。
- `signing-keys:read` のスコープの API アクセストークンで呼び出されたとき、SigningKeys は、署名鍵の一覧と取得だけを許可する。
- `signing-keys:write` を持たないトークンでローテーションまたは無効化を要求された場合、SigningKeys は、403 と `insufficient_scope` で拒否し、`WWW-Authenticate` に `signing-keys:write` を示し、鍵を変えない。
- **例**：EX-SIGNINGKEYS-011-01、EX-SIGNINGKEYS-011-02

### スケジューラーによる期限切れの鍵のアーカイブ

#### REQ-SIGNINGKEYS-002 猶予期間終了後の署名鍵は JWKS から除去してアーカイブする

- スケジューラーがアーカイブを実行したとき、SigningKeys は、`expires_at` が実行の時刻以前でまだアーカイブしていない鍵を `Archived` にし、鍵ごとに `kid`、`retiredAt`、`expiresAt`、`disposedAt` を載せた `SigningKeyArchived` を発行する。
- SigningKeys は、`Archived` の鍵素材を消さず、個別に完全に削除する経路を持たない。
- **例**：EX-SIGNINGKEYS-002-01

### System 管理者によるライフサイクルのバッチの起動

#### REQ-SIGNINGKEYS-003 ライフサイクル設定が不正なバッチは起動しない

- `system_admin` が `idmagic-batch signing-key-lifecycle` を起動したとき、SigningKeys は、`Active` の鍵が作成から `-cadence-days`（デフォルト 90 日）を過ぎた組だけをローテーションし、`-grace-days`（デフォルト 7 日）を重複期間として使う。
- `-grace-days` が `-cadence-days` 以上か、`-cadence-days` が 0 以下の場合、SigningKeys は、設定エラーで終了し、鍵をローテーションしない。
- **例**：EX-SIGNINGKEYS-003-01

### 管理者による検証用の鍵の無効化

#### REQ-SIGNINGKEYS-010 管理者は回転後の検証用鍵だけを即時無効化できる

- 管理者が `Active` でない鍵を無効化したとき、SigningKeys は、その鍵を直ちに JWKS とフェデレーションメタデータから除き、200 と鍵を返し、`Active` の鍵を変えない。
- 管理者が `Active` の鍵の無効化を要求した場合、SigningKeys は、400 と `invalid_request` で拒否し、鍵を変えない。
- 存在しない `kid` の無効化を要求された場合、SigningKeys は、404 と `key_not_found` で拒否する。
- **例**：EX-SIGNINGKEYS-010-01、EX-SIGNINGKEYS-010-02

## セキュリティ上の考慮

鍵の一覧と参照は `AdminKeysRead`（`admin:keys_read`）、ローテーションは `TenantKeysRotate`（`admin:keys_rotate`）、検証用の鍵の無効化は `TenantKeysDisable`（`admin:keys_disable`）を要する。
いずれも、`admin` または `system_admin` のロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行える。

管理 API のレスポンスに秘密鍵の素材を含めることは、どの権限でもできない。
返すのは `kid`、状態、有効期間、証明書、フィンガープリントに限る。
