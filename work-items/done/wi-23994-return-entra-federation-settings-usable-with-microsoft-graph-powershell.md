---
status: completed
---

# Entra の構成の結果を、Microsoft Graph PowerShell でそのまま使える形で返す

`spec-diff` が示す変更は REQ-WSFEDERATION-001 と `ConfigureEntraFederation` である。
構成の応答を Microsoft Graph PowerShell の設定名と有効な XML 署名証明書の base64 に改め、サインアウト先、`wsFed`、`acceptIfMfaDoneByFederatedIdp` と、値を埋めた作成と更新のコマンドを返す。
管理画面は二つのコマンドをコピーでき、制限はコードから辞書の文言へ対応付ける。
PowerShell の引用は通常と曲がった単一引用符を二重化し、純粋な計算のテストとファジングで文字列を保存する性質を検査する。

構文変異に生存と未被覆はなく、コンパイルできない 14 件は文字列連結への算術変異である。
証明書を説明文に取り違える故障は構文変異で表せないため、手動の注入で検査する。
実テナントでのコマンドの実行は wi-79、事前公開する次の鍵と `NextSigningCertificate` の接続は wi-50764、MFA の要求への対応は wi-84807 に残す。
wi-50764 はこの応答を使って次の証明書を登録するため、この項目への依存を追加した。

全文は `git log -1 -- work-items/active/wi-23994-return-entra-federation-settings-usable-with-microsoft-graph-powershell.md` が示すコミットの親、またはそのコミットで読む。
