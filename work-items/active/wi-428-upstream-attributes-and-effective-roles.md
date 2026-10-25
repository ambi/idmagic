---
depends_on: [wi-20073-identify-upstream-attribute-authority]
status: pending
authors: [tn]
risk: high
created_at: 2026-08-27
priority: p1
change_kind: feature
affected_spec:
  - { path: docs/modules/sourcing/features/scim.md, requirement: REQ-SOURCING-004 }
  - { path: docs/modules/identity-management/features/user.md, requirement: REQ-IDMANAGEMENT-001 }
---

# 上流の権威が書き込む属性が、実効ロールへ波及する経路を制限する

## Motivation

`docs/modules/identity-management/glossary.md` は 2 つのことを述べている。`DynamicGroupRule` は「User の中核属性と `TenantUserAttributeSchema` で定義した属性だけを参照し、所属可否を Boolean で返す制限付き CEL 式」であり、`GroupMembership` は `effective_roles(user) = user.roles ∪ ⋃ membership.group.roles` を成り立たせる。

この 2 つを合わせると、**属性から実効ロールへの経路が存在する**。属性が変われば動的な所属が変わり、所属が変われば実効ロールが変わる。

一方 `Sourcing` は、上流の権威から属性を取り込む Context である。`REQ-SOURCING-004` は未対応の `PATCH` パスと読み取り専用属性への書き込みを拒否するが、それは書き込みの形式に対する制限であって、**書き込まれた値が権限へ波及することに対する制限ではない**。上流の IdP でユーザーの部署や職位を書き換えられる者は、その属性を参照する動的グループ規則が存在する限り、IdMagic 側のロールを間接的に得られる。

上流を信頼するという判断自体は正当でありうる。取り込み元は登録された権威だからである。しかし、その信頼の範囲が「アイデンティティの事実」までなのか「権限の決定」までなのかは、どこにも書かれていない。書かれていないため、テナント管理者が動的グループ規則を書くときに、その規則が上流由来の属性を参照しているかどうかを知る手立てがない。

これは[脅威モデル](../../docs/design/security/threat-model.md)の THREAT-067 と THREAT-083 が指す欠落である。`risk` を high とするのは、成立した場合の結果が権限昇格であり、しかも各所の拒否が正しく働いていても成立するためである。

## Scope

- 上流由来の属性と、IdMagic 内で決まる属性を区別できるようにするかを判断する。
- 上流が同期したグループ所属が、動的規則を経ずに実効ロールを直接動かす経路を扱う。`REQ-SOURCING-005` は「`GroupMembership` が同期され User の有効ロールが更新される」と定めており、こちらは属性を介さない分だけ直接的である。
- ロールを付与するグループの動的規則が、上流由来の属性を参照できるかどうかの方針を決める。
- 方針を `docs/modules/identity-management/decisions.md` または `docs/design/security/authorization.md` の判断として書く。どちらが owner かを決める。
- 管理者が動的規則を保存するときに、その規則が権限へ波及することを認識できるようにする。
- 対応する規範シナリオを足す。

## Out of Scope

- 上流の権威そのものの認証と登録。既存の取り込み元のバインディングは変えない。
- 動的グループ規則の CEL 文法の拡張。
- ロールモデルそのものの再設計。

## Design

方針には 3 案がある。

第一案は、上流由来の属性を動的規則から参照できなくすることである。最も安全だが、属性に基づくプロビジョニングという正当な用途をまるごと失う。人事システムを権威として部署でグループを構成することは、この製品が想定する主要な使い方の 1 つである。

第二案は、参照は許すが、ロールを付与するグループの規則に限って上流由来の属性の参照を拒否することである。権限に波及しない所属は自由に組め、波及するものだけを止められる。ただし「ロールを付与するグループ」は後から変わりうるので、グループへロールを付けた時点で既存の規則を検査する必要がある。

第三案は、参照も付与も許したうえで、その組み合わせが成立していることを管理者に見せ、監査に残すことである。制限ではなく可視化であり、判断をテナントへ委ねる。

採る案は本 work item の中心的な判断であり、承認の対象になる。現時点の傾きは第二案と第三案の組み合わせだが、第二案の検査点（グループへロールを付ける時点と、規則を保存する時点の双方）を実装できるかを先に確かめる必要がある。

### 着手前調査で判明した前提

属性ごとの出どころは追跡されていないため、[属性の出どころと上流の書き込み範囲を識別する先行項目](wi-20073-identify-upstream-attribute-authority.md)へ切り出す。
本項目は `pending` のまま依存の解決を待ち、T002 以降の実装と集約検査へ進まない。

