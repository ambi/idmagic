---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: bugfix
affected_spec:
  - { path: docs/contexts/provisioning/standards.md, requirement: RFC7643-OUT-GROUP-RESOURCES }
  - { path: spec/contexts/provisioning/models.tsp, symbol: IdMagic.Contract.GroupPushConfig }
---

# Group の表示名の取得元は保存されるだけで、送出する `displayName` を何も選んでいない

## Motivation

`docs/contexts/provisioning/standards.md` の `RFC7643-OUT-GROUP-RESOURCES` は「`displayName` の取得元は
`GroupPushConfig.display_name_source` が選び、既定は Group の名前である」と宣言する。TypeSpec の
`GroupPushConfig.display_name_source` も「下流の `displayName` へ射影する IdMagic 側の Group 属性キー」と
説明し、管理画面は自由入力の欄として書き込める。

**しかし、この値を読む production の経路が 1 つも無い。** `GroupAttributeSource.ResolveAttributes` は接続を
参照せず、`display_name` を常に `Group.Name` から解決する。管理者が `email` や `description` を入れても、
下流へ届く `displayName` は Group の名前のままである。設定は保存され、画面は入力を受け付け、配信は成功し、
観測できる差だけが無い。

`GroupAttributeSource` の注釈は「`name` が既定であり今日の時点で意味を持つ唯一の値なので、未設定や未知の
取得元は配信を失敗させずに `Group.Name` へ落ちる」と書く。未知の値を既定へ落とす判断そのものは妥当である
（表示名は fail-closed の判断ではない）。成立していないのはその手前で、**既知の値でも選べない**。
`name` 以外に意味を持つ値が無いのは実装がそう決めたからであって、解決済みの属性表は `name`、`description`、
`email` を既に持っている。

[[wi-441-push-groups-produces-no-delivery]] は Scope に「`GroupPushConfig` の表示名の取得元を反映する」を
挙げ、Design にも同じ文を書いたうえで、反映しない実装を残した。宣言だけが先に進み、実装が追いつかなかった
箇所である。

[[wi-507-back-provisioning-standards-rows-with-tests]] が `RFC7643-OUT-GROUP-RESOURCES` にテストを対応付ける
途中でこれを見つけた。同項目は行の他の節（`schemas`、対応付けが解決した属性、メンバーシップを本文に載せず
増分 `add` の PATCH で送ること、除去を送らないこと）を観測して消化しており、この節だけが観測を持てない。

## Scope

- `GroupPushConfig.display_name_source` が選んだ属性キーから `display_name` を解決する。
- 未設定と未知のキーは `Group.Name` へ落とす。表示名は fail-closed の判断ではないので、配信は失敗させない。
- 選べる値の集合を決め、TypeSpec の説明と管理画面の入力をその集合に合わせる。自由入力のままにするか列挙に
  するかは、この項目が決める設計の判断である。
- `RFC7643-OUT-GROUP-RESOURCES` の「取得元を選ぶ」節を区別できるテストを対応付ける。既定とは違うキーを選んだ
  接続で、下流が受け取る `displayName` がそのキーの値になることを観測する。

## Out of Scope

- `RFC7643-OUT-GROUP-RESOURCES` の他の節の観測。[[wi-507-back-provisioning-standards-rows-with-tests]] が持つ。
- User 側の属性解決。`UserAttributeSource` は接続ごとの取得元の選択を持たない。
- 属性対応付け (`AttributeMappingRule`) の表現力。拡張スキーマや多段のフィルターパスは
  `RFC7643-OUT-SCHEMA-EXTENSIONS` が宣言する範囲であり、この項目は触れない。
- メンバーシップの送出。

## Verification

- 既定とは違う取得元を選んだ接続で、下流が受け取る Group リソースの `displayName` がそのキーの値になる。
- 未設定と未知のキーは `Group.Name` になり、配信は成功する。
- `RFC7643-OUT-GROUP-RESOURCES` を名指すテストが、この節を区別する入力と観測を持つ。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **既定値で通るテストだけを書く。** 既定は現状の実装でも通るので、既定だけを観測しても選択が働いている
  証拠にならない。既定とは違うキーを必ず 1 つ通す。
- **未知のキーを失敗にする。** 表示名は fail-closed の判断ではない。未知のキーで配信を止めると、設定の
  打ち間違いが Group の送出全体を落とす。
- **選べる値の集合を決めずに実装する。** 自由入力のまま任意の属性キーを通すと、解決済みの属性表に無い
  キーの扱い（空文字にするのか既定へ落とすのか）が暗黙になる。TypeSpec の説明と画面の入力を、決めた
  集合に合わせる。
