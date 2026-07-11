---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: []
---

# テナントロゴをアップロード後に hosted UI で確実に表示する

## Motivation
ロゴのアップロードが成功しても利用者向け hosted UI に表示されない報告がある。設定保存の成功と表示可能性が乖離すると、テナント管理者は失敗を検知も復旧もできず、wi-89 の主要な利用者価値を満たせない。

## Scope
- `spec/contexts/tenancy.yaml` の `interfaces.GetTenantBrandingAsset`、`invariants.TenantBrandingUploadedAssetIsDisplayable`、および hosted UI への反映 `scenarios`。
- branding asset URL の生成・tenant route 解決・配信・ブラウザでの描画経路の再現と修正。
- 管理画面プレビューと login / consent / account portal に対する回帰テスト。

## Out of Scope
- 画像形式・サイズ上限・任意 SVG の受理方針変更。
- 管理コンソール全体へのテナントテーマ適用。

## Plan
- まず realm path、reverse proxy base path、memory / PostgreSQL の各経路で、アップロード成功後に返る `logo_url` が画像取得・描画可能かを再現する。
- 原因を URL 解決、asset 保存、tenant 解決、キャッシュ、UI の安全化のいずれかへ限定して修正する。成功応答だけでなく実際の画像取得と表示を E2E または同等の統合テストで保証する。

## Tasks
- [x] T001 [SCL] ロゴアップロード後の公開・表示 scenario と失敗時の観測可能な振る舞いを更新する。
- [x] T002 [HTTP/Persistence] 画像 URL の生成・配信・tenant scope を修正する。
- [x] T003 [UI] 管理プレビューと hosted UI のロゴ表示を修正する。
- [x] T004 [Verify] realm / base path を含むアップロードから表示までの回帰テストを追加する。

## Verification
- `just scl-render`
- `just test-go`
- `just test-ui-e2e`
- 手動: ロゴをアップロードし、管理画面プレビュー、login、consent、account portal で表示されることを確認する。

## Risk Notes
公開アセット URL の修正は tenant 境界を越えた取得やキャッシュ混同を招き得る。tenant ID / realm ごとの asset lookup と `nosniff` を回帰テストで維持する。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  realm ごとの tenant branding asset URL を Caddy / Vite が backend へ転送するようにし、
  管理プレビュー、login、consent、account portal が同一 origin の検証済みロゴだけを表示するようにした。
  併せて、E2E fixture の backend 起動先、UUID 化済み demo client、画面遷移・API filter 名を現行実装へ同期し、
  ポートを共有する E2E spec をプロセス単位で直列化した。
- **Affected Guarantees State**:
  ロゴアップロード成功時の `logo_url` は、同じ realm 内の公開 asset endpoint から取得でき、
  tenant 分離と `nosniff` を保ったまま hosted UI と管理プレビューで描画可能である。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just verify-ui` — passed
  - `just test-go` — passed
  - `just test-ui-e2e` — passed
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
