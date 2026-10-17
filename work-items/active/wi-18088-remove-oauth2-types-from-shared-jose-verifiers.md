---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-18
priority: p2
depends_on: [wi-91505-move-the-jwt-token-signer-into-oauth2]
change_kind: refactor
spec_impact: { kind: none, reason: "共有の JWKS 取得とリプレイ検証から OAuth2 の型への依存を外す。取得元の優先順位、SSRF 拒否、署名検証、リプレイ拒否、HTTP 応答、永続状態とイベントは変えない。" }
---

# 共有の JOSE 検証器が OAuth2 の型に依存しないようにする

## 動機

`tools/check/boundary-debt.json` の `shared-dependency:shared->OAuth2` は、JWTSigner の移動後も残る。
`jwks_resolver.go` が OAuth2Client を、Client Assertion と DPoP の検証器が OAuth2 のリプレイ保存ポートを参照しているためである。

## 対象範囲

- JWKS の取得器は取得元の値を受け取り、OAuth2Client からの変換を利用側へ移す。
- 共有の検証器が必要とする最小のリプレイ保存操作を受け取り、OAuth2 の実装を構造的に適合させる。
- 対応する共有依存を台帳から除く。

## 対象外

- JWKS の優先順位、キャッシュ、SSRF 対策、リプレイ保存の寿命と検証規則の変更。

## 設計

既存の `ResolveJWKSSource` と OAuth2Client 用 `Resolve` の空の取得元に対するエラーの差を確認し、既存の結果を維持する。
D5、D6 に従い、OAuth2Client の解釈は利用側に置き、共有側は JWKS の技術的な取得だけを担う。

### 境界の比較

入力は `JWKResolver.Resolve`、その呼び出し元のクライアント認証、二つのリプレイ保存ポートである。
どちらの保存ポートも `RecordIfNew(ctx context.Context, jti string, windowSeconds int, now time.Time) (bool, error)` だけを持つ。
共有検証器は保存の所有者を知らず、受け取った操作でリプレイを拒否する。
状態の所有者、トランザクション、保存期間は変えない。

| 案 | D5、D6 と変更シナリオ | 判断 |
| --- | --- | --- |
| OAuth2Client と OAuth2 のポートを共有側で使い続ける | クライアントモデルの再配置が共有取得器へ波及し、D5 に反する | 不採用 |
| 検証器全体を OAuth2 へ移す | 他のプロトコルも使う鍵取得と署名検証を OAuth2 に結び付ける | 不採用 |
| JWKS の URI と文書だけの入力、および共有側の ReplayStore を定義する | OAuth2Client からの変換はクライアント認証側で行う。既存の保存実装は構造的に適合し、アダプターや保存規則を増やさない | 採用 |

`Resolve(ctx context.Context, source JWKSource) ([]map[string]any, error)` の `JWKSource` は `JWKS map[string]any` と `JwksURI *string` だけを持つ。
外部取得の優先順位と、空の取得元に対する既存のエラー文をそのまま保つ。
`ResolveJWKSSource` は既存の別のエラー文を保つため、この項目では統合しない。

## 計画

取得器とリプレイポートの呼び出し元を確認し、既存テストを保って依存を外す。

## タスク

- [x] T001 [Design] 入力とリプレイポートを比較する。
- [x] T002 [App] 共有の検証器と呼び出し元を更新する。
- [x] T003 [Tooling] 解消した共有依存を台帳から除く。
- [x] T004 [Verify] 既存テストと境界を検証する。

## 検証

- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：共有依存を解消する。
- `mise run test-go-package -- ./backend/shared/security/tokens_jose`：取得と検証の結果を保つ。
- `mise run lint-go` と `mise run test-go-race`：共有コードから影響する全操作の呼び出し元の型と挙動、契約証拠を確認する。
- `mise run check`：台帳、仕様影響と参照の整合。
- `mise run test-go-mutation -- backend/shared/security/tokens_jose 2 off /tmp/idmagic-wi18088-mutation.json main`：変更行の検出能力。型の受け取り方だけの変更で変異が生成されない場合は既存テストと境界検査を証拠とする。

## リスク

空の JWKS と URI のフォールバックを変えると、クライアントの認証結果が変わる。
既存の優先順位とエラーを維持する。

## 要約

規範仕様の差分はない。
JWKS 取得器は URI と JWKS 文書だけを受け取り、OAuth2Client からの変換を OAuth2 のクライアント認証に置いた。
リプレイ検証器は共有側の最小の ReplayStore を使い、既存の保存先が構造的に適合する。
共有の JOSE パッケージからモジュール依存がなくなり、境界台帳の OAuth2 依存を除いた。
変更行は型とシグネチャだけなので自動 mutation は0件であり、検出能力の新しい証拠には数えない。
既存の取得、リプレイ拒否、HTTP 認証のテストを維持した。
