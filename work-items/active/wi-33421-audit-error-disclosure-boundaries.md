---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-01
priority: p1
group: spec-conformance
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/system/features/http-api.md, requirement: REQ-SYSTEM-023 }
---

# 個別のエラー写像を監査し、内部障害の詳細を公開しない

## 動機

[組み込みクライアントの保護](../done/wi-76421-protect-first-party-clients.md)の調査で、共通の内部障害の処理があっても、その手前で生のエラーを別のコードへ変換すると詳細が公開されることが分かった。
同じ種類の候補がほかの機能にも残っているため、今回の修正だけを全面適用の証拠としない。

## 対象範囲

- `backend/provisioning/internal/handlers_http/routes.go` の `writeError` は、未分類のエラーを `invalid_request` と生の `err.Error()` へ変換しているため、入力検証と依存先障害を区別する。
- `backend/authentication/internal/federation/handlers_http/routes.go` の作成と更新は、Repository の `Save` のエラーを一括で入力エラーへ変換しているため、検証と保存障害を区別する。
- `backend/oauth2/internal/token/usecases/userinfo.go` とプロトコルのエラー出力では、`server_error` の説明に入れた内部エラーを公開しない。
- 汎用 API とプロトコルの応答について、エラーを文字列へ変換する箇所を構造的に棚卸しする。
- 型または既知の値で公開可能な拒否を識別し、それ以外は各 API 区分の安全な内部障害の応答と、サーバー側の診断へ渡す。
- SQL、実行時エラー、ライブラリエラー、スタックトレース、ラップした障害を依存先から注入し、HTTP の最終応答と保存状態を検証する。

## 対象外

- エラー文に SQL らしい文字列があるかを調べて隠す実装。
- UI の文字列の置換だけで API の情報公開を防いだとする対応。
- 組み込みクライアントの削除禁止と、実環境のデータの復元。

## 設計

未知のエラーに業務コードを付けることと、業務上の拒否を識別することを区別する。
共通ハンドラーの単体テストだけでは、手前のエラー写像の迂回を検出できない。
棚卸しは監査候補を返し、分類の正しさは所有する操作の故障注入で検証する。
文字列化を一律に禁止する検査は、入力検証と既知の拒否まで巻き込むため、そのまま採用しない。

## 計画

上の候補の最小経路を再現し、API ガイドラインの公開範囲へ適合させる。
既存の正当な拒否の説明を維持しながら、未分類障害の分類と診断を検証する。

## タスク

- [ ] T001 [Acceptance] 各候補で内部障害の公開を再現する。
- [ ] T002 [App] 入力検証、業務上の拒否と内部障害を区別する。
- [ ] T003 [Verify] 各操作の最終応答、診断と保存状態を検証する。

## 検証

`mise run brief` で担当する操作と規範を解決し、操作ごとに `primary_use_cases` を計画してから着手する。
検査の入口は `test-go-one`、`test-go-package`、`verify-changed` を選ぶ。

## リスク

入力検証まで一括で 500 に変えると、利用者が修正できる内容が分からなくなる。
正当な拒否と未知の障害の両方を検証する。
