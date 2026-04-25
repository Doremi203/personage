---
paths:
  - "**/*_test.go"
---

# Go Unit Tests

Table-driven tests with uber/gomock.

## Structure

```go
func TestSomething(t *testing.T) {
    type mocks struct {
        repo *mockrepo.MockRepo
    }
    type args struct {
        input string
    }
    tests := []struct {
        name    string
        args    args
        setup   func(m mocks, a args)
        want    SomeType
        wantErr require.ErrorAssertionFunc
    }{
        {
            name: "success",
            args: args{input: "valid"},
            setup: func(m mocks, a args) {
                m.repo.EXPECT().Find(gomock.Any(), a.input).Return(result, nil)
            },
            want:    expected,
            wantErr: require.NoError,
        },
        {
            name:    "not found",
            args:    args{input: "missing"},
            setup:   func(m mocks, a args) {
                m.repo.EXPECT().Find(gomock.Any(), a.input).Return(nil, repo.ErrNotFound)
            },
            wantErr: require.Error,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctrl := gomock.NewController(t)
            m := mocks{repo: mockrepo.NewMockRepo(ctrl)}
            tt.setup(m, tt.args)
            // ...
            tt.wantErr(t, err)
        })
    }
}
```

## Rules
- If context is needed use t.Context()
- `mocks` and `args` structs defined inside the test function
- `gomock` (`go.uber.org/mock/gomock`) with generated mocks (packages prefixed `mock*`)
- `require`/`assert` from `testify`; `assert.ErrorAssertionFunc` in test table
- Fresh `gomock.Controller` per subtest
- Cover all edge cases: not found, invalid input, tx failure, rollback, success
- Mocks go in a separate directory next to the public interface
- Mocks are auto generated during test build, so NEVER try find them. You MUST write expects according to interface and uber/mock generated methods API
