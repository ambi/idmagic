namespace IdMagic.Contract {

enum SeedEnvironment {
  development: "development",
  test: "test",
  staging: "staging",
  production: "production",
}


enum SeedProfileKind {
  bootstrap: "bootstrap",
  development: "development",
  test: "test",
  performance: "performance",
}


enum SeedMode {
  dry_run: "dry_run",
  apply: "apply",
}


enum SeedOperationKind {
  create: "create",
  update: "update",
  noop: "noop",
  conflict: "conflict",
}


enum SeedSecretProvider {
  env: "env",
  file: "file",
}


@doc("シークレットを具体化するときに使う不透明な参照。`version` は、同じシークレットの意図したローテーションを識別する。")
model SeedSecretReference {
  provider: SeedSecretProvider;
  @minLength(1)
  locator: string;
  @minLength(1)
  version: string;
}


@doc("バージョン付きのマニフェスト。プロファイルごとに、型付きの望ましいリソース、取り込むファイル、決定的な生成器を定める。シークレットは SeedSecretReference としてだけ現れる。")
model SeedManifest {
  @minLength(1)
  schema_version: string;
  profile: SeedProfileKind;
  @doc("ルートのマニフェストのディレクトリからの相対パス。循環、ルートの外を指すパス、リモートの URL はいずれも拒否する。")
  includes?: string[];
  @doc("種類ごとにまとめた望ましいリソース。所有する Context の公開コマンドへ渡す。種類ごとにその Context のリクエストの形をとり、種類の集合が閉じていないので、任意の JSON 値とする。未知の種類と、論理キーの重複はどちらも拒否する。Seeding には HTTP の入口がないので、この形はリクエストボディではなくマニフェストのファイルである。")
  resources: unknown;
  @doc("プロファイルと明示的なシードから解決する、上限付きの生成器。`resources` と同じ理由で任意の JSON 値とし、生成器の種類ごとに固有のパラメーターをとる。生成した性能試験用のユーザーはログインを無効にし、シークレットを保持しない。")
  generators?: unknown;
}


@doc("プロファイルの計画または適用の入力。`profile` と `environment` は必須で、件数を引き上げるときは明示しなければならない。")
model SeedRequest {
  environment: SeedEnvironment;
  profile: SeedProfileKind;
  @doc("明示的に指定するルートのマニフェストのパス。省略すると、そのプロファイルのリポジトリのデフォルトを使う。")
  manifest_path?: string;
  mode: SeedMode;
  tenant_id?: string;
  generator_seed?: string;
  @doc("本番の初期構築でファーストパーティのクライアントに設定する URI。本番の初期構築には一つ以上が必要で、localhost の URI は受け付けない。")
  first_party_redirect_uris?: string[];
  count?: int64;
  @doc("performance プロファイルの通常の上限を超える件数を明示的に許可する。絶対的な上限を超える値は、それでも拒否する。")
  allow_large?: boolean;
  @doc("performance プロファイルのメモリ使用量を一定に抑えるため、一つのバッチで適用する記録の数。省略すると安全なデフォルト値を使い、上限を超える値は拒否する。")
  batch_size?: int64;
}


@doc("シードの実行が作った計画。シークレットも完全な個人情報も含まない。")
model SeedPlan {
  creates: int64;
  updates: int64;
  noops: int64;
  conflicts: int64;
  @doc("計画した操作を順に流せる形で表したもの。論理キー、操作の種類、秘匿した要約からなる。要約の形は操作の種類に従うので、任意の JSON 値とする。")
  operations: unknown;
}


@doc("環境のポリシー、プロファイル、件数の上限、または未知のキーが安全規則に反した。これはシードツールのドメイン条件であり、HTTP のレスポンスボディではない。Seeding には HTTP の入口がない。")
model SeedRejectedError {

}


@doc("シードで管理する対象が手作業で変更されており、明示的な照合なしには上書きできない。これはシードツールのドメイン条件であり、HTTP のレスポンスボディではない。")
model SeedConflictError {

}

}
