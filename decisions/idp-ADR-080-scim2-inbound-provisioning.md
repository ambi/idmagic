# idp-ADR-080: SCIM 2.0 Inbound Provisioning の適用と削除ポリシーの統合

## ステータス

採用。`scl.yaml` の `contexts/scim.yaml` および Go パッケージ `internal/scim` に反映。

## コンテキスト

Okta、Google Workspace (Cloud Identity)、Microsoft Entra ID などの外部 IDP/ID 管理システムから `idmagic` に対してユーザーやグループのプロビジョニングを自動化するため、SCIM 2.0 (RFC 7643 / RFC 7644) プロトコルに基づく Inbound Provisioning サーバー機能が必要である。
手動の管理者 API による CRUD 操作だけでは、外部 IDP での組織変更や退職処理などをタイムリーかつ安全に同期することができない。
SCIM 2.0 サーバーを公開するにあたり、以下の設計課題を決定する必要がある：
1. `idmagic` が SCIM サーバーとして振る舞う範囲
2. SCIM の属性マッピングと `active` 属性のライフサイクルへのマッピング
3. SCIM `DELETE` リクエスト時の `idmagic` 内部の削除ポリシー（ADR-036 / ADR-072 の soft-delete/purge との整合性）
4. Bearer Token による認証とテナント分離の設計

## 決定

1. **SCIM 2.0 サーバーエンドポイント**:
   - 各テナント (Realm) のパスプレフィックス配下に `/scim/v2` エンドポイントをマウントする（例: `/realms/{realm_id}/scim/v2`）。
   - RFC 7644 に準拠し、以下のエンドポイントをサポートする：
     - `/Users` (GET, POST, GET/{id}, PUT/{id}, PATCH/{id}, DELETE/{id})
     - `/Groups` (GET, POST, GET/{id}, PUT/{id}, PATCH/{id}, DELETE/{id})
     - `/ServiceProviderConfig` (GET)
     - `/ResourceTypes` (GET)
     - `/Schemas` (GET)

2. **認証とマルチテナンシー**:
   - 各テナントごとに SCIM プロビジョニング用の Bearer Token を管理する。
   - `Authorization: Bearer <token>` ヘッダーによる認証を行い、トークンからテナント情報を解決する。トークンは管理画面から生成・ローテーション可能とする。

3. **User/Group 属性マッピング**:
   - **User**:
     - SCIM `id` = `User.sub`
     - SCIM `userName` = `User.preferred_username`
     - SCIM `name.formatted` / `displayName` = `User.name`
     - SCIM `emails[type=work].value` = `User.email`
     - SCIM `active` = `UserLifecycle.status == Active`
   - **Group**:
     - SCIM `id` = `Group.id`
     - SCIM `displayName` = `Group.name`
     - SCIM `members` = `GroupMember` 経由の所属ユーザー一覧

4. **`active` 属性とライフサイクル**:
   - SCIM で `active` を `false` に更新する要求（PUT または PATCH）を受信した場合、`User.lifecycle.status` を `Disabled` に遷移させる。
   - `active` を `true` に更新する要求を受信した場合、`User.lifecycle.status` を `Active` に遷移させる。

5. **`DELETE` の取り扱い（削除ポリシーの統合）**:
   - SCIM で `DELETE /scim/v2/Users/{id}` を受信した場合、即時の完全削除 (Purge) は行わず、**soft-delete**（`UserStatus.PendingDeletion` への遷移）を行う。
   - これは、同期設定ミスや外部 IDP 側の操作ミスによる一括データ消失を防止するため、ADR-072 のユーザー削除ポリシー（既定は soft-delete）と統合するためである。
   - soft-delete されたユーザーは 30 日間 (UserSoftDeleteGracePeriod) 保持され、その後自動的に `Purge` (anonymize cascade) される。
   - `DELETE /scim/v2/Groups/{id}` が呼ばれた場合は、PII を含まないため、即時かつ完全にグループを削除しメンバーシップを解除する。

6. **PATCH 操作のサポート**:
   - SCIM PATCH (RFC 7644 §3.5.2) をサポートする。
   - 特に、`active` のトグル（`/Users/{id}` に対する PATCH）および `members` の追加・削除（`/Groups/{id}` に対する PATCH）を適切にハンドリングする。

## 却下した代替案

- **SCIM の `DELETE /Users/{id}` で即時完全削除 (Purge) する**:
  - 却下。外部の同期システムが何らかのエラーや設定変更で大量のユーザーを DELETE した場合に、PII の復旧が不可能になるリスクが極めて高い。既存の ADR-072 が備える誤操作救済措置を無効化してしまうため却下した。
- **グローバルな共有トークンで SCIM サーバーを動かす**:
  - 却下。テナント分離の原則 (ADR-032 / ADR-034) に反する。必ず各テナントごとにスコープされた Bearer Token を発行する。

## 影響

- 新たに SCIM 2.0 連携のための `spec/contexts/scim.yaml` が追加される。
- Go の `/scim/v2` エンドポイントがテナント単位のルーティングに追加され、Bearer Token 認証フィルターが適用される。
- UI にて管理者画面で SCIM 2.0 接続情報（Endpoint URL、Token の生成とローテーション、直近の同期時刻等）の設定および表示が可能になる。
- ユーザー/グループ詳細画面にて、SCIM 由来のオブジェクトについては "SCIM同期元" の表記を追加し、直接編集不可にするための属性制御を行う。
