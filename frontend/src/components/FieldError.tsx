import { useDictionary } from '../lib/i18n'
import type { FieldProblem, FormSchema, FormValidation } from '../lib/form'
import { cn } from '../lib/utils'
import { fieldErrorDictionary } from './FieldError.i18n'

type Dictionary = (typeof fieldErrorDictionary)['en']

function fieldErrorMessage(problem: FieldProblem, t: Dictionary): string {
  switch (problem.kind) {
    case 'required':
      return t.required
    case 'tooLong':
      return t.tooLong.replace('{max}', String(problem.max))
    case 'tooMany':
      return t.tooMany.replace('{max}', String(problem.max))
    case 'invalidCidr':
      return t.invalidCidr.replace('{entry}', problem.entry)
    case 'positiveInteger':
      return t.positiveInteger
    case 'nonNegativeInteger':
      return t.nonNegativeInteger
    case 'integerRange':
      return t.integerRange
        .replace('{min}', String(problem.min))
        .replace('{max}', String(problem.max))
    case 'dateTime':
      return t.dateTime
    case 'invalidUrl':
      return problem.entry === undefined
        ? t.invalidUrl
        : t.invalidUrlEntry.replace('{entry}', problem.entry)
    case 'httpsUrl':
      return problem.entry === undefined
        ? t.httpsUrl
        : t.httpsUrlEntry.replace('{entry}', problem.entry)
    case 'resourceUrl':
      return t.resourceUrl
    case 'invalidScope':
      return t.invalidScope.replace('{entry}', problem.entry)
    case 'invalidPem':
      return t.invalidPem
    case 'invalidJson':
      return t.invalidJson
    case 'invalid':
      return t.invalid
    case 'message':
      return problem.text
  }
}

/**
 * 欄の問題を文章で示す。入力欄は `form.fieldProps(name)` の `aria-describedby` でこの要素を参照する。
 * 入力のたびに読み上げを割り込ませないよう、`role="alert"` は付けない。
 */
export function FieldError<S extends FormSchema>({
  form,
  name,
  className,
}: {
  form: FormValidation<S>
  name: Parameters<FormValidation<S>['problem']>[0]
  className?: string
}) {
  const t = useDictionary(fieldErrorDictionary)
  const problem = form.problem(name)
  if (!problem) return null
  return (
    <p id={form.errorId(name)} className={cn('text-xs leading-5 text-destructive', className)}>
      {fieldErrorMessage(problem, t)}
    </p>
  )
}
