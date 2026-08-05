---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-080: SCIM 2.0 Inbound Provisioning の適用と削除ポリシーの統合

## コンテキスト

Okta、Google Workspace (Cloud Identity)、Microsoft Entra ID などの外部 IDP/ID 管理システムから `idmagic` に対してユーザーやグループのプロビジョニングを自動化するため、SCIM 2.0 (RFC 7643 / RFC 7644) プロトコルに基づく Inbound Provisioning サーバー機能が必要である。
手動の管理者 API による CRUD 操作だけでは、外部 IDP での組織変更や退職処理などをタイムリーかつ安全に同期することができない。
SCIM 2.0 サーバーを公開するにあたり、以下の設計課題を決定する必要がある：
1. `idmagic` が SCIM サーバーとして振る舞う範囲
2. SCIM の属性マッピングと `active` 属性のライフサイクルへのマッピング
3. SCIM `DELETE` リクエスト時の `idmagic` 内部の削除ポリシー（ADR-036 / ADR-072 の soft-delete/purge との整合性）
4. Bearer Token による認証とテナント分離の設計

## 決定

`idmagic` は SCIM 2.0 (RFC 7643/7644) サーバーとして inbound provisioning を受け付ける。テナントごとに
スコープされた Bearer Token で認証し（グローバル共有トークンは却下）、SCIM の `DELETE` は即時の完全削除
ではなく既存の soft-delete ポリシー（ADR-072）に統合する。メカニズムの詳細（エンドポイント一覧、属性
マッピング表、`active` とライフサイクルの対応、PATCH の扱い）は
[backend/sourcing/ARCHITECTURE.md](../backend/sourcing/ARCHITECTURE.md) に移した。

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
