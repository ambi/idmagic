---
status: completed
archived_from: 8fde8e6d80edf2c191cea8b3913e305b829be78d
---

# Group の表示名の取得元は保存されるだけで、送出する `displayName` を何も選んでいない

`mise run spec-diff` は `RFC7643-OUT-GROUP-RESOURCES` の変更と、TypeSpec への
`ProvisioningGroupDisplayNameSource` の追加を報告する。`group_push.display_name_source` が、下流へ送る
Group の `displayName` に実際に反映されるようになった。それまでこの設定を読む production の経路は
**1 つも無く**、管理者が何を入力しても下流が受け取る `displayName` は常に Group の名前だった。
**値域を閉じた。** `display_name_source` は任意の文字列から `name`、`description`、`email` の 3 つへ狭まり、
集合の外は `ProvisioningConnection.Validate()` が拒否する。管理画面の欄も自由入力から選択になった。
自由入力のままにする道もあったが、打ち間違いが「保存され、配信も成功し、下流だけが変わらない」結果に
なる —— **この欠陥そのものの形を再生産する**ので選ばなかった。
**責務を 2 つに分けた。** `ports.AttributeSource` は接続を受け取らないので、取得元の選択を属性源へ届けるには
port の署名を変えることになり、User 側にも波及する。代わりに `GroupAttributeSource` は Group の事実
(`id`、`name`、`description`、`email`) だけを解決し、`display_name` を返さなくなった。どれを表示名にするかは
接続ごとの判断であって Group の事実ではないので、接続を既に手にしている `deliverGroup` が組み立てる。
**未設定、集合の外、選んだ属性が空 —— いずれも Group の名前へ落ちる。** 集合の外は管理 API が拒否するので
今後は入らないが、`DisplayNameSourceKey()` は nil レシーバーと未知の値を受け続ける。`description` と
`email` は Group が持たないことがあり、そこで配信を失敗させると設定の打ち間違い 1 つでその Group の
送出全体が止まる。空文字を送るのも下流の検証に拒否されうるので選ばない。
**検査器の欠陥を 1 件見つけ、先に直した。** `check-api-compat` は `$ref` の傍らに書かれた keyword を
捨てるので、`display_name_source` を列挙にしただけで「default が `"name"` から undefined へ変わった」と
5 件を誤報した。生成物はその位置に `"default": "name"` を持っている。baseline を更新すれば報告は消えるが、
同じ読み落としは逆向き（傍らの `default` が変わっても報告ゼロ）にも効くので、
[[wi-537-api-compat-drops-keywords-written-beside-a-ref]] として切り出し、先に直してから本項目へ戻った。

全文は `git show 8fde8e6d80edf2c191cea8b3913e305b829be78d:<このファイルのパス>` で読む。
