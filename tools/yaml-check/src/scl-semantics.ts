import { type Finding, locatePointer } from './lib.ts'

type Dict = Record<string, unknown>

const BUILTIN_TYPES = new Set([
  'Any',
  'Bool',
  'Boolean',
  'Bytes',
  'Date',
  'DateTime',
  'Decimal',
  'Duration',
  'Float',
  'Int',
  'Integer',
  'JSON',
  'Map',
  'Number',
  'Optional',
  'String',
  'UUID',
  'URI',
  'URL',
  'List',
  'Set',
])

function dict(value: unknown): Dict {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Dict)
    : {}
}

function strings(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []
}

function addFinding(findings: Finding[], text: string, pointer: string, message: string): void {
  findings.push({ line: locatePointer(text, pointer), column: 1, message: `scl: ${message}` })
}

function referencedTypes(type: unknown): string[] {
  if (typeof type === 'string') return type.match(/[A-Za-z_][A-Za-z0-9_]*/g) ?? []
  if (Array.isArray(type)) return type.flatMap(referencedTypes)
  return Object.values(dict(type)).flatMap(referencedTypes)
}

function fieldDefinitions(container: unknown): Dict {
  const source = dict(container)
  const result: Dict = {}
  for (const [name, value] of Object.entries(source)) {
    const definition = dict(value)
    if ('fields' in definition && !('type' in definition)) {
      Object.assign(result, dict(definition.fields))
    } else {
      result[name] = value
    }
  }
  return result
}

function validateFieldTypes(
  fields: unknown,
  models: Dict,
  text: string,
  pointer: string,
  findings: Finding[],
): void {
  for (const [fieldName, fieldValue] of Object.entries(fieldDefinitions(fields))) {
    for (const name of referencedTypes(dict(fieldValue).type)) {
      if (!BUILTIN_TYPES.has(name) && !(name in models)) {
        addFinding(
          findings,
          text,
          `${pointer}/${fieldName}/type`,
          `${pointer.replaceAll('/', '.').slice(1)}.${fieldName} references unknown type '${name}'`,
        )
      }
    }
  }
}

