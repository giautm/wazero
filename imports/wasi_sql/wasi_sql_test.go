package wasi_sql_test

import (
	"context"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_sql"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

var testCtx = context.Background()

func TestInstantiate(t *testing.T) {
	r := wazero.NewRuntime(testCtx)
	defer r.Close(testCtx)

	// Instantiate the wasi:sql module
	closer, err := wasi_sql.Instantiate(testCtx, r)
	require.NoError(t, err)
	require.NotNil(t, closer)
	defer closer.Close(testCtx)
}

func TestMustInstantiate(t *testing.T) {
	r := wazero.NewRuntime(testCtx)
	defer r.Close(testCtx)

	// Should not panic
	wasi_sql.MustInstantiate(testCtx, r)
}

func TestBuilder(t *testing.T) {
	r := wazero.NewRuntime(testCtx)
	defer r.Close(testCtx)

	builder := wasi_sql.NewBuilder(r)
	require.NotNil(t, builder)

	// Test Compile
	compiled, err := builder.Compile(testCtx)
	require.NoError(t, err)
	require.NotNil(t, compiled)

	// Test Instantiate
	closer, err := builder.Instantiate(testCtx)
	require.NoError(t, err)
	require.NotNil(t, closer)
	defer closer.Close(testCtx)
}

func TestFunctionExporter(t *testing.T) {
	r := wazero.NewRuntime(testCtx)
	defer r.Close(testCtx)

	exporter := wasi_sql.NewFunctionExporter()
	require.NotNil(t, exporter)

	// Create a custom module with the exporter
	builder := r.NewHostModuleBuilder("test-sql")
	exporter.ExportFunctions(builder)

	closer, err := builder.Instantiate(testCtx)
	require.NoError(t, err)
	require.NotNil(t, closer)
	defer closer.Close(testCtx)
}

func TestModuleFunctions(t *testing.T) {
	r := wazero.NewRuntime(testCtx)
	defer r.Close(testCtx)

	// Instantiate the wasi:sql module
	_, err := wasi_sql.Instantiate(testCtx, r)
	require.NoError(t, err)

	// Verify the module is registered
	mod := r.Module(wasi_sql.ModuleName)
	require.NotNil(t, mod)

	// The module name should match
	require.Equal(t, wasi_sql.ModuleName, mod.Name())
}
