package spec

import (
	"fmt"
	"strings"
)

func (s *SCL) ValidateCoherence() error {
	if s.System == "" {
		return fmt.Errorf("system is required")
	}
	if s.SpecVersion != "3.0" {
		return fmt.Errorf("spec_version must be 3.0")
	}
	if err := s.validateContextMap(); err != nil {
		return err
	}
	if err := s.validateModels(); err != nil {
		return err
	}
	if err := s.validateStates(); err != nil {
		return err
	}
	if err := s.validateStandardReferences(); err != nil {
		return err
	}
	if err := s.validateAuthorizationAndAccess(); err != nil {
		return err
	}
	if err := s.validateFlows(); err != nil {
		return err
	}
	return nil
}

func (s *SCL) validateModels() error {
	validKinds := map[string]bool{
		"entity": true, "value_object": true, "enum": true, "event": true, "error": true,
	}
	for name, model := range s.Models {
		if !validKinds[model.Kind] {
			return fmt.Errorf("model %s: invalid kind %s", name, model.Kind)
		}
		for fieldName, field := range model.Fields {
			if !s.validFieldType(field.Type) {
				return fmt.Errorf("model %s field %s: invalid type %s", name, fieldName, field.Type)
			}
		}
		for fieldName, field := range model.Payload {
			if !s.validFieldType(field.Type) {
				return fmt.Errorf("model %s payload %s: invalid type %s", name, fieldName, field.Type)
			}
		}
	}
	for name, iface := range s.Interfaces {
		for fieldName, field := range iface.Input {
			if field.Fields != nil {
				for subFieldName, subField := range field.Fields {
					if !s.validFieldType(subField.Type) {
						return fmt.Errorf("interface %s input %s.%s: invalid type %s", name, fieldName, subFieldName, subField.Type)
					}
				}
			} else if !s.validFieldType(field.Type) {
				return fmt.Errorf("interface %s input %s: invalid type %s", name, fieldName, field.Type)
			}
		}
		for fieldName, field := range iface.Output {
			if field.Fields != nil {
				for subFieldName, subField := range field.Fields {
					if !s.validFieldType(subField.Type) {
						return fmt.Errorf("interface %s output %s.%s: invalid type %s", name, fieldName, subFieldName, subField.Type)
					}
				}
			} else if !s.validFieldType(field.Type) {
				return fmt.Errorf("interface %s output %s: invalid type %s", name, fieldName, field.Type)
			}
		}
	}
	return nil
}

func (s *SCL) validFieldType(fieldType string) bool {
	if before, ok := strings.CutSuffix(fieldType, "[]"); ok {
		return s.validFieldType(before)
	}
	if strings.HasPrefix(fieldType, "Set<") && strings.HasSuffix(fieldType, ">") {
		return s.validFieldType(fieldType[4 : len(fieldType)-1])
	}
	if strings.HasPrefix(fieldType, "Map<") && strings.HasSuffix(fieldType, ">") {
		parts := strings.SplitN(fieldType[4:len(fieldType)-1], ",", 2)
		return len(parts) == 2 &&
			s.validFieldType(strings.TrimSpace(parts[0])) &&
			s.validFieldType(strings.TrimSpace(parts[1]))
	}
	builtins := map[string]bool{
		"String": true, "Integer": true, "Float": true, "Boolean": true, "UUID": true,
		"Date": true, "DateTime": true, "Timestamp": true, "Duration": true, "JSON": true,
		"Bytes": true, "URI": true, "URL": true, "Any": true, "Number": true,
	}
	if builtins[fieldType] {
		return true
	}
	_, ok := s.Models[fieldType]
	return ok
}

// validateContextMap は context_map の各エントリ (description / depends_on) を検証し、
// 依存先が既知の context であること、依存に uses が宣言されていること、循環が無いことを確認する。
// 模型・interface 等の単一所有はロード時の合成 (キー衝突拒否) が保証する。
func (s *SCL) validateContextMap() error {
	for name, entry := range s.ContextMap {
		if entry.Description == "" {
			return fmt.Errorf("context %s: description is required", name)
		}
		for depName, dependency := range entry.DependsOn {
			if _, ok := s.ContextMap[depName]; !ok {
				return fmt.Errorf("context %s: depends_on references unknown context %s", name, depName)
			}
			if len(dependency.Uses) == 0 {
				return fmt.Errorf("context %s: dependency on %s requires uses", name, depName)
			}
		}
	}
	return validateContextMapCycles(s.ContextMap)
}

