namespace Personage.Auth.DataAccess.Models;

public class GmailTokenWithId : GmailToken
{
    public Guid Id { get; init; }
}