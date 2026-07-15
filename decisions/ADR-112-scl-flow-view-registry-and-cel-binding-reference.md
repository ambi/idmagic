---
status: accepted
authors: [tn]
created_at: 2026-07-16
---

# ADR-112: flows を views/sees/does 記法へ拡張し CEL root binding を形式定義する

## コンテキスト

[[ADR-103]](2026-07-14)は SCL 2.0 の `user_experience` を廃止し、`flows` を
`entry` と `transitions`(`from`/`action`/`interface?`/`to?`/`external?`)のみからなる
view 間ナビゲーショングラフに限定した。同 ADR は「却下した代替案」として
「`user_experience` の screen 台帳を縮小して残す: route/view state と UI 実装の同期コストが残り、
scenario と flow の責務が重なる」ことを明示的に却下している。

採用から2日後、実際に spec を書く/読む運用で次の2点が判明した。

- `flows.transitions` は `from`/`action`/`to` という遷移そのものだけを表現し、各 view が
  「何を表示するか」を一切表現できない。結果として `transitions` の配列が並ぶだけの資料は、
  遷移(エッジ)だけが強調され、UI フロー資料として要である「画面(ノード)の内容」が
  埋没する。`spec/contexts/authentication.yaml` の `Login` flow のように、同じ画面から
  複数のアクションが分岐する構造では特に読みにくい。
- CEL 式に登場する root binding(`context`/`subject`/`resource`/`principal`/`input`/`output`/
  `response`/`measurement`/`request`/`event`/`emitted`)は、
  `SPECIFICATION_CORE_LANGUAGE.md` の表で「どの位置でどの root 名が使えるか」だけが定義され、
  各 root が実際に何を指すか(どんなフィールドを持つオブジェクトか)が一切定義されていない。
  例えば `states.*.guard` だけが対象モデルのフィールド名を無接頭辞で束縛する例外であることが
  文書化されておらず、`guard: input.approved && status == Pending` の `status` が何を指すか
  読者が推測するしかなかった。

## 決定

### 1. flows を views map + sees/does 記法へ再設計する

`flows.<Name>.transitions`(フラット配列)を廃止し、`flows.<Name>.views`
(view 名をキーとする map)へ置き換える。各 view は必須の `sees`(画面の一言説明)と、
任意の `does`(ユーザー操作のリスト)を持つ。`does` の各要素は `action`(識別子)、
`does`(操作の一言説明)を必須とし、`interface`・`to`・`external: true` を任意で持つ
(`to` と `external` は同時に指定しない、という既存ルールを維持する)。

```yaml
flows:
  UserManagement:
    entry: UserList
    views:
      UserList:
        sees: ユーザー一覧(ユーザー名、有効/無効)画面
        does:
          - action: create
            does: ユーザー作成ボタンをクリックする
            to: UserCreate
      UserCreate:
        sees: ユーザー作成画面(ユーザー名入力フォーム、メールアドレス入力フォーム)
        does:
          - action: submit
            does: ユーザー情報をすべて入力して、作成ボタンをクリックする
            interface: CreateUser
            to: UserList
```

`entry` から到達可能かの検証は、`views.*.does[].to` を辿るグラフ探索に一般化し、
`views` に宣言された全キーが到達可能であることを検証する(view がキーとして明示化された
ことによる自然な強化)。`interface` の参照解決は既存どおり行う。

`sees`/`does` はあくまで画面/操作の**一言の要約**であり、screen 台帳・route・パラメータ・
loading/error state・goal/actor の全量インベントリではない。ADR-103 が却下したのは
「screen 台帳の縮小版」であり、本決定は route/view-state のような可変で実装追従が必要な
情報を持ち込むものではなく、低頻度にしか変わらない1行の説明を2つ(`sees`, `does`)
追加するに留めることで同期コストを抑える。ADR-103 の他の判断
(`objectives` を SLO 専用にする、`goal`/`actor` を flow に持たせない)は変更しない。

### 2. CEL root binding を形式定義する

`SPECIFICATION_CORE_LANGUAGE.md` の CEL セクションに、root ごとに実体を定義する
サブセクションを追加する。

- `input`/`output`: その interface が宣言する request/response フィールドそのもの。
- `context`: 実行時の暗黙コンテキスト。フィールドは実データに現れるものを正準列挙とする
  (`authenticated`, `tenant_id`, `client_id`, `grant_type`, `upload.content_type` 等)。
- `principal`: `authorization` 節で評価される呼び出し元。
- `subject`: `interfaces.requires`/`ensures` で束縛される呼び出し元。`principal` とは
  束縛される節が異なるだけで同じ「呼び出し元」を指すが、`oauth2.yaml` の glossary
  `Subject`(OIDC `sub` claim)とは別概念であることを明記する。
- `resource`: `access.resource.type` が指すモデルインスタンス、またはシングルトンの場合は
  `resource.id` のみが意味を持つ。
- `emitted`: 現在の interface 呼び出しで発行済みのイベント集合。`e is EventName` マクロを
  通してのみアクセスする。
- `measurement`/`request`/`response`/`event`(objectives 専用): `measurement` のみ実使用があり、
  他3つは現時点で実例のない将来枠であることを明記する。

`states.*.guard` が対象モデルのフィールド名を無接頭辞で束縛する例外であることも、
この形式定義の一部として明記する。

## 却下した代替案

- `user_experience` の screen 台帳をそのまま復活させる: route・パラメータ・view state・
  accessibility まで含む従来案は、ADR-103 が指摘した UI 実装との同期コストが再発する。
  `sees`/`does` という2つの一言説明のみに限定することで、この却下理由を回避する。
- flow ごとに `goal`/`actor` を必須にする: ADR-103 の却下理由(goal は scenario の見出しが
  所有する)は本決定でも変わらないため、`views`/`does` に `goal`/`actor` は追加しない。
- CEL root の形を JSON Schema で厳密に型付けする: `context`/`emitted` 等は interface ごとに
  実体が変わりうる暗黙オブジェクトであり、汎用スキーマで型付けすると実態と乖離する。
  ドキュメント上の正準フィールド一覧という形に留め、機械検証は既存の root 名検証のみとする。
- `principal` と `subject` を1つの root 名に統一する: 全 context の `requires`/`ensures`/
  `authorization` 節を書き換える移行コストが本決定のスコープに見合わないため、
  名称は維持しつつ文書上で明確に相互参照するに留める。

## 影響

- `tools/yaml-check/schemas/scl-v3.schema.json` の `Flow` 定義(`$defs.Flow`)を
  `views`/`FlowView`/`FlowAction` へ置き換える。
- `tools/yaml-check/src/scl-semantics.ts` の flow 到達可能性・interface 参照解決ロジックを
  `views[].does[].to` ベースに書き換える。
- `tools/scl-to-html` の `Flow` 型・`renderFlows`・関連テストを新記法に合わせて更新する。
- `spec/contexts/{application,audit,authentication,identity-management,oauth2,saml,signing-keys,system,tenancy,ws-federation}.yaml` の
  `flows` セクション(32件)を新記法へ移行し、実際の画面内容を記述する。
- `SPECIFICATION_CORE_LANGUAGE.md` の §3.8(flows)と §5(CEL expression と binding)を
  本決定に合わせて改訂する。
- `spec_version` は `"3.0"` のまま据え置く(単一 work item 内での一括移行のため新旧共存期間がなく、
  スキーマ世代を新設する必要がない)。
- 実装は `work-items/wi-215-*.md` で追跡する。
