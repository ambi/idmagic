---
depends_on: []
status: pending
authors: [tn]
risk: low
created_at: 2026-08-23
priority: p2
change_kind: docs
affected_spec:
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-007 }
  - { path: docs/contexts/provisioning/scenarios.md, requirement: REQ-PROVISIONING-009 }
---

# Provisioning に `standards.md` を置き、外向き SCIM の準拠範囲を宣言する

## Motivation

IdMagic は SCIM 2.0 を両方向で扱う。内向き（`Sourcing`、SCIM サーバー）と外向き（`Provisioning`、SCIM クライアント）である。**準拠範囲を宣言しているのは内向きだけである。**

`docs/contexts/sourcing/standards.md` は RFC 7643 と RFC 7644 に 8 行の `Adoption` / `Strength` / `Statement` を与えている。`docs/contexts/provisioning/` に `standards.md` は無く、`README.md` の索引にも行が無い。

宣言が無いだけで、外向きも同じ RFC が形を定める部分を実装している。`provisioning/internals.md` 自身がそう書いている。

> 外向き側には、フィルター文字列の組み立てと、対応付けに基づく広い属性集合 (`externalId`、Enterprise 拡張) の直列化が必要になる。重なるのは Discovery の構造体と RFC が定めるスキーマ URN 程度

フィルター構文、`externalId`、Enterprise 拡張、Discovery、スキーマ URN——いずれも RFC 7643 / 7644 が定める。**どこまで従い、どこを送らないかは、いまコードにしか無い。**

これが効くのは連携先を増やすときである。下流 SaaS の SCIM 実装には差があり、PATCH を受けない相手、Enterprise 拡張を無視する相手、`externalId` で相関しない相手が実在する。**「IdMagic は PATCH を送るのか PUT を送るのか」「Enterprise 拡張を常に送るのか」は連携の可否を決める問いだが、答えは仕様のどこにも書かれていない。**

[SPECIFICATION_FORMAT.md](../SPECIFICATION_FORMAT.md) §5 と [DOCUMENTATION_GUIDE.md](../DOCUMENTATION_GUIDE.md) §5.3 は、規範の各行に証拠となるテストを要求し、`excluded` の行には否定テストを要求する。**宣言が無い規範には、この要求が一切かからない。** 送らないと決めたものが送られるようになっても、それを嘘だと言う記述が存在しない。

## Scope

- `docs/contexts/provisioning/standards.md` を作り、外向き SCIM クライアントとしての RFC 7643 / RFC 7644 の準拠範囲を宣言する。
- 少なくとも次を `Adoption` / `Strength` / `Statement` の行として決める。
  - User / Group リソースの作成・置換・削除。
  - 部分更新に PATCH を使うか PUT を使うか、接続設定で選べるのか。
  - Enterprise 拡張（`employeeNumber`、`department`、`manager`）を送るかどうか。
  - `externalId` による相関。
  - 連携先の `ServiceProviderConfig` / `ResourceTypes` / `Schemas` の探索と、探索結果に従うのか無視するのか。
  - フィルターを使う場面と、組み立てる構文の範囲。
  - 連携先が返す SCIM エラーレスポンスの解釈（どの状態を再試行し、どれを隔離するか）。
- `docs/contexts/provisioning/README.md` の索引に行を足す。
- 各行に対応するテストを確かめ、無い行にはテストを足す。

## Out of Scope

- `Sourcing` の `standards.md` の見直し。内向きの宣言は既にあり、この変更では触れない。
- 新しいプロトコルの連携先（`internals.md` が将来として挙げる `entraid`、`googledir`）。それらの規範は、その機能単位を作る変更が持つ。
- 実装の変更。いま送っているものを宣言することに限る。**宣言しようとして「これは仕様として妥当でない」と分かった場合は、欠陥として個別の work item へ切り出す。**
- 他の `standards.md` を持たない Context（audit、jobs、tenancy など）の点検。それらが外部規範に準拠しているかは別の問いであり、Provisioning のように「同じ RFC を宣言している片割れが隣にいる」という明確な非対称は無い。

## Design

未定。着手時に次の 3 点を確定して本節に記録する。

