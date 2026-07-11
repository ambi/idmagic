import { IconPhoto, IconTrash, IconUpload } from '@tabler/icons-react'
import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import {
  AuthenticationAPIError,
  deleteTenantBrandingAsset,
  getBranding,
  updateBranding,
  uploadTenantBrandingAsset,
} from '../../api'
import { Alert } from '../../components/ui/alert'
import { Button } from '../../components/ui/button'
import { Card } from '../../components/ui/card'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'
import { Toast } from '../../components/ui/toast'
import {
  isHTTPSURL,
  isValidHexColor,
  MAX_TENANT_BRANDING_ASSET_BYTES,
  safeTenantBrandingAssetURL,
  validateTenantBrandingAssetFile,
} from '../../lib/tenantBranding'
import type { Branding, TenantBrandingAssetKind } from '../../types'

export function brandingSupportURLError(value: string): string | null {
  if (!value.trim()) return null
  return isHTTPSURL(value.trim()) ? null : 'https:// で始まる URL を指定してください。'
}

export function brandingFooterLinkError(label: string, url: string): string | null {
  const trimmedLabel = label.trim()
  const trimmedURL = url.trim()
  if (!trimmedLabel && !trimmedURL) return null
  if (!trimmedLabel) return 'リンクテキストを指定してください。'
  if (trimmedLabel.length > 80) return 'リンクテキストは80文字以内で指定してください。'
  return (
    brandingSupportURLError(trimmedURL) ?? (trimmedURL ? null : 'HTTPS URL を指定してください。')
  )
}

export function brandingColorError(value: string): string | null {
  if (!value.trim()) return null
  return isValidHexColor(value.trim()) ? null : '#rrggbb 形式の色コードを指定してください。'
}