function expressionRoots(expression: string): Set<string> {
  const withoutStrings = expression.replace(/"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'/g, ' ')
  return new Set(
    [...withoutStrings.matchAll(/(?<!\.)\b([A-Za-z_][A-Za-z0-9_]*)\s*\./g)]
      .map((match) => match[1])
      .filter((name): name is string => name !== undefined),
  )
}

function validateExpressions(
  value: unknown,
  allowedRoots: ReadonlySet<string>,
  text: string,
  pointer: string,
  findings: Finding[],
): void {
  const expressions = typeof value === 'string' ? [value] : strings(value)
  expressions.forEach((expression, index) => {
    for (const root of expressionRoots(expression)) {
      if (!allowedRoots.has(root)) {
        addFinding(
          findings,
          text,
          typeof value === 'string' ? pointer : `${pointer}/${index}`,
          `${pointer.replaceAll('/', '.').slice(1)} uses unavailable CEL binding '${root}'`,
        )
      }
    }
  })
}

function modelKind(models: Dict, name: string): string | undefined {
  const kind = dict(models[name]).kind
  return typeof kind === 'string' ? kind : undefined
}

export function verifySclSemantics(document: unknown, text = ''): Finding[] {
  const findings: Finding[] = []
  const doc = dict(document)
  if (doc.spec_version !== '3.0') return findings

  const models = dict(doc.models)
  const interfaces = dict(doc.interfaces)
  const states = dict(doc.states)
  const authorization = dict(doc.authorization)
  const resources = dict(authorization.resources)
  const principals = dict(authorization.principals)
  const policies = dict(authorization.policies)
  const objectives = dict(doc.objectives)
  const scenarios = dict(doc.scenarios)
  const flows = dict(doc.flows)
  const glossary = dict(doc.glossary)

  for (const [modelName, modelValue] of Object.entries(models)) {
    const model = dict(modelValue)
    const fields = dict(model.fields)
    validateFieldTypes(fields, models, text, `/models/${modelName}/fields`, findings)
    validateFieldTypes(model.payload, models, text, `/models/${modelName}/payload`, findings)

    const identities = typeof model.identity === 'string' ? [model.identity] : strings(model.identity)
    identities.forEach((identity, index) => {
      if (!(identity in fields)) {
        addFinding(
          findings,
          text,
          `/models/${modelName}/identity${identities.length > 1 ? `/${index}` : ''}`,
          `model '${modelName}' identity references unknown field '${identity}'`,
        )
      }
    })
    validateExpressions(
      model.constraints,
      new Set(Object.keys(fields)),
      text,
      `/models/${modelName}/constraints`,
      findings,
    )
  }

  for (const [interfaceName, interfaceValue] of Object.entries(interfaces)) {
    const operation = dict(interfaceValue)
    validateFieldTypes(operation.input, models, text, `/interfaces/${interfaceName}/input`, findings)
    validateFieldTypes(operation.output, models, text, `/interfaces/${interfaceName}/output`, findings)

    strings(operation.errors).forEach((name, index) => {
      if (modelKind(models, name) !== 'error') {
        addFinding(
          findings,
          text,
          `/interfaces/${interfaceName}/errors/${index}`,
          `interface '${interfaceName}' references unknown error model '${name}'`,
        )
      }
    })
    strings(operation.emits).forEach((name, index) => {
      if (modelKind(models, name) !== 'event') {
        addFinding(
          findings,
          text,
          `/interfaces/${interfaceName}/emits/${index}`,
          `interface '${interfaceName}' references unknown event model '${name}'`,
        )
      }
    })

    validateExpressions(
      operation.requires,
      new Set(['input', 'resource', 'subject', 'context']),
      text,
      `/interfaces/${interfaceName}/requires`,
      findings,
    )
    validateExpressions(
      operation.ensures,
      new Set(['input', 'resource', 'subject', 'context', 'output', 'response', 'emitted']),
      text,
      `/interfaces/${interfaceName}/ensures`,
      findings,
    )

    const access = operation.access
    if (access === 'internal' && Array.isArray(operation.bindings) && operation.bindings.length > 0) {
      addFinding(
        findings,
        text,
        `/interfaces/${interfaceName}/bindings`,
        `internal interface '${interfaceName}' must not declare an external binding`,
      )
    }
    if (access !== 'public' && access !== 'internal') {
      const protectedAccess = dict(access)
      strings(protectedAccess.policies).forEach((name, index) => {
        if (!(name in policies)) {
          addFinding(
            findings,
            text,
            `/interfaces/${interfaceName}/access/policies/${index}`,
            `interface '${interfaceName}' references unknown authorization policy '${name}'`,
          )
        }
      })
      const resource = dict(protectedAccess.resource)
      const resourceType = resource.type
      if (
        typeof resourceType === 'string' &&
        !(resourceType in models) &&
        !(resourceType in resources)
      ) {
        addFinding(
          findings,
          text,
          `/interfaces/${interfaceName}/access/resource/type`,
          `interface '${interfaceName}' references unknown resource type '${resourceType}'`,
        )
      }
      validateExpressions(
        resource.id,
        new Set(['input', 'resource', 'subject', 'context']),
        text,
        `/interfaces/${interfaceName}/access/resource/id`,
        findings,
      )
    }
  }

  for (const [principalName, principalValue] of Object.entries(principals)) {
    const principal = dict(principalValue)
    if (typeof principal.type === 'string' && !(principal.type in models)) {
      addFinding(
        findings,
        text,
        `/authorization/principals/${principalName}/type`,
        `principal '${principalName}' references unknown model '${principal.type}'`,
      )
    }
    validateExpressions(
      principal.matches,
      new Set(['principal', 'resource', 'context']),
      text,
      `/authorization/principals/${principalName}/matches`,
      findings,
    )
  }

  for (const [policyName, policyValue] of Object.entries(policies)) {
    const policy = dict(policyValue)
    if (typeof policy.principal === 'string' && !(policy.principal in principals)) {
      addFinding(
        findings,
        text,
        `/authorization/policies/${policyName}/principal`,
        `policy '${policyName}' references unknown principal '${policy.principal}'`,
      )
    }
    validateExpressions(
      policy.when,
      new Set(['principal', 'resource', 'context']),
      text,
      `/authorization/policies/${policyName}/when`,
      findings,
    )
  }

  for (const [stateName, stateValue] of Object.entries(states)) {
    const machine = dict(stateValue)
    const target = typeof machine.target === 'string' ? machine.target : ''
    if (!(target in models)) {
      addFinding(
        findings,
        text,
        `/states/${stateName}/target`,
        `state machine '${stateName}' references unknown target model '${target}'`,
      )
    }
    const targetFields = Object.keys(dict(dict(models[target]).fields))
    const stateValues = new Set<string>()
    for (const fieldValue of Object.values(dict(dict(models[target]).fields))) {
      const type = dict(fieldValue).type
      if (typeof type === 'string' && modelKind(models, type) === 'enum') {
        strings(dict(models[type]).values).forEach((value) => stateValues.add(value))
      }
    }
    const allowed = new Set([...targetFields, 'input'])
    const transitions = Array.isArray(machine.transitions) ? machine.transitions : []
    const terminal = new Set(strings(machine.terminal))
    const stateReferences: Array<[unknown, string]> = [[machine.initial, `/states/${stateName}/initial`]]
    transitions.forEach((transitionValue, index) => {
      const transition = dict(transitionValue)
      stateReferences.push(
        [transition.from, `/states/${stateName}/transitions/${index}/from`],
        [transition.to, `/states/${stateName}/transitions/${index}/to`],
      )
      if (typeof transition.from === 'string' && terminal.has(transition.from)) {
        addFinding(
          findings,
          text,
          `/states/${stateName}/transitions/${index}/from`,
          `state machine '${stateName}' declares a transition from terminal state '${transition.from}'`,
        )
      }
      const event = transition.event
      if (typeof event === 'string' && modelKind(models, event) !== 'event') {
        addFinding(
          findings,
          text,
          `/states/${stateName}/transitions/${index}/event`,
          `state machine '${stateName}' references unknown event model '${event}'`,
        )
      }
      strings(transition.effect).forEach((name, effectIndex) => {
        if (modelKind(models, name) !== 'event') {
          addFinding(
            findings,
            text,
            `/states/${stateName}/transitions/${index}/effect/${effectIndex}`,
            `state machine '${stateName}' effect references unknown event model '${name}'`,
          )
        }
      })
      validateExpressions(
        transition.guard,
        allowed,
        text,
        `/states/${stateName}/transitions/${index}/guard`,
        findings,
      )
    })
    if (stateValues.size > 0) {
      strings(machine.terminal).forEach((value, index) =>
        stateReferences.push([value, `/states/${stateName}/terminal/${index}`]),
      )
      for (const [value, pointer] of stateReferences) {
        if (typeof value === 'string' && !stateValues.has(value)) {
          addFinding(
            findings,
            text,
            pointer,
            `state machine '${stateName}' references unknown state value '${value}'`,
          )
        }
      }
    }
  }

  for (const [objectiveName, objectiveValue] of Object.entries(objectives)) {
    const objective = dict(objectiveValue)
    if (typeof objective.interface === 'string' && !(objective.interface in interfaces)) {
      addFinding(
        findings,
        text,
        `/objectives/${objectiveName}/interface`,
        `objective '${objectiveName}' references unknown interface '${objective.interface}'`,
      )
    }
    validateExpressions(
      objective.indicator,
      new Set(['request', 'response', 'event', 'measurement']),
      text,
      `/objectives/${objectiveName}/indicator`,
      findings,
    )
  }

  for (const [scenarioName, scenarioValue] of Object.entries(scenarios)) {
    const scenario = dict(scenarioValue)
    const actor = scenario.actor
    if (typeof actor === 'string' && !(actor in glossary) && !(actor in principals)) {
      addFinding(
        findings,
        text,
        `/scenarios/${scenarioName}/actor`,
        `scenario '${scenarioName}' references unknown actor '${actor}'`,
      )
    }
    const mainSuccess = strings(scenario.main_success)
    const extensions = Array.isArray(scenario.extensions) ? scenario.extensions : []
    extensions.forEach((extensionValue, index) => {
      const at = dict(extensionValue).at
      if (typeof at === 'number' && at > mainSuccess.length) {
        addFinding(
          findings,
          text,
          `/scenarios/${scenarioName}/extensions/${index}/at`,
          `scenario '${scenarioName}' extension points past main_success step ${mainSuccess.length}`,
        )
      }
    })
  }

  for (const [flowName, flowValue] of Object.entries(flows)) {
    const flow = dict(flowValue)
    const entry = typeof flow.entry === 'string' ? flow.entry : ''
    const views = dict(flow.views)
    const reachable = new Set([entry])
    let changed = true
    while (changed) {
      changed = false
      for (const viewName of reachable) {
        const view = dict(views[viewName])
        const does = Array.isArray(view.does) ? view.does : []
        for (const actionValue of does) {
          const action = dict(actionValue)
          if (typeof action.to === 'string' && action.to in views && !reachable.has(action.to)) {
            reachable.add(action.to)
            changed = true
          }
        }
      }
    }
    for (const viewName of Object.keys(views)) {
      if (!reachable.has(viewName)) {
        addFinding(
          findings,
          text,
          `/flows/${flowName}/views/${viewName}`,
          `flow '${flowName}' contains unreachable view '${viewName}'`,
        )
      }
    }
    Object.entries(views).forEach(([viewName, viewValue]) => {
      const view = dict(viewValue)
      const does = Array.isArray(view.does) ? view.does : []
      const seenActions = new Set<string>()
      does.forEach((actionValue, index) => {
        const action = dict(actionValue)
        if (typeof action.action === 'string') {
          if (seenActions.has(action.action)) {
            addFinding(
              findings,
              text,
              `/flows/${flowName}/views/${viewName}/does/${index}/action`,
              `flow '${flowName}' view '${viewName}' duplicates action '${action.action}'`,
            )
          }
          seenActions.add(action.action)
        }
        if (typeof action.interface === 'string' && !(action.interface in interfaces)) {
          addFinding(
            findings,
            text,
            `/flows/${flowName}/views/${viewName}/does/${index}/interface`,
            `flow '${flowName}' references unknown interface '${action.interface}'`,
          )
        }
      })
    })
  }

  const sectionMaps: Record<string, Dict> = {
    glossary,
    models,
    interfaces,
    states,
    objectives,
    scenarios,
    flows,
  }
  for (const [standardName, standardValue] of Object.entries(dict(doc.standards))) {
    const requirements = Array.isArray(dict(standardValue).requirements)
      ? (dict(standardValue).requirements as unknown[])
      : []
    requirements.forEach((requirementValue, requirementIndex) => {
      strings(dict(requirementValue).refs).forEach((ref, refIndex) => {
        const dot = ref.indexOf('.')
        const section = ref.slice(0, dot)
        const name = ref.slice(dot + 1)
        const target =
          section === 'authorization'
            ? name in principals || name in policies || name in resources
            : name in (sectionMaps[section] ?? {})
        if (!target) {
          addFinding(
            findings,
            text,
            `/standards/${standardName}/requirements/${requirementIndex}/refs/${refIndex}`,
            `standard requirement references unknown SCL element '${ref}'`,
          )
        }
      })
    })
  }

  return findings
}
