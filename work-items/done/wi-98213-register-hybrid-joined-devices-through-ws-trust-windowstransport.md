---
status: cancelled
---

# WS-Trust の windowstransport で、Hybrid Join の端末の登録に要るトークンを発行する

中止した。Microsoft は windowstransport を社内向けの入口にだけ置き、社外へ公開してはならない（MUST NOT）と求める（Microsoft Entra hybrid join の計画）。インターネット上の SaaS である IdMagic はこれに従えず、出口 IP による制限は社外への公開の代わりにならない。
同じ位置の Okta も windowstransport で端末を登録せず、Entra Connect がコンピューターを同期した後の登録（sync join）を前提に案内している。Windows 10 1803 以降は、フェデレーションによる即時の登録に失敗すると sync join で登録を終える（Configure Microsoft Entra hybrid join の Federation caveats）。
Hybrid Join は sync join で行う案内へ切り替え、その変更は wi-99128 が担う。実装の途中で、`dsregcmd` がインターネット上の STS へ Kerberos の資格情報を自動で送るかを一次資料で確認できないことも分かった。
wi-43000 で取り込むようにした AD のコンピューターアカウントは、この項目の発行だけが使う予定だったので、利用側がなくなった。
