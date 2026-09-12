package bootstrap

// Seeding が宣言する拒否について、返るエラーと「拒否が起こさなかったこと」の両方を確かめる。
//
// この Context の拒否が守っているのは実行中のリクエストではなく、本番環境の初期状態である。
// env シークレットプロバイダーの拒否も、development / performance プロファイルの拒否も、
// localhost リダイレクト URI の拒否も、「本番に開発用の資格情報と設定を書き込ませない」
// ための防護であり、素通りすれば既知の秘密を持つ管理者が本番に出来上がる。
//
// 入口は `bootstrap.Seed` である。CLI (`idmagic-seed`) とサーバー起動の両方がここを通り、
// マニフェストの読み込み、シークレットの解決、書き込みはすべてこの中で順に起こる。
// 検証関数を直接呼ぶテストは、その関数が投入経路から呼ばれていることを示さないので、
// この Context の解消条件を満たさない。
//
// # 「シークレットの解決が呼ばれていない」をどう読むか
//
// シナリオ 4 件は拒否の位置まで規範に書いている。解決が走った時点で、秘密はプロセスの
// メモリに載る。`Seed` は `os.Getenv` を内側で束ねており、呼び出し回数を外から数える
// 継ぎ目は無い。そこで、確かめたい拒否のマニフェストには必ず解決できないシークレット参照を
// 置く (`seedRefusalCanaryEnv` はどのテストでも設定しない)。解決へ進んだ実装は
// `seedSecretUnavailable` を返すので、宣言された拒否のエラーが返ったことが、そのまま
// 「解決の手前で止まった」ことの観測になる。観測点を作るために境界を作り替えるより、
// 確かめたい防護をそのまま残せる。

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/seeding/domain"
	manifestadapter "github.com/ambi/idmagic/backend/seeding/manifests_yaml"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	// seedRefusalCanaryEnv はどのテストでも設定しない環境変数名。これを参照する
	// シークレットは解決できないので、解決へ進んだ実装だけが seedSecretUnavailable を返す。
	seedRefusalCanaryEnv = "IDMAGIC_SEED_REFUSAL_CANARY"
	// seedSecretUnavailable は SecretResolver が env シークレットを解決できないときの文言。
	seedSecretUnavailable = "seed env secret is unavailable"

	seedProductionRedirectURI = "https://portal.example.com/callback"
	seedAdminConsoleClientID  = "00000000-0000-4000-8000-000000000022"
)

// seedStoreState は投入対象ストアの中身を、呼び出し元から見える形で数える。
// 「1 件も書き込まれていない」は、投入の前後で両方読まないと何も示さない。
type seedStoreState struct {
	clients int
	users   int
	groups  int
}

func readSeedStoreState(t *testing.T, deps *Dependencies) seedStoreState {
	t.Helper()
	ctx := context.Background()
	clients, err := deps.OAuth2.ClientRepo.FindAll(ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatalf("FindAll(clients): %v", err)
	}
	users, err := deps.IdManagement.UserRepo.Count(ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatalf("Count(users): %v", err)
	}
	groups, err := deps.IdManagement.GroupRepo.ListAll(ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatalf("ListAll(groups): %v", err)
	}
	return seedStoreState{clients: len(clients), users: int(users), groups: len(groups)}
}

func newSeedRefusalDeps(t *testing.T) *Dependencies {
	t.Helper()
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatalf("assembleMemory: %v", err)
	}
	if before := readSeedStoreState(t, deps); before != (seedStoreState{}) {
		t.Fatalf("投入前のストアが空でない: %+v", before)
	}
	return deps
}

// assertRefusedBeforeSecretsAndWrites は、返ったエラーが宣言された拒否であり、
// シークレットの解決へ進んでいないこと、そしてストアが空のままであることを確かめる。
func assertRefusedBeforeSecretsAndWrites(t *testing.T, deps *Dependencies, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("拒否されるべき投入が成功した")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), want)
	}
	if strings.Contains(err.Error(), seedSecretUnavailable) {
		t.Fatalf("拒否よりも先にシークレットの解決へ進んでいる: %q", err.Error())
	}
	if after := readSeedStoreState(t, deps); after != (seedStoreState{}) {
		t.Fatalf("拒否されたのにストアが変わっている: %+v", after)
	}
}

