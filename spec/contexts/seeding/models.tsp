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


@doc("秘密値を materialize するための opaque reference。version は同一 secret の意図した rotation を識別する。")
model SeedSecretReference {
  provider: SeedSecretProvider;
  @minLength(1)
  locator: string;
  @minLength(1)
  version: string;
}


@doc("profile ごとの型付き desired resource、include、決定的 generator を持つ versioned manifest。秘密値は SeedSecretReference だけで表す。")
model SeedManifest {
  @minLength(1)
  schema_version: string;
  profile: SeedProfileKind;
  @doc("root manifest directory 内のローカル相対 path。循環、root 外 path、remote URL は拒否する。")
  includes?: string[];
  @doc("対応する record context の公開 command surface へ渡す型付き desired resource。未知 kind と重複 logical key は拒否する。")
  resources: unknown;
  @doc("profile と明示 seed から決定する bounded generator。performance user は login-disabled で秘密値を持たない。")
  generators?: unknown;
}


@doc("profile を計画または適用する入力。profile と environment は必須で、件数を増やす場合も明示する。")
model SeedRequest {
  environment: SeedEnvironment;
  profile: SeedProfileKind;
  @doc("明示 root manifest path。未指定時は profile ごとの repository default を選ぶ。")
  manifest_path?: string;
  mode: SeedMode;
  tenant_id?: string;
  generator_seed?: string;
  @doc("production bootstrap で first-party client に設定する明示 URI。production では bootstrap 実行時に 1 件以上を必須とし、localhost URI は許可しない。")
  first_party_redirect_uris?: string[];
  count?: int64;
  @doc("performance profile の通常上限を超える件数を明示して許可する。絶対上限を越える値は許可しない。")
  allow_large?: boolean;
  @doc("performance profile を bounded memory で適用する 1 batch の最大件数。未指定時は安全な既定値を使い、上限を超える値は拒否する。")
  batch_size?: int64;
}


@doc("秘密値と PII 全量を含まない seed の計画結果。")
model SeedPlan {
  creates: int64;
  updates: int64;
  noops: int64;
  conflicts: int64;
  @doc("logical key、operation kind、redacted summary の streamable な表現。")
  operations: unknown;
}


@doc("環境 policy、profile、件数上限、または unknown key が安全規則に違反した。")
model SeedRejectedError {

}


@doc("seed 管理対象に manual drift があり、明示 reconcile なしに上書きできない。")
model SeedConflictError {

}

}
