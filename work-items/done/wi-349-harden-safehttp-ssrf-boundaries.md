---
status: completed
archived_from: e92a87008bf012ccdf6ba015e091f38e7fb80c82
---

# 外部 client metadata の fetch を検査済み公開 IP への直接接続に限定する

`safehttp` の outbound request を DNS 検査済み IP への直接接続に限定し、環境 proxy による
SSRF 境界の迂回を防いだ。CGNAT 判定を `100.64.0.0/10` 全体の CIDR membership に修正し、
SCL scenario、OAuth2 Architecture、回帰テスト、SCL 派生物を同期した。CodeQL alert 14 は
custom transport を認識しない `go/request-forgery` の false positive として既に dismiss 済みだった。

全文は `git show e92a87008bf012ccdf6ba015e091f38e7fb80c82:<このファイルのパス>` で読む。
