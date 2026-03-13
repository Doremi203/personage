using FluentAssertions;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Api.Configuration;

namespace Personage.Auth.Tests.Configuration;

[TestClass]
public class LockboxSecretConfigurationProviderTests
{
    [TestMethod]
    public void ParseSecretSpec_ValidFormat_ReturnsComponents()
    {
        // Arrange
        const string spec = "secret:e6q869h32umj7dap12qa:e6q1l84sfb3f05qqu03e:access_key_id";

        // Act
        var (secretId, versionId, key) = LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        // Assert
        secretId.Should().Be("e6q869h32umj7dap12qa");
        versionId.Should().Be("e6q1l84sfb3f05qqu03e");
        key.Should().Be("access_key_id");
    }

    [TestMethod]
    public void ParseSecretSpec_KeyWithUnderscores_ReturnsFullKey()
    {
        // Arrange
        const string spec = "secret:abc123:ver456:postgresql_password";

        // Act
        var (secretId, versionId, key) = LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        // Assert
        secretId.Should().Be("abc123");
        versionId.Should().Be("ver456");
        key.Should().Be("postgresql_password");
    }

    [TestMethod]
    public void ParseSecretSpec_KeyWithColons_ReturnsFullKeyIncludingColons()
    {
        // The key is everything after the third colon, so colons in the key are preserved.
        const string spec = "secret:abc123:ver456:some:complex:key";

        var (secretId, versionId, key) = LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        secretId.Should().Be("abc123");
        versionId.Should().Be("ver456");
        key.Should().Be("some:complex:key");
    }

    [TestMethod]
    public void ParseSecretSpec_TooFewParts_ThrowsFormatException()
    {
        const string spec = "secret:abc123:ver456";

        var act = () => LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        act.Should().Throw<FormatException>()
            .WithMessage("*Expected*secret:{id}:{version}:{key}*");
    }

    [TestMethod]
    public void ParseSecretSpec_WrongPrefix_ThrowsFormatException()
    {
        const string spec = "notsecret:abc123:ver456:key";

        var act = () => LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        act.Should().Throw<FormatException>()
            .WithMessage("*Expected*secret:{id}:{version}:{key}*");
    }

    [TestMethod]
    public void ParseSecretSpec_EmptySecretId_ThrowsFormatException()
    {
        const string spec = "secret::ver456:key";

        var act = () => LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        act.Should().Throw<FormatException>()
            .WithMessage("*Secret ID cannot be empty*");
    }

    [TestMethod]
    public void ParseSecretSpec_EmptyVersionId_ThrowsFormatException()
    {
        const string spec = "secret:abc123::key";

        var act = () => LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        act.Should().Throw<FormatException>()
            .WithMessage("*Version ID cannot be empty*");
    }

    [TestMethod]
    public void ParseSecretSpec_EmptyKey_ThrowsFormatException()
    {
        const string spec = "secret:abc123:ver456:";

        var act = () => LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        act.Should().Throw<FormatException>()
            .WithMessage("*Key cannot be empty*");
    }

    [TestMethod]
    public void ParseSecretSpec_OnlyPrefix_ThrowsFormatException()
    {
        const string spec = "secret:";

        var act = () => LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        act.Should().Throw<FormatException>();
    }

    [TestMethod]
    public void ParseSecretSpec_EmptyString_ThrowsFormatException()
    {
        const string spec = "";

        var act = () => LockboxSecretConfigurationProvider.ParseSecretSpec(spec);

        act.Should().Throw<FormatException>();
    }

    [TestMethod]
    public void SecretPrefix_IsCorrectValue()
    {
        LockboxSecretConfigurationProvider.SecretPrefix.Should().Be("secret:");
    }
}
