---
status: completed
---

# XML フェデレーションの鍵のローテーションで、Entra のサインインを止めない

共有 XML 鍵の切り替えを、Entra の設定有無によらず「準備して公開、全連携先への登録を確認、有効化」の二段階とした（REQ-SIGNINGKEYS-001、006）。
準備中は署名鍵を保ち、確認した kid の有効化で旧鍵の 7 日の猶予を始める。再実行は鍵、イベント、猶予を増やさない。管理 API に操作と鍵集合の選択を加え、Entra の構成結果と Graph コマンドに準備済み証明書を渡す（REQ-WSFEDERATION-001）。

Local、Database、Vault を同じ遷移に揃え、Vault の公開状態と選択を PostgreSQL に保存する。実際の起動時の配線を含め、再起動後も Vault の最新バージョンを無条件に署名へ使わない（REQ-SIGNINGKEYS-007）。
JWT と SAML 専用スコープの即時切り替えは維持する。自動登録、自動の登録確認、実 Entra テナントでの疎通と定期バッチは対象外とした。

変更耐性の限界として、機械的変異には初回生成の猶予の変更 4 件と到達しないメモリの解析エラー 1 件が残り、DB のトランザクションを未解放にする 2 件は単独再実行でも時間超過した。時間超過は検出と数えず、永続化と配線は主要テストと意味上の故障注入で検査した。

全文は `git log -1 -- work-items/active/wi-50764-rotate-xml-federation-keys-without-breaking-entra-sign-in.md` が示すコミットの親、またはそのコミットで読む。
