---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例にテストを対応付け、具体例のとおりに振る舞っていなかった 2 か所を直す作業である。公開契約も運用手順も変えない。
  references: []
initial_context:
  specification:
    - docs/domain/data-keys/scenarios.feature.md#REQ-DATAKEYS-001
    - docs/domain/data-keys/scenarios.feature.md#REQ-DATAKEYS-002
    - docs/domain/data-keys/scenarios.feature.md#REQ-DATAKEYS-003
    - docs/domain/data-keys/scenarios.feature.md#REQ-DATAKEYS-004
    - docs/domain/data-keys/scenarios.feature.md#REQ-DATAKEYS-005
    - docs/domain/data-keys/scenarios.feature.md#REQ-DATAKEYS-006
  typespec:
    - IdMagic.Contract.DataKeyUnavailableError
    - IdMagic.Contract.DataKeyStillReferencedError
    - IdMagic.DataKeys.ListTenantDataKeyHealth
  source:
    - backend/datakeys/usecases/lifecycle.go
    - backend/datakeys/usecases/data_key_cache.go
    - backend/datakeys/usecases/tenant_data_key_health.go
    - backend/datakeys/field_cipher.go
    - backend/datakeys/domain/data_key.go
    - backend/datakeys/db_memory/data_keys.go
    - backend/datakeys/handlers_http/admin_data_key_handler.go
    - backend/datakeys/ports/field_migrator.go
    - backend/shared/security/envelope_crypto/envelope_crypto.go
    - backend/shared/http/support_http/auth.go
    - tools/check/example-coverage-debt.json
  tests:
    - backend/datakeys/usecases/lifecycle_test.go
    - backend/datakeys/field_cipher_test.go
    - backend/shared/http/server_http/control_plane_boundary_test.go
  stop_before_reading:
    - backend/datakeys/db_postgres
    - backend/shared/security/envelope_openbao
    - infra
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
---

# DataKeys が宣言する具体例 10 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/data-keys/scenarios.feature.md` が宣言する 10 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 10 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-DATAKEYS-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装にバグがあって具体例のとおりに振る舞っていないことが分かった場合は、難しくない修正なら本 work item で修正する。難しい修正なら別 work-item に切り出す。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## Design

### 消化の対象と観測の位置

10 件は 2 つの入口に分かれる。具体例 1 件に `//spec:covers` の注記 1 つを対応させる。

| 入口 | 具体例 | 観測の位置 |
| --- | --- | --- |
| DEK のライフサイクル（HTTP 経路を持たない System の操作） | EX-DATAKEYS-001-01、001-02、002-01、003-01、004-01、005-01、005-02 | `backend/datakeys/usecases/lifecycle_test.go` と、復号まで通す具体例は `backend/datakeys/field_cipher_test.go`。復号の可否は、利用側の Context が実際に使う `FieldCipher.Decrypt` で読む |
| `GET /api/admin/v1/data-keys/health` | EX-DATAKEYS-006-01、006-02、006-03 | `backend/shared/http/server_http/control_plane_boundary_test.go` の既存 fixture。`Register` の全配線を通る |

ライフサイクルの操作は HTTP の入口を持たないので、`testing_stack` には載せない。

### 具体例の型と Go の値の対応

ライフサイクルの具体例が名指すエラー型は、HTTP 応答ではなくドメインの失敗条件である。次の対応で読む。

| 具体例の型 | Go の値 |
| --- | --- |
| `DataKeyUnavailableError` | `envelope_crypto.ErrDataKeyUnavailable`（改名前の名前は `ErrDecryptionFailed`） |
| `DataKeyStillReferencedError` | `domain.ErrDataKeyStillReferenced` |
| `InvalidRequestError`（active の disable） | `domain.ErrDataKeyIsActive` |
| `AccessDeniedError`（006-02、006-03） | 403 の Problem Details、`access_denied` |

