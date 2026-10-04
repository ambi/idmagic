# 接続の管理

## 概要

この文書は、管理者がプロビジョニングの接続を登録し、試し、全体を同期し直す仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 接続の登録、更新、削除、接続テスト、On-Demand Provision、フル同期、資格情報のローテーション、隔離の解除、タスクの一覧と再試行 |
| 行為者 | テナント管理者、管理 API のクライアント |
| 扱わないもの | 周期ごとの同期は[同期](../synchronization/README.md)が、タスクの実行は[プロビジョニングタスクの実行](../task/README.md)が扱う |

## 操作

### 管理 API のクライアントによる操作

#### REQ-PROVISIONING-001 管理 API クライアントは Provisioning スコープの範囲でだけ接続とプロビジョニングタスクを操作できる

- `provisioning:read` のスコープの API アクセストークンで呼び出されたとき、Provisioning は、アプリケーションの接続とテナントの接続の一覧の参照だけを許可する。
- `provisioning:write` のスコープの API アクセストークンで呼び出されたとき、Provisioning は、接続の変更とプロビジョニングタスクの操作だけを許可する。
- `provisioning:write` を持たないトークンで接続の変更かプロビジョニングタスクの操作を要求された場合、Provisioning は、403 と `insufficient_scope` で拒否し、接続とタスクを変えない。
- 別のテナントで発行したトークンを提示された場合、Provisioning は、401 と `invalid_token` で拒否する。
- `admin` のロールを持たない利用者が接続かタスクの管理 API を要求した場合、Provisioning は、403 と `access_denied` で拒否する。
- **例**：EX-PROVISIONING-001-01、EX-PROVISIONING-001-02、EX-PROVISIONING-001-03、EX-PROVISIONING-001-04

### 管理者による接続の登録

#### REQ-PROVISIONING-002 管理者は接続を登録し、接続テストで下流の対応機能を取得できる

- 管理者がアプリケーションに接続を登録したとき、Provisioning は、接続を作り、201 と接続を返し、`ProvisioningConnectionRegistered` を発行する。
- 管理者が接続をテストしたとき、Provisioning は、下流の `/ServiceProviderConfig` への到達性を確かめ、対応機能を接続に保存し、200 と結果を返す。
- 管理者が接続を取得または更新したとき、Provisioning は、200 と接続を返す。
- 管理者が資格情報と `status` 以外の設定を指定して接続を更新したとき、Provisioning は、保存の後に `ProvisioningConnectionUpdated` を発行する。
- 管理者が `status` を `disabled` から `active` へ戻したとき、Provisioning は、保存の後に `ProvisioningConnectionUpdated` を発行する。
- 管理者が `status` を `disabled` にして接続を更新したとき、Provisioning は、保存の後に `ProvisioningConnectionDisabled` を発行する。
- 管理者が接続を削除したとき、Provisioning は、接続を消して 204 を返し、`ProvisioningConnectionDeleted` を発行する。
- 管理者がテナントの接続を一覧したとき、Provisioning は、テナントのすべての接続を 200 で返す。
- HTTPS でないか、内部のアドレスかリンクローカルのアドレスを指す下流の URL を指定された場合、Provisioning は、400 と `invalid_request` で拒否し、接続を作らない。
- アプリケーションにすでに接続がある場合、Provisioning は、登録を 409 と `provisioning_conflict` で拒否する。
- **例**：EX-PROVISIONING-002-01、EX-PROVISIONING-002-02、EX-PROVISIONING-002-03

#### REQ-PROVISIONING-015 他テナントの接続とプロビジョニングタスクはテナント境界を越えない

- 別のテナントか存在しない接続かタスクを指定された場合、Provisioning は、404 と `provisioning_not_found` で拒否する。
- **例**：EX-PROVISIONING-015-01

### 管理者による On-Demand Provision

#### REQ-PROVISIONING-012 管理者は On-Demand Provision で 1 人のユーザーを試験的にプロビジョニングできる

- 接続の適用範囲の中の主体を指定して On-Demand Provision を要求されたとき、Provisioning は、直ちに `pending` のプロビジョニングタスクを作り、201 とタスクを返す。
- 適用範囲の外の主体（`scope=assigned_only` で割り当てのない User）を指定された場合、Provisioning は、409 と `provisioning_conflict` で拒否し、タスクを作らない。
- **例**：EX-PROVISIONING-012-01、EX-PROVISIONING-012-02

### 管理者によるフル同期

#### REQ-PROVISIONING-013 管理者はフル同期で適用範囲の全対象を収束できる

- 管理者がフル同期を始めたとき、Provisioning は、適用範囲の中のすべての主体にプロビジョニングタスクを作り、200 と作った件数を返す。
- フル同期のすべてのタスクが終端の状態に達したとき、Provisioning は、`FullResyncCompleted` を一度だけ発行する。
- 適用範囲の中に主体がない間、管理者がフル同期を始めたとき、Provisioning は、直ちに `FullResyncCompleted` を発行する。
- **例**：EX-PROVISIONING-013-01

### 管理者による資格情報のローテーション

#### REQ-PROVISIONING-014 資格情報のローテーションは監査でき、`rotated_at` が更新される

- 管理者が新しい資格情報を指定して接続を更新したとき、Provisioning は、資格情報を置き換え、`rotated_at` を更新し、`ProvisioningCredentialRotated` を発行し、以後のタスクに新しい資格情報で下流へ認証させる。
- **例**：EX-PROVISIONING-014-01

## セキュリティ上の考慮

接続の登録、更新、削除、接続テスト、On-Demand Provision、フル同期、隔離の解除、タスクの一覧と再試行は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが、所属テナントに対して行える。
接続テスト、On-Demand Provision、フル同期、隔離の解除、タスクの再試行は、いずれも下流を変えるので、変更の操作に属する。

接続とタスクはテナントを越えない。
別のテナントの接続の ID やタスクの ID を指定した参照は、存在しないものとして拒否する。

下流の URL は、登録の時点で HTTPS であることと、内部のアドレスやリンクローカルのアドレスを指さないことを検証する。
