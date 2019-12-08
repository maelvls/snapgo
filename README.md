# Jest Inline Snapshots for Gomock

**Warning**: this is just a proof-of-concept, please do not use it.

I find Jest inline snapshotting great and wanted to have the same kind of
experience in Go. The issue in Go is that mocking requires a lot of manual
try-and-error when figuring out what input is expected for a mocked call.

The `snap.InlineSnapshot` can be used in gomock's `EXPECT`. Imagine you
have a deeply nested struct:

```go
mock.EXPECT().
    SomeFunction(snap.InlineSnapshot(deeplyNestedStruct))
```

Running with `.Update()` will update the snapshots and the
deeplyNestedStruct will get filled with the 'got' value:

```go
snap.InlineSnapshot(nil).Update().Matches(&struct{data string}{"foo"})
```

will become

```go
snap.InlineSnapshot(&struct{data string}{"foo"}).Matches(&struct{data string}{"foo"})
```

(remember to remove `Update()`)

## Caveats

- all the fields, even the nil ones, are shown
- problems with `*string` (and any pointer to constant literals)
