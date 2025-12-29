using Personage.Auth.GrpcServices;

namespace Personage.Auth;

public class Program
{
    public static void Main(string[] args)
    {
        var builder = WebApplication.CreateBuilder(args);
        builder.WebHost.ConfigureKestrel(options =>
        {
            options.ConfigureEndpointDefaults(lo => lo.Protocols = Microsoft.AspNetCore.Server.Kestrel.Core.HttpProtocols.Http2);
        });
        builder.Services.AddGrpc();

        builder.Services.AddControllers();
        var app = builder.Build();
        
        app.MapControllers();
        
        app.MapGrpcService<TestGrpcService>();

        app.Run();
    }
}