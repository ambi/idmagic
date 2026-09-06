---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-06
priority: p1
depends_on:
  - wi-468-system-console-privileged-session-assurance
  - wi-469-bound-system-console-cross-tenant-reads
  - wi-470-record-control-plane-state-changes-in-the-audit-log
  - wi-471-align-the-tenant-quota-update-route-with-the-admin-api-conventions
change_kind: feature
affected_spec:
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-003 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-011 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-012 }
  - { path: docs/contexts/tenancy/scenarios.feature.md, requirement: REQ-TENANCY-014 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-005 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: docs/contexts/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-015 }
  - { path: docs/contexts/application/scenarios.feature.md, requirement: REQ-APPLICATION-001 }
  - { path: docs/contexts/oauth2/scenarios.feature.md, requirement: REQ-OAUTH2-035 }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.ListTenants }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.GetTenant }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenant }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.DisableTenant }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.EnableTenant }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantQuota }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.ListAdminUsers }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.GetAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.ListGroups }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.GetGroup }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.ListAgents }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.GetAgent }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.ListAdminApplications }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.GetAdminApplication }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.ListAdminOAuth2Clients }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.GetAdminOAuth2Client }
---

# システムコンソールでテナントの参照、変更、削除、リソース確認を完結させる

## Motivation

システムコンソールのテナント画面は、一覧、選択中テナントの参照、設定変更を一つの画面へ詰め込んでいる。詳細カードに表示名、パスワードポリシー、クォータ、正規ロケーションの入力欄が常時現れるため、参照中なのか変更中なのかを画面構造から判断できず、管理コンソールで採用している一覧、詳細、変更の分離とも一致しない。

無効化と再開の API および一覧行の操作は既に存在するが、対象、影響、理由を確認する詳細画面上の危険操作として提示されていない。行末の操作は見つけにくく、一覧を確認するだけの操作からテナント全体の認証停止へ進めるため、制御面操作として必要な意図確認が画面構造に表れていない。

テナントの完全な利用終了を表す削除操作は存在せず、`TenantLifecycle` は物理削除を対象外としている。このため不要になったテナントを無効化して残すことしかできず、テナントが所有する利用者データ、設定、資格情報、ジョブ成果物を製品上の操作で消去できない。

また、システム運用者が見られるのはテナント属性とクォータ使用量の集計だけであり、使用量の根拠となるユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントを対象テナントごとに確認できない。障害調査や削除前確認で、集計値から実体へたどれない。

## Scope

- システムコンソールのテナント UI を、一覧 `/system/tenants`、読み取り専用の詳細 `/system/tenants/{tenant_id}`、変更専用 `/system/tenants/{tenant_id}/edit` に分ける。
- 一覧行から設定変更と無効化を取り除き、行選択は詳細画面への遷移だけにする。
- 詳細画面にテナント属性、状態、正規ロケーション、クォータと使用量、作成・更新・無効化日時を表示し、変更画面、無効化または再開、削除への明示的な操作入口を置く。
- 変更画面だけに、表示名、パスワードポリシー上書き、クォータ、正規ロケーションの入力欄と保存操作を置く。参照画面から同じ更新 API を直接呼べる UI を残さない。
- 無効化、再開、正規ロケーション切替には、依存する `wi-468` の特権変更保証、対象と影響の確認、操作理由を適用する。
- `default` 以外の無効化済みテナントに対し、復元不能な削除を要求できる制御面 API と確認 UI を追加する。
- 削除要求の受理時にテナントを `Deleting` へ遷移させ、通常のテナント解決、再開、設定変更、リソース変更から直ちに到達不能にしたうえで、再試行可能な非同期ジョブによりテナント所有データと外部資格情報を消去する。
- 消去完了後はテナントを `Deleted` の墓標として保持し、通常一覧から除外し、realm とテナント ID の再利用を禁止する。監査イベント、削除ジョブの結果、削除日時、対象識別子は既存の保持方針に従って残す。
- システムコンソールのテナント詳細から、ユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントの一覧と個別詳細を読み取り専用で確認できるようにする。
- テナント横断読出しには専用のシステム API を追加し、テナント管理 API に `tenant_id` 切替パラメーターを加えない。応答は署名付きカーソルとページサイズ上限を持ち、秘密値、パスワードハッシュ、クライアントシークレット、秘密鍵素材を返さない。
- 削除、削除拒否、消去完了、消去失敗を監査イベントとして記録し、操作者、対象テナント、操作理由、ジョブ ID、結果を制御面の監査画面から追跡できるようにする。
- Tenancy、IdentityManagement、Application、OAuth2、Jobs、Audit、System の規範シナリオ、TypeSpec、決定、状態遷移、内部機構を更新する。
- 新しい対応機能とテナント削除の非互換な運用上の意味をリリースノートとアップグレードノートへ記載する。

