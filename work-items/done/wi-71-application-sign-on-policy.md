---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# アプリケーション単位のサインオンポリシー (条件付きアクセス / step-up) を導入する

ApplicationCatalog に AppSignOnPolicy / SignOnRule / RequiredAuthnLevel / AccessCondition を追加し、
tenant/application 単位で `application_sign_on_policies` に保存する管理 API と UI エディタを実装した。
既存の Application 割当ゲートを拡張し、OIDC / SAML / WS-Fed の federation 開始で同じ policy 評価器を通す。
OIDC は step-up 可能な不足を既存 TOTP 導線へ誘導し、SAML / WS-Fed は初期実装として fail-closed で拒否する。
評価点と所有関係は ADR-079 に記録した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
