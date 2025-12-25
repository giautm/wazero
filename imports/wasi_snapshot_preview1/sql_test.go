package wasi_snapshot_preview1_test

import (
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/internal/testing/require"
)

func TestSQLFunctions(t *testing.T) {
	mod, r, _ := requireProxyModule(t, wazero.NewModuleConfig())
	defer r.Close(testCtx)

	// Test that SQL functions are exported
	t.Run("connection-open", func(t *testing.T) {
		// Allocate memory for parameters
		namePtr := uint32(0)
		nameLen := uint32(0)
		resultPtr := uint32(100)

		maskMemory(t, mod, 200)

		// Call connection-open (should return error = 1)
		results, err := mod.ExportedFunction("connection-open").Call(testCtx, uint64(namePtr), uint64(nameLen), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(1), results[0]) // error
	})

	t.Run("statement-prepare", func(t *testing.T) {
		queryPtr := uint32(0)
		queryLen := uint32(0)
		paramsPtr := uint32(0)
		paramsLen := uint32(0)
		resultPtr := uint32(100)

		maskMemory(t, mod, 200)

		// Call statement-prepare (should return error = 1)
		results, err := mod.ExportedFunction("statement-prepare").Call(testCtx, uint64(queryPtr), uint64(queryLen), uint64(paramsPtr), uint64(paramsLen), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(1), results[0]) // error
	})

	t.Run("error-trace", func(t *testing.T) {
		errorHandle := uint32(0)
		resultPtr := uint32(100)
		resultLenPtr := uint32(104)

		maskMemory(t, mod, 200)

		// Call error-trace (should return 0 and write length 0)
		results, err := mod.ExportedFunction("error-trace").Call(testCtx, uint64(errorHandle), uint64(resultPtr), uint64(resultLenPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// Check that length is 0
		length, ok := mod.Memory().ReadUint32Le(resultLenPtr)
		require.True(t, ok)
		require.Equal(t, uint32(0), length)
	})

	t.Run("query", func(t *testing.T) {
		connHandle := uint32(0)
		stmtHandle := uint32(0)
		resultPtr := uint32(100)

		maskMemory(t, mod, 200)

		// Call query (should return 0 with empty result)
		results, err := mod.ExportedFunction("query").Call(testCtx, uint64(connHandle), uint64(stmtHandle), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// Check empty list: pointer=0, length=0
		listPtr, ok := mod.Memory().ReadUint32Le(resultPtr)
		require.True(t, ok)
		require.Equal(t, uint32(0), listPtr)

		listLen, ok := mod.Memory().ReadUint32Le(resultPtr + 4)
		require.True(t, ok)
		require.Equal(t, uint32(0), listLen)
	})

	t.Run("exec", func(t *testing.T) {
		connHandle := uint32(0)
		stmtHandle := uint32(0)
		resultPtr := uint32(100)

		maskMemory(t, mod, 200)

		// Call exec (should return 0 with 0 affected rows)
		results, err := mod.ExportedFunction("exec").Call(testCtx, uint64(connHandle), uint64(stmtHandle), uint64(resultPtr))
		require.NoError(t, err)
		require.Equal(t, uint64(0), results[0]) // success

		// Check affected rows = 0
		affectedRows, ok := mod.Memory().ReadUint32Le(resultPtr)
		require.True(t, ok)
		require.Equal(t, uint32(0), affectedRows)
	})
}
