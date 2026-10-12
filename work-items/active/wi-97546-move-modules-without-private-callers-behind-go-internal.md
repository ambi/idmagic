---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-50428-move-one-module-behind-go-internal]
change_kind: refactor
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: パッケージの配置と組み立て操作を変更するだけで、製品の API、設定、永続状態、通知を変更しない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - docs/design/architecture/logical.md
    - docs/design/application/backend.md
    - docs/design/application/design-guidelines.md
    - tools/check/boundary-debt.json
    - backend/authorization/module.go
    - backend/cmd/internal/bootstrap/memory.go
    - backend/cmd/internal/bootstrap/postgres.go
    - backend/shared/http/server_http/routes.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/cmd/idmagic-batch/main.go
    - backend/cmd/internal/bootstrap/seeding.go
  tests:
    - backend/cmd/internal/bootstrap/audit_event_record_test.go
    - backend/shared/http/testing_stack/stack_test.go
    - backend/shared/http/server_http/routes_contract_test.go
  stop_before_reading: [frontend, spec, infra/schema]
spec_impact: { kind: none, reason: "外から非公開パッケージへの import がないモジュールのパッケージを Go の internal/ へ移し、責務表の公開方式と公開パッケージを書き換えるだけである。HTTP の応答、認証方式、永続状態、ドメインイベント、外向きの通知は変えない。" }
---

# 外から非公開パッケージへの import がないモジュールを Go の internal/ へ移す

## 動機

モジュール設計の項目は、非公開の実装の最終的な配置を Go の `internal/` と定め、全モジュールを公開方式 `legacy` のまま残した。
最初の一つのモジュールの移行は別の項目で手順を確かめるが、二つ目以降の移行を受け持つ項目がない。
このままでは、文書が定める最終形と実装の乖離が解消されない。

ほかのモジュールが非公開パッケージを import しているモジュールは、その依存を解消するまで `internal/` へ移せない。
移すと Go のビルドが失敗するからである。
起票時の負債台帳では、外から非公開パッケージへ import されるモジュールは次の 8 個だった。

| モジュール | 外から非公開パッケージへの import |
| --- | --- |
| Tenancy | 54 |
| Authentication | 19 |
| ClaimMapping | 11 |
| Jobs | 9 |
| OAuth2 | 6 |
| IdManagement | 6 |
| WsFederation | 4 |
| Application | 2 |

残りの 12 個（IdGovernance、Authorization、Audit、Provisioning、Sourcing、ApiTokens、Seeding、SigningKeys、DataKeys、Saml、WorkloadIdentity、SharedSignals のうち、最初の移行の項目が選んだものを除く）は、ほかのモジュールの依存を変えずに移せる。
この項目はそれらを移し、残る 8 個は負債の解消後に[残りのモジュールの移行の項目](wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)で移す。

## 対象範囲

- 着手時に `mise run check-boundaries` と台帳から、外から非公開パッケージへの import がないモジュールを再計測して対象を確定する。
- 対象の各モジュールの本番パッケージを、ルートパッケージ、公開パッケージ、`internal/` のどれかに分類して移す。
- 責務表の公開方式を `internal` にし、公開パッケージを列挙する。
- 組み立て地点からの非公開パッケージへの直接の import を、ルートパッケージの操作へ移す。
- 最初の移行の項目で見つかった手順の不足が構造の文書へ反映されていなければ、ここで反映する。

## 対象外

- 外から非公開パッケージへ import されるモジュールの移行と、その依存の解消。
- `legacy` の公開方式の撤去。
- モジュールの分割と統合。
- 機能スライス間の規則の追加。

## 設計

公開パッケージを決めることは公開範囲の宣言であり、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)を適用する。
現在 `legacy` の判定で公開扱いの `domain` と `ports` を、そのまま公開パッケージへ列挙することを既定にしない。
ほかのモジュールと組み立て地点が実際に使う型と操作を調べ、使われていない `domain` は `internal/` へ移す（D3）。

モジュールごとに、外側に残るパッケージの分類、公開パッケージとその利用元、組み立て地点の結線の変更を設計の節へ記録する。
移行はモジュールごとに独立しているので、モジュール単位でコミットし、途中で止めても `legacy` と `internal` の併用で検査が通る状態を保つ。

### 再計測と対象の確定

基準 revision は `96d851960` とする。
`mise run check-boundaries` は成功した。
負債台帳の本番の非公開 import は Application 2 件、Authentication 1 件、IdManagement 2 件、OAuth2 3 件、WsFederation 2 件である。
`go list -json ./backend/...` の本番、通常テスト、外部テストの import を分けて調べた。
起票時にこの項目へ割り当てられた 12 モジュールから、完了済みの Authorization を除いた 11 モジュールは、本番の非公開 import がすべて 0 件である。
Tenancy、ClaimMapping、Jobs も現在は 0 件だが、起票時に後続項目へ割り当てた移行として今回の対象外に保つ。
テストにある実装への直接 import は、公開範囲を広げず、モジュールの構築操作または公開契約を使う fixture へ変更する。

