---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-25
priority: p3
depends_on: []
change_kind: feature
initial_context:
  source:
    - backend/audit/usecases
    - backend/audit/ports
    - backend/audit/db_postgres
    - backend/signingkeys/usecases
  tests:
    - backend/audit/usecases
    - backend/audit/db_postgres
  stop_before_reading:
    - frontend
affected_spec:
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Contract.ExportAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Contract.GetAdminAuditEvent }
---

# 監査ログの改ざん検知 (ハッシュチェーンと署名付きチェックポイント) を導入する

## Motivation

IdMagic は監査イベントを PostgreSQL に永続化し、検索・CSV エクスポートを提供する
(`docs/contexts/audit/decisions.md`)。しかし現在の監査ログは
**アプリケーションと同じ DB 権限で UPDATE / DELETE 可能な通常テーブル**であり、
「記録が後から書き換えられていないこと」を証明する仕組みが無い。

IdP の監査ログは「誰が誰に何の権限を与えたか」の一次証拠であり、侵害時のフォレンジックと
コンプライアンス監査の両方で使われる。この性質上、競合製品は不変性を明示的に売る:

- **Okta**: System Log は API で読み取り専用。管理者にも削除手段が無い。
- **Entra ID**: 監査ログは保持期間内は変更不可で、外部への継続エクスポートを推奨。
- **Keycloak**: 不変性の保証は無い (DB 直書きで改変可能) — ここが Keycloak が
  「エンタープライズ監査要件で追加製品を要する」と言われる理由の一つ。

IdMagic が「production-ready / enterprise-ready」を主張するなら、Keycloak と同じ水準では
不足である。特に、**DB へ書き込み権限を持つ攻撃者や内部者が痕跡を消せる**状態は、
監査ログの証拠価値そのものを無効化する。

本 WI は「(1) 各テナントの監査イベントを前件ハッシュで連鎖させ、(2) 定期的に署名付き
チェックポイントを発行し、(3) 検証 API とエクスポートで第三者が完全性を再計算できる」
状態を作る。改ざんを**防止**する (それは DB 権限と外部保全の役割) のではなく、
**検知可能にする**ことを目的とする。

## Scope

- **decision**:
  - `docs/contexts/audit/decisions.md` へ記録する決定 (監査ログ完全性): ハッシュ対象フィールドの正規化規則 (フィールド順序・時刻表現・
    NULL 表現)、チェーンの単位 (テナント単位で独立)、チェックポイント発行間隔と署名鍵
    (`docs/contexts/signing-keys/decisions.md` のテナントごとの署名鍵 の鍵を使うか専用用途鍵を切るか)、
    保持期間による古いイベント削除 (`docs/contexts/authentication/decisions.md` の認証イベント保持期間 /
    `docs/contexts/audit/decisions.md` の保持期間) とチェーン継続の両立方式
    (削除区間を封印済みチェックポイントで代表する)、DB 権限による append-only 強制の適用範囲、
    検証失敗時の運用手順を記録する。
- **specification**:
  - `Audit` に `AuditChainEntry` の概念 (既存 audit event に `sequence_no` / `prev_hash` /
    `entry_hash` を追加)、`AuditCheckpoint` model (tenant_id / sequence_no / chain_head_hash /
    signed_at / key_id / signature)、`AuditIntegrityStatus` enum (Verified / Broken / Unknown) を追加する。
  - `VerifyAuditChain` (範囲指定で再計算し不一致箇所を返す) / `ListAuditCheckpoints` /
    `GetAuditIntegrityStatus` interface を追加する。
  - `ExportAdminAuditEvents` を「イベント本文 + チェーンハッシュ + 該当区間のチェックポイントと
    署名」を含む検証可能エクスポートに拡張する。
  - `states` に AuditCheckpointIssued / AuditIntegrityVerificationFailed event を追加する。
  - `guarantees` に「監査イベントは append-only であり、事後の改変・欠落はチェーン検証で
    検知できる」を明文化する。
  - `scenarios`: 正常なチェーン検証成功 / 1 行改変後に検証が該当 sequence を指して失敗 /
    途中行削除で後続の prev_hash 不一致を検知 / 保持期間削除後も封印チェックポイントで
    検証が成立 / 他テナントのチェーンに影響しない。
