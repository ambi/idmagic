---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-03
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/contexts/sharedsignals/main.tsp, symbol: IdMagic.SharedSignals.Operations.ReceiveSecurityEvent }
---

# ゲートウェイの経路許可リストが実行時の経路表からずれていることを検出し、現在のずれを塞ぐ

## Motivation

ブラウザーから見える境界を同一オリジンに揃えるのはゲートウェイの役目である（`docs/deployment.md`）。そのゲートウェイは、どのパスを Go へ渡すかを**手書きの接頭辞列挙**で決めている。列挙は 3 か所にあり、いずれも実行時の経路表と照合されていない。

| 場所 | 形 |
| --- | --- |
| `frontend/Caddyfile` の `@backend` | パス接頭辞の列挙 |
| `frontend/Caddyfile` の `@realmBackend` | `^/realms/[^/]+/(api\|scim\|saml\|...)` の正規表現 |
| `frontend/vite.config.ts` の `server.proxy` | 上の 2 つとほぼ同じ内容を JavaScript で再掲 |

**3 つとも、実行時に登録されている次の経路を含まない。**

| 欠落している経路 | 何が壊れるか |
| --- | --- |
| `/ssf/streams/{stream_id}/events` | 顧客側の IdP が送るセキュリティイベントの受信。Shared Signals の受信経路はこれ 1 本しかないので、参照ゲートウェイの背後では受信機能そのものが届かない |
| `/session/check` | OIDC Session Management の `check_session_iframe`。RP が埋め込む iframe が SPA の `index.html` を受け取る |
| `/application-icons/{application_id}/{id}` | 管理コンソールが `support.TenantRoute` で組み立てるアプリケーションアイコンの URL |
| `/livez`、`/readyz`、`/startupz`、`/metrics` | ゲートウェイ経由での到達。Kubernetes は Pod へ直接当てるので本番の判定には影響しないが、外形監視は届かない |

**このずれは静かに失敗する。** ゲートウェイの `handle` は最後に SPA へ落ちるので、外れた経路は 404 ではなく `index.html` を 200 で返す。`Content-Type` は `text/html` になり、SET の受信も iframe も画像も、エラーではなく「意味のない成功」として観測される。

`frontend/src/devProxy.test.ts` に検査が 1 つあるが、Vite の開発プロキシだけを対象にしており、しかも 8 本の経路を手で列挙している。`Caddyfile` は何にも検査されていない。列挙を手で足す検査は、列挙を手で足す設定と同じ速度でずれる。

[[wi-459-api-process-plane-separation-decision]] は API の Deployment を種別ごとに分けない判断を記録し、その Out of Scope でこの照合を別の work item へ渡した。同じ work item は、**このずれが存在する限り「後から分けるのは安い」とはみなさない**とも書いている。分割は後段を複数の上流へ振り分ける変更なので、振り分けの正しさを確かめる手段がなければ移行の検証対象が読めない。ただし本 work item が塞ぐのは分割の準備ではなく、単一プロセスのままでも現に壊れている経路である。

## Scope

- 実行時の経路表とゲートウェイの許可リストを照合し、ずれを機械的に検出する検査を追加する。`mise run check-*` の 1 つとして常時走らせる。
- 検査の対象に `frontend/Caddyfile` の `@backend` と `@realmBackend`、`frontend/vite.config.ts` の `server.proxy` の 3 つすべてを含める。
- 現在のずれを塞ぐ。`/ssf/*`、`/session/check`、`/application-icons/*` と、運用経路の扱いを決める。
- 経路表の正をどこから取るかを決める。`backend/shared/spec/operations_gen.go` は TypeSpec から生成された 323 件の経路表を持ち、`mise run check-generated-contract` が実行時の経路メタデータと照合している。この既存の正を使えるかを Design で確かめる。
- ホスト形式（`/` 直下）とパス形式（`/realms/{tenant_id}/` 配下）の両方を照合する。`@realmBackend` の正規表現は `@backend` と別の列挙になっているので、片方だけの照合では足りない。

## Out of Scope

- ゲートウェイ設定そのものを必須のランタイムにすること。`docs/deployment.md` の「Caddy は参照用の設定であり、必須のランタイムではない」という判断は変えない。検査するのはリポジトリが同梱する参照設定である。
- 列挙を 1 か所へ統合し、Caddyfile と Vite の設定を生成物にすること。有力だが、生成に倒すかどうかは Design で判断する。判断が生成なら本 work item で実施し、そうでなければ照合だけを持つ。
- 経路ごとの認可、ヘッダー、キャッシュ方針の照合。本 work item は「Go へ届くか」だけを見る。
- 同一オリジンの前提が満たされているかの実行時検証。[[wi-426-same-origin-deployment-assumption-detection]] が持つ。
- API プロセスを種別ごとに分けること、およびそのときの振り分け設計。[[wi-459-api-process-plane-separation-decision]] が判断を持つ。

## Design

### 正をどこから取るか

`backend/shared/spec/operations_gen.go` は TypeSpec から生成され、`Method` と `Path` を持つ 323 件の表である。`mise run check-generated-contract` がこの表と実行時の経路メタデータを照合しているので、**「実行時に存在する経路」の正としては既にこれが使われている。** ゲートウェイの照合も同じ正から導けるなら、正が 2 つに増えない。

