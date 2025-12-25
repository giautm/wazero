package wasi_snapshot_preview1

import (
	"context"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// SQL functions implement the wasi:sql@0.2.0-draft API as no-ops.
// These allow guest applications to import SQL functions without actual database functionality.
//
// The API follows the WIT specification for wasi:sql@0.2.0-draft which includes:
//   - types interface: connection, statement, error resources and data types
//   - readwrite interface: query and exec functions
//
// All functions are implemented as no-ops and return appropriate default or error values.

// sqlFunc is similar to wasiFunc but for sql functions
// Returns 0 on success, 1 on error
type sqlFunc func(ctx context.Context, mod api.Module, params []uint64) uint32

// Call implements the same method as documented on api.GoModuleFunction.
func (f sqlFunc) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(f(ctx, mod, stack))
}

func newSQLHostFunc(
	name string,
	goFunc sqlFunc,
	paramTypes []api.ValueType,
	paramNames ...string,
) *wasm.HostFunc {
	return &wasm.HostFunc{
		ExportName:  name,
		Name:        name,
		ParamTypes:  paramTypes,
		ParamNames:  paramNames,
		ResultTypes: []api.ValueType{i32},
		ResultNames: []string{"result"},
		Code:        wasm.Code{GoFunc: goFunc},
	}
}

// connection::open - Opens a connection to a database
// Signature: (name: string) -> result<connection, error>
// Parameters: name_ptr: i32, name_len: i32, result_ptr: i32
// Returns: i32 (0 = ok with handle in result_ptr, 1 = error with error handle in result_ptr)
var connectionOpen = newSQLHostFunc(
	"connection-open",
	connectionOpenFn,
	[]wasm.ValueType{i32, i32, i32},
	"name_ptr", "name_len", "result_ptr",
)

func connectionOpenFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	// No-op implementation: return error
	return 1 // error
}

// statement::prepare - Prepares a parameterized SQL statement
// Signature: (query: string, params: list<string>) -> result<statement, error>
// Parameters: query_ptr: i32, query_len: i32, params_ptr: i32, params_len: i32, result_ptr: i32
// Returns: i32 (0 = ok with handle in result_ptr, 1 = error with error handle in result_ptr)
var statementPrepare = newSQLHostFunc(
	"statement-prepare",
	statementPrepareFn,
	[]wasm.ValueType{i32, i32, i32, i32, i32},
	"query_ptr", "query_len", "params_ptr", "params_len", "result_ptr",
)

func statementPrepareFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	// No-op implementation: return error
	return 1 // error
}

// error::trace - Returns string representation of an error
// Signature: (error_handle: i32, result_ptr: i32, result_len_ptr: i32) -> void
// Parameters: error_handle: i32, result_ptr: i32, result_len_ptr: i32
// Returns: i32 (always 0)
var errorTrace = newSQLHostFunc(
	"error-trace",
	errorTraceFn,
	[]wasm.ValueType{i32, i32, i32},
	"error_handle", "result_ptr", "result_len_ptr",
)

func errorTraceFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	// No-op implementation: return empty string
	resultLenPtr := uint32(params[2])
	mem := mod.Memory()
	mem.WriteUint32Le(resultLenPtr, 0) // length = 0
	return 0
}

// query - Executes a query and returns rows
// Signature: (conn: borrow<connection>, stmt: borrow<statement>) -> result<list<row>, error>
// Parameters: conn_handle: i32, stmt_handle: i32, result_ptr: i32
// Returns: i32 (0 = ok with row list in result_ptr, 1 = error with error handle in result_ptr)
var sqlQuery = newSQLHostFunc(
	"query",
	sqlQueryFn,
	[]wasm.ValueType{i32, i32, i32},
	"conn_handle", "stmt_handle", "result_ptr",
)

func sqlQueryFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	// No-op implementation: return empty result
	resultPtr := uint32(params[2])
	mem := mod.Memory()
	// Write empty list: pointer=0, length=0
	mem.WriteUint32Le(resultPtr, 0)   // list pointer
	mem.WriteUint32Le(resultPtr+4, 0) // list length
	return 0                          // success with empty result
}

// exec - Executes a statement that modifies data
// Signature: (conn: borrow<connection>, stmt: borrow<statement>) -> result<u32, error>
// Parameters: conn_handle: i32, stmt_handle: i32, result_ptr: i32
// Returns: i32 (0 = ok with affected rows count in result_ptr, 1 = error with error handle in result_ptr)
var sqlExec = newSQLHostFunc(
	"exec",
	sqlExecFn,
	[]wasm.ValueType{i32, i32, i32},
	"conn_handle", "stmt_handle", "result_ptr",
)

func sqlExecFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	// No-op implementation: return 0 affected rows
	resultPtr := uint32(params[2])
	mem := mod.Memory()
	mem.WriteUint32Le(resultPtr, 0) // 0 affected rows
	return 0                        // success
}