`DataKeyUnavailableError` は、契約の説明で「MasterKey プロバイダーに到達できない、wrap か unwrap が失敗した、AAD が一致しない」ときの失敗条件である。Go の番兵 `ErrDecryptionFailed` は unwrap と AAD の失敗だけを表し、名前も復号に限っている。wrap の失敗を同じ番兵に載せる修正（下記）と合わせて、番兵の名前を契約の型に揃えて `ErrDataKeyUnavailable` へ改める。改名は振る舞いを変えない構造の変更なので、別のコミットに先に置く。

### 具体例ごとの判断

| 具体例 | 既存テストの状態 | 行うこと |
| --- | --- | --- |
| 001-01 | 戻り値の版と状態だけを見ており、保存先も平文 DEK の不在も見ていない | 新しく書く。保存先を読み直して版 1 が `active` であることと、生成された平文 DEK が保存値にもイベントにも現れないことを見る |
| 001-02 | 無い | 新しく書く。wrap が失敗するプロバイダーで `ErrDataKeyUnavailable` になり、保存先に DEK が無く、イベントも出ないことを見る |
| 002-01 | 版 1 の `retiring` と復号は見ているが、版 2 が保存上 `active` であることを見ていない | 既存テストに `FindActive` の観測を足して注記する |
| 003-01 | disable のイベントだけを見ている | `FieldCipher` で新しく書く。版 1 が `disabled` になり、以後の復号が `ErrDataKeyUnavailable` になることを見る |
| 004-01 | 版 1 だけの状態で active の拒否を見ている | 既存テストを具体例の形（版 2 が active）へ直し、拒否の後に版 2 が `active` のままであることとイベントが出ないことを見る |
| 005-01 | 保存値の消去とイベントは別々のテストにあり、復号不能を見ていない | `FieldCipher` で新しく書く。登録済みの移行が未移行 0 件を報告する状態で destroy し、`destroyed` と `wrapped_dek` の消去と、キャッシュを作り直しても復号できないことを見る |
| 005-02 | 拒否と `wrapped_dek` の残存は見ているが、状態とイベントを見ていない | 既存テストに `retiring` のままであることとイベントの不在を足して注記する |
| 006-01 | ユースケース単体のテストだけで、入口を通していない | 新しく書く。複数テナントに DEK を置き、制御面テナントの `system_admin` で 200、各テナントの `active_version`・`status`・`provider_reachable` が返り、鍵素材（`wrapped_dek`、`master_key_id`）が無いことを見る |
| 006-02 | 無い | 新しく書く。制御面テナントの `system_admin` を持たない利用者で 403 `access_denied`、本文に他テナントの識別子と状態が無く、`FindAll` が呼ばれないことを見る |
| 006-03 | 403、本文の漏洩なし、`FindAll` 0 回を見ている | 注記を足す。本文が `access_denied` であることを足す |

### 直す欠陥

- **003-01。** `DataKeyCache.GetByVersion` は `destroyed` の版だけを拒否し、`disabled` の版は unwrap して返す。disable がキャッシュを無効化しても、次の復号で保存先から読み直した `disabled` の DEK で復号が通る。`disabled` も `ErrDataKeyUnavailable` で拒否する。修正は状態の判定を 1 つ足すだけで、設計の判断をやり直さない。
- **001-02。** `TinkEnvelopeCrypto.Wrap` はプロバイダーの失敗を `ErrDecryptionFailed` に載せずに返すので、呼び出し側は `DataKeyUnavailableError` として識別できない。unwrap と同じ番兵に載せる。

どちらも Scope の「難しくない修正」にあたるので本項目で直す。

## Plan

1. 台帳から 10 件を外し、`mise run check-spec` が 10 件を名指しで落とすことを観測する。
2. 番兵を `ErrDataKeyUnavailable` へ改名する（構造の変更、先のコミット）。
3. ライフサイクルの 7 件を消化する。003-01 と 001-02 は Unit RED を観測してから直す。
4. 健全性一覧の 3 件を消化する。
5. 変更した Go の変異を読み、`mise run verify` を通す。

## Tasks