### パッケージの分類と公開目的

すべてのモジュールのルートパッケージを組み立て地点向けに残す。
下表の公開パッケージ以外の本番パッケージは `internal/` へ移す。
SQL のクエリと生成コード、共有契約テストも同じ配置へ移す。

| モジュール | 公開パッケージと利用元 | ルートへ集める組み立て操作 |
| --- | --- | --- |
| IdGovernance | なし。IdManagement は自身の `UserMutationCommitter` ポートを使う | メモリと PostgreSQL の保存と捕捉の構築、HTTP 登録、実行器と dispatcher の構築 |
| Audit | `backend/audit/ports`。Authentication、IdManagement、OAuth2 が監査記録と保存ポートを使う | 保存層の構築、HTTP 登録、イベント検索属性の抽出 |
| Provisioning | なし。IdManagement と Application は自身の通知ポートを使う | 保存層の構築、属性と所属の供給元、worker の実行と同期の構築 |
| Sourcing | `backend/sourcing/scim/ports`。IdManagement が SCIM 所有権とライフサイクルの契約を使う | 保存層の構築、取り込みの所有権ガード、HTTP 登録 |
| ApiTokens | `backend/apitoken/domain`、`backend/apitoken/ports`。OAuth2 と Sourcing、HTTP の共通認証がトークン種別と認証ポートを使う | 保存層と Service の構築、設定、HTTP 登録 |
| Seeding | `backend/seeding/domain`。seed コマンドと contributor が宣言的な要求、計画、manifest の契約を使う | YAML の読み込み、秘密の解決、materialize と実行をまとめる。排他と検証は内部に残す |
| SigningKeys | `backend/signingkeys/domain`、`backend/signingkeys/ports`。OAuth2、WsFederation、SharedSignals が署名の型と KeyStore 契約を使う | メモリ、PostgreSQL、Vault の KeyStore 構築、HTTP 登録、バッチの鍵ライフサイクル |
| DataKeys | なし。暗号化を利用する側は自身の cipher と migrator のポートを実装する | 保存層、cache、registry と FieldCipher の構築、HTTP 登録、再暗号化ジョブ |
| Saml | `backend/saml/domain`、`backend/saml/ports`。Application が SP と profile の公開契約を使う | 保存層と replay store の構築、HTTP 登録。HTTP 固有のポートの型はルートで受け取る |
| WorkloadIdentity | `backend/workloadidentity/domain`。OAuth2 が検証結果の grant を使う | 保存層の構築、JWKS と SVID 検証器の構築、HTTP 登録 |
| SharedSignals | `backend/sharedsignals/domain`、`backend/sharedsignals/ports`。OAuth2 が失効エポックの読み取りを使い、公開ポートの実装はその入力と戻り値の型を使う | 保存層、署名器、検証器、反応器、配信処理と HTTP 登録の構築 |

### 境界の判断

入力は責務表、バックエンド設計の移行規則、現在の本番とテストの import、および既存の組み立て操作である。
保存の更新元、トランザクション、イベント、通知の順序はそのまま移す。
適用する制約は D3 と D6 である。
公開の型を介した既存の循環は負債台帳のまま保ち、新しい辺、再入、原子性の境界を作らないので D4 と D7 の変更はない。

候補 A は現在の `domain` と `ports` を一律に公開する案である。
IdGovernance、Provisioning、DataKeys の保存型とユースケースは外部の本番コードが名指さず、組み立ての都合だけで内部知識を公開するため採らない。
候補 B は実在する公開契約だけを残し、保存と外部技術の選択をルートの操作へ移す案であり、これを採る。
Sourcing の domain と WorkloadIdentity の ports も、他モジュールが名指す必要がないので移す。
Seeding の domain は contributor と要求の契約であり、YAML 読み込みや排他の実装を含む usecases と区別して残す。
公開する SharedSignals の domain は公開ポートの引数と戻り値を実装するために必要な型である。

変更シナリオは保存実装、HTTP 変換、JWT 検証器、YAML 読み込みを差し替えることである。
候補 A では組み立て地点が各実装を import して変わる。
候補 B ではルートの構築操作が差を吸収し、呼び出し側は現在の設定と外部ポートだけを渡す。
残る仮定は既存の公開ポートと構造的適合を維持できることであり、Go のコンパイルと既存テストで確認する。

## 計画

