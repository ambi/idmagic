---
name: integrate-work-item
description: 完了した作業項目のブランチを、マージコミットを作らずに rebase と fast-forward で main または指定のブランチへ取り込み、新しく合成されたツリーだけを検証してから、元のブランチと worktree を削除する。作業項目のブランチを取り込む、統合する、マージすると頼まれたときに使う。
---

# 完了した作業項目の取り込み

1. 完了した `work-item/<work-item-id>` の元のブランチ、それを開いているチェックアウト（リポジトリの根の既定のチェックアウト、または `.worktrees/<work-item-id>`）、利用者が指定した取り込み先のブランチ（指定がなければ `main`）を特定する。
   取り込み先はどこかのチェックアウトで開かれていることも、既定のチェックアウトが元のブランチを開いているためにどこでも開かれていないこともある。
   どのチェックアウトでも、無関係な利用者の変更を保つ。
   元のチェックアウトはクリーンで、作業項目が完了し、最終コミットが `implement-work-item` で検証済みでなければならない。
2. コミットや ref を変える前に、検証済みの元のコミットとツリーを記録する。

   ```sh
   source_commit=$(git rev-parse 'work-item/wi-42^{commit}')
   source_tree=$(git rev-parse 'work-item/wi-42^{tree}')
   ```

3. 履歴を直線に保つ。
   元のブランチが取り込み先の上にまだなければ、取り込み先の上へ rebase する。
   進行中の競合には `resolving-merge-conflicts` を使う。
   rebase 後の元のツリーを、記録した検証済みのツリーと比べる。
   異なれば、rebase が新しいツリーを合成したということである。
   元のチェックアウトで次を実行して検証する。

   ```sh
   mise run verify-integration -- "$source_commit"
   ```

   両側はそれぞれのツリーで検証済みなので、このタスクは、元のブランチ自身の変更と取り込んだ変更の両方が届く層と Go のパッケージだけを再実行する。
   実行したものと省いたものは `.verification/verify-selection.json` に記録される。
   競合の解消そのものが必要とする局所の検査も再実行する。

   マージコミットは作らない。
   元のブランチは一つずつ取り込み、次の作業項目は更新後の取り込み先から始めて rebase を要らなくする。
4. 取り込み先を rebase 後の元のブランチへ fast-forward し、取り込み後のツリーを検証済みのツリーと比べる。
   取り込み先がクリーンなチェックアウトで開かれていればそこで fast-forward し、どこでも開かれていなければ ref だけを動かす。
   どちらも fast-forward でない更新を拒否する。

   ```sh
   git -C <target-checkout> merge --ff-only work-item/wi-42   # 取り込み先がそのチェックアウトで開かれている
   git fetch . work-item/wi-42:main                           # 取り込み先がどこでも開かれていない
   integrated_tree=$(git rev-parse 'main^{tree}')
   test "$integrated_tree" = "$(git rev-parse 'work-item/wi-42^{tree}')"
   ```

   - ツリーが同一なら、すぐに終える。
     取り込み先は検証済みの結果そのものを持ち、`check`、`build-site`、`verify` を再実行しても証拠は増えない。
   - ここでツリーが異なるのは、fast-forward が検証済みの元の結果を作らなかったということである。
     後始末の前に止めて調べる。
     通常の fast-forward なら同一になる。

   この判断を決めるのはコミットの同一性ではなく、ツリーの同一性である。
   rebase はファイルを変えずにコミットのメタデータを変えることがある。
5. 取り込み先が元のブランチを含み、必要な検証が通ったら、取り込んだ元のブランチを片付ける。
   元のブランチが既定のチェックアウトで開かれていたら、そのチェックアウトを取り込み先へ戻す。
   worktree で開かれていたら、worktree を削除する。

   ```sh
   git switch main                        # 元のブランチが既定のチェックアウトにある
   git worktree remove .worktrees/wi-42   # 元のブランチが worktree にある
   git branch -d work-item/wi-42
   ```

   `git branch -d` は取り込み先が含まないブランチの削除を拒否するので、取り込んでいない作業の削除を防げる。
   `git worktree remove` が途中で失敗したら、その worktree の `git status --porcelain` が ` D` の行だけであることを確かめてから、`--force` を付けて再実行する。
6. 取り込みの方法、取り込み先のコミット、ツリーの比較、行った検証、削除したブランチと worktree を報告する。
   `spec/generated/` と `site/` はコミットしない。
   リモートの ref の更新と push は、利用者が明示したときだけ行う。
