---
status: completed
archived_from: dee5fde99fc516dd425980d3aac18c28d4e8e63b
---

# IdMagic の runtime route と SCL/OpenAPI 契約を完全に一致させる

runtime-only 14 operation と誤った admin consent 3 path を SCL の models、interfaces、
access、scenarios へ正規化し、SCIM Users/Groups collection GET と SAML 設定更新、
operational probe を含む生成 OpenAPI を runtime route と一致させた。同一 path の異なる
method は一つの path item に保持し、同一 method/path の重複所有は生成時に拒否する。
assembled Echo router と生成 OpenAPI を realm prefix と path parameter 名だけ正規化して
双方向比較する contract test を追加し、手書き allowlist なしで差分 0 件を固定した。

全文は `git show dee5fde99fc516dd425980d3aac18c28d4e8e63b:<このファイルのパス>` で読む。
