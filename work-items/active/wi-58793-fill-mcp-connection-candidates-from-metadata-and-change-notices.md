---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-11-03
priority: p2
depends_on: [wi-95247-discover-mcp-tools-and-automate-connection-setup]
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/resource-access.md, requirement: REQ-OAUTH2-086 }
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-087 }
  - { path: docs/modules/mcp-gateway/features/runtime.md, requirement: REQ-MCPGATEWAY-011 }
---

# 接続先のメタデータと変更通知から、接続準備の候補と状態を補う

## 動機

[wi-95247](../done/wi-95247-discover-mcp-tools-and-automate-connection-setup.md) は、Gateway の起動時の報告でツールの候補を取り込み、評価用クライアントの作成と登録を一つの画面にまとめた。
次の候補は、取得の経路や信頼の扱いが異なるため、同じ受け入れ単位に含めなかった。

## 対象範囲

- サーバー自身がトークンを検査する構成で、Protected Resource Metadata の `scopes_supported` と認可サーバーの情報を取得し、出所を付けてスコープの候補に示す。
- `notifications/tools/list_changed` を受けた Gateway が再報告し、管理画面で追加、削除、定義変更の差分を示す。
- Gateway のない構成で、IdMagic が接続 URL から一覧を取得する経路の採否を決める。採る場合は宛先の制限、リダイレクト、名前解決、取得量、期限、資格情報の保管を定める。
- 業務ツールを実行せずに、認証、一覧取得、登録、割当とトークンの上限を確認する接続確認と、不足する設定の案内を提供する。

## 対象外

- 報告や説明文から、権限や承認の要否を推測すること。
- 外部データの対象カタログ。

## 設計

着手時に、各候補を独立して受け入れられるかで分け、必要なら更に項目を分ける。
IdMagic の外向き取得を採る場合は、SSRF と資格情報の送信先の検査を先に設計し、境界の比較を記録する。

## 計画

今回は起票だけを行う。

## タスク

- [ ] T001 [Design] 候補ごとの採否、取得主体、信頼の扱いを決める。
- [ ] T002 [Spec] 採る候補の要件と契約を更新する。
- [ ] T003 [App] 採る候補を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

着手時に、候補ごとの主要ユースケースと検証タスクを決める。

## リスク

外部の申告を信頼できる事実として扱うと、意図しない権限の公開につながる。
取得した値は候補として示し、公開と割当は管理者の選択に限る。
