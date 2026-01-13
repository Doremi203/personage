using System;
using System.Collections.Generic;
using System.Threading.Tasks;

namespace Personage.Auth.Tests.Infrastructure;


public sealed class Cleaner
{
    private readonly Stack<Func<Task>> _cleanActions = [];

    public void AddCleanAction(Func<Task> cleanAction) => _cleanActions.Push(cleanAction);

    public async Task CleanCreatedObjects()
    {
        while (_cleanActions.TryPop(out var action))
            await action();
    }
}