package domain

import (
	"fmt"
	"strings"
)

const CurrentManifestSchemaVersion = "1"

type SecretProvider string

const (
	SecretProviderEnv  SecretProvider = "env"
	SecretProviderFile SecretProvider = "file"
)

type SecretReference struct {
	Provider SecretProvider `yaml:"provider" json:"provider"`
	Locator  string         `yaml:"locator" json:"locator"`
	Version  string         `yaml:"version" json:"version"`
}

func (r SecretReference) Validate() error {
	if r.Provider != SecretProviderEnv && r.Provider != SecretProviderFile {
		return fmt.Errorf("unsupported seed secret provider %q", r.Provider)
	}
	if strings.TrimSpace(r.Locator) == "" {
		return fmt.Errorf("seed secret locator must not be empty")
	}
	if strings.TrimSpace(r.Version) == "" {
		return fmt.Errorf("seed secret version must not be empty")
	}
	return nil
}

type ResourceKind string

const (
	ResourceKindFirstPartyClients ResourceKind = "first_party_clients"
	ResourceKindDevelopmentDemo   ResourceKind = "development_demo"
)

type Resource struct {
	Kind       ResourceKind               `yaml:"kind" json:"kind"`
	LogicalKey string                     `yaml:"logical_key" json:"logical_key"`
	Clients    []FirstPartyClientSeed     `yaml:"clients,omitempty" json:"clients,omitempty"`
	Demo       *DevelopmentDemoSeed       `yaml:"demo,omitempty" json:"demo,omitempty"`
	Secrets    map[string]SecretReference `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

type FirstPartyClientSeed struct {
	ID    string `yaml:"id" json:"id"`
	Name  string `yaml:"name" json:"name"`
	Scope string `yaml:"scope" json:"scope"`
}

type DemoUserSeed struct {
	ID                string   `yaml:"id" json:"id"`
	PreferredUsername string   `yaml:"preferred_username" json:"preferred_username"`
	Email             string   `yaml:"email" json:"email"`
	Roles             []string `yaml:"roles" json:"roles"`
}

type DemoGroupSeed struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Roles       []string `yaml:"roles" json:"roles"`
	Members     []string `yaml:"members,omitempty" json:"members,omitempty"`
}

type DemoApplicationSeed struct {
	ID           string `yaml:"id" json:"id"`
	Name         string `yaml:"name" json:"name"`
	LaunchURL    string `yaml:"launch_url,omitempty" json:"launch_url,omitempty"`
	BindingType  string `yaml:"binding_type" json:"binding_type"`
	BindingValue string `yaml:"binding_value" json:"binding_value"`
	AssignedUser string `yaml:"assigned_user" json:"assigned_user"`
}

type DevelopmentDemoSeed struct {
	ClientID                string                `yaml:"client_id" json:"client_id"`
	ClientRedirectURIs      []string              `yaml:"client_redirect_uris" json:"client_redirect_uris"`
	Users                   []DemoUserSeed        `yaml:"users" json:"users"`
	Groups                  []DemoGroupSeed       `yaml:"groups" json:"groups"`
	AuthorizationDetailType string                `yaml:"authorization_detail_type" json:"authorization_detail_type"`
	WsFedRealm              string                `yaml:"wsfed_realm" json:"wsfed_realm"`
	WsFedDisplayName        string                `yaml:"wsfed_display_name" json:"wsfed_display_name"`
	WsFedReplyURLs          []string              `yaml:"wsfed_reply_urls" json:"wsfed_reply_urls"`
	SamlEntityID            string                `yaml:"saml_entity_id" json:"saml_entity_id"`
	SamlDisplayName         string                `yaml:"saml_display_name" json:"saml_display_name"`
	SamlACSURLs             []string              `yaml:"saml_acs_urls" json:"saml_acs_urls"`
	Applications            []DemoApplicationSeed `yaml:"applications" json:"applications"`
}

type Generator struct {
	Kind       string `yaml:"kind" json:"kind"`
	LogicalKey string `yaml:"logical_key" json:"logical_key"`
}

type Manifest struct {
	SchemaVersion string      `yaml:"schema_version" json:"schema_version"`
	Profile       Profile     `yaml:"profile" json:"profile"`
	Includes      []string    `yaml:"includes,omitempty" json:"includes,omitempty"`
	Resources     []Resource  `yaml:"resources,omitempty" json:"resources,omitempty"`
	Generators    []Generator `yaml:"generators,omitempty" json:"generators,omitempty"`
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != CurrentManifestSchemaVersion {
		return fmt.Errorf("unsupported seed manifest schema version %q", m.SchemaVersion)
	}
	if !validProfile(m.Profile) {
		return fmt.Errorf("unsupported seed manifest profile %q", m.Profile)
	}
	logicalKeys := make(map[string]struct{}, len(m.Resources)+len(m.Generators))
	for _, resource := range m.Resources {
		if resource.Kind != ResourceKindFirstPartyClients && resource.Kind != ResourceKindDevelopmentDemo {
			return fmt.Errorf("unsupported seed resource kind %q", resource.Kind)
		}
		if resource.Kind == ResourceKindFirstPartyClients && len(resource.Clients) == 0 {
			return fmt.Errorf("first_party_clients resource requires clients")
		}
		for _, client := range resource.Clients {
			if strings.TrimSpace(client.ID) == "" || strings.TrimSpace(client.Name) == "" || strings.TrimSpace(client.Scope) == "" {
				return fmt.Errorf("first-party client id, name, and scope are required")
			}
		}
		if resource.Kind == ResourceKindDevelopmentDemo && resource.Demo == nil {
			return fmt.Errorf("development_demo resource requires demo")
		}
		if resource.Demo != nil {
			if strings.TrimSpace(resource.Demo.ClientID) == "" || len(resource.Demo.Users) == 0 {
				return fmt.Errorf("development_demo requires client_id and users")
			}
			for _, application := range resource.Demo.Applications {
				if application.BindingType != "oidc" && application.BindingType != "wsfed" {
					return fmt.Errorf("unsupported demo application binding type %q", application.BindingType)
				}
			}
		}
		if err := validateLogicalKey(resource.LogicalKey, logicalKeys); err != nil {
			return err
		}
		for name, reference := range resource.Secrets {
			if err := reference.Validate(); err != nil {
				return fmt.Errorf("seed secret %q: %w", name, err)
			}
		}
	}
	for _, generator := range m.Generators {
		if generator.Kind != "performance_users" {
			return fmt.Errorf("unsupported seed generator kind %q", generator.Kind)
		}
		if err := validateLogicalKey(generator.LogicalKey, logicalKeys); err != nil {
			return err
		}
	}
	return nil
}

func (m Manifest) ValidateForRequest(request Request) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if m.Profile != request.Profile {
		return fmt.Errorf("seed manifest profile %q does not match request profile %q", m.Profile, request.Profile)
	}
	if request.Environment == EnvironmentStaging || request.Environment == EnvironmentProduction {
		for _, resource := range m.Resources {
			for _, reference := range resource.Secrets {
				if reference.Provider != SecretProviderFile {
					return fmt.Errorf("seed secret provider %q is not permitted in %s", reference.Provider, request.Environment)
				}
			}
		}
	}
	return nil
}

func validateLogicalKey(value string, seen map[string]struct{}) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("seed logical key must not be empty")
	}
	if _, ok := seen[value]; ok {
		return fmt.Errorf("duplicate seed logical key %q", value)
	}
	seen[value] = struct{}{}
	return nil
}