// writeSeedManifest はテスト用のマニフェストを書き、その絶対パスを返す。
func writeSeedManifest(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// bootstrapManifestBody は first-party クライアントだけを持つ bootstrap マニフェストを返す。
// secrets には解決できない env 参照を置くので、解決へ進んだ実装だけが別のエラーを返す。
func bootstrapManifestBody(extra string) string {
	return `schema_version: "1"
profile: bootstrap
resources:
  - kind: first_party_clients
    logical_key: first-party-portals
    clients:
      - {id: ` + seedAdminConsoleClientID + `, name: IdMagic Admin Console, scope: "openid profile idmagic.admin offline_access"}
    secrets:
      canary:
        provider: env
        locator: ` + seedRefusalCanaryEnv + `
        version: v1
` + extra
}

// 書き込みの前に拒否される。
//
// 素通りすれば、development のマニフェストが bootstrap のつもりの実行で適用される。
// マニフェストとプロファイルの対応は、どの資格情報がどの環境へ入るかの唯一の対応表である。
//
//spec:covers EX-SEEDING-003-01: 指定したプロファイルと異なるマニフェストは、シークレットの解決と
func TestSeedRefusesManifestProfileMismatchBeforeSecretsAndWrites(t *testing.T) {
	deps := newSeedRefusalDeps(t)
	testManifest, err := manifestadapter.LocateDefaultPath(domain.ProfileTest)
	if err != nil {
		t.Fatal(err)
	}

	// 要求は development プロファイル、マニフェストは test プロファイル。
	// 要求そのものは妥当なので、拒否はマニフェストとの不一致だけに由来する。
	_, seedErr := Seed(context.Background(), deps, domain.Request{
		Environment:  domain.EnvironmentDevelopment,
		Profile:      domain.ProfileDevelopment,
		Mode:         domain.ModeApply,
		ManifestPath: testManifest,
	}, "")
	assertRefusedBeforeSecretsAndWrites(t, deps, seedErr, "does not match request profile")

	// 対照: 同じマニフェストを、対応するプロファイルと環境で計画すると通る。
	// 不一致だけが止めていたことを示す。
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	plan, err := Seed(context.Background(), newSeedRefusalDeps(t), domain.Request{
		Environment:  domain.EnvironmentTest,
		Profile:      domain.ProfileTest,
		Mode:         domain.ModeDryRun,
		ManifestPath: testManifest,
	}, "")
	if err != nil {
		t.Fatalf("対照の計画が失敗した: %v", err)
	}
	if plan.Count(domain.OperationCreate) == 0 {
		t.Fatalf("対照の計画に create が無い: %+v", plan)
	}
}

// `include` の循環、ルート外のパスは、いずれもシークレットの解決と書き込みの前に拒否される。
//
// マニフェストは投入の入力そのものなので、壊れた入力を部分的に適用すると、
// 直すために何が入ったかを先に調べなければならなくなる。
//
//spec:covers EX-SEEDING-004-01: 未知のキー、重複する論理キー、未対応のスキーマバージョン、
func TestSeedRefusesInvalidManifestBeforeSecretsAndWrites(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		build func(t *testing.T) string
		want  string
	}{
		{
			name: "unknown key",
			build: func(t *testing.T) string {
				t.Helper()
				return writeSeedManifest(t, t.TempDir(), "main.yaml", bootstrapManifestBody("unknown_key: surprise\n"))
			},
			want: "decode seed manifest",
		},
		{
			name: "duplicate logical key",
			build: func(t *testing.T) string {
				t.Helper()
				return writeSeedManifest(t, t.TempDir(), "main.yaml", bootstrapManifestBody(
					`  - kind: first_party_clients
    logical_key: first-party-portals
    clients:
      - {id: 00000000-0000-4000-8000-000000000023, name: IdMagic Account Portal, scope: "openid"}
`))
			},
			want: "duplicate seed logical key",
		},
		{
			name: "unsupported schema version",
			build: func(t *testing.T) string {
				t.Helper()
				body := strings.Replace(bootstrapManifestBody(""), `schema_version: "1"`, `schema_version: "999"`, 1)
				return writeSeedManifest(t, t.TempDir(), "main.yaml", body)
			},
			want: "unsupported seed manifest schema version",
		},
		{
			name: "include cycle",
			build: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeSeedManifest(t, dir, "other.yaml", `schema_version: "1"
profile: bootstrap
includes: [main.yaml]
`)
				return writeSeedManifest(t, dir, "main.yaml", bootstrapManifestBody("includes: [other.yaml]\n"))
			},
			want: "include cycle",
		},
		{
			name: "include escapes the root",
			build: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeSeedManifest(t, dir, "outside.yaml", `schema_version: "1"
profile: bootstrap
`)
				return writeSeedManifest(t, filepath.Join(dir, "manifests"), "main.yaml",
					bootstrapManifestBody("includes: [../outside.yaml]\n"))
			},
			want: "escapes root",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newSeedRefusalDeps(t)

			_, err := Seed(context.Background(), deps, domain.Request{
				Environment:  domain.EnvironmentDevelopment,
				Profile:      domain.ProfileBootstrap,
				Mode:         domain.ModeApply,
				ManifestPath: testCase.build(t),
			}, "")

			assertRefusedBeforeSecretsAndWrites(t, deps, err, testCase.want)
		})
	}

	// 対照: 同じ形の正しい bootstrap マニフェストは、同じ入口から通って書き込む。
	// 壊れたマニフェストだけが止められていることを示す。
	t.Run("valid manifest applies", func(t *testing.T) {
		deps := newSeedRefusalDeps(t)
		valid := writeSeedManifest(t, t.TempDir(), "main.yaml", `schema_version: "1"
profile: bootstrap
resources:
  - kind: first_party_clients
    logical_key: first-party-portals
    clients:
      - {id: `+seedAdminConsoleClientID+`, name: IdMagic Admin Console, scope: "openid"}
`)
		if _, err := Seed(context.Background(), deps, domain.Request{
			Environment:  domain.EnvironmentDevelopment,
			Profile:      domain.ProfileBootstrap,
			Mode:         domain.ModeApply,
			ManifestPath: valid,
		}, ""); err != nil {
			t.Fatalf("対照の投入が失敗した: %v", err)
		}
		if state := readSeedStoreState(t, deps); state.clients != 1 {
			t.Fatalf("対照で作られたクライアント数 = %d, want 1", state.clients)
		}
	})
}

