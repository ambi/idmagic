---
status: completed
archived_from: 37f68ef9c7d0e492c8e921c70b57b4a9cea43125
---

# 日本語と英語に対応した I18n 基盤と文言管理を導入する

ja / en に限定した型安全な辞書・ロケール解決・保存済みの明示選択を導入し、hosted auth、account、admin の chrome に言語切替を追加した。
hosted auth の主要画面と共有ドメインラベルを辞書へ移し、OIDC `ui_locales` hint、未対応 locale の ja fallback、英語固定の backend API error 境界を仕様と実装で整合させた。

全文は `git show 37f68ef9c7d0e492c8e921c70b57b4a9cea43125:<このファイルのパス>` で読む。
