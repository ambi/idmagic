package keys_memory_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/signingkeys/internal/keys_memory"
	contract "github.com/ambi/idmagic/backend/signingkeys/internal/testing_contract"
)

//spec:covers REQ-SIGNINGKEYS-006: メモリの提供元の準備、切り替えと並行要求は共有契約を満たすこと。
func TestPreparedKeyContract(t *testing.T) {
	contract.Run(t, memoryFixture)
}

//spec:covers REQ-SIGNINGKEYS-003: メモリの提供元のスコープの列挙は共有契約を満たすこと。
func TestKeyScopesContract(t *testing.T) {
	contract.RunKeyScopes(t, memoryFixture)
}

func memoryFixture(t *testing.T) contract.Fixture {
	t.Helper()
	store, err := keys_memory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	return contract.Fixture{Store: store, Context: tenantCtx("tenant-a")}
}
