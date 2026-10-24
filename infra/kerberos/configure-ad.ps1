param(
    [Parameter(Mandatory=$true)][ValidatePattern('^[A-Z0-9][A-Z0-9.-]+$')][string]$Realm,
    [Parameter(Mandatory=$true)][ValidatePattern('^[a-z0-9][a-z0-9.-]+$')][string]$DnsName,
    [Parameter(Mandatory=$true)][ValidatePattern('^[A-Za-z0-9_-]{1,20}$')][string]$ServiceAccount,
    [string]$OutputFile = '.\http.keytab'
)
$ErrorActionPreference = 'Stop'
Import-Module ActiveDirectory

# 既存アカウントと他サービスの SPN を変更せず、新規の専用アカウントだけを作る。
$Spn = "HTTP/$DnsName"
if (Test-Path -LiteralPath $OutputFile) { throw 'Output file already exists.' }
if (Get-ADUser -Filter "SamAccountName -eq '$ServiceAccount'") { throw 'Choose a new dedicated service account. Existing accounts are not modified.' }
if (Get-ADObject -LDAPFilter "(servicePrincipalName=$Spn)") { throw 'The SPN is already registered. Resolve ownership before continuing.' }
$Domain = Get-ADDomain
if ($Domain.DNSRoot.ToUpperInvariant() -cne $Realm) { throw 'The connected AD domain does not match Realm.' }

Write-Host "Create non-administrator service account $ServiceAccount and SPN $Spn."
if ((Read-Host 'Type CREATE to continue') -cne 'CREATE') { throw 'Cancelled.' }
$AccountPassword = Read-Host 'New service account password (use a long random value)' -AsSecureString
New-ADUser -Name $ServiceAccount -SamAccountName $ServiceAccount -UserPrincipalName "$ServiceAccount@$Realm" `
    -Enabled $true -AccountPassword $AccountPassword -KerberosEncryptionType AES256
Set-ADUser -Identity $ServiceAccount -AccountNotDelegated $true -TrustedForDelegation $false

# /pass * は対話入力であり、パスワードを引数や出力ファイルへ埋め込まない。
Write-Host 'Enter the service account password again for ktpass. The export can reset the service account key.'
& ktpass.exe /out $OutputFile /princ "$Spn@$Realm" /mapuser "$($Domain.NetBIOSName)\$ServiceAccount" `
    /mapop set /pass '*' /crypto AES256-SHA1 /ptype KRB5_NT_PRINCIPAL
if ($LASTEXITCODE -ne 0) { throw 'ktpass failed. The dedicated AD account may already exist; inspect it before retrying.' }
$OwnerSid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
& icacls.exe $OutputFile /inheritance:r /grant:r "*$($OwnerSid):(R,W)"
if ($LASTEXITCODE -ne 0) { throw 'Could not restrict keytab permissions. Secure the output file before using it.' }
Write-Host "Protect $OutputFile as a credential and deliver it to the API operator through an approved secret channel."
Write-Host 'This script has not verified a browser login. Configure the browser allowlist and perform the test procedure.'