1. **クライアント側で `Adoption` の 4 値をどう読むか。** `required` / `optional` / `partial` / `excluded` は、DOCUMENTATION_GUIDE §5.3 が「能力を取り入れるかどうか」——つまり**提供者側**の語彙として定義している。送出側では意味を当て直す必要がある。素直な対応は次だが、これでよいかを決める。

   | 値 | 送出側での読み |
   |---|---|
   | `required` | 常に送る、または常に行う |
   | `optional` | 接続設定または連携先の探索結果に応じて行う |
   | `partial` | 一部だけ行う |
   | `excluded` | 行わない |

   `optional` を「接続設定次第」と読むと、`Statement` に**何が既定でどう切り替わるか**まで書かないと検証できない行になる。

2. **ID 空間を `Sourcing` と分けるか。** `RFC7643-CORE-RESOURCES` は既に `Sourcing` が使っている。SPECIFICATION_FORMAT §5 の一意性は「within the document」なので、同じ ID を Provisioning でも使うことは検査を通る。しかしテスト名に規範 ID を書く運用（§5.3）では、**リポジトリ全体で同じ文字列が 2 つの別々の保証を指すことになる。** どちらのテストなのかがテスト名から読めない。接頭辞で分ける（`RFC7643-OUT-CORE-RESOURCES` のような）案を検討する。分けると Sourcing 側の既存 ID との対称性が崩れるので、Sourcing 側も改名するかどうかまで含めて決める。

3. **行ごとの証拠テストが既にあるか。** 外向き SCIM のテストは実在するはずだが、規範 ID を名指ししていない。**名指しの追加だけで済む行と、テストを書く必要がある行を先に分ける。** `excluded` と決めた行には否定テストが要る。「送らない」ことを確かめるテストは、いま 1 件も無い見込みが高い。

## Plan

- まず外向きの実装が実際に何を送っているかを読み、宣言の草案を作る。**宣言を先に書いて実装を後から確かめる順序にしない。** 願望が仕様になる。
- 草案ができた時点で 1 と 2 を確定する。ID の形が決まらないとテストの名指しが書けない。
- 行を入れるたびに対応するテストを確かめる。証拠の無い行を先にまとめて入れない。
- 「送っているが仕様として妥当でない」が出たら切り出して先へ進む。

## Tasks

- [ ] T001 [Spec] 外向き SCIM の実装を読み、いま何を送り何を送っていないかを列挙する。
- [ ] T002 [Design] `Adoption` の読み替え、ID 空間の分け方、証拠テストの過不足を確定し `## Design` に記録する。
- [ ] T003 [Spec] `docs/contexts/provisioning/standards.md` を作る。
- [ ] T004 [Spec] `docs/contexts/provisioning/README.md` の索引に行を足す。
- [ ] T005 [Test] 各行の証拠テストを確かめ、名指しを足す。無い行にはテストを書く。
- [ ] T006 [Test] `excluded` の行に否定テストを置く。
- [ ] T007 [Triage] 実装が仕様として妥当でない箇所を、欠陥として個別の work item へ切り出す。
- [ ] T008 [Verify] `mise run check-spec`、`mise run test-go`、`mise run verify` を通す。

## Verification

- `mise run check-spec`
  - reason: `standards.md` は書式が *(checked)* である。`Adoption` と `Strength` の値、`excluded` に `MUST` を置いていないこと、ID の一意性がここで落ちる。
- `mise run test-go`
- `mise run verify`
- 手動: `excluded` と宣言した行を 1 つ選び、その機能を実際に送るよう実装を一時的に変えて、否定テストが落ちることを確認する。落ちなければ、その行は誰も守っていない。
- 手動: `docs/contexts/sourcing/standards.md` と並べて読み、内向きと外向きで同じ RFC の同じ条項について矛盾した宣言をしていないことを確認する。

## Risk Notes

リスクは low。文書の追加とテストの名指しであり、製品の振る舞いは変えない。

**この作業の価値は、書いた行の数ではなく `excluded` の行の数で決まる。** 「やっていること」を並べるだけなら、コードを読めば分かることの写しになる（SPECIFICATION_FORMAT §3 が禁じる形）。仕様として意味を持つのは「**やらないと決めたこと**」と、その否定テストである。送らないものを 1 つも挙げずに終わったら、この文書はほぼ無価値だと考えてよい。

もう 1 つの失敗は、**実装を確かめずに RFC の目次から行を起こすこと**である。そうすると `Statement` が標準の側から書かれ（§5 が名指しで警告している形）、製品が実際にやっていないことを宣言することになる。T001 を T003 より先に置いているのはそのためである。
