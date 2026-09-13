package server_http_test

// 運用資材 (infra/) と、組み立てた router が実際に登録する経路との対応を固定する。
//
// kustomize と kubeconform が見るのはスキーマと構文だけなので、プローブのパスを
// /healthz へ書き換えても ServiceMonitor の path を /metric にしても、資材の側の検査は
// 何も落ちない。落ちるのは、資材と Register が登録した経路を突き合わせる検査だけである。
// infra/schema/postgres.sql を読む db_postgres/schema_test.go と同じ形を採る。

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"

	"github.com/goccy/go-yaml"
	"github.com/labstack/echo/v5"
)

// repositoryRoot は backend/shared/http/server_http からリポジトリ root への相対パス。
const repositoryRoot = "../../../../"

// registeredGETRoutes は、組み立てた router が GET で登録しているパスの集合。資材の側が
// 名指すパスをこの集合と突き合わせる。
func registeredGETRoutes(t *testing.T) map[string]bool {
	t.Helper()
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{})
	registered := make(map[string]bool)
	for _, route := range e.Router().Routes() {
		if route.Method == http.MethodGet {
			registered[route.Path] = true
		}
	}
	if len(registered) == 0 {
		t.Fatal("no GET route was assembled; the comparison would pass vacuously")
	}
	return registered
}