- [x] T001 [Acceptance] `mise run check-spec` が DataKeys の 10 件を名指しで落とすことを、消化前に観測する。
- [x] T002 [Refactor] `ErrDecryptionFailed` を `ErrDataKeyUnavailable` へ改名する。
- [x] T003 [Use Cases] 001-01、001-02、002-01、004-01、005-02 を消化し、wrap の失敗を番兵に載せる。
- [x] T004 [Use Cases] 003-01、005-01 を `FieldCipher` で消化し、`disabled` の版の復号を拒否する。
- [x] T005 [Adapters] 006-01、006-02、006-03 を消化する。
- [x] T006 [Verify] 変異の読み取りと `mise run verify`。

RED、GREEN、故障注入は `mise run test-go-test -- <package> <Test>` で 1 本ずつ回す。パッケージごとに GREEN になったら `mise run test-go-package -- <package>` と `mise run lint-go` を回す。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Verification

- `mise run check-spec` が、`docs/domain/data-keys/scenarios.feature.md` の 10 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。注記に「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうち注記だけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。注記へ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範は 1 行も動いていない。
  変わったのは、`docs/domain/data-keys/scenarios.feature.md` が宣言する 10 件すべてが、
  その id を名指しするテストから到達されるようになったことと、具体例のとおりに振る舞っていなかった 2 か所である。
  disable した DEK の版で暗号文を復号できていたのを、フェイルクローズで拒否するようにした。
  MasterKey プロバイダーへの wrap の失敗を、`DataKeyUnavailableError` にあたる番兵で識別できるようにした。
  番兵は `ErrDecryptionFailed` から `ErrDataKeyUnavailable` へ改名し、契約の型と名前を揃えた。
  台帳は 47 件から 37 件へ減った。残した行は無い。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-DATAKEYS-001
  - **Observed Failure**: DataKeys の 10 件を `tools/check/example-coverage-debt.json` から外した状態で exit 1。
    REQ-DATAKEYS-001 から REQ-DATAKEYS-006 までの 10 行それぞれが、
    `EX-DATAKEYS-NNN-MM is declared, but no test names it.` の形で id を名指しした。
  - **Detection Reason**: この検査は、宣言された id と、テストの `//spec:covers` が名指した id の集合を比べる。
    テストを書かずに台帳から外せば必ず落ちるので、「消化した」と「台帳から消した」を取り違えられない。
- **Unit RED Evidence**:
  - **Test**: `TestFieldCipherDecryptFailsClosedAfterDisablingTheRetiringVersion`
    (`backend/datakeys/field_cipher_test.go`)
  - **Requirement**: REQ-DATAKEYS-003
  - **Observed Failure**: `Decrypt under disabled v1 = ("JBSWY3DPEHPK3PXP", <nil>), want ErrDataKeyUnavailable`。
    もう 1 件、`TestBootstrapTenantDataKeyFailsClosedWhenMasterKeyProviderIsUnreachable`（REQ-DATAKEYS-001）が
    `BootstrapTenantDataKey error = envelope_crypto: wrap data key: fake: master key provider unreachable, want ErrDataKeyUnavailable`
    で落ちた。
  - **Detection Reason**: 前者は、disable の後に利用側の Context が実際に使う `FieldCipher.Decrypt` で復号を試みる。
    `DataKeyCache.GetByVersion` は `destroyed` だけを拒否し、`disabled` の版は保存先から読み直して unwrap していた。
    既存テストは disable のイベントだけを見ていたので、ロックアウトが効いていないことを検出できなかった。
    後者は、wrap の失敗が番兵を持たず、呼び出し側が「DEK が使えない」失敗として識別できないことを検出する。