- **go**:
  - 監査イベント書き込み時に、同一テナントの直前エントリの `entry_hash` を `prev_hash` として
    取り込み `entry_hash = SHA-256(canonical(entry) || prev_hash)` を計算する。
    順序と一意性は `(tenant_id, sequence_no)` の一意制約と行ロックで保証する。
  - チェックポイント発行を既存の運用バッチ (`idmagic-batch`) のタスクとして追加し、
    テナントごとにチェーン先頭を署名する。署名鍵は SigningKeys の provider 経由で取得する。
  - `VerifyAuditChain` は範囲を分割して再計算し、最初の不一致 sequence と種別
    (改変 / 欠落 / チェックポイント不一致) を返す。
  - 検証可能エクスポートは JSON Lines + マニフェスト (区間 / チェックポイント / 署名 / 使用鍵の
    公開鍵参照) を出力し、IdMagic 無しでも再計算できる形式にする。
- **persistence**:
  - `infra/schema/postgres.sql` に `sequence_no` / `prev_hash` / `entry_hash` 列と
    `audit_checkpoints` テーブルを追加する。
  - アプリ用ロールから監査テーブルの `UPDATE` / `DELETE` を剥がす方針を導入し、保持期間削除は
    専用ロール (バッチ) のみが行う。`infra/schema/README.md` に権限運用を記載する。
- **http**:
  - 検証 API・チェックポイント一覧・検証可能エクスポートのエンドポイントを追加する。
    検証は重い処理なのでレンジ上限を設ける。
- **ui**:
  - 監査ログ画面に完全性ステータス (最終検証時刻 / 直近チェックポイント / Broken 時の該当位置) と
    「検証を実行」「検証可能エクスポート」を追加する。
- **documentation**:
  - README / `infra/README.md` に、外部検証手順 (エクスポートから独自にハッシュを再計算する
    手順)、append-only 権限運用、検証失敗時のエスカレーション手順を追記する。

## Out of Scope

- WORM ストレージ / 外部不変ストレージへの継続転送。転送機構は
  [[wi-286-outbound-event-hooks-and-audit-log-streaming]] が担い、本 WI は「転送先で
  再検証できる形式」を提供するところまで。
- ブロックチェーン / 外部公証サービスへのアンカリング。
- 認証イベント以外のアプリケーションログの完全性
  (`docs/contexts/audit/decisions.md` により別物として扱う)。
- 監査ログの暗号化。→ [[wi-97-envelope-encryption-at-rest]]
- 改ざんの「防止」。DB 権限と外部保全の責務であり、本 WI は検知可能性を作る。

## Plan

- **チェーンはテナント単位で独立させる**。全テナント単一チェーンにすると書き込みが 1 行に
  直列化して認証系のスループットを壊す。テナント単位なら
  [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]] の分割方針とも整合する。
- **正規化を最初に固定する**。ハッシュ対象の直列化が曖昧だと、後の実装変更で過去の
  チェーンが検証不能になる。フィールド順序・時刻フォーマット (UTC / RFC3339 nano) ・
  NULL 表現を `docs/contexts/audit/decisions.md` に固定し、正規化関数の golden test を最初に書く。
- **保持期間削除との両立が設計の核**。古いイベントを消すとチェーンが切れる。
  「削除区間の直前で封印チェックポイントを発行し、削除後はその区間を
  `(from_seq, to_seq, sealed_hash)` として代表させる」方式を採る。削除前に必ず
  チェックポイントを発行することをバッチ順序で保証する。
- **append-only は DB 権限で効かせる**。アプリコードの規約だけでは内部者・侵害時に無意味。
  アプリロールから UPDATE/DELETE を剥がすのが最も効く。ただしローカル開発 / test 環境で
  同じ権限分離を強制すると開発が止まるため、権限分離は postgres モードの運用手順として
  導入し、コード側は「UPDATE/DELETE を発行しない」ことをテストで固定する。
- **書き込みホットパスへの影響を測る**。直前ハッシュ取得のための行ロックは競合点になる。
  テナント単位の採番テーブル + `INSERT ... RETURNING` で 1 往復に収め、
  [[wi-282-staging-load-testing-and-capacity-validation]] で実測する前提を明示する。
- 未決定: チェックポイント署名鍵を既存 tenant signing key と共用するか専用用途にするかは
  `docs/contexts/audit/decisions.md` で決める。監査用途の鍵をトークン署名と共用すると鍵ローテーションの制約が絡むため、
  用途分離を第一候補とする。