## Out of Scope

- システムコンソールから、対象テナントのユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントを作成、更新、削除すること。テナント横断の変更権限をリソース管理へ広げず、本作業項目は調査と削除前確認に必要な読出しだけを提供する。
- システム管理者が対象テナントの利用者としてログインすること、対象テナントの管理コンソールへ権限を借りて入ること、または利用者を偽装すること。
- セッション、同意、SSF ストリーム、監査イベント、ジョブ、署名鍵、データ鍵、エクスポート成果物に、今回の5種類と同じ汎用リソース詳細画面を追加すること。集計値と既存の専用システム画面への導線は表示するが、各資源固有の閲覧機能の拡張は別の作業項目とする。
- 削除済みテナントの復元、削除要求の取消し、realm またはテナント ID の再利用。
- バックアップ媒体から当該テナントだけを選択して消去すること。オンラインの正となる状態は削除するが、既存バックアップ内のデータはバックアップ保持期間と安全な期限切れに従う。
- 監査イベントの保持期間または改ざん耐性の変更。
- テナントデータのエクスポートまたは法令上の証明書を削除前に生成すること。

## Design

### 参照と変更の画面境界

テナント一覧は対象の発見、詳細画面は現在状態の確認、変更画面は設定値の編集だけを担う。詳細画面にはフォーム部品を置かず、変更操作は `/edit` へ遷移して初めて入力可能にする。管理コンソールの既存ページと同じく、保存後は詳細画面へ戻り、更新後の値を再取得して表示する。

正規ロケーション切替は変更画面に置くが、通常設定の一括保存には混ぜない。issuer、WebAuthn RP ID、Cookie の適用範囲を変える独立操作であるため、選択値を保存する直前に影響、対象、理由を専用確認画面で再掲する。

無効化と再開は一覧の行内操作ではなく、詳細画面の状態操作として置く。削除は同じ詳細画面の危険操作領域へ置くが、無効化や再開と同じボタンにはまとめない。

### テナント削除の意味

利用者が行う「削除」は、テナントのデータを利用不能にして消去する製品操作であり、`tenants` 行の即時物理削除ではない。監査の参照整合性、削除の進捗、realm の再利用禁止を維持するため、最小限の墓標を残す。

状態遷移は `Active -> Disabled -> Deleting -> Deleted` とする。`Active` からの削除、`default` の削除、`Deleting` または `Deleted` からの再開と更新は拒否する。同じ削除要求の再送は既存ジョブを返す冪等な操作とし、新しい消去を重複して開始しない。

削除確認では対象の表示名、realm、テナント ID、主要リソース件数、消去されるデータ、監査とバックアップに残るデータを表示し、操作者に realm の完全一致入力と空でない理由を要求する。依存する `wi-468` の有効な MFA ステップアップがなければ要求を受理しない。

要求を受理する同期境界は、状態を `Deleting` に変えてジョブを永続化するところまでとし、`202 Accepted` とジョブ参照を返す。各コンテキストは自身が所有するテナントデータを消去する冪等な効果境界を提供し、削除ジョブが依存順に呼ぶ。時刻、ジョブ ID 採番、永続化、監査イベント発行、外部鍵または資格情報の破棄はユースケースへ注入する効果として扱う。

途中失敗ではテナントを `Deleting` のまま到達不能に保ち、成功済みの消去段階を再実行しても結果が変わらないようにする。失敗内容と最後に完了した段階はジョブへ記録し、再試行で残りを進める。全段階の完了後にだけ `Deleted` へ遷移する。

墓標にはテナント ID、realm、作成日時、削除要求日時、削除完了日時だけを保持し、表示名、設定、外装、属性スキーマ、通知テンプレート、クォータ、利用量は消去する。監査イベントとジョブ履歴は各保持方針に従って残し、秘密鍵など外部保管された資格情報は参照だけでなく実体の破棄結果を確認する。

物理的に `tenants` 行を削除し、外部キーの `CASCADE` だけへ依存する案は採らない。`RESTRICT` を持つ履歴データ、外部鍵、コンテキストごとの副作用を一つの SQL 文では完了できず、途中失敗の再開点と監査対象も失うためである。

削除を無効化の別名として扱う案も採らない。無効化は復帰可能でデータを保持する運用停止であり、復元不能な消去と同じ意味にすると確認内容と監査結果が曖昧になる。

### テナントリソースの読出し

今回の「各リソース」は、テナントの永続的な主体と連携設定を代表し、既存のクォータにも現れるユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントの5種類とする。セッションや監査のような運用時系列データまで一つの汎用一覧へ押し込まず、それぞれの専用画面へ委ねる。

