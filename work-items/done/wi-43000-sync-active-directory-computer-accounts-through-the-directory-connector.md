---
status: completed
---

# Directory Connector で Active Directory のコンピューターアカウントを取り込む

Directory Connector の取り込みに AD のコンピューターアカウントを加えた（REQ-SOURCING-020、REQ-DIRECTORYCONNECTOR-007）。Connector は任意の `computers` の節（`active_directory` だけ、既定のフィルターは `(objectCategory=computer)`）があれば、User を送った後に同じ実行で `SendDirectoryComputerBatch`（`/ingestion-runs/{run_id}/computer-batches`）へ 100 件ずつ送り、完了の項目数は両方の和で照合する（REQ-SOURCING-017〜019、REQ-DIRECTORYCONNECTOR-002、003、006 を改めた）。
`objectSid` は MS-DTYP の二進表現から `S-1-...` に、ドメインは識別名の `DC=` から小文字の DNS 名に写す。差分の位置にコンピューターの範囲を含め、範囲が変われば full にする。
コンピューターは IdManagement の主体にせず、Sourcing が `directory_computers` に AD の写しとして持つ（境界の判断で、ポートと書き込みの契約を増やさない案を採った）。テナントの有効な記録の間で SID と小文字の主体名を部分一意索引で一意に保ち、同じ Connector の作り直しは古い記録を無効にし、別の Connector の持ち主には `computer_conflict` で失敗させる。イベントは発行せず、full の完了で現れなかったコンピューターを無効にして `disabled_computers` に数える。
WsFederation が引く公開の読み取りの操作は、利用側のない操作を先に公開しないため wi-98213 に残した。管理者向けの一覧 API は作らない。
変更耐性では、生き残った変異のうち SID の長さの境界はテストの不足として直した。`FormatSID` の `< 8` → `<= 8` は副権限の数の検査が同じ値を拒否するので等価、エントリの読み取りの非 `IncompleteEntryError` の分岐は写しの関数が返さないので届かない。CLI の確認のコンピューターの出力は未被覆で、OpenLDAP の fixture では AD のコンピューターを作れない。手の故障注入では、別の Connector の持ち主の上書き、完了での無効化の省略、PostgreSQL の主体名の正規化の省略、PostgreSQL 構成の組み立ての欠落をすべて検出した（最後のものは最初は検出できず、両構成の組み立てを確かめるテストを足した）。実物の AD での確認は wi-79 に残る。
