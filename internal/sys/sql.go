package sys

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
)

// SQLContext holds SQL-related resources for a module.
type SQLContext struct {
	mu          sync.RWMutex
	connections map[int32]*sql.DB
	statements  map[int32]*SQLStatement
	errors      map[int32]string
	nextConnID  int32
	nextStmtID  int32
	nextErrID   int32
}

// SQLStatement represents a prepared SQL statement with parameters.
type SQLStatement struct {
	Query  string
	Params []string
}

// NewSQLContext creates a new SQLContext.
func NewSQLContext() *SQLContext {
	return &SQLContext{
		connections: make(map[int32]*sql.DB),
		statements:  make(map[int32]*SQLStatement),
		errors:      make(map[int32]string),
		nextConnID:  1,
		nextStmtID:  1,
		nextErrID:   1,
	}
}

// AddConnection stores a database connection and returns its handle.
func (c *SQLContext) AddConnection(db *sql.DB) int32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextConnID
	c.nextConnID++
	c.connections[id] = db
	return id
}

// GetConnection retrieves a database connection by handle.
func (c *SQLContext) GetConnection(id int32) (*sql.DB, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	db, ok := c.connections[id]
	return db, ok
}

// CloseConnection closes and removes a database connection.
func (c *SQLContext) CloseConnection(id int32) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	db, ok := c.connections[id]
	if !ok {
		return fmt.Errorf("invalid connection handle: %d", id)
	}
	delete(c.connections, id)
	return db.Close()
}

// AddStatement stores a statement and returns its handle.
func (c *SQLContext) AddStatement(stmt *SQLStatement) int32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextStmtID
	c.nextStmtID++
	c.statements[id] = stmt
	return id
}

// GetStatement retrieves a statement by handle.
func (c *SQLContext) GetStatement(id int32) (*SQLStatement, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stmt, ok := c.statements[id]
	return stmt, ok
}

// RemoveStatement removes a statement by handle.
func (c *SQLContext) RemoveStatement(id int32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.statements, id)
}

// AddError stores an error message and returns its handle.
func (c *SQLContext) AddError(msg string) int32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := c.nextErrID
	c.nextErrID++
	c.errors[id] = msg
	return id
}

// GetError retrieves an error message by handle.
func (c *SQLContext) GetError(id int32) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	msg, ok := c.errors[id]
	return msg, ok
}

// RemoveError removes an error by handle.
func (c *SQLContext) RemoveError(id int32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.errors, id)
}

// OpenConnection opens a new database connection with the given driver and DSN.
func (c *SQLContext) OpenConnection(connStr string) (connHandle int32, errHandle int32, isError bool) {
	// Parse driver and DSN from connection string (format: "driver://dsn")
	driver := "sqlite3"
	dsn := connStr

	// Try to extract driver if connection string has :// separator
	for i := 0; i <= len(connStr)-3; i++ {
		if connStr[i:i+3] == "://" {
			driver = connStr[:i]
			dsn = connStr[i+3:]
			break
		}
	}

	// Open database connection
	db, err := sql.Open(driver, dsn)
	if err != nil {
		errHandle = c.AddError(fmt.Sprintf("failed to open database: %v", err))
		return 0, errHandle, true
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		errHandle = c.AddError(fmt.Sprintf("failed to ping database: %v", err))
		return 0, errHandle, true
	}

	connHandle = c.AddConnection(db)
	return connHandle, 0, false
}

// PrepareStatement creates a new statement with the given query and parameters.
func (c *SQLContext) PrepareStatement(query string, params []string) int32 {
	stmt := &SQLStatement{
		Query:  query,
		Params: params,
	}
	return c.AddStatement(stmt)
}

// Query executes a SELECT query and returns the result.
func (c *SQLContext) Query(ctx context.Context, connHandle, stmtHandle int32) (rows *sql.Rows, errHandle int32, isError bool) {
	// Get connection
	db, ok := c.GetConnection(connHandle)
	if !ok {
		errHandle = c.AddError("invalid connection handle")
		return nil, errHandle, true
	}

	// Get statement
	stmt, ok := c.GetStatement(stmtHandle)
	if !ok {
		errHandle = c.AddError("invalid statement handle")
		return nil, errHandle, true
	}

	// Execute query
	rows, err := db.QueryContext(ctx, stmt.Query)
	if err != nil {
		errHandle = c.AddError(fmt.Sprintf("query failed: %v", err))
		return nil, errHandle, true
	}

	return rows, 0, false
}

// Exec executes an INSERT/UPDATE/DELETE statement and returns affected rows count.
func (c *SQLContext) Exec(ctx context.Context, connHandle, stmtHandle int32) (affectedRows int64, errHandle int32, isError bool) {
	// Get connection
	db, ok := c.GetConnection(connHandle)
	if !ok {
		errHandle = c.AddError("invalid connection handle")
		return 0, errHandle, true
	}

	// Get statement
	stmt, ok := c.GetStatement(stmtHandle)
	if !ok {
		errHandle = c.AddError("invalid statement handle")
		return 0, errHandle, true
	}

	// Execute statement
	result, err := db.ExecContext(ctx, stmt.Query)
	if err != nil {
		errHandle = c.AddError(fmt.Sprintf("exec failed: %v", err))
		return 0, errHandle, true
	}

	// Get affected rows
	affectedRows, err = result.RowsAffected()
	if err != nil {
		errHandle = c.AddError(fmt.Sprintf("failed to get affected rows: %v", err))
		return 0, errHandle, true
	}

	return affectedRows, 0, false
}

// Close closes all database connections.
func (c *SQLContext) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	for id, db := range c.connections {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close connection %d: %w", id, err))
		}
	}

	c.connections = make(map[int32]*sql.DB)
	c.statements = make(map[int32]*SQLStatement)
	c.errors = make(map[int32]string)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}
