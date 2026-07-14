package spec

// SCL ドキュメント全体のローダー。TS の src/spec-bindings/scl.ts に対応する。

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"

	"github.com/goccy/go-yaml"
)

// SCL は context-first SCL 3.0 の集約ビュー。トップレベル scl.yaml は context_map のみを持ち、
// 各 context ファイル (contexts/*.yaml) の glossary / models / interfaces などをロード時に合成する。
// 合成されるセクションは `yaml:"-"` とし、トップレベルの strict デコードでは対象にしない。
type SCL struct {
	System                 string                     `yaml:"system"`
	SpecVersion            string                     `yaml:"spec_version"`
	ContextMap             map[string]ContextMapEntry `yaml:"context_map"`
	Standards              map[string]Standard        `yaml:"-"`
	Vocabulary             map[string]Vocabulary      `yaml:"-"`
	Models                 map[string]Model           `yaml:"-"`
	Interfaces             map[string]Interface       `yaml:"-"`
	InterfaceContexts      map[string]string          `yaml:"-"`
	States                 map[string]StateMachine    `yaml:"-"`
	Scenarios              map[string]Scenario        `yaml:"-"`
	AuthorizationByContext map[string]Authorization   `yaml:"-"`
	Objectives             map[string]Objective       `yaml:"-"`
	Flows                  map[string]Flow            `yaml:"-"`
	Annotations            map[string]any             `yaml:"annotations"`
}

// ContextMapEntry は context_map の 1 つの bounded context エントリ。
// ownership は対応する context ファイルが定義する模型・interface などで暗黙に決まる。
type ContextMapEntry struct {
	Description string                       `yaml:"description"`
	Path        string                       `yaml:"path"`
	Publishes   []string                     `yaml:"publishes"`
	DependsOn   map[string]ContextDependency `yaml:"depends_on"`
	Annotations map[string]any               `yaml:"annotations"`
}

// ContextDependency は depends_on の 1 つの依存関係 (via / uses / reason)。
type ContextDependency struct {
	Via    string   `yaml:"via"`
	Uses   []string `yaml:"uses"`
	Reason string   `yaml:"reason"`
}

// contextDocument は 1 つの context ファイル (contexts/*.yaml) のデコード対象。
// 各セクションは集約 SCL に合成される。
type contextDocument struct {
	System        string                  `yaml:"system"`
	SpecVersion   string                  `yaml:"spec_version"`
	Context       string                  `yaml:"context"`
	Standards     map[string]Standard     `yaml:"standards"`
	Glossary      map[string]Vocabulary   `yaml:"glossary"`
	Models        map[string]Model        `yaml:"models"`
	States        map[string]StateMachine `yaml:"states"`
	Interfaces    map[string]Interface    `yaml:"interfaces"`
	Authorization Authorization           `yaml:"authorization"`
	Objectives    map[string]Objective    `yaml:"objectives"`
	Scenarios     map[string]Scenario     `yaml:"scenarios"`
	Flows         map[string]Flow         `yaml:"flows"`
	Annotations   map[string]any          `yaml:"annotations"`
}

type Standard struct {
	Title        string                `yaml:"title"`
	Version      string                `yaml:"version"`
	URL          string                `yaml:"url"`
	Roles        []string              `yaml:"roles"`
	Scope        string                `yaml:"scope"`
	Requirements []StandardRequirement `yaml:"requirements"`
}

type StandardRequirement struct {
	ID        string   `yaml:"id"`
	Section   string   `yaml:"section"`
	Strength  string   `yaml:"strength"`
	Adoption  string   `yaml:"adoption"`
	Statement string   `yaml:"statement"`
	Reason    string   `yaml:"reason"`
	Refs      []string `yaml:"refs"`
}

type Vocabulary struct {
	Description      string             `yaml:"description"`
	Definition       string             `yaml:"definition"`
	Aliases          []string           `yaml:"aliases"`
	Context          string             `yaml:"context"`
	NotToConfuseWith []NotToConfuseWith `yaml:"not_to_confuse_with"`
	Annotations      map[string]any     `yaml:"annotations"`
}

type NotToConfuseWith struct {
	Term   string `yaml:"term"`
	Reason string `yaml:"reason"`
}

type Model struct {
	Kind        string              `yaml:"kind"`
	Description string              `yaml:"description"`
	Identity    any                 `yaml:"identity"`
	Fields      map[string]FieldDef `yaml:"fields"`
	Values      []string            `yaml:"values"`
	Payload     map[string]FieldDef `yaml:"payload"`
	Constraints []string            `yaml:"constraints"`
	Annotations map[string]any      `yaml:"annotations"`
}

