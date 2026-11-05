---
status: completed
---

# 受信した SET の主体を、最上位の `sub_id` から解決する

最上位の `sub_id` から主体を解決し、イベント内の `subject` を省いた署名済み SET を受理するようにした。
REQ-SHAREDSIGNALS-010 に受理と拒否の規則を追加し、SSF10-RECEIVE-SUB-ID と二つの具体例を追加した。
従来の RFC 9493 形式と IdMagic 独自形式を維持し、両方の表現がある場合は形式と発行者を共通に検査して解決後に照合する。
Agent と束縛先クライアントの別名は同じ主体として受理し、不正な形式、`null`、発行者や主体の不一致は失効を反映せず拒否する。

HTTP 入口での RED と GREEN、主体の比較、User の失効、既存形式の回帰を確認した。
`check-spec`、サイト生成、`verify-changed -- main`、DB を使う検査が成功した。
逆依存を含む Go 試験は 3,068 件成功し、外部環境を必要とする既存の OpenLDAP 試験 1 件は未検証である。
送信側の SSF 1.0 適合と `complex` など未対応の形式は対象外とした。
