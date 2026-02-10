# Using kvx with Custom Structs

This example demonstrates how to use kvx as a library with custom Go structs, which is the solution to the "unsupported conversion to ref.Val" error.

## The Problem

When using kvx as a library and passing custom structs (like `catalog.ArtifactListItem`), you might encounter:

```
Error: CEL evaluation error: eval error: unsupported conversion to ref.Val: (catalog.ArtifactListItem){...}
```

This happens because CEL (the expression language used by kvx) needs to convert Go values to its internal representation (`ref.Val`), and it doesn't natively support arbitrary custom struct types.

## The Solution

kvx now automatically converts custom structs to maps via JSON serialization when you use `LoadObject()`. This ensures:

1. **Full CEL Compatibility**: Expressions can be evaluated on custom data
2. **Data Preservation**: JSON tags are respected, so your struct data is preserved
3. **Transparent**: No special handling needed - just pass your struct to `LoadObject()`

## Usage

```go
import "github.com/oakwood-commons/kvx/pkg/core"

// Define your custom struct
type Artifact struct {
    Name    string
    Version string
    SHA     string
}

// Load it with kvx
artifact := &Artifact{
    Name:    "my-service",
    Version: "1.0.0",
    SHA:     "abc123...",
}

// kvx converts this to a map automatically
root, err := core.LoadObject(artifact)
if err != nil {
    panic(err)
}

// Create an engine and evaluate expressions
engine, _ := core.New()

// Now CEL expressions work!
result, _ := engine.Evaluate("_.Name", root)
fmt.Println(result) // Output: my-service
```

## Running the Example

```bash
go run ./examples/custom_struct/main.go
```

This demonstrates:
- Loading a single custom struct
- Evaluating CEL expressions on the struct
- Loading a slice of custom structs
- Filtering using CEL on multiple artifacts
