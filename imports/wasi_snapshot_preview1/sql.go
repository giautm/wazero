package wasi_snapshot_preview1

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// SQL functions implement the wasi:sql@0.2.0-draft API using Go's database/sql package.
//
// The API follows the WIT specification for wasi:sql@0.2.0-draft which includes:
//   - types interface: connection, statement, error resources and data types
//   - readwrite interface: query and exec functions

// sqlResources manages SQL connections, statements, and errors
type sqlResources struct {
	mu          sync.RWMutex
	connections map[int32]*sql.DB
	statements  map[int32]*sqlStatement
	errors      map[int32]string
	nextConnID  int32
	nextStmtID  int32
	nextErrID   int32
}

type sqlStatement struct {
	query  string
	params []string
}

var resources = &sqlResources{
	connections: make(map[int32]*sql.DB),
	statements:  make(map[int32]*sqlStatement),
	errors:      make(map[int32]string),
	nextConnID:  1,
	nextStmtID:  1,
	nextErrID:   1,
}

func (r *sqlResources) addConnection(db *sql.DB) int32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextConnID
	r.nextConnID++
	r.connections[id] = db
	return id
}

func (r *sqlResources) getConnection(id int32) (*sql.DB, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	db, ok := r.connections[id]
	return db, ok
}

func (r *sqlResources) addStatement(stmt *sqlStatement) int32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextStmtID
	r.nextStmtID++
	r.statements[id] = stmt
	return id
}

func (r *sqlResources) getStatement(id int32) (*sqlStatement, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stmt, ok := r.statements[id]
	return stmt, ok
}

func (r *sqlResources) addError(msg string) int32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextErrID
	r.nextErrID++
	r.errors[id] = msg
	return id
}

func (r *sqlResources) getError(id int32) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	msg, ok := r.errors[id]
	return msg, ok
}

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
	namePtr := uint32(params[0])
	nameLen := uint32(params[1])
	resultPtr := uint32(params[2])

	// Read the connection string from memory
	nameBytes, ok := mem.Read(namePtr, nameLen)
	if !ok {
		errID := resources.addError("failed to read connection string")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	connStr := string(nameBytes)

	// Parse driver and DSN from connection string (format: "driver://dsn")
	// For simplicity, we expect format like "sqlite3://path/to/db.db"
	driver := "sqlite3"
	dsn := connStr

	// Try to extract driver if connection string has :// separator
	if idx := indexOf(connStr, "://"); idx >= 0 {
		driver = connStr[:idx]
		dsn = connStr[idx+3:]
	}

	// Open database connection
	db, err := sql.Open(driver, dsn)
	if err != nil {
		errID := resources.addError(fmt.Sprintf("failed to open database: %v", err))
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		errID := resources.addError(fmt.Sprintf("failed to ping database: %v", err))
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	// Store connection and return handle
	connID := resources.addConnection(db)
	mem.WriteUint32Le(resultPtr, uint32(connID))
	return 0 // success
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
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
	queryPtr := uint32(params[0])
	queryLen := uint32(params[1])
	_ = uint32(params[2]) // paramsPtr - not used in simplified implementation
	paramsLen := uint32(params[3])
	resultPtr := uint32(params[4])

	// Read query string
	queryBytes, ok := mem.Read(queryPtr, queryLen)
	if !ok {
		errID := resources.addError("failed to read query string")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}
	query := string(queryBytes)

	// Read parameters list (for now, we store them but don't parse the complex list structure)
	// In a full implementation, this would parse the list<string> from memory
	var paramsList []string
	if paramsLen > 0 {
		// Simplified: just allocate empty list for now
		// A full implementation would parse the list structure from memory
		paramsList = []string{}
	}

	// Create statement
	stmt := &sqlStatement{
		query:  query,
		params: paramsList,
	}

	stmtID := resources.addStatement(stmt)
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
	errorHandle := int32(params[0])
	resultPtr := uint32(params[1])
	resultLenPtr := uint32(params[2])

	msg, ok := resources.getError(errorHandle)
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
	connHandle := int32(params[0])
	stmtHandle := int32(params[1])
	resultPtr := uint32(params[2])

	// Get connection
	db, ok := resources.getConnection(connHandle)
	if !ok {
		errID := resources.addError("invalid connection handle")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	// Get statement
	stmt, ok := resources.getStatement(stmtHandle)
	if !ok {
		errID := resources.addError("invalid statement handle")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	// Execute query
	rows, err := db.QueryContext(ctx, stmt.query)
	if err != nil {
		errID := resources.addError(fmt.Sprintf("query failed: %v", err))
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}
	defer rows.Close()

	// For now, return empty result (full implementation would serialize rows to memory)
	// A complete implementation would:
	// 1. Iterate through rows
	// 2. Serialize each row according to the WIT row structure
	// 3. Write serialized data to guest memory
	// 4. Return pointer and length
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
	connHandle := int32(params[0])
	stmtHandle := int32(params[1])
	resultPtr := uint32(params[2])

	// Get connection
	db, ok := resources.getConnection(connHandle)
	if !ok {
		errID := resources.addError("invalid connection handle")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	// Get statement
	stmt, ok := resources.getStatement(stmtHandle)
	if !ok {
		errID := resources.addError("invalid statement handle")
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	// Execute statement
	result, err := db.ExecContext(ctx, stmt.query)
	if err != nil {
		errID := resources.addError(fmt.Sprintf("exec failed: %v", err))
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	// Get affected rows
	affectedRows, err := result.RowsAffected()
	if err != nil {
		errID := resources.addError(fmt.Sprintf("failed to get affected rows: %v", err))
		mem.WriteUint32Le(resultPtr, uint32(errID))
		return 1 // error
	}

	mem.WriteUint32Le(resultPtr, uint32(affectedRows))
	return 0 // success
}
