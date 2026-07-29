---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-29
depends_on: [wi-97-envelope-encryption-at-rest]
---

# DataKeys の DEK に scheduled rotation と TenantAdministrator 向け緊急ロックアウト経路を追加する

## Motivation
[[wi-97-envelope-encryption-at-rest]] で `DataKeys` context の
`RotateTenantDataKey`/`DisableTenantDataKey`/`DestroyTenantDataKey` usecase を
実装したが、これらを実際に呼び出す本番経路が一切無い (admin HTTP endpoint も
scheduled batch trigger も無い) ことが、同 WI の完了報告時のユーザー質問を
きっかけに判明した。

調査の結果:
- 他の work item / ADR にはこの gap への言及が一切無い。
- wi-97 自身は T007 と Completion の両方で「rotate/disable/destroy の admin
  endpoint は追加していない (将来 WI での配線を想定)」と開示しているが、
  その「将来 WI」を実際には起票していなかった (宣言だけで未追跡)。
- `spec/contexts/data-keys.yaml` は Rotate/Disable/Destroy/Bootstrap を
  `access: internal` と明記しており、HTTP 経由の手動操作は元々想定していない
  意図的な決定 (wi-97 Scope の「管理面は最小限の可視化のみ」と整合)。
- 一方で **scheduled cadence trigger の欠如は誰にも検討・判断されていない
  単純な欠落**。同じパターンを持つ `SigningKeys` context は admin action
  endpoint (`POST /api/admin/keys/rotate`, `POST /api/admin/keys/{kid}/disable`、
  `TenantAdministrator` scope) と scheduled cadence job
  (`idmagic-batch signing-key-lifecycle`、既定90日) の両方を持つが、
  `DataKeys` にはどちらも無い。現状、テナントの DEK は最初の暗号化操作で
  bootstrap された後、rotation が本番で一切トリガーされない。
  [[wi-97-envelope-encryption-at-rest]] T006 で実装した再暗号化ジョブ・
  destroy fail-closed gate は「Rotate が実際に呼ばれた後」初めて意味を持つ
  機構だが、Rotate 自体を呼ぶ経路が無いため実質的に到達不能なコードパスに
  なっている。

鍵ローテーションはセキュリティ/コンプライアンス上の標準的な要求 (侵害時の
被害限定、定期ローテーションポリシーの遵守) であり、主要な KMS/シークレット
管理システムとの比較検討 (下記 Design) でもこれは業界標準から外れている
ことが確認できた。

## Scope
- **scl**: `spec/contexts/data-keys.yaml` の `RotateTenantDataKey` /
  `DisableTenantDataKey` インターフェースの `access` を `internal` から
  `TenantAdministrator` policy (自テナント限定) に変更し `bindings: http`
  を追加する。`authorization.principals` に `TenantAdministrator` を追加する。
  `DestroyTenantDataKey` は `access: internal` のまま変更しない。
- **go**:
  - `backend/cmd/idmagic-batch` に `data-key-lifecycle` サブコマンド
    (scheduled cadence rotation) を追加する。
  - `backend/datakeys/handlers_http` に
    `POST /api/admin/data-keys/rotate` / `POST /api/admin/data-keys/disable`
    を追加する。
  - `backend/datakeys/usecases` 側で cadence 判定 (「現行 active DEK が
    どれだけ古いか」の算出) が必要なら追加する。
- **documentation**: README に `idmagic-batch data-key-lifecycle` の運用
  (既定 cadence、外部 scheduler での実行例) と、OpenBao 側の master key
  自体の `auto_rotate_period` 設定 (idmagic 側のコード変更不要な運用設定)
  について追記する。

## Out of Scope
- OpenBao 側の master key 自動ローテーション設定そのもの (Vault Transit の
  ciphertext はバージョンを内包し decrypt 側が自動解決するため、
  `auto_rotate_period` を有効化するだけで idmagic 側のコード変更なしに
  master key 層は今日から自動ローテーションできる。運用設定として README に
  言及するに留め、実装は行わない)。
