---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-01
depends_on: []
---

# OIDCクライアントシークレットを複数保持・有効期限管理できるようにする

## Motivation

現状の OIDC アプリケーション編集画面は「クライアントシークレットのローテーション」という
単一操作のみを提供しており、実行すると既存のシークレットがどうなるのか、ローテーション後の
移行期間がどうなるのかが利用者から見えず、不安を伴う操作になっている
（UI 上の配置・見せ方の問題は [[wi-314-hosted-and-admin-console-ui-wording-and-navigation-consistency]]
T018 で対応を試みたが、同じ設定カード・同じ保存フォーム内で位置を下へ移しただけであり、
クライアント ID を含む OIDC 設定から視覚的にも構造的にも独立できていない）。

Entra ID などの実装に倣い、有効期限付きのクライアントシークレットを複数同時に保持できるように
し、現在有効なシークレットの一覧（作成日・有効期限・状態）を確認でき、「ローテーション」は
実質「新しいシークレットを追加発行する」操作として提供し、移行が完了した後に利用者が
明示的に古いシークレットを失効させる、という設計の方がクライアント側の無停止移行がしやすく、
操作の見通しも良い。バックエンドのクライアントシークレット永続化モデルの変更を伴うため、
UI 文言・ナビゲーション整理を扱う wi-314 の範囲からは明示的に外し、本 WI として切り出す。

## Scope

- `spec/contexts/oauth2.yaml` の `models.ClientSecretCredential`、credential lifecycle state、
  発行・失効 interface/event/scenario/authorization
- `spec/contexts/application.yaml` の Application editor 向け credential metadata と発行・失効
  interface
- クライアントシークレットの発行・失効・検証を扱うドメイン・ユースケース・永続化・HTTP 層
- `frontend/src/features/admin-applications/AdminApplicationEditPage.tsx`
  （シークレット管理を OIDC 設定フォーム外の独立したトップレベルセクションへ移す）
- `frontend/src/features/admin-applications/ClientSecretRotationPanel.tsx`
  （複数シークレットの一覧・状態・有効期限表示、新規発行、個別失効の UI に刷新）

## Out of Scope

- クライアントシークレット以外の認証方式（private_key_jwt 等）の追加。
- 既存のシークレットハッシュ保存方式そのものの変更（暗号学的な保存方式は現行を維持し、
  「1個だけ保持」を「複数を有効期限付きで保持」に拡張する部分のみを扱う）。

## Design

### 調査で確認した既存実装

- `oauth2_client_secrets` と `ClientSecretCredential` は既に 1 client : N credentials を保持し、
  `credential_id`、hash、`created_at`、`expires_at`、`revoked_at` を持つ。新規テーブルや hash
  方式の変更は不要である。
- client 作成時は credential レコードも保存され、token endpoint は全 credential のうち
  `revoked_at` がなく期限前のいずれかと一致すれば認証する。期限判定はオンデマンドで行うため、
  期限切れを作るバッチは不要である。
- Application detail は非機密 metadata 一覧を既に返す。一方、現行の rotation は全 active
  credential に grace expiry または revoke を設定してから期限なしの新 credential を作るため、
  「追加発行後、利用者が旧値を個別失効する」運用にはなっていない。
- SCL の最大2件制約はコード・DBで原子的に強制されておらず、並行発行時に上限を超え得る。
  発行・失効専用 HTTP interface、個別監査イベント、adapter/UI テストも存在しない。

### Credential lifecycle

- `ClientSecretCredential` の状態は `Active`、`Expired`、`Revoked` とし、`revoked_at` を最優先、
  次に `expires_at <= now` を `Expired`、それ以外を `Active` として導出する。新規発行 credential
  は `expires_in_days`（1..730、既定90）を必須の期限へ変換する。
- 同時に有効な credential は最大2件とする。発行は既存 credential を一切変更せず、2件が active
  なら拒否する。上限検査と insert は client 行をロックする repository transaction（memory は
  同一 mutex critical section）で原子的に行う。
