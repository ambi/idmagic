---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# ログアウトとセッション管理が宣言する標準 9 行にテストを対応付け、標準の被覆台帳から外す

`mise run spec-diff` は `no normative specification change against main` を返す。規範の差分は無い。
変わったのは、ログアウトとセッション管理が宣言する 9 行に対する観測の有無である。
`tools/check/standards-coverage-debt.json` の `untested` は 43 件から 38 件へ減り、本項目が持つ 9 件は
1 件残らず消えた。テストが名指す ID は 249 件になった。
`OIDC-FRONTCHANNEL-IFRAME` は前提 work item [[wi-257-oidc-front-back-channel-logout-notifications]] が
実装と一緒にテストを置いたため、本項目では確認だけで済んだ。
残る 5 行のうち、`OIDC-LOGOUT-ID-TOKEN-HINT` は「ヒントから対象を解決する」ことを固定するテストが
無かった。既存の受理テストは Cookie とヒントが同じセッションを指すため、ヒントを読まない実装でも通る。
Cookie とヒントが別のセッションを指し、`client_id` パラメーターを付けない要求を新設して区別できるようにした。
`OIDC-FRONTCHANNEL-BEST-EFFORT` と `OIDC-BACKCHANNEL-DELIVERY-RETRY` はどちらも「配信結果に依存させない」
ことを言っているので、到達し得ない front-channel 宛先と、試行を使い切って失敗する back-channel 宛先の
それぞれで、LoginSession と同じ `sid` の RefreshTokenRecord が失効していることを読む形にした。
`OIDC-BACKCHANNEL-LOGOUT-TOKEN` は claim の列挙と署名の 2 つを言っているので、署名器の単体テストと、
RP が実際に受け取ったトークンを OP の鍵で検証する e2e の 2 つを対応付けた。
`OIDC-BACKCHANNEL-REPLAY` は `jti` の一意性と、再試行が `jti` を作り直さないことの 2 つを対応付けた。
製品コードは変えていない。増えたのは e2e fixture のローカル失効の観測、配信ワーカーと通知待ちの
ヘルパー、テスト 3 件、注記 6 件である。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