- `DestroyTenantDataKey` の HTTP 経由での呼び出し (crypto-shredding の
  不可逆性を踏まえ、内部 usecase のままとする。将来必要になれば別 WI で
  CLI/ops tool としての慎重な設計を検討する)。
- BYOK / customer-managed key の管理 UI ([[wi-97-envelope-encryption-at-rest]]
  と同様、引き続き対象外)。
- 署名鍵 (`private_jwk`) 側の話は一切対象外
  ([[wi-32-kms-hsm-and-per-tenant-signing-keys]] が引き続き所有)。

## Design
### 業界比較 (AWS KMS / GCP Cloud KMS / HashiCorp Vault・OpenBao Transit)

| システム | 自動ローテーション | 手動/緊急ローテーション | 旧バージョンの扱い | Destroy/削除 |
|---|---|---|---|---|
| AWS KMS (symmetric CMK) | 既定365日、90日〜7年で設定可 (`EnableKeyRotation`) | オンデマンドで即時ローテーション可 | 全旧バージョンを恒久保持し復号に使用 | 明示的な `ScheduleKeyDeletion` (待機期間付き、慎重な別操作) |
| GCP Cloud KMS (symmetric key) | `rotation_period` を鍵自体に設定 (推奨90日、高感度データ向け) | 手動ローテーションも可 | 新規暗号化のみ新バージョンに切替、既存 ciphertext は旧バージョンのまま復号可能 | 「そのバージョンを使うデータが絶対に無いと確信できるまで destroy しない」ことを明記 (disable を先に検討) |
| HashiCorp Vault / OpenBao Transit | `auto_rotate_period` を鍵に設定 (既定 disabled、min 1h)。NIST 800-38D の暗号化回数上限に基づき運用者が頻度を決める | API で on-demand rotate も可 | 鍵バージョンは保持され、ciphertext 側にバージョンが埋め込まれているため decrypt は自動解決 | 明示的な version 削除操作 (別途、慎重に) |

共通パターン: (1) スケジュールベースの自動ローテーションが既定の主経路、
(2) 侵害対応用の手動/オンデマンドローテーションも別途提供、(3) ローテーションは
既存 ciphertext を壊さず新規操作にのみ影響、(4) destroy/削除は「対象バージョンへの
依存が無いと確信できるまで行わない」別の慎重な操作として明確に切り離されている。

`DestroyTenantDataKey` が `PendingCount` (全 `FieldMigrator` の未移行行数) を
見て fail-closed で拒否する gate は、まさに GCP の "never destroy until
certain no data still uses it" と同じ設計であり、destroy を internal/
運用限定のままにする判断は上記比較でも妥当と裏付けられる。逆に rotation を
internal のまま放置するのは業界標準から外れている — 主要 KMS はすべて
scheduled auto-rotation を標準搭載する。

### 採用する設計
- **scheduled cadence rotation** を主経路とする:
  `backend/cmd/idmagic-batch/main.go` の `runSigningKeyLifecycle`
  (`signingusecases.RotateSigningKeyIfDue`/`ArchiveExpiredSigningKeys` を
  テナントごとに呼ぶ) と同型で、テナントごとの現行 active DEK の
  `activated_at`/`created_at` が cadence (既定90日、`-cadence-days` flag、
  `signing-key-lifecycle` と同じ下限バリデーション) を超えていれば
  `RotateTenantDataKey` を呼ぶ `data-key-lifecycle` サブコマンドを追加する。
  Rotate が成功すれば [[wi-97-envelope-encryption-at-rest]] T006 で実装済みの
  auto-enqueue が再暗号化ジョブを自動的に起動するため、この WI 側での
  追加配線は不要。
