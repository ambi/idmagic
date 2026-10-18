---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で Application → OAuth2 を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# OIDC アプリケーションの管理を Application のポートで要求する

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`Application → OAuth2` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

OIDC と service の作成、表示、設定更新、クレーム、シークレットの発行と失効、Application との関連付け、削除を対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

クライアント設定の公開契約、作成失敗の補償保証、シークレットの上限と有効期間の変更。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[カタログ](../../docs/modules/application/features/catalog.md)の REQ-APPLICATION-001、002、005、006、007 と、`backend/application/internal/handlers_http/application_provisioning.go`、`client_secret_lifecycle.go`、`backend/application/client_display_names.go` を起点にする。

D2、D4、D6、D7 を適用する。
Application はカタログの編成と管理の要求を担当し、保存型を含まない管理操作と表示値のポートを定義する。
OAuth2 の公開するクライアント管理操作へ組み立て地点で変換し、クライアントの不変条件と更新は OAuth2 に残す。
OAuth2 → Application の割り当てとポリシー評価は残す。

Repository を写す案は Client の保存型と検証を漏らし、管理 API 全体を OAuth2 へ移す案はカタログの編成を分散させるため採らない。
クライアントシークレットの表現の変更は OAuth2 と表示への変換で閉じ、Application が秘密の保存形式を知らない境界を選ぶ。

現行のクライアント作成、Application 作成、関連付けの順と各失敗時の状態を照合する。
すべての失敗に補償があるとは仮定しない。
秘密を一度だけ返す経路とクォータ、イベントの回数も保つ。

### 着手時に確定した境界

本番の逆向きの import は Application のルート2ファイルと HTTP の4ファイルにある。
管理操作は既存の OAuth2 の `ClientAdministration` を使う部分と、Repository を直接読む表示、関連付け、クレームの保存、削除に分かれる。
Application の `ports.OIDCClientAdministration` は、作成、設定更新、設定表示、関連付け、クレーム更新、削除、シークレットの発行、互換のローテーション、個別の失効を要求する。
入力と結果は Application の管理用の値とし、OAuth2 の保存型、ハッシュ、登録用の秘密は含めない。
作成の結果にはクライアント ID と一度だけ返す平文だけを含め、表示と資格情報のメタデータには平文を含めない。
表示名は `ClientNames.FindClientName(ctx, tenantID, clientID) (*string, error)` で取得し、空白、未取得と読み取りエラーのときのカタログ名へのフォールバックは Application が維持する。

D2、D4、D6、D7 を適用し、具象の変換は組み立て地点配下の `backend/shared/http/server_http/applicationdeps` へ置く。
OAuth2 の公開する管理操作を再利用し、関連付け、クレーム更新とカタログ経由の削除も所有側の公開操作へ切り出す。
具象の変換は保存値を読み取り専用に射影し、保存と不変条件の担当を OAuth2 から移さない。
Application は不透明なクライアント ID と要求する値だけを渡す。
各プロトコルの管理全体を移す案はカタログの編成を分散させ、Repository のコピーは保存表現と検証を Application へ漏らすため採らない。
OAuth2 の型を Application の型の別名として残す案も、依存が消えないため採らない。
表示名専用の照会は管理依存を要求せず、既存の一度だけの照会と重複排除を維持する。
シークレットの保存表現と認証方式を増やす変更は OAuth2 と境界の変換で閉じ、Application は管理と表示の契約だけを扱う。

作成はクライアントの保存と OAuth2 のイベント、Application の作成とイベント、クライアントへの関連付けの順を保つ。
後段の失敗を理由に前段を取り消す補償と、一括トランザクションは追加しない。
PostgreSQL の同一 Tx の protocol linker は変更せず、最後の関連付けは memory の射影への同期も担う既存の操作として維持する。
設定更新はクレームの公開検証、OAuth2 の通常設定の更新、その後にクレームの読み取りと保存、Application のクレーム更新イベントの順を保つ。
クレーム更新の入力に保存時刻を渡し、Application のイベントも同じ時刻を使う。
削除は Application の削除とイベントの後にクライアントを削除する。
OAuth2 の通常の削除操作はカタログ所有のクライアントを拒否するため、今回のカタログの削除の代わりには使わない。
表示は未取得と検索失敗を省略し、資格情報の一覧のエラーを既存どおり表示失敗へ昇格させない。
資格情報の状態は OAuth2 の `StatusAt` を呼んで射影し、状態の判定を複製しない。
有効数の上限だけを Application のエラーへ対応付け、422 の本文を維持する。

既存の表示名の2ファイルでは、入力と期待値を保った fixture の接続を新しい照会へ追従させる。
`TestCharacterizeClientDisplayNamesResolveEachClientOnce` の本文を変更せず、計数する fixture が新しい照会を通る形にする。
テストを変更しない構造変更の証拠の省略は適用せず、新しい境界の独立した失敗、変換と部分確定の観測、変更行のミューテーションを選ぶ。

