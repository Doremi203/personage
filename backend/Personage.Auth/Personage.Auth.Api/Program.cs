using Personage.Auth.GrpcServices;

namespace Personage.Auth;

public class Program
{
    public static void Main(string[] args)
    {
        var builder = WebApplication.CreateBuilder(args);
        builder.Services.AddGrpc(options =>
        {
            options.EnableDetailedErrors = true;
        });
        builder.Services.AddGrpcReflection();
        
        builder.Services.AddEndpointsApiExplorer();
        builder.Services.AddSwaggerGen(c =>
        {
            c.SwaggerDoc("v1", new Microsoft.OpenApi.Models.OpenApiInfo
            {
                Title = "Personage.Auth API",
                Version = "v1",
                Description = "Authentication API with gRPC and REST endpoints"
            });
        });
        builder.Services.AddControllers();
        var app = builder.Build();

        app.UseSwagger();
        app.UseSwaggerUI(c =>
        {
            c.SwaggerEndpoint("/swagger/v1/swagger.json", "Personage.Auth API v1");
            
            c.DisplayOperationId();
            c.DisplayRequestDuration();
        });
        
        app.MapGrpcReflectionService();
        app.MapControllers();
        
        app.UseGrpcWeb();
        app.MapGrpcService<TestGrpcService>().EnableGrpcWeb();
        
        app.Run();
    }
}