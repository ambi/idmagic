import { IconPlus, IconTrash, IconX } from '@tabler/icons-react'
import { useState } from 'react'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { useDictionary } from '../../lib/i18n'
import { adminAuthorizationDictionary } from './AdminAuthorizationPage.i18n'
import { selectClassName } from './AdminAuthorizationShared'
import {
  formatSubjectForm,
  isValidName,
  type ModelDraft,
  type RelationDraft,
  type ResourceTypeDraft,
  type RewriteDraft,
  type SubjectFormDraft,
} from './authorizationModel'

const REWRITE_DEFAULTS: Record<RewriteDraft['kind'], RewriteDraft> = {
  direct: { kind: 'direct', subjects: [] },
  computed_userset: { kind: 'computed_userset', computedRelation: '' },
  tuple_to_userset: { kind: 'tuple_to_userset', tuplesetRelation: '', computedRelation: '' },
}

function replaceAt<T>(items: T[], index: number, item: T): T[] {
  return items.map((current, position) => (position === index ? item : current))
}

function removeAt<T>(items: T[], index: number): T[] {
  return items.filter((_, position) => position !== index)
}

function NameSelect({
  id,
  label,
  names,
  value,
  onChange,
}: {
  id: string
  label: string
  names: string[]
  value: string
  onChange: (value: string) => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  // JSON から戻った未宣言の名前も、選択肢から消して別の値に見せず、そのまま表示する。
  const options = value && !names.includes(value) ? [...names, value] : names
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <select
        id={id}
        className={selectClassName}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">{t.choose}</option>
        {options.map((name) => (
          <option key={name} value={name}>
            {name}
          </option>
        ))}
      </select>
    </div>
  )
}