管理ポートの具体的な操作は `Create(ctx, CreateOIDCClientInput) (*CreatedOIDCClient, error)`、`Update(ctx, UpdateOIDCClientInput) error`、`Configuration(ctx, tenantID, clientID, now) (*OIDCConfiguration, error)` とする。
所有側の更新は `BindApplication(ctx, tenantID, clientID, applicationID) error`、`UpdateClaims(ctx, tenantID, clientID, OIDCClaimUpdate) error`、`Delete(ctx, tenantID, clientID) error` で要求する。
シークレットの3操作は、操作者、クライアント ID、期限または資格情報 ID、現在時刻の入力と、平文を返す操作だけの結果型を持つ。
`OIDCConfiguration` は設定値、クレームと資格情報の非機密メタデータを持ち、`CreatedOIDCClient` は ID と一度だけ返す平文だけを持つ。

## 計画

着手時に既存の仕様、TypeSpec、名指した実装とテストを読み、操作列ごとの成功、拒否、途中失敗と再実行を照合する。
新しい型と操作のシグネチャ、変換の担当、必須依存と確定の境界をコードの前に記す。
この結果が上の保証維持と両立しない場合は、実装へ進む前に親項目の判断を見直す。
保証の追加や変更が必要になれば別の仕様変更として扱う。

## タスク

- [x] T001 [Design] 本番 import の全入口と作用を調査し、管理と表示の値、所有側の操作、部分確定と失敗時の順序を比較して設計へ記した。
- [x] T002 [App] 要求側のポートと所有側の公開操作を結び、既存の管理 API の96テスト、表示名の特性化、管理操作と射影の局所検査を通した。設定射影は未実装の RED から検証した。
- [x] T003 [Docs] 管理と表示の境界と既存の確定順を所有文書へ反映し、実測で消えた1件だけを負債台帳から除いて依存図を再生成した。
- [x] T004 [Verify] 既存の管理 API と特性化の期待値を保ち、正式な配線と逆依存の race 検査、射影のファズ探索、4パッケージの変更行の変異、値の脱落の手動故障、境界と仕様差分を検査した。

## 検証

- `mise run test-go-package -- ./backend/application/internal/handlers_http` の `client_secret_examples_test.go`、`client_secret_refusal_effects_test.go`、`protocol_config_examples_test.go` を起点に、OAuth2 の管理操作と実際の配線を `mise run test-go-changed` で確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

比較基準は `c90266e5f` とする。
最初に既存の Application の HTTP テストを実行し、新しい境界は設定の値、クレームの保存、関連付け、削除、シークレットと表示名の射影を独立して検査する。
変換が欠けた状態の RED を観測してから本体を実装する。
手書きの射影は任意の文字列と nil、空の列、任意の時刻について既存の値を保つ性質を検査し、フィールドの脱落の手動故障を検出する。
`mise run test-go-mutation -- backend/shared/http/server_http/applicationdeps 2 off .verification/wi-60634-mutation.json c90266e5f` で新しい変換の変更行を検査する。
所有側へ移す条件分岐は `mise run test-go-mutation -- backend/oauth2/internal/client/usecases 2 off .verification/wi-60634-owner-mutation.json c90266e5f` で検査する。
呼び出し側の表示名と HTTP の分岐も、`backend/application` と `backend/application/internal/handlers_http` の変更行を同じ基準でミューテーション検査し、移動した分岐の観測の不足と区別する。
`mise run test-go-changed -- c90266e5f` で正式な組み立てと逆依存を race 付きで実行し、HTTP の待受けを使うためサンドボックス外で実行する。
最終検査は `lint-go`、`check`、`render-docs`、`check-published-api-compat` とし、変更しないブラウザーの経路の検査は追加しない。

## リスク

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。

## 要約

規範仕様の変更はない。
Application が OIDC とサービスクライアントの管理操作、設定と資格情報の表示値、表示名の照会を要求するポートを定め、組み立て地点で OAuth2 の公開操作へ接続した。
保存型とシークレットのハッシュを Application へ渡さず、関連付け、クレーム更新とカタログ経由の削除も OAuth2 の公開操作へ移した。
クライアント作成、Application 作成と関連付け、通常設定とクレームの更新、削除の既存の確定順を維持し、補償と一括トランザクションを追加しなかった。
資格情報の状態は所有側の判定を使い、平文を一度だけ返す経路、クォータ、イベント、拒否応答と表示名のフォールバックを維持した。
既存の表示名のテストは fixture の接続だけを追従させ、入力と期待値、特性化テストの本文を保った。
Application から OAuth2 への本番 import を除き、消えた1件だけを台帳から削除して、循環負債を30件から29件へ減らした。
ほかの6組の逆向きの辺、クライアント設定の公開契約、作成失敗の補償、シークレットの上限と有効期間の変更は対象外とした。
