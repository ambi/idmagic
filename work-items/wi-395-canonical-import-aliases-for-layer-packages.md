---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p3
change_kind: tooling
spec_impact:
  kind: none
  reason: "import 別名の綴りを揃え、それを lint で強制するだけである。パッケージの配置、型、関数、HTTP 契約、データモデルのいずれも変えず、規範的シナリオと TypeSpec シンボルの追加、変更、退役を伴わない。"
---

# 層名パッケージの import 別名を `importas` で正準化する

## Motivation

`domain`、`ports`、`usecases` はパッケージ名が層の名前なので、同じファイルで複数の Context を扱うと衝突する。そのため**インポート側は別名を付けることになり、その別名を決める規則がどこにもない**。

対象の import 行は 1615 行あり、そのうち **24 パッケージで綴りが割れている**。同じパッケージが、読む場所によって違う名前で呼ばれている。

| パッケージ | 実際に使われている別名 |
| --- | --- |
| `tenancy/domain` | `tenancydomain` (192) / `tenantdomain` (1) |
| `authentication/domain` | `authdomain` (71) / `authndomain` (1) |
| `idmanagement/agent/domain` | `agentdomain` (25) / `agentmodel` (3) |
| `application/domain` | `appdomain` (20) / `applicationdomain` (1) |
| `workloadidentity/domain` | `workloaddomain` (18) / `workloadidentitydomain` (1) |
| `tenancy/ports` | `tenantports` (45) / `tenancyports` (1) |
| `tenancy/usecases` | `tenancyusecases` (16) / `tenantusecases` (12) |
| `shared/notification/ports` | `notificationports` (19) / `sharednotification` (16) |
| `authentication/session/ports` | `sessionports` (15) / `authnports` (6) |
| `jobs/ports` | `jobsports` (14) / `jobports` (3) |

少数派の綴りを直すと **78 行**である。全体から見れば小さいが、問題は行数ではない。**ドメイン型やポートの正準な呼び名がリポジトリの中に存在しない**ことである。`Tenant` を指す語が読む場所によって変わる状態は、境界づけられたコンテキストが共有された言語を持つという前提と合わない。

層をまたいだ不揃いもある。`wsfederation/domain` は `feddomain`、`wsfederation/ports` は `wsfederationports` と `wsfedports` の 2 通りで呼ばれている。`feddomain` は `authentication/federation`（インバウンドフェデレーション）と読めてしまうが、実際は WS-Federation を指している。**別名が誤読を招いている**例である。

規約を文書に書いても腐る。この repo は読まれる文書ではなく落ちる検査を選んできたので、同じ形で解く。golangci-lint の `importas` は import 別名を規則で強制でき、現在の `.golangci.yml` では有効化されていない。

## Scope

- `.golangci.yml` に `importas` を追加し、`domain` / `ports` / `usecases` の別名を規則として定める。
- 規則から外れている 78 行を修正する。
- 規則を追加した後に新しく生えるパッケージも網羅されるよう、正規表現による包括規則を置く。
- `docs/structure.md` に、層名パッケージの別名が lint で強制されることを記す。

## Out of Scope

- **パッケージの物理配置。** `domain` / `ports` / `usecases` のディレクトリは動かさない。判断の経緯は Design に残す。
- `backend/` 直下の構成。
- `handlers_http`、`db_postgres` などアダプターパッケージの別名。層名ではなく `<role>_<technology>` なので衝突が起きにくく、実際に綴りが割れていない。
- 別名の要らない import（`tenancy`、`oauth2` などの Context ルートパッケージ）への `importas` 適用。

## Design

### 採る形

`.golangci.yml` の `linters.settings.importas` に規則を置く。