- 個別失効は `client_id` と `credential_id` の組で対象を限定し、指定 credential だけへ
  `revoked_at` を設定する。失効済み credential への再実行は冪等に成功し、別 client／存在しない
  ID は拒否する。最後の1件の失効も incident response のため許可する。
- 既存の `expires_at = null` credential は無停止移行のため期限なしの legacy credential として
  維持する。credential table が空で legacy hash のみを持つ client は、最初の追加発行 transaction
  内で期限なし credential へ一度だけ backfill する。
- 旧 `rotate-secret` interface は互換性のため残すが、新 UI からは使用しない。ADR-125 の
  current+previous 自動 sunset を標準 UI とする判断だけを新 ADR で部分的に置き換え、最大2件、
  Application を唯一の編集面とする判断、秘密値を一度だけ返す判断は維持する。
- `ClientSecretIssued` と `ClientSecretRevoked` を既存の domain-event → AuditEvent 経路へ流す。
  payload は tenant、actor、client、credential、expiry の非機密 metadata のみに限定し、secret と
  hash は含めない。

### HTTP and UI

- Application editor に `POST .../oidc/client-secrets`（追加発行）と
  `DELETE .../oidc/client-secrets/{credential_id}`（個別失効）を追加する。どちらも tenant-scoped
  Application から OIDC binding を解決し、secret-based client 以外を拒否する。発行 response
  だけが平文を一度返し、両 response と Application detail は状態付き metadata 一覧を返す。
- UI は作成日・有効期限・状態を一覧し、active credential だけに個別失効操作を出す。新規発行時の
  平文はコピー確認後に閉じる一度限りの表示とする。
- シークレット管理 UI は、クライアント ID やリダイレクト URI 等を編集する OIDC 設定の
  一項目として扱わない。アプリケーション設定を保存する `<form>` の外へ出し、設定カードと
  兄弟関係にある専用のトップレベル `Card` として配置する。専用カードにはシークレット管理の
  見出し・説明・一覧・発行操作・失効操作を閉じ込め、単なる余白・区切り線・内側の警告枠だけを
  「独立したセクション」と見なさない。
- クライアント ID は引き続き OIDC 設定カード内の参照項目とし、シークレット管理カードへは
  重複表示しない。これにより、変更内容をまとめて保存する通常設定と、個別に即時実行される
  credential lifecycle 操作の境界を、見た目と DOM 構造の両方で一致させる。

## Plan

1. OAuth2/Application SCL に lifecycle、追加発行・個別失効 contract、正常・境界・拒否 scenario、
   既存 rotate interface の互換維持を定義し、派生物を再生成する。ADR-125 の部分上書きを記録する。
2. Domain に状態導出と発行・失効 event を追加し、Use Case に期限検証、上限、legacy backfill、
   個別失効を RED→GREEN で実装する。
3. Memory/Postgres repository に原子的発行を実装し、SQLC を再生成する。HTTP adapter に発行・失効
   route と状態付き DTO を追加し、token authentication の複数 active／期限切れ／失効済み動作を
   回帰テストする。
4. Frontend API/type/i18n を追加し、credential 一覧・発行・一度限り表示・個別失効 UI を実装する。
   管理 UI は通常設定 form の外にある兄弟トップレベル Card とする。
5. 層別テスト、`just verify`、`just test-ui-e2e`、SCL/record 検査を通し、実機相当の DOM・視覚境界を
   確認する。
6. Completion を記録して `work-items/done/` へ移し、差分を Conventional Commit にまとめる。
7. 実機フィードバックで判明した日本語ステータスバッジの内容幅依存と折り返しを、共通最小幅・
   中央揃え・折り返し禁止で修正し、表示回帰テストを追加する。

## Tasks

- [x] T001 [SCL] OAuth2/Application の models、states、interfaces、events、scenarios、authorization
      に期限付き追加発行・個別失効・最大2 active・互換 rotate を定義し、派生物を再生成する。