// シークレットの解決と書き込みの前に拒否され、永続状態は変更されない。
//
// 素通りすれば、本番の資格情報がプロセスの環境変数から入る。環境変数はプロセス一覧、
// コンテナの設定、クラッシュダンプに現れるので、そこに置かれた秘密はもう秘密ではない。
//
//spec:covers EX-SEEDING-005-01: 本番で env シークレットプロバイダーを参照するマニフェストは、
func TestSeedRefusesEnvSecretProviderInProductionBeforeResolvingIt(t *testing.T) {
	for _, mode := range []domain.Mode{domain.ModeDryRun, domain.ModeApply} {
		t.Run(string(mode), func(t *testing.T) {
			deps := newSeedRefusalDeps(t)
			manifest := writeSeedManifest(t, t.TempDir(), "main.yaml", bootstrapManifestBody(""))

			// 要求そのものは本番の bootstrap として妥当 (リダイレクト URI も満たす)。
			// 残る違反は env シークレットプロバイダーだけである。
			_, err := Seed(context.Background(), deps, domain.Request{
				Environment:            domain.EnvironmentProduction,
				Profile:                domain.ProfileBootstrap,
				Mode:                   mode,
				ManifestPath:           manifest,
				FirstPartyRedirectURIs: []string{seedProductionRedirectURI},
			}, "")

			assertRefusedBeforeSecretsAndWrites(t, deps, err, `provider "env" is not permitted in production`)
		})
	}

	// 対照: 同じ本番の要求でも、file プロバイダーなら解決して書き込む。
	// 拒否がプロバイダーの種別に由来し、本番であること自体ではないことを示す。
	t.Run("file provider applies in production", func(t *testing.T) {
		deps := newSeedRefusalDeps(t)
		secretRoot := t.TempDir()
		writeSeedManifest(t, secretRoot, "client_secret", "production-client-secret")
		manifest := writeSeedManifest(t, t.TempDir(), "main.yaml", `schema_version: "1"
profile: bootstrap
resources:
  - kind: first_party_clients
    logical_key: first-party-portals
    clients:
      - {id: `+seedAdminConsoleClientID+`, name: IdMagic Admin Console, scope: "openid"}
    secrets:
      canary:
        provider: file
        locator: client_secret
        version: v1
`)
		if _, err := Seed(context.Background(), deps, domain.Request{
			Environment:            domain.EnvironmentProduction,
			Profile:                domain.ProfileBootstrap,
			Mode:                   domain.ModeApply,
			ManifestPath:           manifest,
			FirstPartyRedirectURIs: []string{seedProductionRedirectURI},
		}, secretRoot); err != nil {
			t.Fatalf("対照の投入が失敗した: %v", err)
		}
		if state := readSeedStoreState(t, deps); state.clients != 1 {
			t.Fatalf("対照で作られたクライアント数 = %d, want 1", state.clients)
		}
	})
}

