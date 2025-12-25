package main

import (
	"bytes"
	"context"
	_ "embed"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

//go:embed testdata/sql_example.wasm
var sqlWasm []byte

func TestSQLExample(t *testing.T) {
	if len(sqlWasm) == 0 {
		t.Skip("sql_example.wasm not built: run 'tinygo build -o testdata/sql_example.wasm -target=wasi testdata/sql_example.go'")
	}

	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	// Capture stdout
	var stdout bytes.Buffer

	// Instantiate WASI with SQL support
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Instantiate the module
	config := wazero.NewModuleConfig().
		WithStdout(&stdout).
		WithStderr(&stdout).
		WithStartFunctions("_start")

	mod, err := r.InstantiateWithConfig(ctx, sqlWasm, config)
	require.NoError(t, err)
	defer mod.Close(ctx)

	// Verify output
	output := stdout.String()
	require.Contains(t, output, "WASI SQL Example")
	require.Contains(t, output, "Database connection opened successfully")
	require.Contains(t, output, "Table 'users' created successfully")
	require.Contains(t, output, "Inserted")
	require.Contains(t, output, "Query executed successfully")
	require.Contains(t, output, "All SQL operations completed successfully")

	// Verify no errors occurred
	require.False(t, strings.Contains(output, "Failed"), "output should not contain error messages")
}
