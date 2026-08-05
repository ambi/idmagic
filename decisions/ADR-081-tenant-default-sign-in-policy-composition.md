---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-081: テナントデフォルトサインインポリシーと上書きモデル

## コンテキスト

ADR-079 で Application ごとの sign-in policy を導入したが、テナント全体の baseline 認証要件を一箇所で
設定する手段が無い。管理者は「全アプリで MFA をデフォルトにし、一部の低リスクアプリだけ緩和する」
「全アプリで一定時間を超えた再認証を要求する」といった横断的方針を持つ。デフォルトポリシーが無いと、
アプリ作成時の設定漏れがそのまま弱いログイン要件になり、運用上の安全性と説明可能性が下がる。

sign-in policy 評価器 (`EvaluateSignInPolicy`) は既に「順序付き `SignInRule` の連言」を fail-closed で
評価する。すべての enabled ルールを順に見て、最初に非 allow となったルールの判定 (deny /
step-up) を返し、全ルールを満たせば allow を返す。デフォルトポリシーはこの語彙と評価器を再利用できる。

デフォルトとアプリ個別の関係は当初「合成 (floor)」で設計したが、管理者にとって
「デフォルト＋アプリ個別の合成結果」は直感的に把握しづらく、実効ポリシーが二重に見えるなど UI が
分かりにくいという指摘を受けた。分かりやすさを優先し、関係を単純な**上書き**に改める。

## 決定

Application context の `models.TenantDefaultSignInPolicy`、
`interfaces.GetTenantDefaultSignInPolicy` / `interfaces.UpdateTenantDefaultSignInPolicy`、
`events.TenantDefaultSignInPolicyUpdated`、`invariants.DefaultPolicyAppliesWhenAppUnset`、
更新した `invariants.AppPolicyEvaluatedAcrossProtocols`、`AppSignInPolicyResponse.weaker_than_default` に反映。
wi-115 で導入。ADR-079 を前提とする。

ApplicationCatalog が tenant 単位で `TenantDefaultSignInPolicy` を所有する (tenancy context では
ない — デフォルトポリシーは Application の sign-in policy に関する概念であり、ADR-079 が定めた
所有を分散させない)。デフォルトとアプリ個別ポリシーの関係は**上書き**とし、アプリが独自の有効
ルールを持てばテナントデフォルトを完全に置換し、持たなければデフォルトをそのまま適用する。当初
案は「合成 (floor)」でアプリ個別がデフォルトを下回れない設計だったが、実効ポリシーが二重に見え
管理者が把握しづらく、低リスクアプリの緩和に別途「例外フラグ」が必要でモデルが複雑になるため、
分かりやすさを優先して上書きへ改めた。上書きモデルではアプリ個別ポリシーがデフォルトより弱く
なりうるため、`weaker_than_default` フラグで UI 側に警告するに留め、強制はしない — 低リスク
アプリの明示的な緩和を単純な操作で行えるようにするため、また ADR-079 の「アプリ個別が最終決定
権を持つ」原則とも整合するため。既存テナントの初期値は空 (allow-all) とし、移行時に安全側の
既定 (MFA 必須等) を一括適用する案は大規模ログイン不能を招くため却下した。

実効ルールの合成規則、評価点の配置、ロールバック手順の詳細は
[`backend/application/ARCHITECTURE.md`](../backend/application/ARCHITECTURE.md) に置く。

## 却下した代替案

- **合成 (floor) で下限を強制する。** デフォルト＋アプリ個別を連結し、アプリ個別ではデフォルトを
  下回れない設計。当初案だが、実効ポリシーが二重に見え管理者が把握しづらい。低リスクアプリの緩和に
  別途「例外フラグ」が必要で、モデルが複雑になる。分かりやすさを優先して不採用。
- **例外フラグ (`exempt_from_tenant_default`)。** floor 前提で「このアプリだけデフォルトを外す」
  ためのフラグ。上書きモデルでは上書き自体が緩和手段になるため不要。導入していたフラグは削除する。
- **ルールを意味的にマージ / 重複排除する。** 優先順位が曖昧で実装が複雑。上書きは挙動が予測可能。
- **デフォルトを tenancy context / Tenant 集約に置く。** デフォルトは Application のサインインに関する概念で、
  ADR-079 が sign-in policy の所有を ApplicationCatalog と定めている。所有を分散させない。
- **移行時に安全側の非空デフォルト (例: MFA 必須) を全テナントに適用する。** 大規模ログイン
  不能・認証強度の予期せぬ変化を招く。安全な段階導入に反する。

## 影響

- `tenant_default_sign_in_policies(tenant_id, rules JSONB, updated_at)` テーブルを追加する。
- 管理 API と UI に tenant デフォルト sign-in policy の編集面 (`/api/admin/default-sign-in-policy`) を持つ。
- `GetAppSignInPolicy` 応答をアプリ個別ポリシー・テナントデフォルト・上書き後の effective ルール列・
  `weaker_than_default` 警告フラグを区別して返すよう拡張し、アプリ詳細で 3 種を表示できるようにする。
- federation 開始時の Application gate は、アプリが独自ポリシーを持てばそれを、持たなければデフォルトを
  実効ポリシーとして評価する。
- UI のサインインポリシー画面はデフォルトを詳細表示 → 編集モードで編集し、各アプリの上書き有無・警告・
  実効ポリシーを一覧する。
