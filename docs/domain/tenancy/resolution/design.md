# テナントの解決の設計

この文書は、[テナントの解決](README.md)の要件を保証する仕組みを扱う。

## アーキテクチャ

解決は Host ヘッダーとパスだけを見て、次の順に進む。

1. `tenant_base_domain` が設定され、Host が `{label}.{tenant_base_domain}` に一致するなら、ラベルを realm として対応付ける。見つかったテナントの `endpoint_style` が `Subdomain` でなければ不在として扱う。
2. パスが `/realms/{realm}/...` に一致するなら realm を対応付ける。見つかったテナントの `endpoint_style` が `Path` でなければ不在として扱う。
3. どちらにも一致しないリクエストは、テナントが存在しないものとして扱う。任意の Host や接頭辞のないパスをデフォルトテナントへ退避させない。

| 構成要素 | 責務 |
| --- | --- |
| `backend/shared/http/support_http` のテナント解決ミドルウェア | パスの `/realms/{realm}` の区間または Host から realm を取り出し、`TenantRepository.FindByRealm` で `Tenant` を引き、`endpoint_style` と到達経路を照合する |
| `Deps.CanonicalLocation` | テナントの正規ロケーションから、発行者と URL の接頭辞を組み立てる |
| リクエストコンテキスト | 解決した `Tenant`、発行者、URL の接頭辞を運ぶ。発行者、URL の接頭辞、Cookie のスコープ、WebAuthn の RP ID は、いずれもここから組み立てる |

プロトコルと管理のルートは、テナントをまたぐ制御面のテナント管理（`/api/admin/v1/tenants/...`）も含めて、すべて解決済みのテナントの下に置く。
制御面の操作を制御面テナントからの呼び出しに限るのは、ルーティングではなく `RequireControlPlaneUser` の判定である。
制御面の操作は `/realms/default/...` から呼ばれるので、デフォルトテナントのセッション Cookie のパスだけで対象を覆え、Cookie のスコープをルートパスまで広げずに済む。

## インフラストラクチャ

`Subdomain` を選べるのは、デプロイ時に基底ドメイン `tenant_base_domain` を設定した場合だけである。
設定しないデプロイ先は `Path` のままで、ワイルドカード DNS もテナントごとの証明書も要らない。
