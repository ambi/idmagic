# 監査イベント

この文書は、監査イベントの操作と状態を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Audit](../README.md) / [ページ一覧](README.md) / 監査イベント

分類：[管理コンソール](../../../design/application/pages/admin/README.md) / [監査](../../../design/application/pages/admin/audit/README.md)

`admin-audit-events`　`/admin/audit_events`

テナントの操作と関係者を検索して確認する

| 項目 | 内容 |
| --- | --- |
| 主担当 | [Audit](../README.md) |
| 利用者 | テナント管理者 |
| 入口 | 管理メニュー |
| 利用条件 | 対象テナントの管理権限で参照し、操作の認可は API が判断する。 |
| 表示内容 | 日時、種類、結果、主体と対象による検索。条件は URL に保持、イベントの詳細と関係者からの再検索、結果のページ送り、出力ジョブの開始、再取得、停止とダウンロード |

利用手順は[Audit のページ一覧](README.md)にまとめる。

個々の入力項目は実装の画面が持ち、受理範囲は API 契約と[共通の UI 設計](../../../design/application/user-interface.md)の検証の分担が定める。

## 操作と結果の詳細

| 操作 | 起点 | 契機 | 条件 | API | 結果 | 到達先 | データ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| search | ready | 検索・再取得・ページ送り | 常時 | IdMagic.Audit.Operations.ListAdminAuditEvents | 条件に合う監査イベントを表示する | [監査イベント](admin-audit-events.md) / ready | 対象の識別子と操作の入力値 |
| detail | ready | 詳細または関係者を選択 | 常時 | UI 内の操作 | イベント詳細またはその関係者を含む検索結果を表示する | [監査イベント](admin-audit-events.md) / ready | 対象の識別子と操作の入力値 |
| exportCsv | ready | 出力を開始・再取得・キャンセル | 常時 | IdMagic.Audit.Operations.StartAdminAuditExport、IdMagic.Audit.Operations.ListAdminAuditExports、IdMagic.Audit.Operations.CancelAdminAuditExport | 検索条件に対応する出力ジョブを更新する | [監査イベント](admin-audit-events.md) / ready | 対象の識別子と操作の入力値 |
| download | ready | 出力を取得 | 出力が完了し期限内 | IdMagic.Audit.Operations.DownloadAdminAuditExport | 監査イベントのファイルを取得する | ダウンロード / external | なし |

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
| empty | 対象がないことと、作成または検索条件を変える入口を示す。 |

## 関連する設計

- [TypeSpec の定義](../../../../spec/modules/audit/pages.tsp)
- [共通の UI 設計](../../../design/application/user-interface.md)
- [共通の UI 規則](../../../design/application/user-interface.md)

[Auditのページ一覧へ戻る](README.md)
