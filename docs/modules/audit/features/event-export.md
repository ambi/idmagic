# 監査エクスポート

## 概要

この文書は、監査の検索結果を非同期に生成し、期限付きの JSON 成果物として取得する仕様である。
検索の文法は[監査イベントの検索](event-search.md)に従う。

## モデル

ジョブの種類は `audit_export`、実行レーンは `bulk` に固定する。
成果物は `{"events":[...]}` の JSON とし、各要素は監査イベント一覧と同じ表現にする。
要求者、要求時刻、経路による範囲、正規化した検索条件、形式と制限値をジョブへ固定する。
要求時刻より後に発生したイベントを含めない。
ページをまたぐ同時点スナップショットは保証しない。
履歴の一覧は、所属テナントの直近 200 ジョブから要求者と経路の範囲が一致するものを最大 20 件返す。

| 設定 | デフォルト | 意味 |
| --- | --- | --- |
| `AUDIT_EXPORT_MAX_EVENTS` | 1,000,000 | 成果物一件の最大イベント数 |
| `AUDIT_EXPORT_MAX_BYTES` | 268,435,456 | JSON 全体の最大バイト数 |
| `AUDIT_EXPORT_TTL` | `24h` | 完成からダウンロード可能な期間 |

設定値は正の数または正の期間に限り、ジョブを作成した時点の値を以後の実行にも使う。
読出しは 500 件のページを上限とし、一時ファイルと 64 KiB 以下の永続チャンクを使う。
成功した Job と完成した成果物の組だけを公開し、メモリへの全イベントの蓄積を避ける。

## 状態遷移

### ExportLifecycle

| State | Kind | Meaning |
| --- | --- | --- |
| queued | initial | 実行待ち |
| running | — | 生成中 |
| succeeded | — | 完成し期限内 |
| failed | terminal | 生成を失敗として確定 |
| canceled | terminal | 未完了の要求を取消し |
| expired | terminal | 成功した成果物の有効期限に到達 |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| queued | Started | — | running | 開始を監査へ記録 |
| queued | Canceled | — | canceled | 取消しを監査へ記録 |
| running | Succeeded | — | succeeded | 成果物を公開 |
| running | Failed | — | failed | 部分成果物を非公開のまま破棄 |
| running | Canceled | — | canceled | 部分成果物を非公開のまま破棄 |
| succeeded | Expired | 現在時刻が expires_at 以上 | expired | ダウンロードを拒否 |

| State | 状態の取得 | 生成 | 取消し | ダウンロード | 期限到達 |
| --- | --- | --- | --- | --- | --- |
| queued | 何もしない | → running | → canceled | 拒否：409 audit_export_unavailable | 何もしない |
| running | 何もしない | → succeeded（生成成功）<br>→ failed（生成失敗） | → canceled | 拒否：409 audit_export_unavailable | 何もしない |
| succeeded | 何もしない | 何もしない | 拒否：409 job_not_cancelable | 何もしない | → expired |
| failed | 何もしない | 何もしない | 拒否：409 job_not_cancelable | 拒否：409 audit_export_unavailable | 何もしない |
| canceled | 何もしない | 何もしない | 拒否：409 job_not_cancelable | 拒否：409 audit_export_unavailable | 何もしない |
| expired | 何もしない | 何もしない | 拒否：409 job_not_cancelable | 拒否：409 audit_export_unavailable | 何もしない |

## 操作

### 管理者によるエクスポートの作成

#### REQ-AUDIT-009 エクスポートの作成は条件を固定し bulk のジョブを返す

- 管理者がエクスポートを作成したとき、Audit は、経路のテナント範囲と正規化した条件を保存し、監査レコードの読出しを行わず、`bulk` の `audit_export` ジョブを 202 で返し、`AuditExportRequested` を発行する。
- システムの経路で作成したとき、Audit は、現在の制御面主体を求め、全テナントを範囲とし、ジョブと成果物を制御面テナントへ所属させる。
- テナント管理の経路で作成したとき、Audit は、認証したテナントの監査読出し権限を求め、範囲を所属テナントに固定する。
- 不正な検索条件を指定された場合、Audit は、400 と `invalid_request` で拒否し、ジョブを作らない。
- `active_jobs` の上限を超える場合、Audit は、429 と `active_job_quota_exceeded` で拒否し、ジョブを作らない。

