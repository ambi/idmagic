---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-18
priority: p3
depends_on: [wi-21934-decide-module-dependency-order-and-break-cycles]
change_kind: refactor
spec_impact: { kind: none, reason: "要求側のポートと境界の変換で Tenancy → IdManagement を切り離し、HTTP の応答、認証と認可の判定、永続状態、イベントと既存の確定順を保つ。" }
---

# テナント属性スキーマを IdManagement の保存型から切り離す

## 動機

[依存の向きの決定](../done/wi-21934-decide-module-dependency-order-and-break-cycles.md)で、`Tenancy → IdManagement` を除く向きを選んだ。
公開パッケージの利用でもモジュールの循環は残るため、この組を独立して解消する。

## 対象範囲

User と Group のスキーマの保存型、Repository、HTTP 表現、組み込み属性の照会、定義検証、動的 Group の参照保護を対象にする。
モジュールのルート、公開型、HTTP、アダプターを含め、この辺を生む本番 import をすべて対象にする。
実装後に循環を再計測し、消えた違反 ID だけを台帳から除き、依存の図を生成し直す。

## 対象外

属性定義の形式の変更、スキーマ更新の原子性の追加、テーブルの移管。
ほかの組の循環の解消は、この項目へ混ぜない。

## 設計

[属性スキーマ](../../docs/modules/tenancy/features/attribute-schema.md)の REQ-TENANCY-002、020、043 と、`backend/tenancy/internal/usecases/manage_user_attribute_schema.go`、`manage_group_attribute_schema.go`、`internal/handlers_http/admin_user_attribute_schema_handler.go` を起点にする。

D1、D2、D4、D6 を適用する。
Tenancy はスキーマの保存と管理 API を維持し、要求側の表現とポートを定義する。
IdManagement は属性定義の形式、組み込み属性と動的 Group の参照情報を引き続き定め、組み立て地点の変換で Tenancy のポートへ接続する。
定義検証と参照保護を通してから保存し、拒否時は保存とイベント発行を行わない既存の保証を保つ。
IdManagement → Tenancy のスキーマ照会は残し、表の所有者と保存形式は変えない。

全体を IdManagement へ移す案は保存と API の担当を変え、属性定義を shared へ移す案は D5 に反するため採らない。
Tenancy の型に IdManagement の別名を残す案では循環が消えない。
属性型を増やす変更では IdManagement の検証と境界の変換だけを見直し、Tenancy に同じ検証を複製しない。

保存型の表現を選ぶ際は、定義の意味と保存の担当を混同しない。
User と Group のスキーマを統合しない。

### 着手時に確定した境界

`tenant_user_attribute_schemas` と `tenant_group_attribute_schemas` のテーブル所有者と保存アダプターは、データベース設計の宣言どおり IdManagement である。
親項目の「保存は Tenancy」は管理上の要求を指すものとして読み、SQL の担当や保存場所を移さない。
既存の IdManagement の保存型を使う Repository は、同じ公開パッケージ内の `user/ports` と `group/ports` へ移す。
IdManagement とプロトコルなどの既存の照会元は、同じ保存操作と値をそのポートから使う。

Tenancy は `domain.UserAttributeDef`、`GroupAttributeDef`、`TenantUserAttributeSchema`、`TenantGroupAttributeSchema` を管理用の値として定義する。
JSON のフィールドと省略条件を維持し、属性の型と公開範囲は文字列として受け取り、意味の検証を IdManagement に委ねる。
Tenancy の保存ポートは、既存の `FindByTenant(ctx, tenantID)`、`Save(ctx, schema)`、`Delete(ctx, tenantID)` に加え、`Validate(schema) error` と User の `Builtin() []UserAttributeDef` を要求する。
ユースケースの既存の取得、作成日時の維持、検証、保存という順序を変えず、検証だけをポートへ渡す。

変換は既存の組み立て地点配下の `backend/shared/http/server_http/tenancydeps` に置く。
この末端パッケージは IdManagement と Tenancy の公開型だけを結び、HTTP ルーターやモジュールのルートには依存しない。
ルーター本体へ置く案は Tenancy のユースケース検査から import 循環を生むため採らず、IdManagement の中へ置く案は利用側の変換の担当を提供側へ移すため採らない（D4、D6）。
二つの層の値の変換では全フィールド、nil、空の配列、任意の文字列、ポインターの値を保つ。
検証は変換後の IdManagement の `Validate` をそのまま呼び、形式、組み込みとの衝突、重複の規則を複製しない。

動的 Group は `DynamicGroupAttributeReferences(ctx, tenantID) ([][]string, error)` で属性キーだけを返す Tenancy の関数型ポートへ変換する。
未配線時の扱い、読み取りエラー、参照された属性の削除と型変更の 409、拒否後の状態とイベントの不変を保つ。
通知テンプレートの試し送りも UserRepository を使っているため、`FindTestNotificationActor(ctx, sub)` で表示名と検証済みメールだけを返す照会へ置き換える。
操作者が存在しない場合の拒否と、読み取りエラーの伝播を維持し、本人と異なる宛先を受け取らない。

正式な HTTP の組み立てと memory、PostgreSQL の起動時配線から、同じ IdManagement の保存先を変換して Tenancy へ渡す。
worker、削除、プロトコルの照会は IdManagement の元のポートを使い続ける。
スキーマ更新、参照検査、イベント発行の順序、別々の確定と既存の任意依存は変えない。
新しい必須の作用と規範仕様は追加しない。

### テストの追従についての利用者の指示

利用者は、今回に限り、既存テストの期待結果を維持した fixture と参照の追従を明示的に認めた。
入力値、表明、期待値、規範と操作の引用、skip を変えず、新しいポートへの接続だけを追従させる。
検査のために本番へ旧境界の互換経路を残さない。
既存テストの成功を比較基準とし、値の変換と正式な配線についても観測する。