```yaml
importas:
  no-unaliased: true      # 規則を定めたパッケージは必ずその別名で import する
  no-extra-aliases: false # 規則の無いパッケージへの別名付けは禁じない
  alias:
    # 1. 確立した略称を明示エントリで正準として固定する（順序が先）
    - pkg: github.com/ambi/idmagic/backend/idmanagement/domain
      alias: idmdomain
    - pkg: github.com/ambi/idmagic/backend/sharedsignals/domain
      alias: ssdomain
    # ...
    # 2. 残りは正規表現で機械的に導く（後段）
    - pkg: github.com/ambi/idmagic/backend/(\w+)/(\w+)/domain
      alias: ${2}domain
    - pkg: github.com/ambi/idmagic/backend/(\w+)/domain
      alias: ${1}domain
```

### 既存の略称は温存する

規則を `<ディレクトリ名>+<層名>` の機械的な導出だけにすると、`idmdomain` → `idmanagementdomain`、`ssdomain` → `sharedsignalsdomain`、`authdomain` → `authenticationdomain`、`signingdomain` → `signingkeysdomain`、`appdomain` → `applicationdomain`、`igdomain` → `idgovernancedomain`、`workloaddomain` → `workloadidentitydomain` となり、**350 行以上が長くなる方向に書き換わる**。

これは目的と釣り合わない。解きたいのは綴りが割れていることであって、綴りが短いことではない。したがって**確立した略称は明示エントリで正準として固定し、機械的導出は規則の無いパッケージへの受け皿として後段に置く**。修正は 78 行に収まる。

明示エントリの一覧そのものが、この repo における別名の正準な登録簿になる。それが `importas` を選ぶ理由でもある。

### 綴りが割れているものの決め方

24 パッケージのうち、多数派が明らかなものは多数派を採る。**判断が要るのは 2 つだけ**である。

- `tenancy/usecases`：`tenancyusecases` (16) と `tenantusecases` (12) が拮抗している。`tenancy/domain` が `tenancydomain` で、`tenancy/ports` が `tenantports` なので、Context 内でも既に割れている。**Context 名（`tenancy`）に揃える**のが筋である。`tenant` は集約の名前であってパッケージの名前ではない。
- `wsfederation/*`：`feddomain` / `wsfederationports` / `wsfedports` の 3 通り。`authentication/federation` と誤読される `feddomain` は採らない。`wsfed` に揃えるか `wsfederation` に揃えるかを着手時に決める。

`shared/notification/ports` の `sharednotification` (16) は、他と違って層名を含まない別名である。`shared` 配下は Context ではなく技術的な共通機能なので、`<capability>ports` に揃えるか別扱いにするかを決める。

### 正規表現の衝突

`backend/authorization/domain` と `backend/oauth2/authorization/domain` は、機械的導出だとどちらも `authorizationdomain` になる。両方を import するファイルではコンパイルが通らない。**片方は明示エントリで別の別名を与える**（`oauth2/authorization` 側を `oauthauthorizationdomain` などにする）。同じ形の衝突が他に無いことを T001 の棚卸しで確認する。

### `domain/` ディレクトリを廃止する案を採らなかった理由

当初は `backend/<context>/(<feature>/)domain/` を廃止し、ドメイン層を Context / Feature 直下へ引き上げる案を検討した。DDD / Clean Architecture で最も中心となる層が境界の名前をそのまま名乗る、という点で筋は通っている。採らなかったのは次の理由による。

- **別名の問題を解かない。** 原因は「`domain` という名前」ではなく「パッケージ名が層名であること」で、それは `ports`（520 行）と `usecases`（332 行）でも同一である。ドメイン層だけ動かしても 58% しか触れず、残りは同じ形で残る。非対称を `docs/structure.md` で説明する負債だけが増える。
- **語彙の観点でも利得が小さい。** `userdomain.User` の `User` は既に語彙を持っている。`user.User` は短いが stutter は残る。
- **`tools/check/src/check-boundaries.ts` の層判定が名前判定から位置判定へ緩む。** 移動の途中で検査が誤って通ると、層の逆流に気づけない。
- 代わりに検討した「core の 3 層を Context / Feature の 1 パッケージへ統合する」案（`tenancy.Tenant` / `tenancy.TenantRepository` / `tenancy.CreateTenant`）は別名を完全に消し、元の発想を最後まで通した形になる。ただし `domain` が `usecases` を import しないという規則がパッケージ境界を失って強制不能になり、リポジトリ全体の書き換えを要する。**本 work item はこの案を否定しない。** `importas` は統合の前提を壊さないので、統合をやる際は明示エントリを捨てるだけで済む。

