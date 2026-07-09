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

1. **所有と語彙。** ApplicationCatalog が tenant 単位で `TenantDefaultSignInPolicy` を所有する。
   フィールドは既存の `SignInRule` (`RequiredAuthnLevel` / `AccessCondition`) をそのまま使い、
   語彙・条件モデルは再設計しない (wi-115 Out of Scope)。デフォルトポリシーは Application の
   sign-in policy に関する概念なので、tenancy context ではなく ADR-079 と同じ
   ApplicationCatalog が所有する。DB テーブルは `tenant_default_sign_in_policies`、
   監査イベントは `TenantDefaultSignInPolicyUpdated`。

2. **関係 = 上書き (override)。** アプリが独自の sign-in policy (有効ルール) を持てば、それが
   テナントデフォルトを**完全に置換**して評価される。持たなければデフォルトをそのまま適用する。
   合成・連結はしない。実効ルールは `EffectiveSignInRules(default, app)` が
   「アプリに有効ルールがあれば app.rules、なければ default.rules」を返し、
   既存の `EvaluateSignInPolicy` に渡す。どちらの経路でも評価器は fail-closed のまま。
   「合成後の二重表示」が無くなり、管理者は 1 つの実効ポリシーだけを見ればよい。

3. **デフォルトより弱い上書きは許可し、警告する。** 上書きモデルではアプリ個別ポリシーは
   デフォルトを下回れる。強制はせず、デフォルトより弱いとき (認証強度の引き下げ・再認証を求めるまでの時間の
   延長や撤廃・許可ネットワークの緩和のいずれか) に `AppSignInPolicyResponse.weaker_than_default` を
   true にして UI で警告する。判定は `AppPolicyWeakerThanDefault(default, app)` に集約する。
   floor を強制しないのは、低リスクアプリの明示的な緩和を単純な操作 (アプリ側で上書き) で
   行えるようにするため。ADR-079 の「アプリ個別が最終決定権を持つ」原則とも整合する。

4. **既存テナントの初期値は空 (ルール無し)。** 空のデフォルトポリシーは allow-all で、導入時点では
   挙動を変えない。管理者が明示的にルールを設定して初めて baseline が有効化される。移行時に
   MFA 等の安全側デフォルトを一括適用すると大規模ログイン不能・認証強度の予期せぬ変化を招く
   ため採らない。

5. **評価点は既存の Application gate と同じ federation 開始経路。** OIDC authorize、WS-Fed sign-in、
   SAML SSO は同じ gate を通し、gate が実効ポリシー (アプリ独自があればそれ、なければデフォルト) を
   組み立てて評価器に渡す。クライアント IP も従来どおり全経路で渡す。protocol ごとの個別分岐で
   迂回できない。

6. **ロールバックはデータ操作で即時・可逆。** デフォルトポリシーはテーブル行であり、ルールを空に
   するか行を削除すれば allow-all に戻る。スキーマ破壊的変更を伴わず、`TenantDefaultSignInPolicyUpdated`
   で各変更 (クリアを含む) を監査追跡できる。個別アプリの緊急緩和はアプリ側ポリシーの上書きで即時に行える。

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