- **Change-Resistance Results**:
  リスクは low である。`mise run test-go-mutation` を変更した 3 パッケージに回した。
  - `backend/datakeys`：5 個すべてを殺した。
  - `backend/shared/security/envelope_crypto`：10 個すべてを殺した。
  - `backend/datakeys/usecases`：49 個のうち 43 個を殺し、3 個が生き残った（3 個はビルド不能）。
    生き残りはどれも今回変えていない行にある。
    - `data_key_cache.go` 104 行の `Invalidate` のテナント比較：複数テナントを載せたキャッシュの無効化をどのテストも見ていない。
    - `lifecycle.go` 77 行の再暗号化ジョブ投入失敗の分岐：ログだけで、観測できる結果を持たない。
    - `reencrypt.go` 144 行：本項目の具体例の外にある。

  変異器が表せない故障を 2 件手で入れた。2 件とも検出された。
  1. 健全性一覧のハンドラーから `RequireControlPlaneUser` の拒否を外す → `TestControlPlaneDataKeyHealthRejectsSystemAdminOutsideControlPlaneTenant` と
     `TestControlPlaneDataKeyHealthRejectsControlPlaneMemberWithoutSystemAdmin` が落ちる。
  2. 健全性の `provider` へ `master_key_id` を混ぜる → `TestControlPlaneDataKeyHealthListsEveryTenantWithoutKeyMaterial` が落ちる。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run check-spec` - passed（DataKeys の行は台帳に残っていない）
  - `mise run test-ui-e2e` - 実行していない。変えたのは DataKeys の復号可否の判定、wrap の失敗の番兵、テストだけで、
    フロントエンドから到達する応答は変えていない。

### 消化した具体例とテストの対応

| 具体例 | テスト | 行ったこと |
| --- | --- | --- |
| EX-DATAKEYS-001-01 | `TestBootstrapTenantDataKeyPersistsOnlyTheWrappedVersionOne` | 新しく書いた |
| EX-DATAKEYS-001-02 | `TestBootstrapTenantDataKeyFailsClosedWhenMasterKeyProviderIsUnreachable` | 新しく書き、wrap の失敗を直した |
| EX-DATAKEYS-002-01 | `TestRotateTenantDataKeyThenDecryptStillWorksForOldVersion` | 版 2 が保存上 active であることの観測を足した |
| EX-DATAKEYS-003-01 | `TestFieldCipherDecryptFailsClosedAfterDisablingTheRetiringVersion` | 新しく書き、disabled の版の復号を直した |
| EX-DATAKEYS-004-01 | `TestDisableTenantDataKeyRejectsActiveVersion` | 具体例の形（版 2 が active）へ直し、状態とイベントの不在を足した |
| EX-DATAKEYS-005-01 | `TestFieldCipherCannotDecryptUnderADestroyedVersion` | 新しく書いた |
| EX-DATAKEYS-005-02 | `TestDestroyTenantDataKeyRejectsWhenMigratorReportsPendingRecords` | retiring のままであることとイベントの不在を足した |
| EX-DATAKEYS-006-01 | `TestControlPlaneDataKeyHealthListsEveryTenantWithoutKeyMaterial` | 新しく書いた |
| EX-DATAKEYS-006-02 | `TestControlPlaneDataKeyHealthRejectsControlPlaneMemberWithoutSystemAdmin` | 新しく書いた |
| EX-DATAKEYS-006-03 | `TestControlPlaneDataKeyHealthRejectsSystemAdminOutsideControlPlaneTenant` | `access_denied` の観測を足して注記した |

### 親項目の測定への追加

DataKeys で注記だけで済んだ件は 0 件だった。既存テストがあった 6 件（001-01、002-01、004-01、005-01、005-02、006-03）は、
どれも `Then` の一部しか観測していなかった。003-01 の復号の観測が、disable によるロックアウトが効いていないという既存の欠陥を見つけた。

### 残したこと

- `RotateTenantDataKey`、`DisableTenantDataKey`、`DestroyTenantDataKey` を本番のコードから呼ぶ入口（CLI、ジョブ、管理 API）は無い。
  具体例の Primary actor は `System` であり、本項目はユースケースと `FieldCipher` の境界で観測した。入口を設けるかは本項目の範囲外である。
- disable が無効化するのは、同じプロセスの `DataKeyCache` だけである。別のレプリカがキャッシュ済みの DEK は、そのレプリカのキャッシュが作り直されるまで残る。
  上の入口が無い現在は顕在化しない。
