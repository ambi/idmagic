---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で OAuth2 → ApiTokens を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# 組み込みリソースの公開情報と失効の識別情報を OAuth2 へ渡す

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`OAuth2 → ApiTokens` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

組み込みリソースの公開スコープと、失効要求の組み込み client_id の例外を、OAuth2 の正式な構築入力へ置き換える。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

スコープ語彙の共有への移設、外部設定による認証例外の追加、署名と検証の担当変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

`backend/oauth2/internal/token/usecases/protected_resource_metadata.go` と `internal/handlers_http/token_handler.go:handleRevoke`、[API トークン](../../docs/modules/api-tokens/README.md)、[OAuth2](../../docs/modules/oauth2/README.md)を起点にする。

D1、D4、D5、D6 を適用する。
業務のスコープ語彙と組み込みクライアント識別子は ApiTokens が引き続き担当する。
OAuth2 が必要とするリソース情報と識別情報の入力を定義し、正式な組み立て地点から渡す。
ApiTokens → OAuth2 の署名と検証の依存を残す。

全スコープを shared に移す案は D5 に反し、同じ識別子を OAuth2 にハードコードする案は管理 API の知識を複製する。
組み込みリソースの設定を一般の外部入力で上書きできる案も採らない。
管理スコープを追加する変更は ApiTokens の宣言と配線から導き、OAuth2 のスコープ一覧を手で更新しない。

空の resource で返すメタデータと、登録済みリソースの照会を分ける。
失効の例外は既存の client_id 一致、Authorization ヘッダーなし、client_secret なしの条件に限定し、一般のクライアント認証を弱めない。
必須の構築入力を API、worker、batch、テストの各組み立てで照合する。

### 境界の判断の記録

入力の参照：[依存の向きを選ぶ基準](../../docs/design/application/backend.md#依存の向きを選ぶ基準)の「プロトコルから必要になる管理発行トークンの情報」の行、[依存の向きと差し替え点](../../docs/design/application/design-guidelines.md#依存の向きと差し替え点)。

着手時の再調査では、OAuth2 から ApiTokens への本番 import は次の 2 箇所だけである。
`_test.go` の import は循環の集計に含まれない。

| 箇所 | 使う値 | 用途 |
| --- | --- | --- |
| `internal/token/usecases/protected_resource_metadata.go` | `apitokendomain.AllScopes()` | resource を省いた Protected Resource Metadata の `scopes_supported` |
| `internal/handlers_http/token_handler.go:handleRevoke` | `apitokendomain.BuiltinClientID` | クライアント認証なしで `/revoke` を受け付ける組み込みの公開クライアントの判定 |

OAuth2 の経路を組み立てる地点は `backend/shared/http/server_http/routes.go` の `oauth2.RouteDeps` だけである。
worker と batch は OAuth2 の経路を登録しない。
HTTP のテストも `server_http.Register` を通るため、正式な組み立てと同じ値を受け取る。

| 候補 | 制約への適合 | 変更シナリオの波及 | 採否 |
| --- | --- | --- | --- |
| A. OAuth2 が値の型を定義し、組み立て地点が ApiTokens の値から作って `RouteDeps` で渡す | 状態の所有者を変えず、逆向きの import を除く（D4、D6）。作用を持たない固定値なのでポートは要らない | 管理スコープの追加は ApiTokens の宣言だけで届く | 採用 |
| B. OAuth2 が値を返すインターフェースを定義し、ApiTokens の Service が実装する | 実装が一つしかなく、差し替えの必要がない差し替え点を足す | A と同じだが、呼び出し側の学ぶ量が増える | 不採用 |
| C. スコープと識別子を shared へ移す | D5 に反し、業務の語彙を共有ライブラリへ置く | 管理 API の知識が二箇所から読める | 不採用 |
| D. 識別子とスコープを OAuth2 に複製する | 管理 API の知識を複製する | スコープの追加で二箇所の更新が要る | 不採用 |

型と操作：

```go
// package tokenusecases (backend/oauth2/internal/token/usecases)
type BuiltinResource struct {
    Scopes             []string // resource を省いたメタデータの scopes_supported
    RevocationClientID string   // クライアント認証なしで /revoke を呼べる組み込みの公開クライアント
}

func BuildProtectedResourceMetadata(
    ctx context.Context, repo ports.McpResourceServerRepository, builtin BuiltinResource,
    tenantID, resource, issuer string,
) (*ProtectedResourceMetadata, error)
```

`oauth2.RouteDeps` と `oauth2http.Deps` に `BuiltinResource tokenusecases.BuiltinResource` を加え、組み立て地点は `apitokendomain.AllScopes()` と `apitokendomain.BuiltinClientID` から値を作る。
失効の例外は `RevocationClientID` が空でないことを条件に加える。
未配線のまま空の client_id と一致して認証を省く経路を作らないためである。
登録済みリソースの照会、確定順、イベントは変えない。

残る仮定：組み込みリソースの値は起動中に変わらない。

## 計画

着手時に既存の仕様、TypeSpec、名指した実装とテストを読み、操作列ごとの成功、拒否、途中失敗と再実行を照合する。
新しい型と操作のシグネチャ、変換の担当、必須依存と確定の境界をコードの前に記す。
この結果が上の保証維持と両立しない場合は、実装へ進む前に親項目の判断を見直す。
保証の追加や変更が必要になれば別の仕様変更として扱う。

## タスク

- [x] T001 [Design] import の全入口と作用を再調査し、境界の型と操作、変換、確定順を設計へ記す。
- [x] T002 [App] 要求側のポートと所有側の公開操作を結び、逆向きの辺を解消する。
- [x] T003 [Docs] 公開範囲と現在の設計を同期し、実測で消えた負債だけを削除して依存図を再生成する。
- [x] T004 [Verify] 既存の保証、正式な配線と境界を検査し、仕様差分を確認する。

## 検証

- `mise run test-go-package -- ./backend/oauth2/internal/token/usecases` の `protected_resource_metadata_test.go` と、HTTP の組み込み client の失効と通常 client の認証検査を起点にする。
  正式な組み立てから公開スコープが届くことを `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。

## 要約

規範仕様の差分はない。
OAuth2 から ApiTokens への本番 import 2 箇所を除き、組み込みリソースの公開スコープと失効用の組み込みクライアントの ID を、OAuth2 が定義する値 `BuiltinResource` として組み立て地点から渡すようにした。
固定値だけを渡すためポートは設けず、resource を省いたメタデータの作成は登録済みリソースの照会から分けた。
失効の例外は値が空なら当てず、未配線のまま空の client_id で認証を省く経路を作らない。

`OAuth2 → ApiTokens` と、同じ循環にあった `ApiTokens → OAuth2` の 2 件が実測で消え、台帳は 16 件から 14 件になった。
依存図を生成し直し、バックエンド設計の実装状況を実装済みへ更新した。

組み込みクライアントの ID に資格情報を添えた要求と client_id を欠く要求が一般のクライアント認証で拒否されることを、本番コードの変更前にテストで示した。
組み立て地点の値の切断を手で注入し、失効とメタデータの既存テストが検出することを確かめた。