## 計画

着手時に既存の仕様、TypeSpec、名指した実装とテストを読み、操作列ごとの成功、拒否、途中失敗と再実行を照合する。
新しい型と操作のシグネチャ、変換の担当、必須依存と確定の境界をコードの前に記す。
この結果が上の保証維持と両立しない場合は、実装へ進む前に親項目の判断を見直す。
保証の追加や変更が必要になれば別の仕様変更として扱う。

## タスク

- [x] T001 [Design] 保存先と型、定義の検証、参照保護、通知の宛先照会の担当と確定順を比較した。
- [x] T002 [App] 管理用の値と要求側のポートを組み立て地点で結び、Tenancy から IdManagement への本番 import を解消した。
- [x] T003 [Docs] 所有文書と依存図を同期し、消えた14件の違反だけを台帳から除いた。
- [x] T004 [Verify] 作業開始点からの逆依存の race 検査、正式な配線、性質、変更行の変異と手動故障、境界、文書と仕様影響を検査した。

## 検証

- `mise run report-contract-evidence -- --changed --base-revision main --target <変更予定ファイル>`：共有のルートの組み立てを含む影響する操作と参照不足を実装前に照合する。
- `mise run test-go-changed -- a0ed7efd6`：チェックポイントに入れた変更も含め、Tenancy、IdManagement、組み立てと既存の逆依存を race 付きで検査する。HTTP の待受けを含むためサンドボックス外で実行する。

- `mise run test-go-package -- ./backend/tenancy/internal/usecases` と該当 HTTP テストで、`TestUpdateUserAttributeSchemaRejectsBuiltinCollision`、重複キーの拒否、動的参照の 409 と状態不変を確認する。
- `mise run lint-go`：変更した Go の静的検査を行う。
- `mise run check-boundaries` と `mise run check-boundary-debt-ratchet`：辺の解消、公開範囲と負債の非増加を確認する。
- `mise run check-contract-drift`、`mise run check-status-drift` と `mise run check-published-api-compat`：HTTP の表現と応答を変えていないことを確認する。
- `mise run check-work-items`、`mise run check-links`、`mise run spec-diff`：記録と参照、規範への影響を確認する。

テストを変更しない構造変更は既存テストの成功を証拠にする。
保証を表すテストが欠けている場合は、公開状態に応じた特性化または要件のテストを実装より先に用意し、その段階の証拠と変更耐性の手順を選ぶ。

今回の比較基準は `a0ed7efd6` である。
既存テストの追従は利用者の指示に従い、新しい変換だけに `FuzzAttributeSchemaProjection` と照会のアダプター検査を加える。
`mise run test-go-fuzz -- ./backend/shared/http/server_http/tenancydeps FuzzAttributeSchemaProjection 10s` で任意の文字列、各フィールドと nil、空配列の往復を検査する。
新しいアダプターと、変更した条件がある Tenancy のユースケース、HTTP のアダプター、正式なルートの組み立てには、変更行を対象とする `test-go-mutation` を使う。
具体的な対象は `backend/shared/http/server_http/tenancydeps`、`backend/tenancy/internal/usecases`、`backend/tenancy/internal/handlers_http`、`backend/shared/http/server_http` とし、worker 2、cache off、第5引数を `a0ed7efd6` にする。
型参照と import だけを変えたパッケージには、変異させる式がないため、コンパイル、既存テストと境界検査を使う。
ツールが表現できない PII の値の脱落を手で注入し、性質の検査で検出する。
参照保護の正式な配線の切断を手で注入し、`TestUserAttributeSchemaRefusesChangingAReferencedAttribute`（EX-TENANCY-043-01、02）で検出する。
server パッケージ単体の変異検査で、User の保存先の上書きと Group の保存先の未配線に当たる2変異が生き残ったため、`TestTenantAttributeSchemaRoutesReadIdentityOwnedSchemas` を追加する。
Tenancy の既存検査とは別に、変異を加える組み立て地点のパッケージで、正式な GET の応答が同じ IdManagement の保存先から返ることを観測する。
対象は保存済みの1件の参照に絞り、スキーマのすべての成功と拒否のシナリオを複製しない。
文書は `check-spec`、`check-links`、`render-docs` と `check-published-api-compat` で、記録と境界は `check` で検査する。
ブラウザーのコードと HTTP 契約は変えず、正式なルーターを通す既存の HTTP 検査で配線を観測する。

## リスク

PII の値の変換と試し送りの宛先に触れるため、着手時のリスクを high に引き上げる。

import を消しただけで実行時の再入と必須の作用を失わないよう、変換前後の公開経路と各確定点を比較する。
必須の作用を省略できる未配線経路を増やさず、既存の構築条件とテナント境界を保つ。

## 要約

規範仕様の変更はない。
Tenancy の属性スキーマを管理用の値と要求側のポートにし、IdManagement の保存型と検証、組み込み属性、動的 Group の参照へ組み立て地点で接続した。
通知テンプレートの試し送りも、表示名と検証済みのメールだけを返す操作者照会に置き換えた。
保存先とテーブル所有者は IdManagement に維持し、User、Group、クレーム、CSV と worker の照会は IdManagement の保存ポートを使う。
Tenancy から IdManagement への本番 import を除き、消えた14件の負債だけを削除して、循環負債を45件から31件へ減らした。
利用者の指示に従って既存テストの入力と期待結果を保ち、fixture と参照を新しい境界へ追従させた。
ほかの8組の逆向きの辺と、属性の仕様、スキーマ更新の原子性、テーブルの移管は対象外とした。