ただし 2 点を確かめる必要がある。第一に、`operations_gen.go` に現れない経路が実行時にあるかどうか。`/metrics` は TypeSpec に宣言があるが、Echo が内部で持つ経路や、ミドルウェアが応答する経路が漏れていないかを見る。第二に、`/realms/{tenant_id}` 接頭辞を持つ形は表の中でどう表現されているか。表は既定テナントの形だけを持ち、realm 形は登録側で導出している可能性がある。

### 照合の向き

2 方向のうち、**「実行時にあるがゲートウェイが通さない」だけを失敗にする。** 逆向き（ゲートウェイが通すが実行時にない）は、廃止した経路の掃除漏れであって、機能は壊れない。警告として出し、失敗にはしない。片方向にしておくのは、失敗の意味を「利用者に届かない機能がある」の 1 つに保つためである。

### 生成に倒すか、照合にとどめるか

| 案 | 利点 | 欠点 |
| --- | --- | --- |
| 照合だけを持つ | 小さい。既存の設定ファイルの形と読みやすさを変えない。参照設定であるという位置付けとも整合する | ずれは検出できるが、直すのは人である。3 か所を手で同期する作業は残る |
| 接頭辞の列挙を生成物にする | 3 か所のずれが構造的に起きなくなる | 生成物を追跡するか、生成を検査に含めるかを決める必要がある。`Caddyfile` は運用者が読んで理解する参照設定でもあるので、生成物にすると読みにくくなりうる |

**照合を先に入れる。** 現在のずれを塞ぐことと、再発を検出することが目的であり、それは照合で達成できる。生成は、照合が「毎回同じ 3 か所を直す」作業に落ちたときに改めて判断する。判断材料が出るまで、読みやすさを確実に失う変更はしない。

### 運用経路の扱い

`/livez`、`/readyz`、`/startupz`、`/metrics` は Kubernetes が Pod へ直接当てるので、ゲートウェイを通す必要は必ずしもない。`/metrics` は認証を持たないため、`docs/contexts/system/decisions.md` が「公開先は折り返しアドレス、管理用ネットワーク、認証付きプロキシの背後に限る」と定めている。**つまり `/metrics` はゲートウェイを通してはならない経路である。** 照合は「通すべき」と「通してはならない」を区別できる必要があり、単なる集合の差では表せない。分類は経路表の側に持たせるか、照合の設定として持つかを着手時に決める。

## Plan

1. 実行時の経路表の正を確定する。`operations_gen.go` で足りるか、realm 形と非宣言経路の扱いを確かめる。
2. 「通すべき」「通してはならない」「どちらでもよい」の分類をどこに置くかを決める。
3. 現在のずれの全量を出す。上の表は既知の 7 件だが、網羅したものではない。
4. 検査を書き、現在のずれで RED になることを確かめる。
5. 3 つの設定を直して GREEN にする。`/ssf/*` と `/session/check` は機能が届いていないので、経路が通ることを観測可能な境界で確かめる。
6. `mise run check` の依存へ加える。

## Tasks

- [ ] T001 [Research] 実行時経路の正を確定し、realm 形と非宣言経路の扱いを決める。
- [ ] T002 [Design] 「通すべき/通してはならない/どちらでもよい」の分類の置き場所を決める。
- [ ] T003 [Research] 3 つの設定と経路表のずれを全量出す。
- [ ] T004 [Acceptance] 現在のずれで失敗する照合検査を書き、RED を確かめる。
- [ ] T005 [Gateway] `frontend/Caddyfile` の `@backend` と `@realmBackend` を直す。
- [ ] T006 [Gateway] `frontend/vite.config.ts` の `server.proxy` を直す。
- [ ] T007 [Acceptance] `/ssf/streams/{stream_id}/events` と `/session/check` がゲートウェイ経由で Go に届くことを確かめる。
- [ ] T008 [Tooling] 検査を `mise` タスクにし、`check` の依存へ加える。
- [ ] T009 [Verify] 検査を通す。

## Risk Notes

リスクは medium。壊れているのは Shared Signals の受信と OIDC Session Management という、どちらも外部の当事者が呼ぶ経路である。届いていないことに現在誰も気づいていないのは、失敗が 200 の HTML として返るためである。**修正すると、これまで届かなかった要求が届くようになる。** 受信側の処理が本番の負荷を受けるのは初めてになるので、`docs/capacity.md` の Non-protocol request profile に置いた Shared Signals の見積もりが最初に試されるのはこの変更の後である。

**照合の網羅性そのものは検査できない。** 「実行時経路の正」が実際にすべての経路を含んでいることは、その正の作り方に依存する。`operations_gen.go` に現れない経路があれば、照合を通っても届かない経路が残る。T001 でこの網羅性を確かめ、確かめられない部分は Risk として残す。

**運用経路の分類を誤ると、`/metrics` を公開しうる。** `docs/contexts/system/decisions.md` は `/metrics` を認証なしと定めているので、「通すべき経路」の集合へ誤って入れると認証のない指標が公開入口から読める。分類は既定を「通してはならない」にし、通す経路だけを明示する向きにする。

`reversibility` は reversible。設定の変更であり、データも公開契約も変えない。

## Verification

- `mise run check`
- `mise run verify`
