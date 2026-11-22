# アプリケーションのプロビジョニング

この文書は、アプリケーションのプロビジョニングの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Provisioning](../README.md) / [ページ一覧](README.md) / アプリケーションのプロビジョニング

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [プロビジョニング](../../../design/application/pages/admin/provisioning/README.md)

`admin-application-provisioning`　`/admin/applications/$applicationId/provisioning`

接続先の同期設定、状態と処理タスクを管理する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Provisioning](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | アプリケーション詳細 |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 未接続時の登録、接続設定、資格情報と属性対応、接続テスト、停止状態からの再開、個別実行と全件再同期、状態によるタスク一覧、詳細と再試行、接続解除の確認 |

利用手順は[Provisioning のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| save | ready | 接続または設定を保存 | 常時 | IdMagic.Provisioning.Operations.RegisterProvisioningConnection、IdMagic.Provisioning.Operations.UpdateProvisioningConnection | 接続状態と設定を更新する | [アプリケーションのプロビジョニング](admin-application-provisioning.md) / ready | 対象の識別子と操作の入力値 |
| test | ready | 接続をテストまたは再開 | 常時 | IdMagic.Provisioning.Operations.TestProvisioningConnection、IdMagic.Provisioning.Operations.ResumeProvisioningConnection | テスト結果または接続の状態を表示する | [アプリケーションのプロビジョニング](admin-application-provisioning.md) / ready | 対象の識別子と操作の入力値 |
| run | ready | 個別実行または全件再同期 | 常時 | IdMagic.Provisioning.Operations.ProvisionOnDemand、IdMagic.Provisioning.Operations.StartFullResync | 実行結果と開始したジョブを表示する | [アプリケーションのプロビジョニング](admin-application-provisioning.md) / ready | 対象の識別子と操作の入力値 |
| tasks | ready | タスクを取得または再試行 | 常時 | IdMagic.Provisioning.Operations.ListProvisioningTasks、IdMagic.Provisioning.Operations.GetProvisioningTask、IdMagic.Provisioning.Operations.RetryProvisioningTask | 対象タスクの詳細と状態を更新する | [アプリケーションのプロビジョニング](admin-application-provisioning.md) / ready | 対象の識別子と操作の入力値 |
| disconnect | ready | 接続解除を確認 | 常時 | IdMagic.Provisioning.Operations.DeleteProvisioningConnection | 接続情報を削除して未接続の表示に戻す | [アプリケーションのプロビジョニング](admin-application-provisioning.md) / ready | 対象の識別子と操作の入力値 |
| back | ready | 戻る | 常時 | UI 内の操作 | 移動先を表示する | [アプリケーション詳細](../../application/pages/admin-application-detail.md) / ready | なし |

## 表示状態

| 状態 | 表示 |
| --- | --- |
| ready | 取得した内容を表示する。 |
| loading | ページ移動中は直前の表示を保つ。初回の取得はルーターが扱う。 |
| error | 表示できなければ共通エラーを表示し、操作失敗は操作の近くに残す。 |
| forbidden | 権限不足を表示し、未反映の状態を保つ。 |
| expired | 認証が必要なページでは再認可し、復帰できなければログインへ移る。 |
| success | 成功した操作を一度通知し、表示内容を再取得する。 |
| invalid | 入力不備の表示は共通フォーム規則に従う。項目ごとの規則は別に定義する。 |
| submitting | 実行中の操作を示し、二重送信を防ぐ。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/provisioning/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Provisioningのページ一覧へ戻る](README.md)
