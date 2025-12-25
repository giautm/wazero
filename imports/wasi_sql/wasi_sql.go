// Package wasi_sql implements the WASI SQL API as a no-op runtime module.
// This allows guest applications to import wasi:sql@0.2.0-draft functions
// without actual database functionality.
//
// The API follows the WIT specification for wasi:sql@0.2.0-draft which includes:
//   - types interface: connection, statement, error resources and data types
//   - readwrite interface: query and exec functions
//
// All functions are implemented as no-ops and return appropriate default or error values.
//
// See https://github.com/WebAssembly/wasi-sql for the specification.
package wasi_sql

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/wasm"
)

const (
	// ModuleName is the name of the wasi:sql module
	ModuleName = "wasi:sql@0.2.0-draft"

	i32 = wasm.ValueTypeI32
)

// MustInstantiate calls Instantiate or panics on error.
//
// This is a simpler function for those who know the module ModuleName is not
// already instantiated, and don't need to unload it.
func MustInstantiate(ctx context.Context, r wazero.Runtime) {
	if _, err := Instantiate(ctx, r); err != nil {
		panic(err)
	}
}

// Instantiate instantiates the ModuleName module into the runtime.
//
// # Notes
//
//   - Failure cases are documented on wazero.Runtime InstantiateModule.
//   - Closing the wazero.Runtime has the same effect as closing the result.
func Instantiate(ctx context.Context, r wazero.Runtime) (api.Closer, error) {
	return NewBuilder(r).Instantiate(ctx)
}

// Builder configures the ModuleName module for later use via Compile or Instantiate.
//
// # Notes
//
//   - This is an interface for decoupling, not third-party implementations.
//     All implementations are in wazero.
type Builder interface {
	// Compile compiles the ModuleName module. Call this before Instantiate.
	//
	// Note: This has the same effect as the same function on wazero.HostModuleBuilder.
	Compile(context.Context) (wazero.CompiledModule, error)

	// Instantiate instantiates the ModuleName module and returns a function to close it.
	//
	// Note: This has the same effect as the same function on wazero.HostModuleBuilder.
	Instantiate(context.Context) (api.Closer, error)
}

// NewBuilder returns a new Builder.
func NewBuilder(r wazero.Runtime) Builder {
	return &builder{r}
}

type builder struct{ r wazero.Runtime }

// hostModuleBuilder returns a new wazero.HostModuleBuilder for ModuleName
func (b *builder) hostModuleBuilder() wazero.HostModuleBuilder {
	ret := b.r.NewHostModuleBuilder(ModuleName)
	exportFunctions(ret)
	return ret
}

// Compile implements Builder.Compile
func (b *builder) Compile(ctx context.Context) (wazero.CompiledModule, error) {
	return b.hostModuleBuilder().Compile(ctx)
}

// Instantiate implements Builder.Instantiate
func (b *builder) Instantiate(ctx context.Context) (api.Closer, error) {
	return b.hostModuleBuilder().Instantiate(ctx)
}

// FunctionExporter exports functions into a wazero.HostModuleBuilder.
//
// # Notes
//
//   - This is an interface for decoupling, not third-party implementations.
//     All implementations are in wazero.
type FunctionExporter interface {
	ExportFunctions(wazero.HostModuleBuilder)
}

// NewFunctionExporter returns a new FunctionExporter.
func NewFunctionExporter() FunctionExporter {
	return &functionExporter{}
}

type functionExporter struct{}

// ExportFunctions implements FunctionExporter.ExportFunctions
func (functionExporter) ExportFunctions(builder wazero.HostModuleBuilder) {
	exportFunctions(builder)
}

// exportFunctions adds all go functions that implement wasi:sql.
// These should be exported in the module named ModuleName.
func exportFunctions(builder wazero.HostModuleBuilder) {
	exporter := builder.(wasm.HostFuncExporter)

	// Export resource constructor and method functions
	// connection resource
	exporter.ExportHostFunc(connectionOpen)

	// statement resource
	exporter.ExportHostFunc(statementPrepare)

	// error resource
	exporter.ExportHostFunc(errorTrace)

	// readwrite interface functions
	exporter.ExportHostFunc(query)
	exporter.ExportHostFunc(exec)
}

// sqlFunc is similar to wasiFunc but for sql functions
// Returns 0 on success, 1 on error
type sqlFunc func(ctx context.Context, mod api.Module, params []uint64) uint32

// Call implements the same method as documented on api.GoModuleFunction.
func (f sqlFunc) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(f(ctx, mod, stack))
}

func newHostFunc(
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
var connectionOpen = newHostFunc(
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
var statementPrepare = newHostFunc(
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
var errorTrace = newHostFunc(
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
var query = newHostFunc(
	"query",
	queryFn,
	[]wasm.ValueType{i32, i32, i32},
	"conn_handle", "stmt_handle", "result_ptr",
)

func queryFn(_ context.Context, mod api.Module, params []uint64) uint32 {
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
var exec = newHostFunc(
	"exec",
	execFn,
	[]wasm.ValueType{i32, i32, i32},
	"conn_handle", "stmt_handle", "result_ptr",
)

func execFn(_ context.Context, mod api.Module, params []uint64) uint32 {
	// No-op implementation: return 0 affected rows
	resultPtr := uint32(params[2])
	mem := mod.Memory()
	mem.WriteUint32Le(resultPtr, 0) // 0 affected rows
	return 0                        // success
}