export function BrandingTab({ csrfToken }: { csrfToken: string }) {
  const [branding, setBranding] = useState<Branding | null>(null)
  const [productName, setProductName] = useState('')
  const [primaryColor, setPrimaryColor] = useState('')
  const [accentColor, setAccentColor] = useState('')
  const [footerLink1Label, setFooterLink1Label] = useState('')
  const [footerLink1URL, setFooterLink1URL] = useState('')
  const [footerLink2Label, setFooterLink2Label] = useState('')
  const [footerLink2URL, setFooterLink2URL] = useState('')
  const [footerText, setFooterText] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  // biome-ignore lint/correctness/useExhaustiveDependencies: 初回マウント時のみ取得する
  useEffect(() => {
    let cancelled = false
    getBranding()
      .then((result) => {
        if (cancelled) return
        applyBranding(result)
      })
      .catch((cause) => {
        if (cancelled) return
        setError(
          cause instanceof AuthenticationAPIError
            ? cause.message
            : 'ブランディング設定を取得できませんでした。',
        )
      })
    return () => {
      cancelled = true
    }
  }, [])

  function applyBranding(next: Branding) {
    setBranding(next)
    setProductName(next.product_name ?? '')
    setPrimaryColor(next.primary_color ?? '')
    setAccentColor(next.accent_color ?? '')
    setFooterLink1Label(next.footer_link_1?.label ?? '')
    setFooterLink1URL(next.footer_link_1?.url ?? '')
    setFooterLink2Label(next.footer_link_2?.label ?? '')
    setFooterLink2URL(next.footer_link_2?.url ?? '')
    setFooterText(next.footer_text ?? '')
  }

  async function handleSave() {
    const footerLink1Error = brandingFooterLinkError(footerLink1Label, footerLink1URL)
    const footerLink2Error = brandingFooterLinkError(footerLink2Label, footerLink2URL)
    const primaryColorError = brandingColorError(primaryColor)
    const accentColorError = brandingColorError(accentColor)
    const firstError = footerLink1Error ?? footerLink2Error ?? primaryColorError ?? accentColorError
    if (firstError) {
      setError(firstError)
      return
    }
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const next = await updateBranding(csrfToken, {
        product_name: productName.trim(),
        primary_color: primaryColor.trim(),
        accent_color: accentColor.trim(),
        footer_link_1: { label: footerLink1Label.trim(), url: footerLink1URL.trim() },
        footer_link_2: { label: footerLink2Label.trim(), url: footerLink2URL.trim() },
        footer_text: footerText.trim(),
      })
      applyBranding(next)
      setNotice('ブランディング設定を更新しました。')
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : 'ブランディング設定を更新できませんでした。',
      )
    } finally {
      setSaving(false)
    }
  }

  if (!branding) {
    return (
      <Card className="p-6">{error ? <Alert variant="destructive">{error}</Alert> : null}</Card>
    )
  }

  return (
    <div className="grid gap-6">
      <Card className="grid gap-5 p-6 sm:grid-cols-2">
        <AssetUploader
          csrfToken={csrfToken}
          kind="logo"
          label="ロゴ"
          hint="login / consent / account portal のヘッダーに表示されます。"
          currentURL={branding.logo_url}
          onUpdated={applyBranding}
        />
        <AssetUploader
          csrfToken={csrfToken}
          kind="favicon"
          label="favicon"
          hint="ブラウザタブに表示されます。"
          currentURL={branding.favicon_url}
          onUpdated={applyBranding}
        />
      </Card>

      <Card className="p-6">
        <header>
          <h2 className="text-base font-semibold text-slate-900">表示名・配色・リンク</h2>
          <p className="mt-1 text-sm text-slate-600">
            未設定の項目は IdMagic の標準表示にフォールバックします。
          </p>
        </header>
        <div className="mt-5 grid gap-4">
          {error ? <Alert variant="destructive">{error}</Alert> : null}
          <Toast message={notice} onDismiss={() => setNotice('')} />
          <div className="grid gap-1.5">
            <Label htmlFor="branding-product-name">製品名</Label>
            <Input
              id="branding-product-name"
              value={productName}
              onChange={(event) => setProductName(event.target.value)}
              placeholder="IdMagic"
              maxLength={80}
            />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <ColorField
              id="branding-primary-color"
              label="プライマリカラー"
              value={primaryColor}
              onChange={setPrimaryColor}
              error={brandingColorError(primaryColor)}
            />
            <ColorField
              id="branding-accent-color"
              label="アクセントカラー"
              value={accentColor}
              onChange={setAccentColor}
              error={brandingColorError(accentColor)}
            />
          </div>
          <FooterLinkFields
            number={1}
            label={footerLink1Label}
            url={footerLink1URL}
            onLabelChange={setFooterLink1Label}
            onURLChange={setFooterLink1URL}
          />
          <FooterLinkFields
            number={2}
            label={footerLink2Label}
            url={footerLink2URL}
            onLabelChange={setFooterLink2Label}
            onURLChange={setFooterLink2URL}
          />
          <div className="grid gap-1.5">
            <Label htmlFor="branding-footer-text">フッターテキスト</Label>
            <Input
              id="branding-footer-text"
              value={footerText}
              onChange={(event) => setFooterText(event.target.value)}
              placeholder="(c) Acme Inc."
              maxLength={280}
            />
          </div>
          <div>
            <Button type="button" disabled={saving} onClick={() => void handleSave()}>
              {saving ? '保存中…' : '保存'}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  )
}