func readAsset(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(repositoryRoot + path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

// decodeDocuments は複数ドキュメントの YAML を 1 つずつ T へ読む。
func decodeDocuments[T any](t *testing.T, path string, raw []byte) []T {
	t.Helper()
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	documents := make([]T, 0, 2)
	for {
		var document T
		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		documents = append(documents, document)
	}
	if len(documents) == 0 {
		t.Fatalf("%s declared no document", path)
	}
	return documents
}

type probe struct {
	HTTPGet struct {
		Path string `yaml:"path"`
	} `yaml:"httpGet"`
}

type container struct {
	Name           string `yaml:"name"`
	StartupProbe   probe  `yaml:"startupProbe"`
	LivenessProbe  probe  `yaml:"livenessProbe"`
	ReadinessProbe probe  `yaml:"readinessProbe"`
}

type workloadManifest struct {
	Kind string `yaml:"kind"`
	Spec struct {
		Template struct {
			Spec struct {
				Containers []container `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

// 運用マニフェストの 3 つのプローブが、それぞれ StartupProbe / LivenessProbe /
// ReadinessProbe として登録された経路を呼ぶことを、両方向で確かめる。資材が名指す
// パスが登録されていないこと (綴り違い) と、登録されているのに資材が呼んでいないこと
// (プローブの欠落) は別の誤りであり、片方向の照合では後者が残る。
//
//spec:covers EX-SYSTEM-001-01: 運用マニフェストの生存・受付可否・起動完了の各プローブが、登録済みの /livez /readyz /startupz をそれぞれ呼ぶこと。
func TestOperationalManifestProbesCallTheRegisteredEndpoints(t *testing.T) {
	t.Parallel()

	registered := registeredGETRoutes(t)
	const manifest = "infra/k8s/base/api.yaml"

	var api container
	found := false
	for _, document := range decodeDocuments[workloadManifest](t, manifest, readAsset(t, manifest)) {
		if document.Kind != "Deployment" {
			continue
		}
		for _, declared := range document.Spec.Template.Spec.Containers {
			if declared.Name == "api" {
				api, found = declared, true
			}
		}
	}
	if !found {
		t.Fatalf("%s declares no Deployment container named api", manifest)
	}

	declared := map[string]string{
		"startupProbe":   api.StartupProbe.HTTPGet.Path,
		"livenessProbe":  api.LivenessProbe.HTTPGet.Path,
		"readinessProbe": api.ReadinessProbe.HTTPGet.Path,
	}
	for name, path := range declared {
		if path == "" {
			t.Errorf("%s declares no httpGet path for %s", manifest, name)
			continue
		}
		if !registered[path] {
			t.Errorf("%s probes %s at %q, which the assembled router does not register", manifest, name, path)
		}
	}
	for _, want := range []string{"/startupz", "/livez", "/readyz"} {
		if !registered[want] {
			t.Fatalf("the assembled router no longer registers %q; the manifest comparison lost its subject", want)
		}
		probed := false
		for _, path := range declared {
			if path == want {
				probed = true
			}
		}
		if !probed {
			t.Errorf("%s: no probe calls the registered endpoint %q", manifest, want)
		}
	}
}

type serviceMonitor struct {
	Kind string `yaml:"kind"`
	Spec struct {
		Endpoints []struct {
			Path string `yaml:"path"`
		} `yaml:"endpoints"`
	} `yaml:"spec"`
}

type prometheusConfig struct {
	ScrapeConfigs []struct {
		JobName     string `yaml:"job_name"`
		MetricsPath string `yaml:"metrics_path"`
	} `yaml:"scrape_configs"`
}

// standardScrapeTargetsMetrics は、Operator を持たない配備が使う標準のスクレイプ設定が
// 登録済みの MetricsExposition を指すことを確かめる。2 つの具体例が同じ観測を要求するので
// 1 か所に置く。
func standardScrapeTargetsMetrics(t *testing.T, registered map[string]bool) {
	t.Helper()
	const standard = "infra/docker/prometheus.yml"
	config := decodeDocuments[prometheusConfig](t, standard, readAsset(t, standard))[0]
	if len(config.ScrapeConfigs) == 0 {
		t.Fatalf("%s declares no scrape_configs; a deployment without the Operator would collect nothing", standard)
	}
	for _, scrape := range config.ScrapeConfigs {
		if !registered[scrape.MetricsPath] {
			t.Errorf("%s job %q scrapes %q, which the assembled router does not register", standard, scrape.JobName, scrape.MetricsPath)
		}
	}
}

// Prometheus の 2 つの収集経路 — Operator の ServiceMonitor と、Operator を持たない配備が
// 使う標準のスクレイプ設定 — が、どちらも登録済みの MetricsExposition を指すことを確かめる。
// 片方だけを見る検査は、一方の配備だけが収集を失う書き換えを通す。
//
//spec:covers EX-SYSTEM-001-01: Prometheus の収集設定が、登録済みの /metrics を収集対象として名指すこと。
func TestMonitoringAssetsScrapeTheRegisteredMetricsEndpoint(t *testing.T) {
	t.Parallel()

	registered := registeredGETRoutes(t)
	if !registered["/metrics"] {
		t.Fatal("the assembled router no longer registers /metrics; the monitoring comparison lost its subject")
	}

	const monitor = "infra/k8s/monitoring/operator/servicemonitor.yaml"
	endpoints := 0
	for _, document := range decodeDocuments[serviceMonitor](t, monitor, readAsset(t, monitor)) {
		if document.Kind != "ServiceMonitor" {
			continue
		}
		for _, endpoint := range document.Spec.Endpoints {
			endpoints++
			if !registered[endpoint.Path] {
				t.Errorf("%s scrapes %q, which the assembled router does not register", monitor, endpoint.Path)
			}
		}
	}
	if endpoints == 0 {
		t.Errorf("%s declares no ServiceMonitor endpoint", monitor)
	}

	standardScrapeTargetsMetrics(t, registered)
}

type kustomization struct {
	Resources []string `yaml:"resources"`
}

// Prometheus Operator を持たない配備では ServiceMonitor を適用できない。既定の監視資材が
// operator の overlay を含まず、標準のスクレイプ設定が同じ MetricsExposition を収集する
// ことを確かめる。既定へ operator を足すと、CRD の無いクラスターでは kustomize が通った
// あとの適用が落ちる。
//
//spec:covers EX-SYSTEM-001-03: 既定の監視資材が ServiceMonitor を含まず、標準の Prometheus スクレイプ設定が同じ /metrics を収集すること。
func TestServiceMonitorStaysOutOfTheDefaultMonitoringSet(t *testing.T) {
	t.Parallel()

	const base = "infra/k8s/monitoring/kustomization.yaml"
	baseSet := decodeDocuments[kustomization](t, base, readAsset(t, base))[0]
	if len(baseSet.Resources) == 0 {
		t.Fatalf("%s declares no resource; the default monitoring set would apply nothing", base)
	}
	for _, resource := range baseSet.Resources {
		if strings.Contains(resource, "operator") {
			t.Errorf("%s includes %q; the ServiceMonitor must stay opt-in for clusters without the Prometheus Operator", base, resource)
		}
	}

	const overlay = "infra/k8s/monitoring/operator/kustomization.yaml"
	overlaySet := decodeDocuments[kustomization](t, overlay, readAsset(t, overlay))[0]
	if len(overlaySet.Resources) == 0 {
		t.Fatalf("%s declares no resource; the opt-in overlay would apply nothing", overlay)
	}

	standardScrapeTargetsMetrics(t, registeredGETRoutes(t))
}
