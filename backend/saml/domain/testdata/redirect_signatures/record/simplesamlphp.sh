#!/bin/sh
# SimpleSAMLphp の SP が HTTP-Redirect で送る、署名した AuthnRequest と LogoutRequest を記録する。
# composer:2 のコンテナの中で実行し、記録の JSON を標準出力へ書く。
set -eu
SSP_VERSION=2.5.5
composer config -g allow-plugins true >/dev/null
composer create-project -q --no-interaction --no-dev --ignore-platform-reqs "simplesamlphp/simplesamlphp:$SSP_VERSION" /w >&2
cd /w
cp config/config.php.dist config/config.php
mkdir -p cert
# 記録のために作る鍵。記録には証明書だけを残し、秘密鍵はコンテナとともに捨てる。
openssl req -x509 -newkey rsa:2048 -nodes -keyout cert/sp.key -out cert/sp.crt -days 3650 \
  -subj "/CN=simplesamlphp recording sp" 2>/dev/null

cat > record.php <<'PHP'
<?php
require '/w/vendor/autoload.php';

use SimpleSAML\Configuration;
use SimpleSAML\Module\saml\Message;

Configuration::setConfigDir('/w/config');

$sp = Configuration::loadFromArray([
    'entityID' => 'https://sp.example.com',
    'entityid' => 'https://sp.example.com',
    'privatekey' => '/w/cert/sp.key',
    'certificate' => '/w/cert/sp.crt',
    'sign.authnrequest' => true,
    'sign.logout' => true,
    'redirect.sign' => true,
    'signature.algorithm' => 'http://www.w3.org/2001/04/xmldsig-more#rsa-sha256',
    'metadata-set' => 'saml20-sp-hosted',
], 'sp');
$idp = Configuration::loadFromArray([
    'entityID' => 'https://idp.example/realms/default/saml/metadata',
    'entityid' => 'https://idp.example/realms/default/saml/metadata',
    'SingleSignOnService' => [[
        'Binding' => 'urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect',
        'Location' => 'https://idp.example/realms/default/saml/sso',
    ]],
    'SingleLogoutService' => [[
        'Binding' => 'urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect',
        'Location' => 'https://idp.example/realms/default/saml/slo',
    ]],
    'metadata-set' => 'saml20-idp-remote',
], 'idp');

// SP の saml:SP が送る値に合わせ、RelayState には戻り先の URL を入れる。`~` と空白は、
// 符号化の規則の違いが綴りに現れる文字である。
$relayState = 'https://sp.example.com/~alice/start page.php?step=1';
$binding = new \SAML2\HTTPRedirect();

$ar = Message::buildAuthnRequest($sp, $idp);
$ar->setDestination('https://idp.example/realms/default/saml/sso');
$ar->setAssertionConsumerServiceURL('https://sp.example.com/acs');
$ar->setProtocolBinding('urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST');
$ar->setRelayState($relayState);
$authn = $binding->getRedirectURL($ar);

$lr = Message::buildLogoutRequest($sp, $idp);
$lr->setDestination('https://idp.example/realms/default/saml/slo');
$lr->setNameId(new \SAML2\XML\saml\NameID());
$lr->getNameId()->setValue('user-1');
$lr->setRelayState($relayState);
$logout = $binding->getRedirectURL($lr);

echo json_encode([
    'service_provider' => 'SimpleSAMLphp ' . getenv('SSP_VERSION'),
    'php' => PHP_VERSION,
    'certificate_pem' => file_get_contents('/w/cert/sp.crt'),
    'requests' => [
        ['message' => 'AuthnRequest', 'location' => $authn],
        ['message' => 'LogoutRequest', 'location' => $logout],
    ],
], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES) . "\n";
PHP
SSP_VERSION="$SSP_VERSION" php record.php
