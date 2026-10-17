---
status: completed
---

# プロトコル設定とアプリケーションの結び付けをテーブルの所有者へ移す

規範仕様の差分はない。
OAuth2、Saml、WsFederation が application_id の条件付き更新を所有し、起動の組み立てが同じ Tx へ参加する操作を Application に渡す。
元のテナントと未結合の条件、Application 側の再読、確定と巻き戻しの範囲を保ち、台帳の3件を除いた。
三つのプロトコルの正常系、参照先がない作成、所有者の更新直後の故障による両側のロールバックを PostgreSQL で検証する。
Application の変更行の5変異はすべて検出し、生存した変異はない。
CSV の残り4件は、クォータと発行を省く再評価の経路をそのまま公開しないため wi-55044 を前提として wi-88544 に残した。

全文は `git log -1 -- work-items/active/wi-55774-move-protocol-application-links-to-table-owners.md` が示すコミットの親、またはそのコミットで読む。