- [x] T002 [ADR] ADR-125 の自動 sunset UI 判断を、明示的な追加発行・個別失効へ部分的に
      置き換える判断を記録する。
- [x] T003 [Domain] `ClientSecretCredential` の状態導出を追加する。
      RED: 状態優先順位と期限境界の table test（SCL state `ClientSecretCredentialLifecycle`）→ GREEN。
      `TestClientSecretCredentialStatusAt` で Active / Expired の期限境界 / Revoked 優先を固定した。
- [x] T004 [UseCase] 期限付き追加発行、最大2 active、legacy backfill、個別失効、非機密 audit event
      を実装する。RED: 発行正常系・期限境界・上限・非 secret client・個別失効・冪等失効・別
      credential 拒否・event payload の tests（SCL scenarios）→ GREEN。
      `client_secret_lifecycle_test.go` の発行・backfill・拒否・個別失効 tests と、互換
      rotation の上限回帰 test を RED → GREEN で通した。event struct は secret/hash を持たない。
- [x] T005 [Adapter] Memory/Postgres repository の発行上限を原子的にし、credential roundtrip と
      並行発行 contract を実装する。RED: repository tests（model constraint）→ GREEN。
      memory mutex critical section と PostgreSQL の client row `FOR UPDATE` transaction を実装し、
      並行発行で成功が1件だけになる repository tests を RED → GREEN で通した。
- [x] T006 [Adapter] Application HTTP の発行・個別失効 route、状態付き metadata、tenant/CSRF
      境界を実装する。RED: handler contract tests（Application interfaces）→ GREEN。
      `client_secret_lifecycle_test.go` で追加発行 201、上限 409、個別失効、冪等失効、状態 metadata、
      `Cache-Control: no-store` を RED → GREEN で固定した。
- [x] T007 [Adapter] token endpoint が複数 active credential を受理し、expired/revoked credential
      を拒否する回帰テストを追加する。RED ではなく既存 SCL contract の characterization test とする。
      `client_auth_test.go` の characterization test で2件の active を受理し expired/revoked を
      拒否することを確認した。
- [x] T008 [App] `ClientSecretRotationPanel` をシークレット一覧・新規発行・個別失効の UI に
      刷新する。RED: 状態表示・発行・一度限り表示・失効・上限・error の component tests → GREEN。
      期限プリセット、状態一覧、最大2件制御、一度限り表示、確認付き個別失効を実装し、4 component
      tests を RED → GREEN で通した。
- [x] T009 [App] シークレット管理 UI を `AdminApplicationEditPage` の設定保存 `<form>` から
      完全に外し、クライアント ID を含む設定カードとは別のトップレベル `Card` に配置する。
      余白や区切り線による位置調整だけでは完了としない。
      `grid max-w-3xl gap-6` の直下で通常設定 Card の次に専用 Card を置き、専用見出し・説明・一覧・
      操作を閉じ込めた。panel 自身は Card を重ねず、トップレベル Card の責務を明確にした。
- [x] T010 [Test] シークレット管理の見出し・操作が設定保存 `<form>` に含まれず、クライアント
      ID と異なるトップレベルカードに属することをコンポーネントテストで固定する。
      `places client secret management outside the settings form in its own top-level card` を先に
      失敗確認し、専用 region が form 外・クライアント ID と別 Card であることを GREEN で固定した。
- [x] T011 [Verify] `just verify`、`just test-ui-e2e`、`just check` を通し、実機相当レビューを行う。
      全コマンド成功。DOM 境界テスト、24px の sibling Card 間隔、E2E 4 spec / 22 tests で
      視覚・構造境界と主要ブラウザー導線を実機相当確認した。
- [x] T012 [App] 日本語ステータスバッジの幅を状態間で揃え、「失効済み」を1行に保つ。
      RED: 日本語の「有効」「失効済み」に共通最小幅・中央揃え・折り返し禁止がないことを
      component test で確認 → `min-w-20 justify-center whitespace-nowrap` と状態列の
      `whitespace-nowrap` を追加して GREEN。`just verify-ui` も成功した。