モジュールごとに移動、構築操作、呼び出し側、責務表を一つの変更として検証し、正式コミットを作る。
配線を外す故障を境界検査と組み立てテストで観測し、演算子のある新しい配線の分岐だけを基準 revision からのミューテーション検査へ選ぶ。
移動したテストが差分に含まれるため、振る舞いを保つ変更の RED 免除は使わない。
受け入れと単体の代替 RED は、責務表だけを `internal` にしたときの移し忘れと非公開 import を `check-boundaries` が拒否する結果とする。
業務ロジックは移動と import の置換だけに保ち、配線の既存の規範被覆と不足をモジュールごとに確認する。

## タスク

- [ ] T001 [Inventory] 対象のモジュールを再計測して確定し、モジュールごとに外部の利用元と組み立て地点からの import を記録する。
- [ ] T002 [Design] モジュールごとにパッケージの分類と公開パッケージを判断手順で決める。
- [ ] T003 [App] モジュールごとにパッケージを移し、利用元と組み立て地点を書き換え、責務表を `internal` にする。
- [ ] T004 [Verify] 変更を検証する。

## 検証

### モジュールごとの記録

- DataKeys: 公開パッケージを持たず、7 パッケージを移した。Module が保存先、暗号化、cache、migrator registry を組み、FieldCipher と再暗号化 handler に同じ依存を渡す。外部テストもこの構築操作と初期化、回転の操作を使う。`test-go-package` は DataKeys 54 件、MFA の PostgreSQL 9 件、外部 IdP の PostgreSQL 11 件が race 付きで成功した。境界検査、仕様影響検査と lint も成功した。PostgreSQL の保存先の旧構築関数は常に成功するため、移したエラー伝播の分岐には現在到達する入力がない。
- SigningKeys: `domain` と `ports` を公開し、6 パッケージを移した。メモリ、PostgreSQL、Vault の構築、HTTP 登録とバッチの鍵ライフサイクルをルートへ集めた。テストの保存型は公開 KeyStore へ替え、HTTP の検査は独立した JSON の読み取り型を使う。外部鍵の JWK と信頼しない JWT 鍵の証明書も製品の内部変換へ依存しない fixture にした。`test-go-package` は SigningKeys 41 件、batch 8 件、`test-go-test` の XML の署名と鍵回転は 2 件が成功し、PostgreSQL、境界検査と lint も成功した。
- Audit: `ports` だけを公開し、保存層、HTTP、usecases と共有契約を移した。既存の `TestCharacterize*` の本文を保ち、保存先の構築と検索属性の抽出をルートへ移した。`test-go-package` は Audit 81 件、bootstrap 119 件、Authentication/usecases 13 件が race 付きで成功した。`check-boundaries`、`check-event-contract` と lint も成功した。監査保存先の retention 用の操作を、構築操作の戻り値にも保つ。
- 移行の前提となる検査: 特性化テストの移動を削除として拒否する故障を受け入れテストで RED にした。削除されたファイルの本文が追加されたテストファイルにすべて一致する場合だけ移動として扱い、本文変更と削除は引き続き拒否する。`test-tools-file -- check/src/spec-impact.acceptance.test.ts` は 30 テスト、`typecheck-tools`、`lint-tools` が成功した。
- Sourcing: `scim/ports` だけを公開し、ほかの 7 パッケージを移した。worker の User と Group のガード、正式な SCIM のライフサイクル操作をルートから組む。`test-go-package -- ./backend/sourcing/...` は 482 テスト、`./backend/cmd/idmagic` は 4 テストが race 付きで成功し、PostgreSQL も実行した。lint が移動した保存アダプターの `(nil, nil)` を 3 件拒否したため、既存の保存ポート用の除外パスへ任意の `internal/` 区画を加え、同じ lint が成功した。
- WorkloadIdentity: 責務表だけの切り替えで、外側の 7 パッケージと System からの非公開 import 5 件を境界検査が拒否した。移動後は成功した。`test-go-package -- ./backend/workloadidentity/...` は race 付きで 90 テストが成功し、PostgreSQL も実行した。`compile-spec` 後の `TestAssembledRoutesMatchGeneratedOpenAPI` は成功した。ルートの変更行に対するミューテーション検査は演算子がなく 0 件だった。HTTP の経路登録の切断は最後に共通の組み立てテストで調べる。

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 外部の利用元に合わせて公開パッケージを広げると、`legacy` の判定より公開範囲が広がる。
  公開パッケージの追加ごとに D3 で、公開操作で表せるかを比べる。
- 移すパッケージ数が多く、差分が大きくなる。
  モジュール単位のコミットに分け、パッケージの移動と結線の変更を読み分けられるようにする。