| 確認した入口と保存型 | 現在の情報 | 制限の判断に不足する情報 |
| --- | --- | --- |
| `backend/idmanagement/user/domain/users.go` の `User`、`AttributeValue`、`UserAttributeDef` | 属性の値、型、本人による編集可否 | 属性ごとの上流の識別と書き込み範囲 |
| `backend/sourcing/internal/sourceuser/attributes.go` の `Attributes.Apply` | SCIM と Directory Connector の属性を同じ User のフィールドへ投影する | 値の出どころを投影後も識別する情報 |
| `backend/sourcing/internal/scim/source_idmanagement/user_lookup.go` の `IsSCIMUser` | SCIM の対応表に User があるか | 各属性を上流が更新できるか。User 全体の由来表示では区別できない |
| `backend/idmanagement/internal/user/usecases/federated_user.go` の `ProvisionFederatedUser` | JIT の属性を通常の `AttributeValue` として作成する | JIT で渡された各属性の出どころ |

規則とロールの検査を追加する入口は存在する。
`backend/idmanagement/internal/group/usecases/dynamic_groups.go` の `UpdateDynamicGroupRule` は、コンパイル済みの `References()` と Group のロールを保存前に参照できる。
同じファイルの `SetDynamicGroupRuleEnabled` も、有効化時の検査候補になる。
`backend/idmanagement/internal/group/usecases/admin_groups.go` の `CreateGroup` と `UpdateGroup` は、Group のロールを検査して保存する入口である。
CSV の `group_import_planner.go` と適用経路も、管理 API だけに拒否を置いて回避されないように調査する必要がある。

直接所属の経路は、属性の追跡とは別に残る。
`backend/idmanagement/source_commands.go` の `SetGroupMembership` は管理用の所属操作へ委譲し、`REQ-SOURCING-005` は Group のロールを実効ロールへ反映することを定めている。
再開時には、SCIM 管理の Group へのロール追加と、ロールのある Group を上流管理へ結び付ける操作の双方を扱う。

現在の値に出どころを付けるだけで、第二案の保存時の検査が成立するとは限らない。
User がまだいないときや、後から上流の属性マッピングが変わるときにも、規則の参照属性を上流が更新できるか判断できる必要がある。
先行項目では、この利用条件を満たす分類を決めてから追跡を実装する。
第一案、第二案、第三案の採否は未決であり、今回の調査だけで第二案と第三案を採用したことにはしない。

## Plan

1. 属性の出どころを現在追跡しているかを確認する。追跡していなければ、追跡が先行作業になる。
2. ロールを付与するグループと、上流由来の属性を参照する動的規則の現在の組み合わせを、実データではなくモデルの上で洗い出す。
3. 方針を決め、判断として書く。
4. 検査点を実装し、拒否を状態と効果の両方でテストする。

## Tasks

- [x] T001 [Baseline] 属性の出どころの追跡の有無と、検査点として使える箇所を確認する。追跡はなく、wi-20073 を前提として切り出す。規則保存、有効化とロール変更の入口は上記の調査に記す。
- [ ] T002 [Acceptance] 上流由来の属性を参照する規則がロール付与グループへ結び付く経路が、現在は何にも拒否されないことを観測する。
- [ ] T003 [Spec] 方針を判断として書き、規範シナリオを足す。
- [ ] T004 [App] 検査点を実装する。
- [ ] T005 [Verify] 拒否が、返した状態と、拒否が触らなかったものの両方で確かめられることを確認する。

## Verification

- 着手前調査と依存の追加だけの差分は、`mise run check-work-items` で形式と依存先、`mise run check-links` で追加したリンクを検査する。
- 以下は先行項目の完了と方針の確定後に行う。本段階では製品コード、仕様とテストを変えず、`mise run verify` を実行しない。
- 方針が拒否と定めた組み合わせが、規則の保存時とグループへのロール付与時の双方で拒否される。
- 拒否の後に、動的な所属も実効ロールも変わっていない。
- 正当な用途（権限へ波及しない属性ベースの所属）が引き続き成立する。
- `mise run verify`

## Risk Notes

この経路は各所の拒否が正しく働いていても成立する。テストは「拒否されること」ではなく「上流の属性変更が実効ロールを変えないこと」を確かめる必要があり、確認の境界を間違えやすい。

第一案を採ると既存のテナントの構成を壊しうる。既存の組み合わせを洗い出す T002 の前に方針を決めない。

属性の出どころを追跡していない場合、先行作業が本体より大きくなる可能性がある。その場合は本 work item を分割し、追跡の導入を先に切り出す。
