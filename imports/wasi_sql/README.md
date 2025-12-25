# wasi:sql

This package implements the WASI SQL API (`wasi:sql@0.2.0-draft`) as a no-op runtime module for wazero.

## Overview

The WASI SQL API provides a standard interface for WebAssembly modules to interact with SQL databases. This implementation exposes the API to guest applications but provides only no-op implementations.

## Usage

```go
package main

import (
	"context"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_sql"
)

func main() {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Instantiate the wasi:sql module
	wasi_sql.MustInstantiate(ctx, r)
	
	// Now instantiate your wasm module that imports wasi:sql functions
	// mod, _ := r.Instantiate(ctx, wasmBytes)
}
```

## API

The module exports the following functions according to the WIT specification:

### Types Interface

- **connection-open**: Opens a connection to a database (no-op, returns error)
- **statement-prepare**: Prepares a parameterized SQL statement (no-op, returns error)
- **error-trace**: Returns string representation of an error (no-op, returns empty string)

### Readwrite Interface

- **query**: Executes a query and returns rows (no-op, returns empty result)
- **exec**: Executes a statement that modifies data (no-op, returns 0 affected rows)

## WIT Specification

This implementation follows the `wasi:sql@0.2.0-draft` specification:

```wit
package wasi:sql@0.2.0-draft;

world imports {
    import readwrite;
}

interface readwrite {
    use types.{statement, row, error, connection};
    
    query: func(c: borrow<connection>, q: borrow<statement>) -> result<list<row>, error>;
    exec: func(c: borrow<connection>, q: borrow<statement>) -> result<u32, error>;
}

interface types {
    record row {
        field-name: string,
        value: data-type,
    }
    
    variant data-type {
        int32(s32),
        int64(s64),
        uint32(u32),
        uint64(u64),
        float(float64),
        double(float64),
        str(string),
        boolean(bool),
        date(string),
        time(string),
        timestamp(string),
        binary(list<u8>),
        null
    }

    resource statement {
        prepare: static func(query: string, params: list<string>) -> result<statement, error>;
    }
    
    resource error {
        trace: func() -> string;
    }
    
    resource connection {
        open: static func(name: string) -> result<connection, error>;
    }
}
```

## Notes

- All functions are no-op implementations
- Resource handles are not managed (as this is a no-op implementation)
- This module is intended to allow guest applications to link against the wasi:sql API without requiring actual database functionality
- For production use, you would need to implement actual database operations
