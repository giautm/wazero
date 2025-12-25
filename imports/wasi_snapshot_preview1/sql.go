package wasi_snapshot_preview1

import (
	"context"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// SQL functions implement the wasi:sql@0.2.0-draft API using Go's database/sql package.
//
// The API follows the WIT specification for wasi:sql@0.2.0-draft which includes:
//   - types interface: connection, statement, error resources and data types
//   - readwrite interface: query and exec functions

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
	[]api.ValueType{i32, i32, i32},
	"name_ptr", "name_len", "result_ptr",
)

func connectionOpenFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	mem := mod.Memory()
	sqlCtx := mod.(*wasm.ModuleInstance).Sys.SQL()
	namePtr := uint32(params[0])
	nameLen := uint32(params[1])
	resultPtr := uint32(params[2])

	// Read the connection string from memory
	nameBytes, ok := mem.Read(namePtr, nameLen)
	if !ok {
		errID := sqlCtx.AddError("failed to read connection string")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	connStr := string(nameBytes)

	// Open connection using SQLContext
	connHandle, errHandle, isError := sqlCtx.OpenConnection(connStr)
	if isError {
		mem.WriteUint32Le(resultPtr, uint32(errHandle))
		return 1 // error
	}

	mem.WriteUint32Le(resultPtr, uint32(connHandle))
	return 0 // success
}

// statement::prepare - Prepares a parameterized SQL statement
// Signature: (query: string, params: list<string>) -> result<statement, error>
// Parameters: query_ptr: i32, query_len: i32, params_ptr: i32, params_len: i32, result_ptr: i32
// Returns: i32 (0 = ok with handle in result_ptr, 1 = error with error handle in result_ptr)
var statementPrepare = newSQLHostFunc(
	"statement-prepare",
	statementPrepareFn,
	[]api.ValueType{i32, i32, i32, i32, i32},
	"query_ptr", "query_len", "params_ptr", "params_len", "result_ptr",
)

func statementPrepareFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	mem := mod.Memory()
	sqlCtx := mod.(*wasm.ModuleInstance).Sys.SQL()
	queryPtr := uint32(params[0])
	queryLen := uint32(params[1])
	_ = uint32(params[2]) // paramsPtr - not used in simplified implementation
	paramsLen := uint32(params[3])
	resultPtr := uint32(params[4])

	// Read query string
	queryBytes, ok := mem.Read(queryPtr, queryLen)
	if !ok {
		errID := sqlCtx.AddError("failed to read query string")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}
	query := string(queryBytes)

	// Prepare parameters list (simplified for now)
	var paramsList []string
	if paramsLen > 0 {
		paramsList = []string{}
	}

	// Create statement
	stmtID := sqlCtx.PrepareStatement(query, paramsList)
	mem.WriteUint32Le(resultPtr, uint32(stmtID))
	return 0 // success
}

// error::trace - Returns string representation of an error
// Signature: (error_handle: i32, result_ptr: i32, result_len_ptr: i32) -> void
// Parameters: error_handle: i32, result_ptr: i32, result_len_ptr: i32
// Returns: i32 (always 0)
var errorTrace = newSQLHostFunc(
	"error-trace",
	errorTraceFn,
	[]api.ValueType{i32, i32, i32},
	"error_handle", "result_ptr", "result_len_ptr",
)

func errorTraceFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	mem := mod.Memory()
	sqlCtx := mod.(*wasm.ModuleInstance).Sys.SQL()
	errorHandle := int32(params[0])
	resultPtr := uint32(params[1])
	resultLenPtr := uint32(params[2])

	msg, ok := sqlCtx.GetError(errorHandle)
	if !ok {
		msg = "unknown error"
	}

	// Write error message to memory
	msgBytes := []byte(msg)
	if !mem.Write(resultPtr, msgBytes) {
		mem.WriteUint32Le(resultLenPtr, 0)
		return 0
	}

	mem.WriteUint32Le(resultLenPtr, uint32(len(msgBytes)))
	return 0
}

// query - Executes a query and returns rows
// Signature: (conn: borrow<connection>, stmt: borrow<statement>) -> result<list<row>, error>
// Parameters: conn_handle: i32, stmt_handle: i32, result_ptr: i32
// Returns: i32 (0 = ok with row list in result_ptr, 1 = error with error handle in result_ptr)
var sqlQuery = newSQLHostFunc(
	"query",
	sqlQueryFn,
	[]api.ValueType{i32, i32, i32},
	"conn_handle", "stmt_handle", "result_ptr",
)

func sqlQueryFn(ctx context.Context, mod api.Module, params []uint64) uint32 {
	mem := mod.Memory()
	sqlCtx := mod.(*wasm.ModuleInstance).Sys.SQL()
	connHandle := int32(params[0])
	stmtHandle := int32(params[1])
	resultPtr := uint32(params[2])

	// Execute query
	rows, errHandle, isError := sqlCtx.Query(ctx, connHandle, stmtHandle)
	if isError {
		mem.WriteUint32Le(resultPtr, uint32(errHandle))
		return 1 // error
	}
	defer rows.Close()

	// For now, return empty result (full implementation would serialize rows to memory)
	mem.WriteUint32Le(resultPtr, 0)   // list pointer
	mem.WriteUint32Le(resultPtr+4, 0) // list length
	return 0                          // success
}

// exec - Executes a statement that modifies data
// Signature: (conn: borrow<connection>, stmt: borrow<statement>) -> result<u32, error>
// Parameters: conn_handle: i32, stmt_handle: i32, result_ptr: i32
// Returns: i32 (0 = ok with affected rows count in result_ptr, 1 = error with error handle in result_ptr)
var sqlExec = newSQLHostFunc(
	"exec",
	sqlExecFn,
	[]api.ValueType{i32, i32, i32},
	"conn_handle", "stmt_handle", "result_ptr",
)

func sqlExecFn(ctx context.Context, mod api.Module, params []uint64) uint32 {
	mem := mod.Memory()
	sqlCtx := mod.(*wasm.ModuleInstance).Sys.SQL()
	connHandle := int32(params[0])
	stmtHandle := int32(params[1])
	resultPtr := uint32(params[2])

	// Execute statement
	affectedRows, errHandle, isError := sqlCtx.Exec(ctx, connHandle, stmtHandle)
	if isError {
		mem.WriteUint32Le(resultPtr, uint32(errHandle))
		return 1 // error
	}

	mem.WriteUint32Le(resultPtr, uint32(affectedRows))
	return 0 // success
}