- **緊急時の手動ローテーション/ロックアウト**も別経路として用意する:
  `backend/datakeys/handlers_http` に
  `POST /api/admin/data-keys/rotate` (`TenantAdministrator`、自テナントの
  DEK を rotate) と `POST /api/admin/data-keys/disable` (同、対象 version
  を即時ロックアウト) を追加する。`backend/signingkeys/handlers_http/
  admin_key_handler.go` の `requireTenantKeyManager`/`DisableTenantKey`
  ハンドラパターンをそのまま踏襲する。
- **Destroy は internal のまま**: 上記業界比較の通り、destroy/削除は
  自動化・セルフサービス化しない別枠の操作として扱う。

### 代替案として検討したが不採用
- Destroy も HTTP 経由でセルフサービス化する案: crypto-shredding の
  不可逆性とリスクの高さから見送り、internal のまま維持する
  (AWS/GCP の「delete は慎重な別操作」という設計とも整合)。
- master key 層 (OpenBao) の自動ローテーションを idmagic 側で明示的に
  実装する案: Vault Transit の仕様上 idmagic のコード変更なしで運用設定
  のみで実現できるため、実装せず README への言及に留める。

## Plan
1. SCL: `spec/contexts/data-keys.yaml` を更新し `just check` を通す
   (`scl-change` Skill)。
2. Go (Adapters, test-first): `backend/datakeys/handlers_http` に
   rotate/disable ハンドラを追加し、`backend/signingkeys/handlers_http` の
   既存テスト (`admin_key_handler_test.go` 相当) に倣った HTTP レベル
   テストを先に書く。
3. Go (Infrastructure, test-first 免除): `backend/cmd/idmagic-batch` に
   `data-key-lifecycle` サブコマンドを追加する。cadence 判定ロジックが
   usecase 層に必要ならそこは test-first で先に書く。
4. Documentation: README 更新。
5. `just verify` 一式を通し、work item を完了させる。

## Tasks
- [ ] T001 [SCL] `RotateTenantDataKey`/`DisableTenantDataKey` の access を
  `TenantAdministrator` + `bindings: http` に変更し、`DestroyTenantDataKey`
  は internal のまま維持する。`authorization.principals` に
  `TenantAdministrator` を追加する。`just check` で検証する。
- [ ] T002 [Go/Adapters] `backend/datakeys/handlers_http` に
  `POST /api/admin/data-keys/rotate` / `POST /api/admin/data-keys/disable`
  を実装する (test-first)。
- [ ] T003 [Go/Infrastructure] `idmagic-batch data-key-lifecycle`
  サブコマンドを実装する (cadence-days flag、下限バリデーション)。
- [ ] T004 [Documentation] README に運用手順 (cadence job の実行例、
  OpenBao 側 `auto_rotate_period` の言及) を追記する。
- [ ] T005 [Verify] rotate/disable の認可境界 (自テナントのみ、他テナント
  からは拒否)、cadence job が実際に `RotateTenantDataKey` を起動し
  T006 の再暗号化ジョブが自動 enqueue されることを確認する。

## Verification
- `just check`
- `just verify-go`
- `just build-go`
- 手動: `idmagic-batch data-key-lifecycle` を実行し、cadence を超えた
  テナントの DEK が rotate され、再暗号化ジョブが enqueue されることを
  確認する。

## Risk Notes
Rotate は非破壊的操作 (旧 version は retiring として残り復号可能) であり、
GCP/AWS/Vault いずれも同じ設計のため相対的にリスクは低い。ただし cadence
job が意図せず高頻度で走ると再暗号化ジョブの負荷が積み重なるため、
`-cadence-days` の下限バリデーション (signing-key-lifecycle と同様) を
入れる。TenantAdministrator への rotate/disable の権限付与は、既存の
signing key に対する同スコープの権限 (`admin`/`system_admin` ロールで
自テナントのみ) とまったく同じ認可境界を再利用するため、新規のリスクは
限定的。