function FooterLinkFields({
  number,
  label,
  url,
  onLabelChange,
  onURLChange,
}: {
  number: 1 | 2
  label: string
  url: string
  onLabelChange: (value: string) => void
  onURLChange: (value: string) => void
}) {
  const error = brandingFooterLinkError(label, url)
  return (
    <fieldset className="grid gap-3 rounded-md border border-slate-200 p-3">
      <legend className="px-1 text-sm font-medium text-slate-800">フッターリンク {number}</legend>
      <div className="grid gap-1.5">
        <Label htmlFor={`branding-footer-link-${number}-label`}>リンクテキスト</Label>
        <Input
          id={`branding-footer-link-${number}-label`}
          value={label}
          onChange={(event) => onLabelChange(event.target.value)}
          placeholder="ヘルプ"
          maxLength={80}
        />
      </div>
      <div className="grid gap-1.5">
        <Label htmlFor={`branding-footer-link-${number}-url`}>HTTPS URL</Label>
        <Input
          id={`branding-footer-link-${number}-url`}
          value={url}
          onChange={(event) => onURLChange(event.target.value)}
          placeholder="https://example.com/help"
        />
      </div>
      {error ? <p className="text-xs text-red-700">{error}</p> : null}
    </fieldset>
  )
}

function ColorField({
  id,
  label,
  value,
  onChange,
  error,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  error: string | null
}) {
  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex items-center gap-2">
        <input
          type="color"
          aria-label={`${label}を選択`}
          value={isValidHexColor(value) ? value : '#0f172a'}
          onChange={(event) => onChange(event.target.value)}
          className="size-10 shrink-0 cursor-pointer rounded-md border border-slate-200"
        />
        <Input
          id={id}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="#0f172a"
          className="font-mono"
        />
      </div>
      {error ? <p className="text-xs text-red-700">{error}</p> : null}
    </div>
  )
}

function AssetUploader({
  csrfToken,
  kind,
  label,
  hint,
  currentURL,
  onUpdated,
}: {
  csrfToken: string
  kind: TenantBrandingAssetKind
  label: string
  hint: string
  currentURL?: string
  onUpdated: (next: Branding) => void
}) {
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const previewURL = safeTenantBrandingAssetURL(currentURL)

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setError('')
    const validationError = await validateTenantBrandingAssetFile(file)
    if (validationError) {
      setError(
        validationError === 'too-large'
          ? `画像は ${MAX_TENANT_BRANDING_ASSET_BYTES / 1024} KiB 以下にしてください。`
          : '画像は PNG / JPEG / WebP / GIF を選択してください。',
      )
      return
    }
    setUploading(true)
    try {
      onUpdated(await uploadTenantBrandingAsset(csrfToken, kind, file))
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError
          ? cause.message
          : '画像をアップロードできませんでした。',
      )
    } finally {
      setUploading(false)
    }
  }

  async function handleDelete() {
    setUploading(true)
    setError('')
    try {
      onUpdated(await deleteTenantBrandingAsset(csrfToken, kind))
    } catch (cause) {
      setError(
        cause instanceof AuthenticationAPIError ? cause.message : '画像を削除できませんでした。',
      )
    } finally {
      setUploading(false)
    }
  }

  return (
    <div className="grid gap-2">
      <p className="text-sm font-semibold text-slate-900">{label}</p>
      <p className="text-xs text-slate-500">{hint}</p>
      {error ? <p className="text-xs text-red-700">{error}</p> : null}
      <div className="flex items-center gap-3">
        <div className="flex size-14 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-slate-200 bg-slate-50">
          {previewURL ? (
            <img src={previewURL} alt="" className="size-full object-contain p-1" />
          ) : (
            <IconPhoto size={20} className="text-slate-400" aria-hidden="true" />
          )}
        </div>
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="default"
            disabled={uploading}
            onClick={() => inputRef.current?.click()}
          >
            <IconUpload size={16} aria-hidden="true" />
            アップロード
          </Button>
          {previewURL ? (
            <Button
              type="button"
              variant="ghost"
              size="default"
              disabled={uploading}
              onClick={() => void handleDelete()}
            >
              <IconTrash size={16} aria-hidden="true" />
              削除
            </Button>
          ) : null}
        </div>
        <input
          ref={inputRef}
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          className="hidden"
          onChange={(event) => void handleFileChange(event)}
        />
      </div>
    </div>
  )
}