type FieldDef struct {
	Type        string         `yaml:"type"`
	Optional    bool           `yaml:"optional"`
	Default     any            `yaml:"default"`
	Constraints []any          `yaml:"constraints"`
	Description string         `yaml:"description"`
	Annotations map[string]any `yaml:"annotations"`
	// Inline Schema
	Fields map[string]FieldDef `yaml:"fields"`
}

type Interface struct {
	Description string              `yaml:"description"`
	Steps       []string            `yaml:"steps"`
	Input       map[string]FieldDef `yaml:"input"`
	Output      map[string]FieldDef `yaml:"output"`
	Errors      []string            `yaml:"errors"`
	Emits       []string            `yaml:"emits"`
	Requires    []string            `yaml:"requires"`
	Ensures     []string            `yaml:"ensures"`
	Access      any                 `yaml:"access"`
	Idempotent  bool                `yaml:"idempotent"`
	ReadOnly    bool                `yaml:"read_only"`
	Bindings    []Binding           `yaml:"bindings"`
	Annotations map[string]any      `yaml:"annotations"`
}

type ProtectedAccess struct {
	Policies []string
	Resource AccessResource
}

type AccessResource struct {
	Type string
	ID   string
}

// ProtectedInterfaceAccess returns the structured form of a protected SCL 3.0 access declaration.
func ProtectedInterfaceAccess(iface Interface) (ProtectedAccess, bool) {
	value, ok := iface.Access.(map[string]any)
	if !ok {
		return ProtectedAccess{}, false
	}
	policies, ok := stringSlice(value["policies"])
	if !ok || len(policies) == 0 {
		return ProtectedAccess{}, false
	}
	resource, ok := value["resource"].(map[string]any)
	if !ok {
		return ProtectedAccess{}, false
	}
	resourceType, typeOK := resource["type"].(string)
	resourceID, idOK := resource["id"].(string)
	if !typeOK || !idOK {
		return ProtectedAccess{}, false
	}
	return ProtectedAccess{
		Policies: policies,
		Resource: AccessResource{Type: resourceType, ID: resourceID},
	}, true
}

func stringSlice(value any) ([]string, bool) {
	switch values := value.(type) {
	case []string:
		return values, true
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			item, ok := value.(string)
			if !ok {
				return nil, false
			}
			out = append(out, item)
		}
		return out, true
	default:
		return nil, false
	}
}

// Binding は generic な map で受け、kind に応じて型変換するスタイル。
// Go の sum type 表現は冗長なため、kind ベース field アクセスを許容する。
type Binding map[string]any

func (b Binding) Kind() string {
	if s, ok := b["kind"].(string); ok {
		return s
	}
	return ""
}

func (b Binding) String(k string) string {
	if s, ok := b[k].(string); ok {
		return s
	}
	return ""
}

type StateMachine struct {
	Description string         `yaml:"description"`
	Target      string         `yaml:"target"`
	Initial     string         `yaml:"initial"`
	Terminal    []string       `yaml:"terminal"`
	Transitions []Transition   `yaml:"transitions"`
	Annotations map[string]any `yaml:"annotations"`
}

type Transition struct {
	From   string   `yaml:"from"`
	Event  string   `yaml:"event"`
	To     string   `yaml:"to"`
	Guard  any      `yaml:"guard"`
	Effect []string `yaml:"effect"`
}

type Scenario struct {
	Actor       string              `yaml:"actor"`
	Given       []string            `yaml:"given"`
	MainSuccess []string            `yaml:"main_success"`
	Extensions  []ScenarioExtension `yaml:"extensions"`
	Tags        []string            `yaml:"tags"`
	Description string              `yaml:"description"`
	Annotations map[string]any      `yaml:"annotations"`
}

type ScenarioExtension struct {
	At        any      `yaml:"at"`
	Condition string   `yaml:"condition"`
	Steps     []string `yaml:"steps"`
}

type Authorization struct {
	Resources  map[string]AuthorizationResource  `yaml:"resources"`
	Principals map[string]AuthorizationPrincipal `yaml:"principals"`
	Policies   map[string]AuthorizationPolicy    `yaml:"policies"`
}

type AuthorizationResource struct {
	Description string `yaml:"description"`
}

type AuthorizationPrincipal struct {
	Type        string         `yaml:"type"`
	Matches     []string       `yaml:"matches"`
	Description string         `yaml:"description"`
	Annotations map[string]any `yaml:"annotations"`
}

type AuthorizationPolicy struct {
	Effect      string         `yaml:"effect"`
	Principal   string         `yaml:"principal"`
	When        string         `yaml:"when"`
	Description string         `yaml:"description"`
	Annotations map[string]any `yaml:"annotations"`
}