### ワーカーによる生成

#### REQ-AUDIT-010 生成は全ページを読み上限超過と取消しで部分成果物を公開しない

- ワーカーがジョブを実行したとき、Audit は、範囲と条件に束縛した署名付きカーソルで全ページを発生時刻と ID の降順に読み、JSON を逐次書き込む。
- 全イベントが制限値以内の間、生成を完了したとき、Audit は、件数、バイト数、SHA-256 と有効期限を持つ不変の成果物を確定する。
- 最大イベント数を超える場合、Audit は、`event_limit_exceeded` でジョブを失敗させ、部分成果物を公開しない。
- JSON 全体が最大バイト数を超える場合、Audit は、`byte_limit_exceeded` でジョブを失敗させ、部分成果物を公開しない。
- 生成または保存が失敗した場合、Audit は、`audit_export_failed` でジョブを失敗させ、例外の本文と部分成果物を公開しない。
- ジョブを取り消された場合、Audit は、次のページ境界または確定前に生成を中断し、成果物をダウンロード可能にしない。
- 同じジョブを再配送されたとき、Audit は、確定済みの同じ Job ID の成果物を再利用する。

### 要求者による状態の確認と取得

#### REQ-AUDIT-011 状態と成果物の取得は現在の権限と所有者を確認する

- 要求者が状態、一覧、取消しまたはダウンロードを要求したとき、Audit は、要求時と同じ経路の範囲と現在の権限を検査する。
- 別の要求者、別テナントまたは異なる範囲の ID を指定された場合、Audit は、404 と `audit_export_not_found` で拒否し、状態の変更と成果物の返却を行わない。
- 成功かつ期限内の成果物をダウンロードされたとき、Audit は、`Content-Disposition: attachment` と `Cache-Control: no-store` の JSON を返し、`AuditExportDownloaded` を発行する。
- 未完了、失敗、取消しまたは期限切れの成果物を要求された場合、Audit は、409 と `audit_export_unavailable` で拒否し、成果物を返さない。
- 未完了のジョブを取り消されたとき、Audit は、ジョブを canceled にし、active_jobs の枠を解放する。
- 終端状態のジョブを取り消された場合、Audit は、409 と `job_not_cancelable` で拒否し、状態を変えない。
- Jobs が開始、成功、失敗または取消しを確定したとき、Audit は、対応する `AuditExportStarted`、`AuditExportSucceeded`、`AuditExportFailed` または `AuditExportCanceled` を発行する。
- 保持期限バッチが期限切れの成果物を削除したとき、Audit は、`AuditExportExpired` を発行し、以後の保存バイト数の計測から除く。
- メトリクスを収集したとき、Audit は、処理の段階と失敗、上限超過、成果物の保存バイト数を公開し、要求者、条件とテナントをラベルへ載せない。

### 同期 API の互換利用

#### REQ-AUDIT-012 同期 API は有界な互換経路として非推奨を通知する

- 同期 API を要求されたとき、Audit は、最大 10,000 件の JSON を返し、2026-10-25 の `Deprecation` と 2027-10-25 の `Sunset` を付ける。
- 同期の生成が API プロセス全体で 2 件に達している場合、Audit は、429 と `audit_export_busy` で拒否し、監査の読出しを行わない。
- 同期 JSON が 16 MiB を超える場合、Audit は、422 と `audit_export_limit_exceeded` で拒否し、部分 JSON を返さない。
- **判断**：削除は Sunset と初回公開から 12 か月の両方を満たした後に別の変更で行う。非同期 API を先に提供し、同期の実効 1,000 件への切詰めを修正する。

## セキュリティ上の考慮

横断の操作は現在の制御面主体に限定し、保存した過去の権限を再利用しない。
ジョブの params と生成用カーソルを公開しない。
作成と取消しはブラウザー要求の検証を行い、API アクセストークンには `audit:read` を求める。
監査の通常の配信失敗は REQ-AUDIT-008 に従う。