各コンテキストは既存のテナント管理 API と別に、制御面主体と明示的な対象テナントを要求する読み取り専用の System 操作を所有する。既存の応答モデルと権限判断を持たない表示部品は再利用できるが、テナント管理ページへ `systemMode` や `targetTenantID` を渡して横断能力を切り替える形にはしない。

一覧は対象テナント、絞り込み条件、ページサイズに束縛した署名付きカーソルを使う。別テナントまたは別リソース種別のカーソルは拒否し、一回の要求の永続化処理量をページサイズの定数倍に制限する。詳細取得も対象テナントを明示的に照合し、別テナントの同じ識別子を返さない。

システム画面は調査用の読み取り専用表示とし、編集、削除、秘密表示、資格情報発行への導線を置かない。`Deleted` は常に未存在として返し、`Deleting` は削除進行中であることを明示して新しいリソース読出しを拒否する。

対象テナントの管理コンソールをそのまま開く案は採らない。制御面 User は `default` テナントに所属しており、別テナントのセッションやロールを借りなければテナント内 API を正しく呼べない。暗黙の偽装を導入せず、横断読出しであることが経路、TypeSpec 操作、シェルから分かる専用境界を使う。

### 仕様の所有

Tenancy は詳細、変更、無効化、再開、削除状態と削除オーケストレーションを所有する。IdentityManagement、Application、OAuth2 は自身のリソースを制御面から読み出す操作と返してよい項目を所有する。System はシステムコンソールだけが横断 UI の入口となる不変条件を所有し、Jobs と Audit は削除の実行状況と記録の観測を所有する。

仕様段階で既存シナリオを改変して異なる主体を詰め込まず、SystemAdministrator を主体とする新しい規範シナリオと TypeSpec 操作を割り当てる。割り当て後は `affected_spec` を新しい ID と操作記号へ更新する。

## Plan

1. テナント参照、設定変更、状態操作、削除、5種類のリソース読出しについて、主体、経路、応答、拒否、状態遷移、保持データを規範シナリオ、TypeSpec、決定、状態、内部機構へ先に定義する。
2. 詳細画面に入力欄が存在すること、一覧行から確認なしに無効化できること、削除 API が存在しないこと、制御面から対象テナントの5種類のリソースを読めないことを、UI と HTTP の受け入れ境界で RED として固定する。
3. テナントの `Deleting` と `Deleted`、削除要求、消去段階、墓標を純粋な状態遷移として実装し、禁止遷移と冪等性の Unit RED から GREEN にする。
4. コンテキストごとの冪等な消去効果と再開可能な削除ジョブを実装し、オンラインデータ、外部資格情報、監査・ジョブ保持の境界を検証する。
5. 5種類のリソースに、制御面主体限定、対象テナント固定、カーソル有界、秘密を含まない System 読出し API を実装する。
6. テナント UI を一覧、詳細、変更へ分け、危険操作の確認、削除ジョブ進捗、読み取り専用リソース一覧と詳細を日英で実装する。
7. 認証、対象照合、別テナントカーソル、削除拒否のそれぞれで読出しも状態変更も起きないことを確認し、成功経路では目的の対象だけが変化または返却されることを確認する。
8. 仕様生成物、SQL 生成物、経路ツリー、リリース文書を再生成し、互換性、セキュリティ、境界、標準検査を通す。

## Tasks

- [ ] T001 [Spec] Tenancy、IdentityManagement、Application、OAuth2、Jobs、Audit、System に新しい規範シナリオ、TypeSpec 操作、決定、状態遷移、内部機構を追加し、`affected_spec` を割り当てた ID と記号へ更新する。
- [ ] T002 [Acceptance] 現在の参照・変更混在、行内無効化、削除欠落、対象テナントのリソース読出し欠落を HTTP と UI の境界で観測し、RED を確認する。
- [ ] T003 [Domain] `Active`、`Disabled`、`Deleting`、`Deleted` の遷移、削除要求の冪等性、既定テナントと有効テナントの拒否を Unit RED から実装する。
- [ ] T004 [App] テナント墓標と削除ジョブを永続化し、要求受理時にテナント解決と変更を停止する。
- [ ] T005 [App] 各コンテキストが所有するテナントデータと外部資格情報を消去する冪等な効果境界を実装し、削除ジョブから再開可能な順序で実行する。
- [ ] T006 [Acceptance] 消去途中の代表的な失敗を注入し、テナントが到達不能のまま保たれ、再試行で重複副作用なく `Deleted` へ到達することを確認する。
- [ ] T007 [App] ユーザー、グループ、エージェント、アプリケーション、OAuth 2.0 クライアントの System 一覧・詳細 API を、制御面主体、対象テナント、ページサイズへ固定して実装する。
- [ ] T008 [Acceptance] 別テナントの識別子とカーソル、秘密項目の要求、MFA 未成立の主体を拒否し、対象リソースやテナント状態に副作用がないことを確認する。
- [ ] T009 [UI] テナント一覧、読み取り専用詳細、変更専用画面を別のルートとページ境界へ分け、保存後の再取得と戻り先を実装する。
- [ ] T010 [UI] 無効化、再開、正規ロケーション切替、削除の対象・影響・理由・MFA 確認と、削除ジョブの状態表示を日英で実装する。
- [ ] T011 [UI] 5種類のリソース一覧・詳細を読み取り専用で実装し、既存の表示部品だけを権限判断なしで共有する。
- [ ] T012 [Acceptance] 一覧から詳細、変更、状態操作、削除、リソース調査までの主要経路を E2E で確認し、参照画面に変更可能な入力欄がないことを確認する。
- [ ] T013 [Docs] リリースノートとアップグレードノートに新しい画面構成、削除の不可逆性、墓標、realm 再利用禁止、バックアップ保持、System API を記載する。
- [ ] T014 [Verify] 仕様、生成物、API 互換性、セキュリティ制御、コンテキスト境界、UI、データベーススキーマの検査を通す。

