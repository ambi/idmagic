package mcpgateway_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/goccy/go-yaml"
)

type exampleMatcher struct {
	Exact  string `yaml:"exact"`
	Prefix string `yaml:"prefix"`
}

type exampleMatchers struct {
	Patterns []exampleMatcher `yaml:"patterns"`
}

type exampleExtAuthz struct {
	FailureModeAllow *bool `yaml:"failure_mode_allow"`
	StatusOnError    struct {
		Code int `yaml:"code"`
	} `yaml:"status_on_error"`
	WithRequestBody struct {
		MaxRequestBytes     int   `yaml:"max_request_bytes"`
		AllowPartialMessage *bool `yaml:"allow_partial_message"`
	} `yaml:"with_request_body"`
	AllowedHeaders exampleMatchers `yaml:"allowed_headers"`
	HTTPService    struct {
		AuthorizationResponse struct {
			AllowedUpstreamHeaders exampleMatchers `yaml:"allowed_upstream_headers"`
			AllowedClientHeaders   exampleMatchers `yaml:"allowed_client_headers"`
		} `yaml:"authorization_response"`
	} `yaml:"http_service"`
	Disabled bool `yaml:"disabled"`
}

type exampleRoute struct {
	Match struct {
		Path string `yaml:"path"`
	} `yaml:"match"`
	Route struct {
		Cluster string `yaml:"cluster"`
	} `yaml:"route"`
	TypedPerFilterConfig    map[string]exampleExtAuthz `yaml:"typed_per_filter_config"`
	ResponseHeadersToRemove []string                   `yaml:"response_headers_to_remove"`
}

type exampleConfig struct {
	StaticResources struct {
		Listeners []struct {
			FilterChains []struct {
				Filters []struct {
					TypedConfig struct {
						RouteConfig struct {
							VirtualHosts []struct {
								Routes []exampleRoute `yaml:"routes"`
							} `yaml:"virtual_hosts"`
						} `yaml:"route_config"`
						HTTPFilters []struct {
							Name        string          `yaml:"name"`
							TypedConfig exampleExtAuthz `yaml:"typed_config"`
						} `yaml:"http_filters"`
					} `yaml:"typed_config"`
				} `yaml:"filters"`
			} `yaml:"filter_chains"`
		} `yaml:"listeners"`
	} `yaml:"static_resources"`
}

const extAuthzFilter = "envoy.filters.http.ext_authz"

func (m exampleMatchers) allows(want exampleMatcher) bool { return slices.Contains(m.Patterns, want) }

// 実物を起動しなくても、迂回につながる設定の誤りを毎回の実行で検出する。Envoy が設定どおりに動くことは
// TestEnvoy* が確かめる。
//
//spec:covers REQ-MCPGATEWAY-007: 設定例は失敗時と断片の本文を通さず、判定に要るヘッダーを渡し、上流への経路で判定を省かない
func TestEnvoyExampleDoesNotAllowABypass(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "envoy-mcp-gateway", "envoy.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var config exampleConfig
	if err := yaml.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.StaticResources.Listeners) != 1 {
		t.Fatalf("listeners = %d; check every listener before adding one", len(config.StaticResources.Listeners))
	}
	manager := config.StaticResources.Listeners[0].FilterChains[0].Filters[0].TypedConfig

	filters := manager.HTTPFilters
	if len(filters) != 2 || filters[0].Name != extAuthzFilter || filters[1].Name != "envoy.filters.http.router" {
		t.Fatalf("http filters must run ext_authz before the router: %+v", filters)
	}
	authz := filters[0].TypedConfig
	if authz.FailureModeAllow == nil || *authz.FailureModeAllow {
		t.Error("failure_mode_allow must be explicitly false")
	}
	if authz.StatusOnError.Code != 503 {
		t.Errorf("status_on_error = %d", authz.StatusOnError.Code)
	}
	body := authz.WithRequestBody
	if body.AllowPartialMessage == nil || *body.AllowPartialMessage {
		t.Error("allow_partial_message must be explicitly false")
	}
	if body.MaxRequestBytes != 4194304 {
		t.Errorf("max_request_bytes = %d, want the gateway default max_body_bytes", body.MaxRequestBytes)
	}
	for _, want := range []exampleMatcher{
		{Exact: "authorization"}, {Exact: "dpop"}, {Exact: "content-type"}, {Exact: "content-encoding"}, {Exact: "origin"}, {Prefix: "mcp-"},
	} {
		if !authz.AllowedHeaders.allows(want) {
			t.Errorf("allowed_headers lacks %+v", want)
		}
	}
	response := authz.HTTPService.AuthorizationResponse
	if len(response.AllowedUpstreamHeaders.Patterns) != 1 || !response.AllowedUpstreamHeaders.allows(exampleMatcher{Exact: "x-request-id"}) {
		t.Errorf("allowed_upstream_headers = %+v, want x-request-id only", response.AllowedUpstreamHeaders)
	}
	for _, want := range []string{"www-authenticate", "allow", "retry-after", "x-request-id", "content-type", "cache-control"} {
		if !response.AllowedClientHeaders.allows(exampleMatcher{Exact: want}) {
			t.Errorf("allowed_client_headers lacks %s", want)
		}
	}

	upstream := 0
	for _, host := range manager.RouteConfig.VirtualHosts {
		for _, route := range host.Routes {
			disabled := route.TypedPerFilterConfig[extAuthzFilter].Disabled
			switch route.Route.Cluster {
			case "mcp_server":
				upstream++
				if disabled {
					t.Errorf("route %s reaches the upstream without ext_authz", route.Match.Path)
				}
				if !slices.Contains(route.ResponseHeadersToRemove, "x-request-id") {
					t.Errorf("route %s keeps the upstream X-Request-ID", route.Match.Path)
				}
			case "mcp_gateway":
				if !disabled {
					t.Errorf("metadata route %s must skip ext_authz, because Envoy reads its 200 as a permit", route.Match.Path)
				}
			default:
				t.Errorf("route %s goes to unknown cluster %q", route.Match.Path, route.Route.Cluster)
			}
		}
	}
	if upstream != 1 {
		t.Errorf("upstream routes = %d", upstream)
	}
}