// SubjectFormAdder は主体の形の 3 通りの追加を選択肢で受け、文字列の組み立てを変換関数に任せる。
function SubjectFormAdder({
  idPrefix,
  draft,
  onAdd,
}: {
  idPrefix: string
  draft: ModelDraft
  onAdd: (form: SubjectFormDraft) => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const [kind, setKind] = useState<SubjectFormDraft['kind']>('object')
  const [type, setType] = useState('')
  const [relation, setRelation] = useState('')
  const typeNames = draft.resourceTypes.map((candidate) => candidate.name).filter(Boolean)
  const relationNames =
    draft.resourceTypes
      .find((candidate) => candidate.name === type)
      ?.relations.map((candidate) => candidate.name)
      .filter(Boolean) ?? []
  const ready = type !== '' && (kind !== 'subject_set' || relation !== '')

  function add() {
    onAdd(kind === 'subject_set' ? { kind, type, relation } : { kind, type })
    setRelation('')
  }

  return (
    <div className="flex flex-wrap items-end gap-2">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor={`${idPrefix}-kind`}>{t.subjectFormKind}</Label>
        <select
          id={`${idPrefix}-kind`}
          className={selectClassName}
          value={kind}
          onChange={(event) => setKind(event.target.value as SubjectFormDraft['kind'])}
        >
          <option value="object">{t.subjectObject}</option>
          <option value="subject_set">{t.subjectSet}</option>
          <option value="wildcard">{t.subjectWildcard}</option>
        </select>
      </div>
      <NameSelect
        id={`${idPrefix}-type`}
        label={t.subjectType}
        names={typeNames}
        value={type}
        onChange={(next) => {
          setType(next)
          setRelation('')
        }}
      />
      {kind === 'subject_set' ? (
        <NameSelect
          id={`${idPrefix}-relation`}
          label={t.subjectRelation}
          names={relationNames}
          value={relation}
          onChange={setRelation}
        />
      ) : null}
      <Button type="button" variant="outline" disabled={!ready} onClick={add}>
        {t.addSubjectForm}
      </Button>
    </div>
  )
}

function RewriteEditor({
  idPrefix,
  draft,
  type,
  rewrite,
  onChange,
  onRemove,
}: {
  idPrefix: string
  draft: ModelDraft
  type: ResourceTypeDraft
  rewrite: RewriteDraft
  onChange: (rewrite: RewriteDraft) => void
  onRemove: () => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  const relationNames = type.relations.map((relation) => relation.name).filter(Boolean)
  return (
    <div className="flex flex-col gap-3 rounded-md border border-slate-200 p-3">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={`${idPrefix}-rewrite-kind`}>{t.rewriteKind}</Label>
          <select
            id={`${idPrefix}-rewrite-kind`}
            className={selectClassName}
            value={rewrite.kind}
            onChange={(event) =>
              onChange(REWRITE_DEFAULTS[event.target.value as RewriteDraft['kind']])
            }
          >
            <option value="direct">{t.rewriteDirect}</option>
            <option value="computed_userset">{t.rewriteComputed}</option>
            <option value="tuple_to_userset">{t.rewriteTupleToUserset}</option>
          </select>
        </div>
        <Button type="button" variant="ghost" onClick={onRemove}>
          <IconX size={16} aria-hidden="true" />
          {t.removeRewrite}
        </Button>
      </div>
      {rewrite.kind === 'direct' ? (
        <>
          <div>
            <p className="text-sm font-medium">{t.subjectFormsLabel}</p>
            <ul className="mt-1 flex flex-wrap gap-2">
              {rewrite.subjects.map((form, index) => {
                const label = formatSubjectForm(form)
                return (
                  <li
                    key={label}
                    className="flex items-center gap-1 rounded-md bg-slate-100 px-2 py-1 font-mono text-xs"
                  >
                    {label}
                    <button
                      type="button"
                      aria-label={t.removeSubjectForm.replace('{form}', label)}
                      onClick={() =>
                        onChange({ ...rewrite, subjects: removeAt(rewrite.subjects, index) })
                      }
                    >
                      <IconX size={14} aria-hidden="true" />
                    </button>
                  </li>
                )
              })}
            </ul>
          </div>
          <SubjectFormAdder
            idPrefix={idPrefix}
            draft={draft}
            onAdd={(form) => {
              const label = formatSubjectForm(form)
              if (rewrite.subjects.some((current) => formatSubjectForm(current) === label)) return
              onChange({ ...rewrite, subjects: [...rewrite.subjects, form] })
            }}
          />
        </>
      ) : null}
      {rewrite.kind === 'tuple_to_userset' ? (
        <NameSelect
          id={`${idPrefix}-tupleset`}
          label={t.tuplesetRelation}
          names={relationNames}
          value={rewrite.tuplesetRelation}
          onChange={(tuplesetRelation) => onChange({ ...rewrite, tuplesetRelation })}
        />
      ) : null}
      {rewrite.kind === 'computed_userset' ? (
        <NameSelect
          id={`${idPrefix}-computed`}
          label={t.computedRelation}
          names={relationNames}
          value={rewrite.computedRelation}
          onChange={(computedRelation) => onChange({ ...rewrite, computedRelation })}
        />
      ) : null}
      {rewrite.kind === 'tuple_to_userset' ? (
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={`${idPrefix}-computed`}>{t.computedRelation}</Label>
          <Input
            id={`${idPrefix}-computed`}
            autoComplete="off"
            value={rewrite.computedRelation}
            onChange={(event) => onChange({ ...rewrite, computedRelation: event.target.value })}
          />
        </div>
      ) : null}
    </div>
  )
}

function RelationEditor({
  idPrefix,
  draft,
  type,
  relation,
  onChange,
  onRemove,
}: {
  idPrefix: string
  draft: ModelDraft
  type: ResourceTypeDraft
  relation: RelationDraft
  onChange: (relation: RelationDraft) => void
  onRemove: () => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)
  return (
    <div className="flex flex-col gap-3 border-l-2 border-slate-200 pl-3">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor={`${idPrefix}-name`}>{t.relationName}</Label>
          <Input
            id={`${idPrefix}-name`}
            autoComplete="off"
            aria-invalid={!isValidName(relation.name)}
            value={relation.name}
            onChange={(event) => onChange({ ...relation, name: event.target.value })}
          />
        </div>
        <Button type="button" variant="ghost" onClick={onRemove}>
          <IconTrash size={16} aria-hidden="true" />
          {t.removeRelation}
        </Button>
      </div>
      {relation.rewrites.map((rewrite, index) => (
        <RewriteEditor
          // biome-ignore lint/suspicious/noArrayIndexKey: rules have no identity other than their position
          key={`${index}-${rewrite.kind}`}
          idPrefix={`${idPrefix}-rewrite-${index}`}
          draft={draft}
          type={type}
          rewrite={rewrite}
          onChange={(next) =>
            onChange({ ...relation, rewrites: replaceAt(relation.rewrites, index, next) })
          }
          onRemove={() => onChange({ ...relation, rewrites: removeAt(relation.rewrites, index) })}
        />
      ))}
      <div>
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            onChange({ ...relation, rewrites: [...relation.rewrites, REWRITE_DEFAULTS.direct] })
          }
        >
          <IconPlus size={16} aria-hidden="true" />
          {t.addRewrite}
        </Button>
      </div>
    </div>
  )
}

export function ModelDraftForm({
  draft,
  onChange,
}: {
  draft: ModelDraft
  onChange: (draft: ModelDraft) => void
}) {
  const t = useDictionary(adminAuthorizationDictionary)

  function updateType(index: number, type: ResourceTypeDraft) {
    onChange({ resourceTypes: replaceAt(draft.resourceTypes, index, type) })
  }

  return (
    <div className="flex flex-col gap-3">
      {draft.resourceTypes.map((type, typeIndex) => (
        // biome-ignore lint/suspicious/noArrayIndexKey: the name is the edited value, so only the position is stable
        <Card key={typeIndex} className="flex flex-col gap-4 p-4">
          <div className="flex flex-wrap items-end justify-between gap-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor={`type-${typeIndex}-name`}>{t.resourceTypeName}</Label>
              <Input
                id={`type-${typeIndex}-name`}
                autoComplete="off"
                aria-invalid={!isValidName(type.name)}
                value={type.name}
                onChange={(event) => updateType(typeIndex, { ...type, name: event.target.value })}
              />
            </div>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onChange({ resourceTypes: removeAt(draft.resourceTypes, typeIndex) })}
            >
              <IconTrash size={16} aria-hidden="true" />
              {t.removeResourceType}
            </Button>
          </div>
          {type.relations.map((relation, relationIndex) => (
            <RelationEditor
              // biome-ignore lint/suspicious/noArrayIndexKey: the name is the edited value, so only the position is stable
              key={relationIndex}
              idPrefix={`type-${typeIndex}-relation-${relationIndex}`}
              draft={draft}
              type={type}
              relation={relation}
              onChange={(next) =>
                updateType(typeIndex, {
                  ...type,
                  relations: replaceAt(type.relations, relationIndex, next),
                })
              }
              onRemove={() =>
                updateType(typeIndex, {
                  ...type,
                  relations: removeAt(type.relations, relationIndex),
                })
              }
            />
          ))}
          <div>
            <Button
              type="button"
              variant="outline"
              onClick={() =>
                updateType(typeIndex, {
                  ...type,
                  relations: [...type.relations, { name: '', rewrites: [] }],
                })
              }
            >
              <IconPlus size={16} aria-hidden="true" />
              {t.addRelation}
            </Button>
          </div>
        </Card>
      ))}
      <div>
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            onChange({ resourceTypes: [...draft.resourceTypes, { name: '', relations: [] }] })
          }
        >
          <IconPlus size={16} aria-hidden="true" />
          {t.addResourceType}
        </Button>
      </div>
    </div>
  )
}
