#!/bin/sh
# Sustainsys.Saml2 の SP が HTTP-Redirect で送る、署名した AuthnRequest と LogoutRequest を記録する。
# mcr.microsoft.com/dotnet/sdk のコンテナの中で実行し、記録の JSON を標準出力へ書く。
set -eu
SUSTAINSYS_VERSION=2.11.2
export DOTNET_CLI_TELEMETRY_OPTOUT=1 DOTNET_NOLOGO=1
mkdir /w && cd /w
dotnet new console -o rec >/dev/null
cd rec
dotnet add package Sustainsys.Saml2 --version "$SUSTAINSYS_VERSION" >&2
cat > Program.cs <<'CS'
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
using System.Text.Json;
using Microsoft.IdentityModel.Tokens.Saml2;
using Sustainsys.Saml2;
using Sustainsys.Saml2.Metadata;
using Sustainsys.Saml2.Saml2P;
using Sustainsys.Saml2.WebSso;

// 記録のために作る鍵。記録には証明書だけを残し、秘密鍵はコンテナとともに捨てる。
using var rsa = RSA.Create(2048);
var csr = new CertificateRequest("CN=sustainsys recording sp", rsa, HashAlgorithmName.SHA256, RSASignaturePadding.Pkcs1);
var cert = csr.CreateSelfSigned(DateTimeOffset.UtcNow.AddDays(-1), DateTimeOffset.UtcNow.AddYears(10));
const string sigAlg = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256";
// RelayState は利用側が設定できず、Sustainsys.Saml2 が要求ごとに作る。

var binding = Saml2Binding.Get(Saml2BindingType.HttpRedirect);

var authn = new Saml2AuthenticationRequest
{
    DestinationUrl = new Uri("https://idp.example/realms/default/saml/sso"),
    AssertionConsumerServiceUrl = new Uri("https://sp.example.com/acs"),
    Issuer = new EntityId("https://sp.example.com"),
    SigningCertificate = cert,
    SigningAlgorithm = sigAlg,
};
var authnLocation = binding.Bind(authn).Location;

var logout = new Saml2LogoutRequest
{
    DestinationUrl = new Uri("https://idp.example/realms/default/saml/slo"),
    Issuer = new EntityId("https://sp.example.com"),
    NameId = new Saml2NameIdentifier("user-1"),
    SessionIndex = "session-1",
    SigningCertificate = cert,
    SigningAlgorithm = sigAlg,
};
var logoutLocation = binding.Bind(logout).Location;

var pem = "-----BEGIN CERTIFICATE-----\n" +
    Convert.ToBase64String(cert.RawData, Base64FormattingOptions.InsertLineBreaks).Replace("\r\n", "\n") +
    "\n-----END CERTIFICATE-----\n";
Console.WriteLine(JsonSerializer.Serialize(new
{
    service_provider = "Sustainsys.Saml2 " + Environment.GetEnvironmentVariable("SUSTAINSYS_VERSION"),
    dotnet = Environment.Version.ToString(),
    certificate_pem = pem,
    requests = new[]
    {
        new { message = "AuthnRequest", location = authnLocation.OriginalString },
        new { message = "LogoutRequest", location = logoutLocation.OriginalString },
    },
}, new JsonSerializerOptions { WriteIndented = true }));
CS
SUSTAINSYS_VERSION="$SUSTAINSYS_VERSION" dotnet run 2>&1 >/tmp/out.json | cat >&2
cat /tmp/out.json