### 却下した案

- **規約を `docs/structure.md` に書く。** 機械検査が無い規約は腐る。今の 24 パッケージの割れがその実例である。
- **`importas` を機械的導出だけで構成する。** 350 行以上を長くする方向に書き換えることになり、目的と釣り合わない。
- **`no-extra-aliases: true` にする。** 規則の無いパッケージへの別名付けまで禁じると、標準ライブラリや外部依存の import にまで影響する。今回解きたい範囲を超える。

## Plan

- **棚卸しを先に行う。** 24 パッケージそれぞれについて正準とする別名を決め、明示エントリの一覧を作る。ここで決め切らずに設定を書き始めると、lint が落ちるたびに場当たりで別名を足すことになり、登録簿の意味が無くなる。
- 設定を入れてから 78 行を直す。逆順にすると、直した綴りが正しいことを機械で確認できない。
- 書き換えは `sd` で行う。78 行を手で触らない。
- `importas` の正規表現がアンカーされるか（部分一致か完全一致か）、明示エントリと正規表現エントリの評価順が定義されているかを、実際に落ちる例で確認してから本採用する。ドキュメントの記述だけを根拠にしない。

### 着手時に決める未解決点

1. `wsfederation/*` を `wsfed` と `wsfederation` のどちらに揃えるか。
2. `shared/` 配下（`notification`、`ratelimit`、`security` など）を Context と同じ規則に含めるか、別扱いにするか。
3. `oauth2/authorization` 側に与える別名。

## Tasks

- [ ] T001 [Inventory] 対象 1615 行を棚卸しし、24 パッケージの正準な別名と、正規表現が衝突するパッケージを確定する。
- [ ] T002 [Tooling] `.golangci.yml` に `importas` を追加する。明示エントリを先に、正規表現の受け皿を後段に置く。
- [ ] T003 [Tooling] 正規表現のアンカーと評価順を、意図的に規則違反の import を書いて確認する。
- [ ] T004 [Refactor] 規則から外れた 78 行を `sd` で修正する。
- [ ] T005 [Spec] `docs/structure.md` に、層名パッケージの別名が lint で強制されることと、正準な別名の決め方を記す。
- [ ] T006 [Verify] `mise run lint-go` と `mise run verify` を通す。

## Verification

- `mise run lint-go`
  - reason: 追加した `importas` 規則が既存コード全体に対して通ること。
- `mise run verify`
- 手動: 任意のファイルで `tenancydomain` を `tenantdomain` に書き換え、`mise run lint-go` が落ちることを確認する。落ちなければ規則は強制されていない。
- 手動: 規則を定めていない新しいパッケージを `domain` 配下に作り、正規表現の受け皿が効いて別名を強制することを確認する。効かなければ、次に生えるパッケージから再び綴りが割れる。
- 手動: `backend/authorization/domain` と `backend/oauth2/authorization/domain` を同時に import するファイルがコンパイルできることを確認する。

## Risk Notes

リスクは low。振る舞いを変えず、別名の綴りだけを触る。誤りがあればコンパイルが落ちるので、静かに壊れる経路が無い。

唯一の実質的な危険は、**明示エントリの一覧が最初から不完全なまま固定されること**である。棚卸し（T001）を省いて設定を書き始めると、lint が落ちるたびに場当たりで別名を足すことになり、「登録簿を機械で守る」という目的が「lint を黙らせるための設定」に変わる。T001 を先頭に置いたのはこのためである。

`importas` の挙動（正規表現のアンカー、明示エントリと正規表現の評価順）を実例で確認せずに採用すると、規則が一部のパッケージにだけ効いている状態に気づけない。T003 をタスクとして独立させたのはこのためである。
