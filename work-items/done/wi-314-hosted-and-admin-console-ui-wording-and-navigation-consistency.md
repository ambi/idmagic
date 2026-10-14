---
status: completed
archived_from: da06dbb83b2d01dc5129a8a76ffc9f79c6c69d30
---

# ホスト認証画面と管理コンソールの文言・レイアウト・参照編集分離を実機レビューの指摘に沿って改善する

ホスト認証画面と管理コンソールの文言・ナビゲーション・参照編集分離を T001-T022 で横断的に
改善し、実機レビュー第2弾の T023-T031 も反映した。第2弾では IdP ホーム導線、同意表示、
検索可能な共通ピッカー、ライフサイクルワークフローのアプリ内 dialog、グループ push の
入力補助、監査イベント名、ユーザー属性文言、ダッシュボード情報量を是正した。
Out of Scope としたグループ属性拡張と OIDC client secret lifecycle はそれぞれ
[[wi-315-group-contact-and-custom-attributes]] / [[wi-316-oidc-client-secret-lifecycle-management]]
に分割済み。GUI 属性マッピング、SAML/WS-Federation 署名鍵画面、デザインシステム刷新は
本 WI では実装していない。
test-first からの逸脱 (self-attest): 第2弾の T023/T024/T028-T031 は既存の部分実装を確認後に
回帰テストを補ったため、RED を先行確認していない。T025 の共通 picker accessibility、T026、
T027 は RED → GREEN を確認した。
Verification 節の手動確認は本セッションでは実施せず、同じ主要導線を E2E 22件と UI unit
516件で自動検証した。

全文は `git show da06dbb83b2d01dc5129a8a76ffc9f79c6c69d30:<このファイルのパス>` で読む。
