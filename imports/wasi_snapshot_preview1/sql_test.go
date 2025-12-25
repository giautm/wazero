package wasi_snapshot_preview1_test

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestSQLFunctions(t *testing.T) {
	mod, r, _ := requireProxyModule(t, wazero.NewModuleConfig())
	defer r.Close(testCtx)

	t.Run("connection-open-success", func(t *testing.T) {
		mem := mod.Memory()

		// Write connection string to memory
		connStr := "sqlite3://:memory:"
		namePtr := uint32(1000)
		resultPtr := uint32(2000)

		maskMemory(t, mod, 3000)
		mem.Write(namePtr, []byte(connStr))

		// Call connection-open (should succeed)
		results, err := mod.ExportedFunction("connection-open").Call(testCtx, uint64(namePtr), uint64(len(connStr)), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// Check that a connection handle was written
		connHandle, ok := mem.ReadUint32Le(resultPtr)
		require.True(t, ok)
		require.NotEqual(t, uint32(0), connHandle) // should be non-zero handle
	})

	t.Run("connection-open-invalid-driver", func(t *testing.T) {
		mem := mod.Memory()

		// Write invalid connection string
		connStr := "invalid-driver://test"
		namePtr := uint32(1000)
		resultPtr := uint32(2000)

		maskMemory(t, mod, 3000)
		mem.Write(namePtr, []byte(connStr))

		// Call connection-open (should fail)
		results, err := mod.ExportedFunction("connection-open").Call(testCtx, uint64(namePtr), uint64(len(connStr)), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(1), results[0]) // error

		// Check that an error handle was written
		errHandle, ok := mem.ReadUint32Le(resultPtr)
		require.True(t, ok)
		require.NotEqual(t, uint32(0), errHandle) // should be non-zero error handle
	})

	t.Run("statement-prepare", func(t *testing.T) {
		mem := mod.Memory()

		query := "SELECT * FROM users WHERE id = ?"
		queryPtr := uint32(1000)
		paramsPtr := uint32(2000)
		paramsLen := uint32(0)
		resultPtr := uint32(3000)

		maskMemory(t, mod, 4000)
		mem.Write(queryPtr, []byte(query))

		// Call statement-prepare (should succeed)
		results, err := mod.ExportedFunction("statement-prepare").Call(testCtx, uint64(queryPtr), uint64(len(query)), uint64(paramsPtr), uint64(paramsLen), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// Check that a statement handle was written
		stmtHandle, ok := mem.ReadUint32Le(resultPtr)
		require.True(t, ok)
		require.NotEqual(t, uint32(0), stmtHandle) // should be non-zero handle
	})

	t.Run("error-trace", func(t *testing.T) {
		mem := mod.Memory()

		// First create an error by trying to open invalid connection
		connStr := "invalid://test"
		namePtr := uint32(1000)
		resultPtr := uint32(2000)

		maskMemory(t, mod, 5000)
		mem.Write(namePtr, []byte(connStr))

		// This should create an error
		results, err := mod.ExportedFunction("connection-open").Call(testCtx, uint64(namePtr), uint64(len(connStr)), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(1), results[0]) // error

		// Get the error handle
		errorHandle, ok := mem.ReadUint32Le(resultPtr)
		require.True(t, ok)

		// Now call error-trace
		traceResultPtr := uint32(3000)
		traceLenPtr := uint32(4000)

		results, err = mod.ExportedFunction("error-trace").Call(testCtx, uint64(errorHandle), uint64(traceResultPtr), uint64(traceLenPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// Check that error message length was written
		msgLen, ok := mem.ReadUint32Le(traceLenPtr)
		require.True(t, ok)
		require.NotEqual(t, uint32(0), msgLen) // should have some error message
	})

	t.Run("exec", func(t *testing.T) {
		mem := mod.Memory()

		// First open a connection
		connStr := "sqlite3://:memory:"
		namePtr := uint32(1000)
		connResultPtr := uint32(2000)

		maskMemory(t, mod, 10000)
		mem.Write(namePtr, []byte(connStr))

		results, err := mod.ExportedFunction("connection-open").Call(testCtx, uint64(namePtr), uint64(len(connStr)), uint64(connResultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0])

		connHandle, _ := mem.ReadUint32Le(connResultPtr)

		// Prepare a CREATE TABLE statement
		query := "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)"
		queryPtr := uint32(3000)
		stmtResultPtr := uint32(4000)

		mem.Write(queryPtr, []byte(query))

		results, err = mod.ExportedFunction("statement-prepare").Call(testCtx, uint64(queryPtr), uint64(len(query)), uint64(0), uint64(0), uint64(stmtResultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0])

		stmtHandle, _ := mem.ReadUint32Le(stmtResultPtr)

		// Execute the statement
		execResultPtr := uint32(5000)
		results, err = mod.ExportedFunction("exec").Call(testCtx, uint64(connHandle), uint64(stmtHandle), uint64(execResultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// Check affected rows (should be 0 for CREATE TABLE)
		affectedRows, ok := mem.ReadUint32Le(execResultPtr)
		require.True(t, ok)
		require.Equal(t, uint32(0), affectedRows)
	})

	t.Run("query", func(t *testing.T) {
		mem := mod.Memory()

		// First open a connection
		connStr := "sqlite3://:memory:"
		namePtr := uint32(1000)
		connResultPtr := uint32(2000)

		maskMemory(t, mod, 10000)
		mem.Write(namePtr, []byte(connStr))

		results, err := mod.ExportedFunction("connection-open").Call(testCtx, uint64(namePtr), uint64(len(connStr)), uint64(connResultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0])

		connHandle, _ := mem.ReadUint32Le(connResultPtr)

		// Prepare a SELECT statement
		query := "SELECT 1 as num, 'test' as str"
		queryPtr := uint32(3000)
		stmtResultPtr := uint32(4000)

		mem.Write(queryPtr, []byte(query))

		results, err = mod.ExportedFunction("statement-prepare").Call(testCtx, uint64(queryPtr), uint64(len(query)), uint64(0), uint64(0), uint64(stmtResultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0])

		stmtHandle, _ := mem.ReadUint32Le(stmtResultPtr)

		// Execute query
		queryResultPtr := uint32(5000)
		results, err = mod.ExportedFunction("query").Call(testCtx, uint64(connHandle), uint64(stmtHandle), uint64(queryResultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// For now, query returns empty list (full row serialization not implemented)
		listPtr, ok := mem.ReadUint32Le(queryResultPtr)
		require.True(t, ok)
		require.Equal(t, uint32(0), listPtr)
	})
}