## Verification

- `mise run check-spec`
- `mise run spec-render`
- `mise run check-api-compat`
- `mise run check-contract-drift`
- `mise run check-generated-contract`
- `mise run check-event-contract`
- `mise run check-security-controls`
- `mise run report-security-test-gaps`
- `mise run check-boundaries`
- `mise run check-schema`
- `mise run test-go-race`
- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run check-work-items`
- `mise run check-ids`
- `mise run verify`

受け入れ検証では、少なくとも次の観測結果を固定する。

- テナント詳細画面は現在値だけを表示し、変更用入力と保存操作を持たない。変更画面で保存すると詳細へ戻り、再取得した値が表示される。
- 無効化、再開、正規ロケーション切替、削除は、対象、影響、理由、必要な MFA ステップアップを確認しなければ状態を変えない。
- `default`、`Active`、権限不足、理由欠落、realm 確認不一致に対する削除拒否では、テナント状態、データ、資格情報、ジョブ、成功監査イベントが増減しない。
- 削除要求の受理後は対象テナントの認証、プロトコル、管理、リソース読出しが直ちに到達不能になり、同じ要求を再送しても消去ジョブが一つだけ存在する。
- 消去ジョブ完了後はオンラインのテナント所有データと外部資格情報が残らず、墓標、realm の予約、監査イベント、ジョブ結果だけが保持方針に従って残る。
- 消去途中の失敗後に再試行すると未完了段階だけが進み、すでに消去した外部資格情報の再破棄が誤った成功や別対象への副作用を生まない。
- 5種類の一覧と詳細は選択したテナントの情報だけを返し、ページサイズを超えて読み出さず、別テナントへ束縛されたカーソルを拒否し、秘密値を含まない。
- システムコンソールのリソース画面には作成、更新、削除、資格情報発行の操作がなく、テナント管理 API を横断読出しへ流用しない。

## Risk Notes

リスクは high とする。削除オーケストレーションまたは対象テナント照合を誤ると、別テナントのデータや資格情報を復元不能に消去する可能性がある。すべての消去効果は明示的な `tenant_id` を要求し、対象不一致、既定テナント、状態不一致を永続化や外部呼出しより前に拒否する。

データベースには `ON DELETE CASCADE` と `ON DELETE RESTRICT` が混在し、外部 KeyProvider や成果物ストアもあるため、SQL の外部キーだけでは完全な消去を証明できない。各コンテキストの消去結果をテストし、代表的な段階で失敗を注入して再開可能性と誤対象への副作用不在を確認する。

墓標を消すと realm を再利用でき、過去の issuer、Cookie、クライアント設定、監査記録が新しいテナントと混同される。墓標と realm 予約は残し、通常の詳細応答には個人情報や設定を残さない。

制御面のリソース読出しは新しいテナント横断能力である。専用 System 操作と `privileged_read` をサーバー側へ置き、画面の非表示だけを認可にしない。応答モデルを既存管理 API から機械的に流用せず、秘密と変更用メタデータを除いた項目を TypeSpec で明示する。

一つの画面コンポーネントをモード切替で共有すると、テナント内画面に横断対象指定が漏れる。共有するのは表、状態バッジ、読み取り専用の詳細表示など権限判断を持たない部品だけとし、ページ、ローダー、API クライアント、ルートガードは分離する。

新しい削除状態、公開 API 操作、監査イベント、規範 ID、および実行済み削除は撤回できないため、`reversibility` は irreversible とする。UI の配置だけは戻せても、削除済みデータと外部資格情報は復元しない。
