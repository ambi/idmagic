---
status: completed
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-10-01
priority: p2
depends_on: [wi-95161-declare-spec-impact-and-find-implicit-specifications]
change_kind: docs
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 既存の挙動を規則として明示し、利用者が依存してよい境界値、既定値、応答の形を初めて文書で約束する。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-78471-transcribe-implicit-specifications-of-tenancy.md }
initial_context:
  specification:
    - docs/domain/tenancy/resolution/scenarios.feature.md#REQ-TENANCY-006
  typespec: []
  source:
    - docs/domain/tenancy
    - backend/tenancy/domain/tenancy.go
    - backend/tenancy/usecases
    - backend/tenancy/handlers_http
    - backend/tenancy/db_memory/quota_repository.go
    - backend/tenancy/db_postgres/quota_repository.go
    - backend/tenancy/db_postgres/tenant_usages.sql
    - backend/shared/http/support_http/tenant_middleware.go
    - backend/shared/mediavalidation/image.go
    - SPECIFICATION_FORMAT.md
  tests:
    - backend/tenancy/handlers_http
  stop_before_reading: [frontend, backend/oauth2, backend/authentication, spec/generated]
affected_spec:
  - { path: docs/domain/tenancy/resolution/scenarios.feature.md, requirement: REQ-TENANCY-022 }
  - { path: docs/domain/tenancy/resolution/scenarios.feature.md, requirement: REQ-TENANCY-023 }
  - { path: docs/domain/tenancy/resolution/scenarios.feature.md, requirement: REQ-TENANCY-024 }
  - { path: docs/domain/tenancy/lifecycle/scenarios.feature.md, requirement: REQ-TENANCY-025 }
  - { path: docs/domain/tenancy/lifecycle/scenarios.feature.md, requirement: REQ-TENANCY-026 }
  - { path: docs/domain/tenancy/lifecycle/scenarios.feature.md, requirement: REQ-TENANCY-027 }
  - { path: docs/domain/tenancy/settings/scenarios.feature.md, requirement: REQ-TENANCY-028 }
  - { path: docs/domain/tenancy/settings/scenarios.feature.md, requirement: REQ-TENANCY-029 }
  - { path: docs/domain/tenancy/settings/scenarios.feature.md, requirement: REQ-TENANCY-030 }
  - { path: docs/domain/tenancy/settings/scenarios.feature.md, requirement: REQ-TENANCY-031 }
  - { path: docs/domain/tenancy/branding/scenarios.feature.md, requirement: REQ-TENANCY-032 }
  - { path: docs/domain/tenancy/branding/scenarios.feature.md, requirement: REQ-TENANCY-033 }
  - { path: docs/domain/tenancy/branding/scenarios.feature.md, requirement: REQ-TENANCY-034 }
  - { path: docs/domain/tenancy/branding/scenarios.feature.md, requirement: REQ-TENANCY-035 }
  - { path: docs/domain/tenancy/quota/scenarios.feature.md, requirement: REQ-TENANCY-036 }
  - { path: docs/domain/tenancy/quota/scenarios.feature.md, requirement: REQ-TENANCY-037 }
  - { path: docs/domain/tenancy/notification-template/scenarios.feature.md, requirement: REQ-TENANCY-038 }
  - { path: docs/domain/tenancy/notification-template/scenarios.feature.md, requirement: REQ-TENANCY-039 }
  - { path: docs/domain/tenancy/notification-template/scenarios.feature.md, requirement: REQ-TENANCY-040 }
  - { path: docs/domain/tenancy/integration-endpoints/scenarios.feature.md, requirement: REQ-TENANCY-041 }
  - { path: docs/domain/tenancy/integration-endpoints/scenarios.feature.md, requirement: REQ-TENANCY-042 }
  - { path: docs/domain/tenancy/attribute-schema/scenarios.feature.md, requirement: REQ-TENANCY-043 }
---

# Tenancy の既存コードにある暗黙の仕様を書き起こす

## 動機