// 既知のデモ資格情報は作成されない。
//
// development マニフェストは alice と root を固定の UUID と固定のパスワードで作る。
// この拒否が素通りすれば、公開リポジトリに書いてある資格情報を持つ system_admin が
// 本番に出来上がる。
//
//spec:covers EX-SEEDING-007-01: 本番では development と performance のプロファイルを書き込み前に拒否し、
func TestSeedRefusesDevelopmentAndPerformanceProfilesInProduction(t *testing.T) {
	for _, testCase := range []struct {
		profile domain.Profile
		count   int
	}{
		{profile: domain.ProfileDevelopment},
		{profile: domain.ProfilePerformance, count: 5},
	} {
		t.Run(string(testCase.profile), func(t *testing.T) {
			for _, mode := range []domain.Mode{domain.ModeDryRun, domain.ModeApply} {
				t.Run(string(mode), func(t *testing.T) {
					deps := newSeedRefusalDeps(t)

					_, err := Seed(context.Background(), deps, domain.Request{
						Environment: domain.EnvironmentProduction,
						Profile:     testCase.profile,
						Mode:        mode,
						Count:       testCase.count,
					}, "")

					assertRefusedBeforeSecretsAndWrites(t, deps, err,
						`profile "`+string(testCase.profile)+`" is not permitted in production`)
					// デモの利用者そのものを名指しで確かめる。件数だけでは、
					// 別の理由で 0 件だったのか拒否が効いたのかを取り違える。
					alice, err := deps.IdManagement.UserRepo.FindBySub(context.Background(), seedUserAliceID)
					if err != nil || alice != nil {
						t.Fatalf("デモ利用者 = %#v, err = %v; want nil, nil", alice, err)
					}
				})
			}
		})
	}

	// 対照: 同じ development プロファイルを development 環境で適用するとデモ利用者が作られる。
	// 拒否が本番であることに由来し、マニフェストの不備ではないことを示す。
	t.Run("development applies outside production", func(t *testing.T) {
		t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
		t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
		deps := newSeedRefusalDeps(t)
		if _, err := Seed(context.Background(), deps, domain.Request{
			Environment: domain.EnvironmentDevelopment,
			Profile:     domain.ProfileDevelopment,
			Mode:        domain.ModeApply,
		}, ""); err != nil {
			t.Fatalf("対照の投入が失敗した: %v", err)
		}
		alice, err := deps.IdManagement.UserRepo.FindBySub(context.Background(), seedUserAliceID)
		if err != nil || alice == nil {
			t.Fatalf("対照でデモ利用者が作られていない: %#v, %v", alice, err)
		}
	})
}