type Objective struct {
	Description string         `yaml:"description"`
	Interface   string         `yaml:"interface"`
	Indicator   string         `yaml:"indicator"`
	Target      float64        `yaml:"target"`
	Window      string         `yaml:"window"`
	Budgeting   string         `yaml:"budgeting"`
	Slice       string         `yaml:"slice"`
	Annotations map[string]any `yaml:"annotations"`
}

type Flow struct {
	Description string           `yaml:"description"`
	Entry       string           `yaml:"entry"`
	Transitions []FlowTransition `yaml:"transitions"`
	Annotations map[string]any   `yaml:"annotations"`
}

type FlowTransition struct {
	From      string `yaml:"from"`
	Action    string `yaml:"action"`
	Interface string `yaml:"interface"`
	To        string `yaml:"to"`
	External  bool   `yaml:"external"`
}

// =====================================================================
// ローダー
// =====================================================================

var loaded *SCL

func LoadSCL() (*SCL, error) {
	if loaded != nil {
		return loaded, nil
	}
	path := os.Getenv("SCL_PATH")
	if path == "" {
		_, here, _, ok := runtime.Caller(0)
		if !ok {
			return nil, fmt.Errorf("loader: cannot determine caller path")
		}
		root := filepath.Join(filepath.Dir(here), "..", "..", "..")
		path = filepath.Join(root, "spec", "scl.yaml")
	}
	raw, err := os.ReadFile(path) //nolint:gosec // SCL_PATH is an explicit operator-controlled configuration path.
	if err != nil {
		return nil, fmt.Errorf("loader: read %s: %w", path, err)
	}
	s, err := DecodeSCL(raw)
	if err != nil {
		return nil, fmt.Errorf("loader: unmarshal scl.yaml: %w", err)
	}
	if err := s.loadContexts(filepath.Dir(path)); err != nil {
		return nil, err
	}
	loaded = s
	return loaded, nil
}

// DecodeSCL はトップレベル scl.yaml (context_map) を strict にデコードする。
// glossary / models / interfaces などの合成セクションは loadContexts が context ファイルから取り込む。
func DecodeSCL(raw []byte) (*SCL, error) {
	var s SCL
	if err := yaml.NewDecoder(bytes.NewReader(raw), yaml.Strict()).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// loadContexts は context_map の各 path を読み込み、合成セクションへマージする。
// 同一キーが複数 context で定義されれば衝突として拒否する (暗黙の単一所有を保証する)。
func (s *SCL) loadContexts(dir string) error {
	s.Standards = map[string]Standard{}
	s.Vocabulary = map[string]Vocabulary{}
	s.Models = map[string]Model{}
	s.Interfaces = map[string]Interface{}
	s.InterfaceContexts = map[string]string{}
	s.States = map[string]StateMachine{}
	s.Scenarios = map[string]Scenario{}
	s.AuthorizationByContext = map[string]Authorization{}
	s.Objectives = map[string]Objective{}
	s.Flows = map[string]Flow{}

	// 決定論的なマージ順 (衝突メッセージの安定化) のため context 名を並べる。
	names := make([]string, 0, len(s.ContextMap))
	for name := range s.ContextMap {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := s.ContextMap[name]
		if entry.Path == "" {
			continue
		}
		path := filepath.Join(dir, filepath.FromSlash(entry.Path))
		raw, err := os.ReadFile(path) //nolint:gosec // path は context_map が宣言する spec ファイル。
		if err != nil {
			return fmt.Errorf("loader: read context %s: %w", name, err)
		}
		var doc contextDocument
		if err := yaml.NewDecoder(bytes.NewReader(raw), yaml.Strict()).Decode(&doc); err != nil {
			return fmt.Errorf("loader: unmarshal context %s (%s): %w", name, entry.Path, err)
		}
		if err := s.mergeContext(name, doc); err != nil {
			return err
		}
	}
	return nil
}

// mergeContext は 1 つの context ファイルの各セクションを集約 SCL へ取り込む。
func (s *SCL) mergeContext(ctxName string, doc contextDocument) error {
	mergeStandards(s.Standards, doc.Standards)
	for name, entry := range doc.Glossary {
		if entry.Context == "" {
			entry.Context = ctxName
		}
		current, ok := s.Vocabulary[name]
		if !ok || vocabularyDetailScore(entry) > vocabularyDetailScore(current) {
			s.Vocabulary[name] = entry
		}
	}
	mergeModels(s.Models, doc.Models)
	if err := mergeMap(s.Interfaces, doc.Interfaces, ctxName, "interface"); err != nil {
		return err
	}
	for name := range doc.Interfaces {
		s.InterfaceContexts[name] = ctxName
	}
	if err := mergeMap(s.States, doc.States, ctxName, "state"); err != nil {
		return err
	}
	if err := mergeMap(s.Scenarios, doc.Scenarios, ctxName, "scenario"); err != nil {
		return err
	}
	s.AuthorizationByContext[ctxName] = doc.Authorization
	if err := mergeMap(s.Objectives, doc.Objectives, ctxName, "objective"); err != nil {
		return err
	}
	if err := mergeMap(s.Flows, doc.Flows, ctxName, "flow"); err != nil {
		return err
	}
	return nil
}

// mergeMap は src の各エントリを dst へ移し、キー衝突を拒否する。
func mergeMap[V any](dst, src map[string]V, ctxName, kind string) error {
	for name, value := range src {
		if _, ok := dst[name]; ok {
			return fmt.Errorf("loader: %s %s defined by multiple contexts (last %s)", kind, name, ctxName)
		}
		dst[name] = value
	}
	return nil
}

// mergeModels builds a convenient cross-context lookup while allowing each context to carry
// published-language stubs. When names overlap, the structurally richer definition wins.
func mergeModels(dst, src map[string]Model) {
	for name, value := range src {
		current, ok := dst[name]
		if !ok || modelDetailScore(value) > modelDetailScore(current) {
			dst[name] = value
		}
	}
}

func mergeStandards(dst, src map[string]Standard) {
	for name, value := range src {
		current, ok := dst[name]
		if !ok || len(value.Requirements) > len(current.Requirements) {
			dst[name] = value
		}
	}
}

func vocabularyDetailScore(entry Vocabulary) int {
	return len(entry.Definition) + len(entry.Description) + len(entry.Aliases)*8 + len(entry.NotToConfuseWith)*8
}

func modelDetailScore(model Model) int {
	return len(model.Fields)*4 + len(model.Payload)*4 + len(model.Constraints)*2 + len(model.Values)
}

// MustLoadSCL は LoadSCL の panic 版（main 配線で使う）。
func MustLoadSCL() *SCL {
	s, err := LoadSCL()
	if err != nil {
		panic(err)
	}
	return s
}

// =====================================================================
// 派生ビュー
// =====================================================================

var wireAliasPattern = regexp.MustCompile(`^[a-z][a-z0-9_:.-]*$`)

// ToWire は PascalCase 名をワイヤ形式 (snake_case 等) に変換する。
// vocabulary[].aliases から WIRE_ALIAS_PATTERN に最初に一致するものを返す。
func (s *SCL) ToWire(name string) string {
	entry, ok := s.Vocabulary[name]
	if !ok {
		return name
	}
	for _, a := range entry.Aliases {
		if wireAliasPattern.MatchString(a) {
			return a
		}
	}
	return name
}

func (s *SCL) ToWireAll(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = s.ToWire(n)
	}
	return out
}

