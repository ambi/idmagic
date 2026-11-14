---
status: completed
---

# データエクスポートの結果のイベントを、エクスポートが実際に至った結果に対してだけ発行する

REQ-IDMANAGEMENT-086 と DataExportLifecycle を変更し、`succeeded` または最終的な `failed` の保存後だけ、対応する結果イベントを発行する。
Jobs のハンドラー登録に確定後の通知を含め、IdManagement が保存済みの結果をイベントへ変換する。
再試行は既存の `JobFailed` と `JobRetried` で記録し、取り消し、リース喪失、保存失敗では結果イベントを発行しないため、従来より結果イベントの件数が減る。
Jobs の再試行方針とイベントの payload は変更しない。
保存後の停止による監査記録の欠落は既存の許容範囲を維持する。
