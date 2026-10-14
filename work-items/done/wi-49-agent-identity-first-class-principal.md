---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# AI エージェントを第一級の非人間プリンシパルとして導入する

ADR-048 に基づき、Agent を User / OAuth2Client と別の第一級プリンシパルとして
spec-first で end-to-end 実装した。SCL に Agent 系 model / enum (AgentStatus =
Active/Disabled/Killed, AgentKind)、9 イベント、AgentLifecycle 状態機械、10 admin
interface、AdminAgentsManage 権限、vocabulary を追加し、Authentication component の
owns_* と Go の policy action / interface マッピングを更新した。Go は domain 型・
検証・repository port・Postgres / in-memory adapter・migration 0013・admin CRUD
use case / HTTP ハンドラ (register / update / disable・enable / kill / delete /
credential bind・unbind) を実装。トークン発行経路 (client_credentials) に、束縛
client から Agent を解決し !IsActive() なら invalid_client で拒否する fail-closed
ゲートを入れ、束縛時は agent_id / principal_type=agent クレームを付与する。管理 UI
(一覧・登録・所有者表示・無効化 / kill / 削除・資格情報束縛) は groups を踏襲して追加した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
