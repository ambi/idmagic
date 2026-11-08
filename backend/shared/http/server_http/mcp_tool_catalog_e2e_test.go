package server_http_test

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-MCPGATEWAY-011, REQ-OAUTH2-086: 配布する設定の Gateway が実在の MCP サーバーの一覧をカタログし、カタログは公開設定と割当を変えず、未公開のツールは転送されない。
//verify:operation PutResourceToolCatalog: フォームの tools と complete を受け、保存したカタログの名前と引数が上流の定義と一致する。
func TestGatewaySendsToolCatalogWithoutPublishing(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	ctx := context.Background()
	before, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || before == nil {
		t.Fatalf("server=%v err=%v", before, err)
	}

	result, err := f.loaded.SendToolCatalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tools != 2 || !result.Complete {
		t.Fatalf("result=%+v, want two complete tools", result)
	}

	sendCatalog, err := f.s.McpToolCatalogs.Find(ctx, tenancydomain.DefaultTenantID, before.ID)
	if err != nil || sendCatalog == nil {
		t.Fatalf("sendCatalog=%v err=%v", sendCatalog, err)
	}
	names := []string{}
	for _, tool := range sendCatalog.Tools {
		names = append(names, tool.Name)
		if tool.Name == "read_document" && !slices.Equal(tool.Arguments, []string{"document_id"}) {
			t.Fatalf("read_document arguments=%v", tool.Arguments)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"delete_document", "read_document"}) || !sendCatalog.Complete {
		t.Fatalf("sent=%v complete=%v", names, sendCatalog.Complete)
	}

	after, err := f.s.McpResourceServers.FindByID(ctx, tenancydomain.DefaultTenantID, before.ID)
	if err != nil || !after.UpdatedAt.Equal(before.UpdatedAt) || !reflect.DeepEqual(after.AccessRules, before.AccessRules) {
		t.Fatalf("the sendCatalog changed the registration: before=%+v after=%+v err=%v", before, after, err)
	}

	// カタログしただけのツールは公開されておらず、割当もないので上流で実行されない。
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	response, body := f.post(t, "delete_document", gatewayBearer(token))
	if response.StatusCode != http.StatusForbidden || f.deleted.Load() != 0 {
		t.Fatalf("status=%d body=%s deleted=%d", response.StatusCode, body, f.deleted.Load())
	}
}

//spec:covers REQ-OAUTH2-086: 上限を越えるカタログと、Active の登録に束縛されていないクライアントのカタログを拒否し、前回のカタログを保つ。
//verify:operation PutResourceToolCatalog: 204 と no-store でカタログを保存し、形の誤りは 400、Active の登録がないクライアントは 403 で拒否する。
func TestToolCatalogRejectsInvalidInputAndKeepsThePreviousOne(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	ctx := context.Background()
	if _, err := f.loaded.SendToolCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("server=%v err=%v", server, err)
	}
	sendCatalog := func(tools, complete string) int {
		t.Helper()
		form := url.Values{"tools": {tools}, "complete": {complete}}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, f.idmagic.URL+"/realms/default/resource-access/tools", strings.NewReader(form.Encode()))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		request.SetBasicAuth(evalResourceServerClient, evalResourceServerSecret)
		response, err := f.idmagic.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("Cache-Control=%q", response.Header.Get("Cache-Control"))
		}
		return response.StatusCode
	}
	kept := func() {
		t.Helper()
		stored, err := f.s.McpToolCatalogs.Find(ctx, tenancydomain.DefaultTenantID, server.ID)
		if err != nil || stored == nil || len(stored.Tools) != 2 || !stored.Complete {
			t.Fatalf("the previous sendCatalog was not kept: %+v %v", stored, err)
		}
	}
	for name, body := range map[string][2]string{
		"not an array":     {`{"name":"a"}`, "true"},
		"colon in a name":  {`[{"name":"a:b"}]`, "true"},
		"unknown complete": {`[{"name":"a"}]`, "yes"},
	} {
		if status := sendCatalog(body[0], body[1]); status != http.StatusBadRequest {
			t.Fatalf("%s: status=%d, want 400", name, status)
		}
		kept()
	}

	server.State = oauthdomain.McpResourceServerDisabled
	if err := f.s.McpResourceServers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	if status := sendCatalog(`[{"name":"a"}]`, "true"); status != http.StatusForbidden {
		t.Fatalf("disabled registration status=%d, want 403", status)
	}
	kept()
}

//spec:covers REQ-MCPGATEWAY-012, REQ-OAUTH2-086: 配布する設定の Gateway が実在の MCP サーバーの変更通知を受けて一覧をカタログし直し、IdMagic は直前の一覧を変更前として残す。
//verify:operation PutResourceToolCatalog: 変更通知の後のカタログで、保存した一覧に追加したツールが現れ、変更前の一覧が残る。
func TestGatewayResendsToolCatalogWhenTheUpstreamToolsChange(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	var log bytes.Buffer
	go func() {
		f.loaded.ListenForToolChanges(ctx, &log)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("server=%v err=%v", server, err)
	}
	waitForSendCatalog := func(want int) *oauthdomain.McpToolCatalog {
		t.Helper()
		// 二回目のカタログは、REQ-MCPGATEWAY-012 の間隔のため最初のカタログから10秒後に届く。
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			sendCatalog, err := f.s.McpToolCatalogs.Find(ctx, tenancydomain.DefaultTenantID, server.ID)
			if err != nil {
				t.Fatal(err)
			}
			if sendCatalog != nil && len(sendCatalog.Tools) == want {
				return sendCatalog
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("no sendCatalog with %d tools; log=%s", want, log.String())
		return nil
	}
	waitForSendCatalog(2)

	mcp.AddTool(f.mcp, &mcp.Tool{Name: "write_document"}, func(context.Context, *mcp.CallToolRequest, readDocumentInput) (*mcp.CallToolResult, readDocumentOutput, error) {
		return nil, readDocumentOutput{}, nil
	})
	sendCatalog := waitForSendCatalog(3)
	changes := sendCatalog.Changes()
	if changes == nil || !slices.Equal(changes.Added, []string{"write_document"}) || len(changes.Removed) != 0 || len(sendCatalog.Previous.Tools) != 2 {
		t.Fatalf("changes=%+v previous=%+v", changes, sendCatalog.Previous)
	}
}
