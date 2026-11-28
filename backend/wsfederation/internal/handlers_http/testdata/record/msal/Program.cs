// MSAL.NET の AcquireTokenByUsernamePassword を、連携した（WS-Trust の）アカウントとして実行する。
// mise run record-msal が docker の .NET SDK で実行し、IdMagic が受け取った要求は Go の側が記録する。
//
// MSAL は Microsoft Entra ID の userrealm に問い合わせてアカウントの種類を決め、連携なら MEX を取得し、
// MEX から選んだ WS-Trust のエンドポイントへ RST を送る。userrealm だけを WithHttpClientFactory の
// 手元の応答で返す。MEX と RST は、MSAL 4.92 では WithHttpClientFactory を通らずに送られるので、
// コンテナの名前の解決で idp.example を IdMagic へ向ける。RSTR の後の Entra ID への要求には誤りを返して終える。
using System.Net;
using System.Text;
using Microsoft.Identity.Client;

string Env(string name) => Environment.GetEnvironmentVariable(name) ?? throw new InvalidOperationException($"{name} is required");

var app = PublicClientApplicationBuilder.Create(Env("CLIENT_ID"))
    .WithAuthority("https://login.microsoftonline.com/organizations")
    .WithInstanceDiscovery(false)
    .WithHttpClientFactory(new Factory(new EntraStub(Env("USERREALM_JSON"))))
    .Build();

string outcome;
try
{
#pragma warning disable CS0618 // ROPC は MSAL で非推奨だが、WS-Trust の RST を送るのはこの経路だけである。
    await app.AcquireTokenByUsernamePassword(new[] { "https://graph.microsoft.com/.default" }, Env("USERNAME"), Env("PASSWORD")).ExecuteAsync();
#pragma warning restore CS0618
    outcome = "token acquired";
}
catch (Exception e)
{
    outcome = $"{e.GetType().Name}: {e.Message}";
}
var msalVersion = typeof(PublicClientApplication).Assembly.GetName().Version?.ToString() ?? "unknown";
await File.WriteAllTextAsync("/out/outcome.txt", $"{msalVersion}\n{Environment.Version}\n{outcome}\n");
Console.WriteLine($"MSAL {msalVersion}: {outcome}");

sealed class Factory(HttpMessageHandler handler) : IMsalHttpClientFactory
{
    public HttpClient GetHttpClient() => new(handler, disposeHandler: false);
}

// EntraStub は login.microsoftonline.com への要求に答える。userrealm には連携のアカウントを答え、
// それ以外（RSTR の後のトークンの要求など）には誤りを返す。
sealed class EntraStub(string userRealmJSON) : HttpMessageHandler
{
    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
    {
        var uri = request.RequestUri!;
        Console.WriteLine($"-> {request.Method} {uri}");
        if (uri.Host == "login.microsoftonline.com" && uri.AbsolutePath.Contains("/userrealm/", StringComparison.OrdinalIgnoreCase))
        {
            return Task.FromResult(new HttpResponseMessage(HttpStatusCode.OK) { Content = new StringContent(userRealmJSON, Encoding.UTF8, "application/json") });
        }
        return Task.FromResult(new HttpResponseMessage(HttpStatusCode.BadRequest)
        {
            Content = new StringContent("{\"error\":\"invalid_request\",\"error_description\":\"not part of the recording\"}", Encoding.UTF8, "application/json"),
        });
    }
}