Tenancy の実装には、仕様影響の宣言を導入する前から、コードにだけ存在する挙動がある。
エラー時のフォールバック、上限、順序、作用を起こさない条件などがコードにだけあると、実装を変えた人がそれを仕様変更だと認識できない。
仕様影響の宣言はこれからの変更だけを扱うので、既存の挙動は一度読んで分類する必要がある。

導入時点（2026-10-01）の `spec-review-candidates` の結果は次のとおりである。

| 項目 | 値 |
| --- | --- |
| 宣言済みの規則 | 21 件 |
| 仕様 ID を引くテストが実行しない本番コードの位置 | 281 箇所（5 パッケージ） |
| 対象にした仕様を引くテスト | 932 件 |

この数は読み始める位置の量を示すもので、仕様漏れの件数ではない。

## 対象範囲

- コンテキストの文書を機能ノードへ再編する。規範の意味を変えない構造変更として、書き起こしより先のコミットにする。
- 次の候補を再取得し、パッケージ全体を[既存コードからの書き起こし](../../docs/development/specification-first-workflow.md#既存コードからの書き起こし)の観点表で読む。
  - `mise run spec-review-candidates -- backend/tenancy`
- 見つけた細部を、維持すべき仕様、意図が疑わしいが外部から依存され得る挙動、実装詳細、不要なコードの四種類に分類する。
- 維持すべき仕様は、現在の挙動を機能ノードの規則として書き、その規範 ID を引くテストを書く。
- 意図が疑わしい挙動は、現在の挙動を規則として書き、要判断の欄に未決定の点を残す。
- 不要なコードを削除する。

## 対象外

- 挙動の是正。
  現在の挙動を明示する変更と、挙動を修正する変更を分けるため、是正は見つけた点ごとに別の work item で扱う。
- 実装詳細の文書化。
  その細部だけが異なる二つの実装をどちらも正しいと判断できるなら、仕様へ書かない。
- 候補をゼロにすること。
  候補は着手点であり、実行されないコードが実装詳細の場合もある。
- `frontend/` の既存コードの探索。

## 設計

手順、分類、観点表は[仕様先行の開発ワークフロー](../../docs/development/specification-first-workflow.md#既存コードからの書き起こし)が定める。
何を仕様として書くかは `SPECIFICATION_FORMAT.md` §6 の「仕様として書く実装上の細部」で判断し、規則は同じ節の「規則一件の書式」で書く。

着手時の `affected_spec` は起票時の仮の参照である。
着手時に、書き起こして宣言する規範要素へ置き換える。
新しい規則は `impact: modifies` として差分に現れる。

### 機能ノード

`backend/tenancy` は機能スライスを持たないので、機能ノードはコードではなく規則のまとまりで分けた。
テナントの解決の実装は `backend/shared/http/support_http` にあるが、規則は Tenancy が持つ。

| 機能ノード | 移した規則 | 節の語彙 |
| --- | --- | --- |
| `resolution` | REQ-TENANCY-006〜011 | API |
| `lifecycle` | REQ-TENANCY-003、014 と `TenantLifecycle` の状態遷移 | ライフサイクル |
| `settings` | REQ-TENANCY-019、021 | API |
| `branding` | REQ-TENANCY-004、005 | API |
| `attribute-schema` | REQ-TENANCY-002、020 | API |
| `quota` | REQ-TENANCY-012、013 | API |
| `notification-template` | REQ-TENANCY-015〜018 | API |
| `integration-endpoints` | REQ-TENANCY-001 | API |

一つの機能だけの判断と仕組みは機能ノードへ移し、複数の機能にまたがるもの（管理の認可、識別子の分割、`TenantBranding` と属性スキーマを独立させる判断、`Supporting` の分類）はルートに残した。

### 分類の結果

観点表で `backend/tenancy` とテナント解決のミドルウェアを読み、見つけた細部を次のとおり分類した。

| 細部 | 分類 | 反映先 |
| --- | --- | --- |
| Host の正規化と、多段のラベルを realm として読まないこと | 維持すべき仕様 | REQ-TENANCY-022 |
| 解決の拒否の本文、無効なテナントへの経路ごとの応答 | 維持すべき仕様 | REQ-TENANCY-023 |
| `Vary: Host`、subdomain の発行者が scheme と port を引き継ぐこと | 維持すべき仕様 | REQ-TENANCY-024 |
| realm の DNS ラベル規則、予約語、`xn--`、作成時の初期状態 | 維持すべき仕様 | REQ-TENANCY-025 |
| 一覧の id の昇順と、上限を読めないテナントの項目の省略 | 意図が疑わしい | REQ-TENANCY-026、wi-12979 |
| すでにその状態にあるテナントの無効化と再開 | 意図が疑わしい | REQ-TENANCY-027、wi-12979 |
| 信頼済みデバイスの有効期間の範囲と、範囲外の保存値の読み方 | 維持すべき仕様 | REQ-TENANCY-028 |
| 通知のデフォルト言語の検証 | 維持すべき仕様 | REQ-TENANCY-029 |
| パスワードポリシーの上書きの正規化と基準時刻 | 維持すべき仕様 | REQ-TENANCY-030 |
| 値が変わらない項目も `changed_fields` に載ること | 意図が疑わしい | REQ-TENANCY-031、wi-12979 |
| ブランド設定の文字列の正規化と上限、ラベルのバイト数の上限 | 維持すべき仕様、上限の一部は意図が疑わしい | REQ-TENANCY-032、wi-12979 |
| 画像の形式と大きさ、アップロードと削除の結果 | 維持すべき仕様 | REQ-TENANCY-033 |
| 公開ブランド設定の ETag とキャッシュ | 維持すべき仕様 | REQ-TENANCY-034 |
| 画像配信のヘッダー | 維持すべき仕様 | REQ-TENANCY-035 |
| Hard Quota のデフォルト値、実効値、未知のリソース、減算の下限 | 維持すべき仕様 | REQ-TENANCY-036 |
| 上限の更新が上書きの全体を置き換えること、負の上限 | 維持すべき仕様、負の上限は意図が疑わしい | REQ-TENANCY-037、wi-12979 |
| プレビューの補完とテナントの製品名 | 維持すべき仕様 | REQ-TENANCY-038 |
| 何も削除しなかったリセットのイベント | 意図が疑わしい | REQ-TENANCY-039、wi-12979 |
| 試し送りの言語と、送信失敗時の応答 | 維持すべき仕様 | REQ-TENANCY-040 |
| 署名証明書のフィンガープリントの形式 | 維持すべき仕様 | REQ-TENANCY-041 |
| 署名用の資格情報がないときの 503 | 維持すべき仕様 | REQ-TENANCY-042 |
| 動的グループが参照する属性の保護と、スキーマの全置換 | 維持すべき仕様 | REQ-TENANCY-043 |
| クォータ更新の経路が realm を解決せず、内部エラーの文字列を返すこと | 意図が疑わしい | 既存の wi-471 |
| クォータ更新と正規ロケーション切替が監査イベントを発行しないこと | 意図が疑わしい | 既存の wi-470 |
| Soft Quota の警告が実装されておらず、バックエンドの扱いも異なること | 文書と実装の食い違い | wi-38516 |
| 文脈にテナントがないとき `tenancy.TenantID` が default テナントを返すこと | 意図が疑わしい | wi-46256 |
| 未定義の属性スキーマの取得が、要求のたびに現在時刻を `created_at` に入れること | 実装詳細 | なし |
| ブランド設定の更新イベントの時刻を二度取得すること | 実装詳細 | なし |
| `TenantRepo` を配線しない構成で default テナントだけを解決すること | 実装詳細 | なし |
| `CheckQuotaAndIncrement` と `DecrementQuota` の委譲 | 実装詳細（ほかの Context が呼ぶので不要なコードではない） | なし |

不要と判断したコードはなかった。
内部設計のうち、規則と重なった値（Hard Quota のデフォルト値、信頼済みデバイスの上限）は規則へのリンクに置き換えた。
実装と食い違っていた記述（realm を取り出す正規表現、無効なテナントの 400 をプロトコルの経路に限る記述）は、実装に合わせて直した。

## 計画

1. 候補を再取得し、観点表でパッケージを読んで、細部を四種類に分類する。
2. 分類の一覧をこの work item に記録し、是正が必要な点は別の work item として起票する。
3. 維持すべき仕様と意図が疑わしい挙動を、機能ノードの規則として書く。
4. 書いた規則を引くテストを追加する。
5. 不要なコードを削除する。

未解決の問いはない。
個々の細部が仕様か実装詳細かは、着手後の分類で決める。

## タスク

- [x] T001 [Plan] 候補を再取得し、観点表でパッケージを読んで、見つけた細部を四種類に分類した一覧をこの work item に記録する。コンテキストの文書を機能ノードへ再編する。
- [x] T002 [Spec] 維持すべき仕様と意図が疑わしい挙動を、機能ノードの規則として書く。
- [x] T003 [Acceptance] 書いた規則の例を引くテストを追加し、各テストが現在の挙動を固定することを、挙動を変える誤実装の注入で確認する。単体と受け入れの確認は `mise run test-go-test -- <package> <test>`、パッケージ単位は `mise run test-go-package -- <package>`、PostgreSQL の契約はサンドボックスの外で実行する。
- [x] T004 [App] 不要と分類したコードを削除する。不要と分類したコードはなかった。
- [x] T005 [Plan] 是正が必要な点を、別の work item として起票する。
- [x] T006 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run check`
- `mise run verify`

## リスク

- 現在の偶然の挙動を仕様として固定する危険がある。
  将来の実装が維持する義務を負うかを先に判断し、義務のない細部は実装詳細として仕様へ書かない。
- 被覆は、実行されているが仕様に書かれていない挙動を見つけない。
  候補だけに頼らず、観点表でパッケージ全体を読む。
- 割り当てた規範 ID は取り消せない。
  規則の単位を決めてから ID を割り当てる。

## 完了

- **Completed At**: 2026-10-01
- **Summary**:
  `mise run spec-diff` は、REQ-TENANCY-022 から REQ-TENANCY-043 までの 22 件の規則の追加を報告し、既存の規則の変更、削除、状態遷移の変更は報告しない。
  Tenancy の文書を、テナントの解決、ライフサイクル、設定、ブランド設定、属性スキーマ、リソース上限、通知テンプレート、連携エンドポイントの 8 つの機能ノードへ分けた。
  この再編は規範の意味を変えない構造変更で、再編だけの時点で `spec-diff` は差分を報告しなかった。
  既存コードを観点表で読み、実装にだけあった挙動を 22 件の規則と 50 件の例として書き起こした。
  現在の挙動を規則として書き、振る舞いは変えていない。
  意図が疑わしい 6 点は規則に要判断として残し、wi-12979 で決める。
  Soft Quota の文書と実装の食い違いは wi-38516、文脈にテナントがないときに default テナントへ落ちる `tenancy.TenantID` は wi-46256 として起票した。
  クォータ更新の経路と監査イベントの欠落は、既存の wi-471 と wi-470 が扱う。
  内部設計のうち規則と重なった値は規則へのリンクに置き換え、実装と食い違っていた記述は実装に合わせた。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（規範 ID をテストが引いているかの検査）
  - **Requirement**: REQ-TENANCY-022
  - **Observed Failure**: 規則を書いた直後、50 件の例すべてについて `EX-TENANCY-0NN-MM is declared, but no test names it.` で失敗した。
  - **Detection Reason**: 書き起こしは既存の挙動を固定する作業なので、実装より先に失敗するのは、宣言した例をどのテストも引いていないという検査である。挙動そのものを固定できていることは、下のフォールト注入で確かめた。
- **Unit RED Evidence**:
  - **Test**: `backend/tenancy/domain/tenancy_test.go` の `TestEffectiveTrustedDeviceMaxAgeTreatsOutOfRangeAsDisabled`、`backend/tenancy/db_memory/contract_test.go` と `backend/tenancy/db_postgres/contract_test.go` の `TestQuotaContract`
  - **Requirement**: REQ-TENANCY-036
  - **Observed Failure**: N/A: 既存の挙動を固定するテストなので、書いた時点で通過した。代わりに、範囲外の値を有効として読む誤実装、0 を下回る減算、未知のリソースの計数、上書きを全置換しない `SetQuota` を注入し、それぞれのテストが失敗することを観測した。
  - **Detection Reason**: 規則が定める境界（範囲外の保存値を無効として読む、使用量は 0 で止まる、未知のリソースは数えない、更新は上書きの全体を置き換える）を、どれか一つでも外した実装はテストが拒否する。
- **Change-Resistance Results**:
  `mise run test-go-mutation` を `backend/tenancy/usecases`、`backend/tenancy/domain`、`backend/tenancy/db_memory` に実行した。
  検出は usecases が 147 件中 127 件、domain が 20 件中 17 件、db_memory が 10 件中 9 件である。
  このツールは各パッケージ自身のテストだけを実行するので、HTTP の入口から固定した規則の変異は生き残る。
  生き残った変異は次のとおり分類した。

  | 生き残った変異 | 分類 |
  | --- | --- |
  | `previewVars` の条件の否定 4 件 | HTTP のテストが固定する。`settings.ProductName` の条件を外す誤実装を手で注入し、`TestPreviewUsesTheTenantProductName` が失敗した |
  | `normalizeOverride` の境界 4 件、`Update` の委譲深さの境界 1 件 | HTTP のテストが固定する（EX-TENANCY-030-01、REQ-TENANCY-021 の既存テスト） |
  | `enforcePolicyFloor` の境界 8 件、`Update` の信頼済みデバイスの上限の境界 1 件 | テストの不足だった。基準値ちょうどと一つ弱めた値を読む `TestPasswordPolicyOverrideBoundaries` と、上限ちょうどを読む `TestTrustedDeviceMaxAgeAcceptsExactlyTheCeiling` を追加した |
  | `Update` の `maxAge < 0` の境界 1 件、`EffectiveTrustedDeviceMaxAge` の 0 の境界 1 件、メモリーの `Decrement` の 0 の境界 1 件 | 等価な変異。0 はどちらの分岐でも同じ結果になる |
  | `ListNotificationTemplates` の容量の算術 1 件 | 等価な変異。スライスの初期容量だけが変わる |
  | `ValidateNewRealm` の長さの境界 1 件 | HTTP のテストが固定する（63 文字の realm を受け付ける対照） |
  | `tenantSchema` の `EndpointStyle` の否定 1 件 | この作業の規則の外。未知の `endpoint_style` は、保存前に `ValidateEndpointStyleSelectable` が拒否する |

  ツールが表せない配線と作用の誤実装は手で注入した。
  `Vary: Host` の削除、無効なテナントの判定を到達経路の判定より先にする変更、404 の本文への項目の追加、subdomain の発行者の port の欠落、ブランド設定と画像配信の `Cache-Control` の変更、ETag の固定、試し送りの言語をテナントのデフォルトへ変える変更、プレビューの空欄を補わない変更、動的グループの参照の確認の配線外し、型の変更の見逃し、フィンガープリントの小文字化、有効期間の UTC 化の欠落、署名用の資格情報の解決失敗の見逃し、上限の読み取り失敗で一覧を失敗させる変更、`TenantCreated` の発行の削除、realm の空白の除去の欠落、予約語の照合の迂回、再度の無効化を何もしない操作にする変更、表示名だけの更新で基準時刻を進める変更を注入し、どれも対応するテストが失敗した。
  多段のラベルを realm として受け付ける変更だけは検出されなかった。
  realm は DNS ラベルの規則でドットを含められないので、多段のラベルは realm の検索でも見つからず、同じ 404 になる。
  現在の realm の制約のもとでは等価であり、この確認は realm の規則が緩んだときの多層防御として残した。
- **Verification Results**:
  - `mise run check-work-items` - 成功
  - `mise run check` - 成功
  - `mise run verify` - 成功（サンドボックスの外で実行）
