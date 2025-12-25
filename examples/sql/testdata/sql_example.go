//go:build tinygo.wasm

package main

import "unsafe"

// WASI SQL host function imports
//
//go:wasm-module wasi_snapshot_preview1
//export connection-open
func connectionOpen(namePtr, nameLen, resultPtr uint32) uint32

//go:wasm-module wasi_snapshot_preview1
//export statement-prepare
func statementPrepare(queryPtr, queryLen, paramsPtr, paramsLen, resultPtr uint32) uint32

//go:wasm-module wasi_snapshot_preview1
//export query
func sqlQuery(connHandle, stmtHandle, resultPtr uint32) uint32

//go:wasm-module wasi_snapshot_preview1
//export exec
func sqlExec(connHandle, stmtHandle, resultPtr uint32) uint32

//go:wasm-module wasi_snapshot_preview1
//export error-trace
func errorTrace(errorHandle, resultPtr, resultLenPtr uint32) uint32

func main() {
	println("WASI SQL Example")
	println("================")

	// Open a SQLite in-memory database
	connStr := "sqlite3://:memory:"
	connStrPtr := unsafe.Pointer(unsafe.StringData(connStr))
	var connResult uint32

	result := connectionOpen(
		uint32(uintptr(connStrPtr)),
		uint32(len(connStr)),
		uint32(uintptr(unsafe.Pointer(&connResult))),
	)

	if result != 0 {
		println("Failed to open database connection")
		printError(connResult)
		return
	}

	connHandle := connResult
	println("✓ Database connection opened successfully")

	// Create a table
	createTableSQL := "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"
	createTablePtr := unsafe.Pointer(unsafe.StringData(createTableSQL))
	var createStmtResult uint32

	result = statementPrepare(
		uint32(uintptr(createTablePtr)),
		uint32(len(createTableSQL)),
		0, 0, // no params
		uint32(uintptr(unsafe.Pointer(&createStmtResult))),
	)

	if result != 0 {
		println("Failed to prepare CREATE TABLE statement")
		printError(createStmtResult)
		return
	}

	createStmtHandle := createStmtResult
	println("✓ CREATE TABLE statement prepared")

	// Execute CREATE TABLE
	var execResult uint32
	result = sqlExec(
		connHandle,
		createStmtHandle,
		uint32(uintptr(unsafe.Pointer(&execResult))),
	)

	if result != 0 {
		println("Failed to execute CREATE TABLE")
		printError(execResult)
		return
	}

	println("✓ Table 'users' created successfully")

	// Insert data
	insertSQL := "INSERT INTO users (name, age) VALUES ('Alice', 30)"
	insertSQLPtr := unsafe.Pointer(unsafe.StringData(insertSQL))
	var insertStmtResult uint32

	result = statementPrepare(
		uint32(uintptr(insertSQLPtr)),
		uint32(len(insertSQL)),
		0, 0,
		uint32(uintptr(unsafe.Pointer(&insertStmtResult))),
	)

	if result != 0 {
		println("Failed to prepare INSERT statement")
		printError(insertStmtResult)
		return
	}

	insertStmtHandle := insertStmtResult

	result = sqlExec(
		connHandle,
		insertStmtHandle,
		uint32(uintptr(unsafe.Pointer(&execResult))),
	)

	if result != 0 {
		println("Failed to execute INSERT")
		printError(execResult)
		return
	}

	affectedRows := execResult
	println("✓ Inserted", affectedRows, "row(s)")

	// Query data
	selectSQL := "SELECT * FROM users"
	selectSQLPtr := unsafe.Pointer(unsafe.StringData(selectSQL))
	var selectStmtResult uint32

	result = statementPrepare(
		uint32(uintptr(selectSQLPtr)),
		uint32(len(selectSQL)),
		0, 0,
		uint32(uintptr(unsafe.Pointer(&selectStmtResult))),
	)

	if result != 0 {
		println("Failed to prepare SELECT statement")
		printError(selectStmtResult)
		return
	}

	selectStmtHandle := selectStmtResult

	var queryResultBuf [8]byte
	result = sqlQuery(
		connHandle,
		selectStmtHandle,
		uint32(uintptr(unsafe.Pointer(&queryResultBuf[0]))),
	)

	if result != 0 {
		println("Failed to execute SELECT")
		printError(queryResultBuf[0])
		return
	}

	println("✓ Query executed successfully")
	println("")
	println("All SQL operations completed successfully!")
}

func printError(errorHandle uint32) {
	var errorMsg [256]byte
	var errorLen uint32

	errorTrace(
		errorHandle,
		uint32(uintptr(unsafe.Pointer(&errorMsg[0]))),
		uint32(uintptr(unsafe.Pointer(&errorLen))),
	)

	if errorLen > 0 {
		println("Error:", string(errorMsg[:errorLen]))
	}
}