// 書き込み前に拒否され、bootstrap のクライアントは作られない。
//
// 指定した URI だけをリダイレクト URI として持つ。
//
// 拒否が素通りすると、`seedContributor.operations` の既定値が効いて localhost の
// リダイレクト URI を持つ本番クライアントが出来上がる。localhost へ返す認可コードは、
// 利用者の端末で動く任意のプロセスが受け取れる。
//
//spec:covers EX-SEEDING-008-02: 本番の bootstrap で、未指定・localhost・HTTP のリダイレクト URI は
//spec:covers EX-SEEDING-008-01: 明示した https の URI を指定すると、ファーストパーティークライアントは
func TestSeedRefusesProductionBootstrapRedirectURIsBeforeWriting(t *testing.T) {
	for _, testCase := range []struct {
		name string
		uris []string
	}{
		{name: "unspecified", uris: nil},
		{name: "localhost", uris: []string{"https://localhost:3000/callback"}},
		{name: "http", uris: []string{"http://portal.example.com/callback"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			deps := newSeedRefusalDeps(t)
			manifest := writeSeedManifest(t, t.TempDir(), "main.yaml", bootstrapManifestBody(""))

			_, err := Seed(context.Background(), deps, domain.Request{
				Environment:            domain.EnvironmentProduction,
				Profile:                domain.ProfileBootstrap,
				Mode:                   domain.ModeApply,
				ManifestPath:           manifest,
				FirstPartyRedirectURIs: testCase.uris,
			}, "")

			assertRefusedBeforeSecretsAndWrites(t, deps, err, "redirect URI")
		})
	}

	// EX-SEEDING-008-01 の通常経路。拒否の対照でもあり、宣言そのものの検証でもある。
	t.Run("explicit https URIs are the only ones configured", func(t *testing.T) {
		deps := newSeedRefusalDeps(t)
		manifest := writeSeedManifest(t, t.TempDir(), "main.yaml", `schema_version: "1"
profile: bootstrap
resources:
  - kind: first_party_clients
    logical_key: first-party-portals
    clients:
      - {id: `+seedAdminConsoleClientID+`, name: IdMagic Admin Console, scope: "openid"}
`)
		if _, err := Seed(context.Background(), deps, domain.Request{
			Environment:            domain.EnvironmentProduction,
			Profile:                domain.ProfileBootstrap,
			Mode:                   domain.ModeApply,
			ManifestPath:           manifest,
			FirstPartyRedirectURIs: []string{seedProductionRedirectURI},
		}, ""); err != nil {
			t.Fatalf("本番 bootstrap の投入が失敗した: %v", err)
		}
		client, err := deps.OAuth2.ClientRepo.FindByID(
			context.Background(), tenancydomain.DefaultTenantID, seedAdminConsoleClientID)
		if err != nil || client == nil {
			t.Fatalf("bootstrap クライアントが作られていない: %#v, %v", client, err)
		}
		if !slices.Equal(client.RedirectURIs, []string{seedProductionRedirectURI}) {
			t.Fatalf("redirect URIs = %v, want exactly the requested one", client.RedirectURIs)
		}
	})
}

// 扱われ、手動変更は維持される。
//
// 規範は「投入が失敗したか」ではなく「値が守られたか」である。運用者が本番で直した値を
// 次の投入が黙って戻すなら、seed は運用の敵になる。
//
// マニフェスト由来のファーストパーティークライアントとデモクライアントは、それぞれ別の
// 判定でドリフトを見ている。片方だけを確かめると、もう片方が上書きに退化しても気づけない。
//
//spec:covers EX-SEEDING-009-01: seed 管理対象の論理キーが手動で変更されているとき、再適用は競合として
func TestSeedRefusesManualDriftAndKeepsTheChangedValue(t *testing.T) {
	for _, clientID := range []string{seedAdminConsoleClientID, seedDemoClientID} {
		t.Run(clientID, func(t *testing.T) {
			t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
			t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
			ctx := context.Background()
			deps := newSeedRefusalDeps(t)
			request := domain.Request{
				Environment: domain.EnvironmentDevelopment, Profile: domain.ProfileDevelopment, Mode: domain.ModeApply,
			}
			if _, err := Seed(ctx, deps, request, ""); err != nil {
				t.Fatalf("初回の投入が失敗した: %v", err)
			}

			// 対照: ドリフトが無ければ、同じ要求は何度でも収束する。
			// これが無いと、常に失敗する投入と「ドリフトを拒否する投入」を区別できない。
			if _, err := Seed(ctx, deps, request, ""); err != nil {
				t.Fatalf("ドリフトの無い再適用が失敗した: %v", err)
			}

			client, err := deps.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, clientID)
			if err != nil || client == nil {
				t.Fatalf("FindByID(%s) = %#v, %v", clientID, client, err)
			}
			const manual = "manual-drift"
			client.Scope = manual
			if err := deps.OAuth2.ClientRepo.Save(ctx, client); err != nil {
				t.Fatalf("Save(manual drift): %v", err)
			}

			if _, err := Seed(ctx, deps, request, ""); err == nil {
				t.Fatal("手動変更があるのに再適用が成功した")
			} else if !strings.Contains(err.Error(), "seed drift") {
				t.Fatalf("error = %q, want the drift conflict", err.Error())
			}

			kept, err := deps.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, clientID)
			if err != nil || kept == nil {
				t.Fatalf("FindByID(after drift) = %#v, %v", kept, err)
			}
			if kept.Scope != manual {
				t.Fatalf("scope = %q, want the manual value %q to survive", kept.Scope, manual)
			}
		})
	}
}
