# wi-595-declare-uniqueness-conflict-responses

管理 API の利用者、グループ、エージェント、ワークロード・トラストバンドル、ライフサイクルワークフローについて、名前または発行者の重複で返る 409 Problem Details を API 契約へ追加した。

生成クライアントは重複を未知の応答として扱わず、操作ごとに宣言された競合エラー型として処理できる。
対象には [IdMagic.WorkloadIdentity.Operations.RegisterWorkloadTrustBundle](../../../spec/contexts/workloadidentity/main.tsp) を含む。