## Tasks

- [ ] T001 [Spec] `Audit` に sequence_no / prev_hash / entry_hash、AuditCheckpoint、
      AuditIntegrityStatus、interface 3 件、event 2 件、guarantee、scenario 5 件を追加し
      `mise run check-spec` を通す。
- [ ] T002 [Spec] 監査ログ完全性の決定を `docs/contexts/audit/decisions.md` に記録する (正規化規則・チェーン単位・チェックポイント
      間隔と鍵・保持期間削除との両立・append-only 権限・検証失敗時手順)。
- [ ] T003 [Domain] 正規化関数とハッシュ計算、チェーン検証ロジック (不一致種別の判別) を実装する。
      RED: golden な正規化バイト列テストと、改変 / 欠落を検知するテストを先に書く
      (scenario `Audit.chain_verification_detects_mutation`) → GREEN。
- [ ] T004 [Persistence] `sequence_no` / `prev_hash` / `entry_hash` 列、`(tenant_id, sequence_no)`
      一意制約、`audit_checkpoints` テーブルを `infra/schema/postgres.sql` に追加し、sqlc
      クエリを再生成する (`mise run sqlc-generate`)。RED: 並行 INSERT で sequence が重複しない
      テスト (`mise run test-go-race`) → GREEN。
- [ ] T005 [Write path] 監査イベント書き込みにチェーン連結を組み込む。RED: 連続書き込みで
      prev_hash が繋がるテスト → GREEN。既存の監査書き込み経路すべてを通ることを確認する。
- [ ] T006 [Batch] `idmagic-batch` にチェックポイント発行タスクを追加し、保持期間削除の
      **前**に封印チェックポイントを発行する順序を保証する。RED: 削除後も検証が成立する
      テスト → GREEN。
- [ ] T007 [Usecase/HTTP] `VerifyAuditChain` / `ListAuditCheckpoints` /
      `GetAuditIntegrityStatus` と検証可能エクスポートを実装する。レンジ上限と 403/404 を
      handler テストで固定する。RED → GREEN。
- [ ] T008 [Permissions] アプリロールから監査テーブルの UPDATE/DELETE を剥がす運用手順を
      `infra/schema/README.md` に追記し、コードが UPDATE/DELETE を発行しないことを
      静的に確認するテスト / lint を追加する。
- [ ] T009 [UI] 監査ログ画面に完全性ステータス、検証実行、検証可能エクスポートを追加する。
      RED: presentation logic の unit test → GREEN。
- [ ] T010 [Docs] README に外部検証手順と検証失敗時のエスカレーションを追記する。
- [ ] T011 [Verify] 下記 Verification を緑にする。`mise run spec-render` を実行する。

## Verification

- `mise run check` / `mise run check-spec` / `mise run check-work-items` / `mise run check-ids`
- `mise run test-go` / `mise run test-go-race` / `mise run verify-go`
- `mise run verify-ui`
- 手動: `mise run dev` (postgres モード) で (1) 数件の監査イベントを作り検証が成功すること、
  (2) DB で 1 行の内容を直接書き換えると検証が該当 sequence を指して失敗すること、
  (3) 1 行を直接削除すると欠落として検知されること、(4) 検証可能エクスポートを
  IdMagic 外のスクリプトで再計算して一致すること、を確認する。

## Risk Notes

チェーン連結を監査書き込みパスに入れるため、**認証系の書き込みレイテンシに直接影響する**。
テナント単位採番と 1 往復での連結に限定し、負荷実測を
[[wi-282-staging-load-testing-and-capacity-validation]] に引き継ぐ。
正規化規則を後から変えると過去のチェーンが検証不能になる不可逆な変更になるため、
`docs/contexts/audit/decisions.md` に固定し golden test で守る。将来の変更は「バージョン付き正規化」で扱う余地を残す。
保持期間削除とチェーンの両立は順序依存 (削除前に封印) であり、バッチ順序が崩れると
検証不能な区間が生まれる。封印済みでない区間の削除を拒否する fail-closed を実装する。
append-only の DB 権限分離は運用手順であり、適用漏れがあると本 WI の保証が名目上のものに
なる。README / infra ドキュメントで前提として明記し、検証手順に含める。