func (s *SCL) EnumValues(modelName string) ([]string, error) {
	m, ok := s.Models[modelName]
	if !ok {
		return nil, fmt.Errorf("model %s not found", modelName)
	}
	if m.Kind != "enum" {
		return nil, fmt.Errorf("%s is not an enum", modelName)
	}
	return m.Values, nil
}

func (s *SCL) EnumWireValues(modelName string) ([]string, error) {
	v, err := s.EnumValues(modelName)
	if err != nil {
		return nil, err
	}
	return s.ToWireAll(v), nil
}

func (s *SCL) StatesOf(machineName string) ([]string, error) {
	sm, ok := s.States[machineName]
	if !ok {
		return nil, fmt.Errorf("state machine %s not found", machineName)
	}
	seen := map[string]struct{}{sm.Initial: {}}
	out := []string{sm.Initial}
	for _, t := range sm.Terminal {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	for _, tr := range sm.Transitions {
		for _, n := range []string{tr.From, tr.To} {
			if _, ok := seen[n]; !ok {
				seen[n] = struct{}{}
				out = append(out, n)
			}
		}
	}
	return out, nil
}

func (s *SCL) EventsOf(machineName string) ([]string, error) {
	sm, ok := s.States[machineName]
	if !ok {
		return nil, fmt.Errorf("state machine %s not found", machineName)
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, tr := range sm.Transitions {
		if _, ok := seen[tr.Event]; !ok {
			seen[tr.Event] = struct{}{}
			out = append(out, tr.Event)
		}
	}
	return out, nil
}

func (s *SCL) VocabularyCodes() map[string]struct{} {
	out := map[string]struct{}{}
	for name := range s.Vocabulary {
		out[s.ToWire(name)] = struct{}{}
	}
	return out
}

func (s *SCL) HTTPBinding(iface Interface) (Binding, bool) {
	for _, b := range iface.Bindings {
		if b.Kind() == "http" {
			return b, true
		}
	}
	return nil, false
}

const (
	AuthCodeFlowName   = "AuthorizationCodeFlow"
	DeviceCodeFlowName = "DeviceCodeFlow"
)