func validateContextMapCycles(contextMap map[string]ContextMapEntry) error {
	const (
		unvisited = iota
		visiting
		visited
	)
	state := map[string]int{}
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case visiting:
			return fmt.Errorf("context dependency cycle includes %s", name)
		case visited:
			return nil
		}
		state[name] = visiting
		for depName := range contextMap[name].DependsOn {
			if err := visit(depName); err != nil {
				return err
			}
		}
		state[name] = visited
		return nil
	}
	for name := range contextMap {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}

func (s *SCL) validateStates() error {
	for name, machine := range s.States {
		if _, ok := s.Models[machine.Target]; !ok {
			return fmt.Errorf("state %s: unknown target model %s", name, machine.Target)
		}
		states := map[string]struct{}{machine.Initial: {}}
		for _, terminal := range machine.Terminal {
			states[terminal] = struct{}{}
		}
		transitions := map[string]struct{}{}
		for _, transition := range machine.Transitions {
			states[transition.From] = struct{}{}
			states[transition.To] = struct{}{}
			key := transition.From + "\x00" + transition.Event
			if _, ok := transitions[key]; ok {
				return fmt.Errorf("state %s: duplicate transition from %s on %s", name, transition.From, transition.Event)
			}
			transitions[key] = struct{}{}
			if _, ok := s.Vocabulary[transition.Event]; !ok {
				model, modelOK := s.Models[transition.Event]
				if !modelOK || model.Kind != "event" {
					return fmt.Errorf("state %s: event %s is unresolved", name, transition.Event)
				}
			}
		}
		for state := range states {
			if state == "" {
				return fmt.Errorf("state %s: state names must not be empty", name)
			}
		}
		for _, terminal := range machine.Terminal {
			for _, transition := range machine.Transitions {
				if transition.From == terminal {
					return fmt.Errorf("state %s: terminal state %s has outgoing transition", name, terminal)
				}
			}
		}
	}
	return nil
}

func (s *SCL) validateStandardReferences() error {
	for standardName, standard := range s.Standards {
		for _, requirement := range standard.Requirements {
			for _, ref := range requirement.Refs {
				section, name, ok := strings.Cut(ref, ".")
				if !ok || !s.referenceExists(section, name) {
					return fmt.Errorf("standard %s requirement %s: unknown reference %s", standardName, requirement.ID, ref)
				}
			}
		}
	}
	return nil
}

func (s *SCL) validateAuthorizationAndAccess() error {
	for contextName, authorization := range s.AuthorizationByContext {
		for name, policy := range authorization.Policies {
			if _, ok := authorization.Principals[policy.Principal]; !ok {
				return fmt.Errorf("authorization %s policy %s: unknown principal %s", contextName, name, policy.Principal)
			}
		}
	}
	for name, iface := range s.Interfaces {
		contextName := s.InterfaceContexts[name]
		authorization := s.AuthorizationByContext[contextName]
		access, protected := ProtectedInterfaceAccess(iface)
		if !protected {
			if kind, ok := iface.Access.(string); !ok || (kind != "public" && kind != "internal") {
				return fmt.Errorf("interface %s: invalid access", name)
			}
			continue
		}
		for _, policy := range access.Policies {
			if _, ok := authorization.Policies[policy]; !ok {
				return fmt.Errorf("interface %s: unknown authorization policy %s", name, policy)
			}
		}
		if _, ok := s.Models[access.Resource.Type]; !ok {
			if _, ok := authorization.Resources[access.Resource.Type]; !ok {
				return fmt.Errorf("interface %s: unknown authorization resource %s", name, access.Resource.Type)
			}
		}
	}
	return nil
}

func (s *SCL) validateFlows() error {
	for name, flow := range s.Flows {
		for _, transition := range flow.Transitions {
			if transition.Interface == "" {
				continue
			}
			if _, ok := s.Interfaces[transition.Interface]; !ok {
				return fmt.Errorf("flow %s: unknown interface %s", name, transition.Interface)
			}
		}
	}
	return nil
}

func (s *SCL) referenceExists(section, name string) bool {
	switch section {
	case "standards":
		_, ok := s.Standards[name]
		return ok
	case "glossary":
		_, ok := s.Vocabulary[name]
		return ok
	case "models":
		_, ok := s.Models[name]
		return ok
	case "interfaces":
		_, ok := s.Interfaces[name]
		return ok
	case "states":
		_, ok := s.States[name]
		return ok
	case "scenarios":
		_, ok := s.Scenarios[name]
		return ok
	case "objectives":
		_, ok := s.Objectives[name]
		return ok
	case "flows":
		_, ok := s.Flows[name]
		return ok
	default:
		return false
	}
}
