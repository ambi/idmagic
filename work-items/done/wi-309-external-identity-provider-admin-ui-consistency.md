---
status: completed
archived_from: 0500683499be50c11cc042b1d4814431d5d36788
---

# 外部IDプロバイダー管理画面を他リソースと同じ一覧/編集分離UIに揃え、下書き状態・設定テスト・シークレット参照の分かりにくさを解消する

Motivation で指摘した4点全てに対応した。(1) 一覧/詳細/編集/作成の別ルート分割と
`adminNav.ts` トップレベル項目化 ([[ADR-086]] /
[[wi-126-admin-and-account-ui-consistency-and-navigation-policy]] を Entra 連携と同じ形で
適用)。(2) Draft 廃止と `Active`/`Disabled` の2値化により未着手接続の即時削除を可能化
([[ADR-149]])。(3) 「設定テスト」を OIDC discovery/JWKS 到達性・SAML 証明書検証・secret
解決可能性の実処理に置き換え、構造化結果 (成功/失敗理由) を画面表示。(4) `secret_reference`
を `EnvelopeCrypto`/`DataKeys` による実値暗号化保存に切替え、UI をシークレット実値入力
(write-only) に変更 ([[ADR-150]])。
Out of Scope として明記済みの未対応: 「設定」タブに残る他項目の整理、`entra-federation`
との画面統合、authorization_code フロー全体のシミュレート、`EnvelopeCrypto`/`DataKeys`
自体の新機能追加 (いずれも Scope の Out of Scope 節に記載済み)。
test-first からの逸脱 (self-attest): T007 (secrets アダプタ) と T009 (UI) は、配線の
複雑さ・実装とテストの一体的な組み立てを理由に、事前の RED 確認を経ずに実装とテストを
ほぼ同時に書いた (Tasks の該当項目に理由を明記)。それ以外の層 (Domain/Usecase/Handlers) は
RED → GREEN を確認した。
手動検証 (Verification 節の (1)〜(5)) はブラウザでの実機確認を本セッションでは実施しておらず、
各項目に対応する自動テストで同等の挙動を検証した点をユーザーに開示する。

全文は `git show 0500683499be50c11cc042b1d4814431d5d36788:<このファイルのパス>` で読む。
