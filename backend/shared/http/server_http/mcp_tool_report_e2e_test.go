package server_http_test

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-MCPGATEWAY-011, REQ-OAUTH2-086: 配布する設定の Gateway が実在の MCP サーバーの一覧を報告し、報告は公開設定と割当を変えず、未公開のツールは転送されない。
//verify:operation ReportResourceTools: フォームの tools と complete を受け、保存した報告の名前と引数が上流の定義と一致する。
func TestGatewayReportsUpstreamToolsWithoutPublishing(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	ctx := context.Background()
	before, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || before == nil {
		t.Fatalf("server=%v err=%v", before, err)
	}

	result, err := f.loaded.ReportTools(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Tools != 2 || !result.Complete {
		t.Fatalf("result=%+v, want two complete tools", result)
	}

	report, err := f.s.McpToolReports.Find(ctx, tenancydomain.DefaultTenantID, before.ID)
	if err != nil || report == nil {
		t.Fatalf("report=%v err=%v", report, err)
	}
	names := []string{}
	for _, tool := range report.Tools {
		names = append(names, tool.Name)
		if tool.Name == "read_document" && !slices.Equal(tool.Arguments, []string{"document_id"}) {
			t.Fatalf("read_document arguments=%v", tool.Arguments)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"delete_document", "read_document"}) || !report.Complete {
		t.Fatalf("reported=%v complete=%v", names, report.Complete)
	}

	after, err := f.s.McpResourceServers.FindByID(ctx, tenancydomain.DefaultTenantID, before.ID)
	if err != nil || !after.UpdatedAt.Equal(before.UpdatedAt) || !reflect.DeepEqual(after.AccessRules, before.AccessRules) {
		t.Fatalf("the report changed the registration: before=%+v after=%+v err=%v", before, after, err)
	}

	// 報告しただけのツールは公開されておらず、割当もないので上流で実行されない。
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	response, body := f.post(t, "delete_document", gatewayBearer(token))
	if response.StatusCode != http.StatusForbidden || f.deleted.Load() != 0 {
		t.Fatalf("status=%d body=%s deleted=%d", response.StatusCode, body, f.deleted.Load())
	}
}

//spec:covers REQ-OAUTH2-086: 上限を越える報告と、Active の登録に束縛されていないクライアントの報告を拒否し、前回の報告を保つ。
//verify:operation ReportResourceTools: 204 と no-store で報告を保存し、形の誤りは 400、Active の登録がないクライアントは 403 で拒否する。
func TestToolReportsRejectInvalidReportsAndKeepThePreviousOne(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	ctx := context.Background()
	if _, err := f.loaded.ReportTools(ctx); err != nil {
		t.Fatal(err)
	}
	server, err := f.s.McpResourceServers.FindByResource(ctx, tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("server=%v err=%v", server, err)
	}
	report := func(tools, complete string) int {
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
		stored, err := f.s.McpToolReports.Find(ctx, tenancydomain.DefaultTenantID, server.ID)
		if err != nil || stored == nil || len(stored.Tools) != 2 || !stored.Complete {
			t.Fatalf("the previous report was not kept: %+v %v", stored, err)
		}
	}
	for name, body := range map[string][2]string{
		"not an array":     {`{"name":"a"}`, "true"},
		"colon in a name":  {`[{"name":"a:b"}]`, "true"},
		"unknown complete": {`[{"name":"a"}]`, "yes"},
	} {
		if status := report(body[0], body[1]); status != http.StatusBadRequest {
			t.Fatalf("%s: status=%d, want 400", name, status)
		}
		kept()
	}

	server.State = oauthdomain.McpResourceServerDisabled
	if err := f.s.McpResourceServers.Save(ctx, server); err != nil {
		t.Fatal(err)
	}
	if status := report(`[{"name":"a"}]`, "true"); status != http.StatusForbidden {
		t.Fatalf("disabled registration status=%d, want 403", status)
	}
	kept()
}
