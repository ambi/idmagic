---
status: completed
---

# RC4 の Kerberos のサービスチケットを、PAC の解析の失敗ではなく明示の規則で扱う

REQ-AUTHENTICATION-042 に二つの規則を加えた。
AES でないチケットとセッションの鍵は、復号の前に 401 で拒否し、`NegotiateAuthRejected` の `reason` を `encryption` にする。
AES の鍵のない keytab では、起動を失敗させる。
RC4 を許す設定は設けない。
RFC 8429 が RC4 を非推奨とし、Microsoft の KDC も CVE-2026-20833 の更新で既定を AES に限ったためである。OpenJDK と MIT krb5 も RC4 を既定で無効にしている。
`TestVerifierAgainstSambaAD` の失敗の原因は、Samba AD が RC4 で出したチケットの PAC の署名（HMAC-MD5、-138）を gokrb5 v8.6.0 が符号なしで読んで拒否することだった。
復号の前の判定でこの欠陥へ到達しなくなったので、gokrb5 は直していない。
検査用の Samba AD には二つのサービスアカウントを置く。AES に限った `svc-sso` と、暗号方式を設定しない（RC4 でチケットを出す）`svc-legacy` である。
`test-kerberos-ad` を CI の Go のジョブへ加えた。
手順書には、既存のサービスアカウントを AES に限って keytab を出し直す手順と、`encryption` の拒否の切り分けを書いた。
残るリスクが二つある。実物の Windows の AD と Windows のブラウザーでは確認していない。また、AD 以外の KDC が出す PAC のない RC4 のチケットは、受理から拒否へ変わる。
