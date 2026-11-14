---
status: completed
---

# PKCE をすべてのクライアントに求めるか、クライアントごとに定めるかを決めて揃える

PKCE 非対応の連携が多いという利用者の判断に従い、通常の confidential は PKCE を任意とする C 案を採用した。
public、認証方式 `none`、FAPI 2.0 は設定によらず必須とし、通常の confidential は登録または管理 API の `require_pkce=true` で必須化できる。

REQ-OAUTH2-005 の認可と交換、REQ-OAUTH2-016 と REQ-OAUTH2-035 の設定保存、REQ-OAUTH2-014 の広告、RFC9700-AUTHORIZATION-CODE の適用範囲と対応する TypeSpec を揃えた。
保存した challenge があるコードは必ず verifier を照合し、ないコードへの verifier の追加は拒否する。
Discovery の `require_pkce` は false とし、追加カラムは false をデフォルトにする。

管理 API による登録から非 PKCE のトークン取得、必須化と解除、public／FAPI の拒否、DB 保存を検証し、重要な必須条件とダウングレード拒否の故障をテストが検出することも確認した。
最終差分の検査、スキーマ収束、生成サイトを確認した。
通常検証には既存の Samba AD、Envoy、対象外 fuzz 入力の Skip があり、この範囲は未検証である。

設定 UI の追加と `plain` への対応は対象外とした。
PKCE を省く confidential OIDC 連携ではクライアント側の nonce 検証などのコード注入対策が必要であり、その実装の検証は今回の範囲に含めない。