## Verification

着手時にバックエンドの層別検証コマンドを確定する。既存クライアントの無停止移行（新シークレット
発行後、旧シークレットが失効するまで両方が有効であること）に加え、UI では次を確認する。

- コンポーネントテストで、シークレット管理セクションがアプリケーション設定の `<form>` 外に
  あり、クライアント ID と同一のトップレベル `Card` に含まれないこと。
- 実機レビューで、通常設定カードとシークレット管理カードの間に明確なカード境界と余白があり、
  シークレット管理が OIDC 設定項目の続きに見えないこと。
- `just verify-ui`、`just test-ui-e2e`、`just check` が成功すること。

## Risk Notes

トークンエンドポイントの認証ロジックに直接関わる変更であり、既存の稼働中クライアントの
認証を壊すと影響が大きい。移行期間中は新旧シークレットが両方有効であることを保証し、
既存の期限なし credential は grandfather する。発行上限は use case の事前検査だけでなく
repository transaction で原子的に保証する。

未信頼入力は UUID と整数の単純な固定形式で、再帰文法や組み合わせ爆発を持たない。高リスク箇所は
認証状態遷移と並行上限であるため、fuzz/property test は採用せず、table test、repository concurrency
test、HTTP contract test、token endpoint 回帰テストで検証する。

## Completion

- **Completed At**: 2026-08-01
- **Summary**:
  OIDC client secret を有効期限付きで追加発行し、最大2件を同時利用しながら credential 単位で
  明示失効できる lifecycle を、SCL から domain/use case、memory/PostgreSQL、Application HTTP、
  token authentication、audit、UI まで実装した。既存の期限なし credential は維持し、table が
  空の legacy client だけを最初の発行 transaction 内で backfill する。ADR-152 で ADR-125 の
  自動 sunset UI 判断だけを部分的に置き換え、旧 rotate API は互換経路として残した。

  UI はクライアント ID を含む通常設定 Card と別のトップレベル Card にし、設定 `<form>` の外へ
  完全に分離した。専用 Card には見出し・説明・credential 一覧・発行・失効だけを置き、両 Card は
  `gap-6` の兄弟要素である。単なる下方向への移動や内側 section 化ではないことを DOM test で
  固定した。完了後の実機フィードバックで判明した状態バッジの内容幅依存も解消し、「有効」だけの
  場合も十分な楕円幅を保ち、「失効済み」は折り返さない表示へ修正した。
- **Affected Guarantees State**:
  新規発行は既存 credential を変更せず、repository transaction が同時 active 最大2件を原子的に
  保証する。token endpoint は active credential のどちらも受理し、`expires_at <= now` または
  revoked credential を拒否する。発行時の平文は no-store response と一度限り UI にのみ現れ、
  audit event には secret/hash を含めない。
- **Verification Results**:
  - `just check` — passed（SCL、work-items、IDs、architecture、traceability）。
  - `just verify-go` — passed（lint 0 issues、全 Go tests with race）。
  - `just verify-ui` — passed（format、lint、unit、typecheck、build）。
  - `just test-ui-e2e` — passed（4 spec files、22 tests）。
  - `just verify` — passed（全 check を並行実行、25秒）。
- **Test-first self-attestation**:
  domain、use case、memory/PostgreSQL concurrency、HTTP、frontend API/component、Card 境界は
  RED → GREEN を確認した。token endpoint と audit category は既存契約の characterization test と
  して追加した。手動ブラウザー目視は実施せず、DOM/Card layout 検査と E2E で代替した。
- **Out of Scope として意図的に対応しなかったこと**:
  private_key_jwt 等の認証方式追加、既存 hash 保存方式の変更、active 上限2件の設定可能化、
  期限切れ credential のバッチ削除は実装していない。いずれも本 WI の Scope/Design どおりであり、
  SCL standards の adoption を partial/excluded へ狭める変更はない。
