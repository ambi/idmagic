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


@doc("An opaque reference used to materialize a secret. `version` identifies an intended rotation of the same secret.")
model SeedSecretReference {
  provider: SeedSecretProvider;
  @minLength(1)
  locator: string;
  @minLength(1)
  version: string;
}


@doc("A versioned manifest holding, per profile, the typed desired resources, the includes, and the deterministic generators. A secret appears only as a SeedSecretReference.")
model SeedManifest {
  @minLength(1)
  schema_version: string;
  profile: SeedProfileKind;
  @doc("A path relative to the root manifest directory. A cycle, a path outside the root, and a remote URL are all refused.")
  includes?: string[];
  @doc("The desired resources, keyed by kind, handed to the public command surface of the owning context. Any JSON value, because each kind carries the request shape of its own context and the set of kinds is open; an unknown kind and a duplicate logical key are both rejected. Seeding has no HTTP surface, so this shape is the manifest file rather than a request body.")
  resources: unknown;
  @doc("The bounded generators the profile and the explicit seed resolve to. Any JSON value, for the same reason as `resources`: each generator kind carries its own parameters. Generated performance users are login-disabled and hold no secret.")
  generators?: unknown;
}


@doc("The input for planning or applying a profile. `profile` and `environment` are required, and raising a count has to be stated explicitly.")
model SeedRequest {
  environment: SeedEnvironment;
  profile: SeedProfileKind;
  @doc("An explicit root manifest path. Left out, the repository default for the profile is used.")
  manifest_path?: string;
  mode: SeedMode;
  tenant_id?: string;
  generator_seed?: string;
  @doc("The URIs to configure on the first-party client during a production bootstrap. In production a bootstrap run requires at least one, and a localhost URI is not accepted.")
  first_party_redirect_uris?: string[];
  count?: int64;
  @doc("Explicitly allows a count above the usual ceiling of the performance profile. A value above the absolute ceiling is still refused.")
  allow_large?: boolean;
  @doc("How many records one batch applies while keeping the performance profile within bounded memory. Left out, a safe default is used; a value above the ceiling is refused.")
  batch_size?: int64;
}


@doc("The plan a seed run produced. It carries no secret and no full PII.")
model SeedPlan {
  creates: int64;
  updates: int64;
  noops: int64;
  conflicts: int64;
  @doc("A streamable rendering of the planned operations: logical key, operation kind, and a redacted summary. Any JSON value, because the summary shape follows the kind of each operation.")
  operations: unknown;
}


@doc("The environment policy, the profile, a count ceiling, or an unknown key broke a safety rule. A seeding-tool domain condition, not an HTTP response body: seeding has no HTTP surface.")
model SeedRejectedError {

}


@doc("A seed-managed target has manual drift and cannot be overwritten without an explicit reconcile. A seeding-tool domain condition, not an HTTP response body.")
model SeedConflictError {

}

}
