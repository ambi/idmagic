#!/bin/sh
# Shibboleth SP が HTTP-Redirect で送る、署名した AuthnRequest を記録する。
# debian のコンテナの中で Apache と shibd を起動し、SessionInitiator のリダイレクトを受け取る。
# 記録の JSON を標準出力へ書く。
set -eu
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq >&2
apt-get install -y -qq apache2 libapache2-mod-shib curl jq >&2
SHIB_VERSION=$(dpkg-query -W -f='${Version}' shibboleth-sp-utils 2>/dev/null || dpkg-query -W -f='${Version}' libapache2-mod-shib)

cd /etc/shibboleth
# 記録のために作る鍵。記録には証明書だけを残し、秘密鍵はコンテナとともに捨てる。
rm -f sp-signing-key.pem sp-signing-cert.pem sp-encrypt-key.pem sp-encrypt-cert.pem
shib-keygen -f -n sp-signing -h sp.example.com -e https://sp.example.com/shibboleth >&2
shib-keygen -f -n sp-encrypt -h sp.example.com -e https://sp.example.com/shibboleth >&2

cat > idp-metadata.xml <<'XML'
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://idp.example/realms/default/saml/metadata">
  <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol" WantAuthnRequestsSigned="true">
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example/realms/default/saml/slo"/>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example/realms/default/saml/sso"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>
XML

cat > shibboleth2.xml <<'XML'
<SPConfig xmlns="urn:mace:shibboleth:3.0:native:sp:config"
    xmlns:conf="urn:mace:shibboleth:3.0:native:sp:config"
    clockSkew="180">
    <OutOfProcess tranLogFormat="%u|%s|%IDP|%i|%ac|%t|%attr|%n|%b|%E|%S|%SS|%L|%UA|%a" />
    <ApplicationDefaults entityID="https://sp.example.com/shibboleth"
        REMOTE_USER="eppn" signing="true" signingAlg="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256">
        <Sessions lifetime="28800" timeout="3600" relayState="ss:mem"
                  checkAddress="false" handlerSSL="false" cookieProps="http">
            <SSO entityID="https://idp.example/realms/default/saml/metadata">SAML2</SSO>
            <Logout>SAML2 Local</Logout>
            <Handler type="Status" Location="/Status" acl="127.0.0.1 ::1"/>
            <Handler type="Session" Location="/Session" showAttributeValues="false"/>
        </Sessions>
        <Errors supportContact="root@localhost" helpLocation="/about.html" styleSheet="/shibboleth-sp/main.css"/>
        <MetadataProvider type="XML" validate="true" path="idp-metadata.xml"/>
        <AttributeExtractor type="XML" validate="true" reloadChanges="false" path="attribute-map.xml"/>
        <AttributeFilter type="XML" validate="true" path="attribute-policy.xml"/>
        <CredentialResolver type="File" use="signing" key="sp-signing-key.pem" certificate="sp-signing-cert.pem"/>
        <CredentialResolver type="File" use="encryption" key="sp-encrypt-key.pem" certificate="sp-encrypt-cert.pem"/>
    </ApplicationDefaults>
    <SecurityPolicyProvider type="XML" validate="true" path="security-policy.xml"/>
    <ProtocolProvider type="XML" validate="true" reloadChanges="false" path="protocols.xml"/>
</SPConfig>
XML

chown _shibd sp-*.pem 2>/dev/null || true
shibd -t >&2
mkdir -p /run/shibboleth
shibd -f -p /run/shibboleth/shibd.pid >&2
a2enmod shib >&2
apache2ctl start >&2
sleep 3

# SessionInitiator は target を状態に保存し、RelayState には保存した状態への参照を入れる。
# `~` と空白は、符号化の規則の違いが綴りに現れる文字である。
location=$(curl -s -o /dev/null -w '%{redirect_url}' \
  'http://localhost/Shibboleth.sso/Login?target=http%3A%2F%2Flocalhost%2F~alice%2Fstart%20page.html%3Fstep%3D1')
[ -n "$location" ] || { echo "no redirect from the SessionInitiator" >&2; exit 1; }

jq -n --arg sp "Shibboleth SP (Debian libapache2-mod-shib $SHIB_VERSION)" \
  --rawfile cert sp-signing-cert.pem --arg location "$location" \
  '{service_provider: $sp, certificate_pem: $cert, requests: [{message: "AuthnRequest", location: $location}]}'
