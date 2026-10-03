# Feature: CSV の転送の例

## Rule: REQ-IDMANAGEMENT-034 CSV の見出しは機械キーと完全に一致する名前だけを受け付ける

### Example: EX-IDMANAGEMENT-034-01 BOM で始まる見出し

- Given ファイルは UTF-8 の BOM の直後に見出し `preferred_username` を持つ
- When 管理者がそのファイルを事前検証へ投入する
- Then 見出しは `preferred_username` として受け付けられる

### Example: EX-IDMANAGEMENT-034-02 大文字を含む列名

- When 管理者が見出し `Preferred_Username` を持つファイルを事前検証へ投入する
- Then ファイルは `invalid_header` で拒否される

### Example: EX-IDMANAGEMENT-034-03 秘密情報の列名

- When 管理者が見出しに `mfa_secret` を含むファイルを事前検証へ投入する
- Then ファイルは `invalid_header` で拒否される

## Rule: REQ-IDMANAGEMENT-035 列の数が見出しと異なる行は、その行だけを拒否する

### Example: EX-IDMANAGEMENT-035-01 列の足りない行を挟むファイル

- Given ファイルの見出しは二列であり、2 行目のデータだけが一列である
- When 管理者がそのファイルを事前検証へ投入する
- Then 2 行目は `invalid_column_count` で `rejected` となり、1 行目と 3 行目は計画される

## Rule: REQ-IDMANAGEMENT-037 CSV の転送ポリシーのデフォルト値と上限の境界

### Example: EX-IDMANAGEMENT-037-01 上限と等しい行の数

- Given 転送ポリシーのデータ行の上限は 2 行である
- When 見出しと 2 行のデータを持つファイルを読む
- Then 2 行とも読み取られる

### Example: EX-IDMANAGEMENT-037-02 上限を超える行の数

- Given 転送ポリシーのデータ行の上限は 2 行である
- When 見出しと 3 行のデータを持つファイルを読む
- Then 3 行目で `too_many_rows` となり解析を止める

## Rule: REQ-IDMANAGEMENT-036 先頭のアポストロフィーは、数式の保護として付けたものだけを取り除く

### Example: EX-IDMANAGEMENT-036-01 保護として付けたアポストロフィー

- When インポートが `'=SUM(A1)` のセルを読む
- Then 値は `=SUM(A1)` である

### Example: EX-IDMANAGEMENT-036-02 値の一部であるアポストロフィー

- When インポートが `'abc` のセルを読む
- Then 値は `'abc` のまま変わらない

## Rule: REQ-IDMANAGEMENT-038 属性値のセルは型ごとの正規の字句形だけを受け付ける

### Example: EX-IDMANAGEMENT-038-01 正規の字句形の値

- Given 属性 `level` の型は `number` である
- When インポートが `1.5` のセルを読む
- Then 値は数値の 1.5 である

### Example: EX-IDMANAGEMENT-038-02 正規でない字句形の値

- Given 属性 `level` の型は `number` である
- When インポートが `1.0` のセルを読む
- Then セルは拒否される

### Example: EX-IDMANAGEMENT-038-03 任意の属性の空のセル

- Given 属性 `nickname` は任意である
- When インポートが空のセルを読む
- Then その属性を消す
