# WASI SQL Example

This example demonstrates using the `wasi:sql@0.2.0-draft` API from a WebAssembly guest application compiled with TinyGo.

## Overview

The guest application (`testdata/sql_example.go`) performs the following SQL operations:
1. Opens a SQLite in-memory database connection
2. Creates a `users` table
3. Inserts a row into the table
4. Queries the table
5. Displays results

All database operations are performed through the WASI SQL API, which is implemented by the wazero host.

## Building the Guest Application

To build the WebAssembly module, you need TinyGo installed:

```bash
cd testdata
tinygo build -o sql_example.wasm -target=wasi sql_example.go
```

## Running the Example

```bash
cd examples/sql
go run .
```

Expected output:
```
WASI SQL Example
================
✓ Database connection opened successfully
✓ CREATE TABLE statement prepared
✓ Table 'users' created successfully
✓ Inserted 1 row(s)
✓ Query executed successfully

All SQL operations completed successfully!

WebAssembly SQL example completed successfully!
```

## Running the Test

```bash
go test -v
```

## How It Works

### Guest Application (WebAssembly)

The guest application imports WASI SQL host functions:
- `connection-open`: Opens a database connection
- `statement-prepare`: Prepares an SQL statement
- `exec`: Executes INSERT/UPDATE/DELETE statements
- `query`: Executes SELECT queries
- `error-trace`: Retrieves error messages

These functions are defined using `//go:wasm-module` and `//export` directives for TinyGo.

### Host Application (Go)

The host application:
1. Creates a wazero runtime
2. Instantiates WASI (including SQL support) using `wasi_snapshot_preview1.MustInstantiate`
3. Loads and runs the guest WebAssembly module
4. The SQL functions automatically connect to the SQLite3 driver

## Database Support

The current implementation supports any Go `database/sql` compatible driver. The connection string format is:

```
driver://dsn
```

Examples:
- SQLite in-memory: `sqlite3://:memory:`
- SQLite file: `sqlite3://path/to/database.db`
- MySQL: `mysql://user:password@tcp(localhost:3306)/dbname`
- PostgreSQL: `postgres://user:password@localhost/dbname?sslmode=disable`

Note: The appropriate driver must be imported in the host application.

## Limitations

The current implementation has some limitations:
- Query results are not yet fully serialized back to guest memory (returns empty list)
- Prepared statement parameters are not yet fully supported
- Transactions are not yet implemented
- Connection pooling is basic

These features can be added in future iterations of the wasi:sql implementation.

## Architecture

```
┌─────────────────────────────────┐
│   Guest Application (Wasm)      │
│   - TinyGo compiled             │
│   - Calls WASI SQL functions    │
└──────────────┬──────────────────┘
               │
               │ WASI SQL API
               │
┌──────────────▼──────────────────┐
│   wazero Runtime (Host)         │
│   - wasi_snapshot_preview1      │
│   - SQL host functions          │
└──────────────┬──────────────────┘
               │
               │ database/sql
               │
┌──────────────▼──────────────────┐
│   SQLite3 Driver                │
│   (or other SQL driver)         │
└─────────────────────────────────┘
```

## See Also

- [wasi:sql@0.2.0-draft specification](https://github.com/WebAssembly/wasi-sql)
- [wazero documentation](https://wazero.io)
- [TinyGo WASI target](https://tinygo.org/docs/guides/webassembly/wasi/)
