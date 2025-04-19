package adapter

import (
	"database/sql"
	"fmt"
)

// DBAdapter adapts a database
type DBAdapter struct {
	BaseAdapter
	DB          *sql.DB
	DriverName  string
	DataSource  string
	TablePrefix string
}

// NewDBAdapter creates a new database adapter
func NewDBAdapter(name, driverName, dataSource, tablePrefix string, config map[string]interface{}) *DBAdapter {
	base := NewBaseAdapter(name, DB, "Database Adapter", config)
	return &DBAdapter{
		BaseAdapter: *base,
		DriverName:  driverName,
		DataSource:  dataSource,
		TablePrefix: tablePrefix,
	}
}

// Initialize sets up the database adapter
func (a *DBAdapter) Initialize() error {
	db, err := sql.Open(a.DriverName, a.DataSource)
	if err != nil {
		return err
	}
	
	// Check connection
	err = db.Ping()
	if err != nil {
		return err
	}
	
	a.DB = db
	return nil
}

// GetCapabilities returns the capabilities of the database
func (a *DBAdapter) GetCapabilities() (map[string]interface{}, error) {
	// Query for tables
	query := "SELECT table_name FROM information_schema.tables WHERE table_name LIKE ?"
	rows, err := a.DB.Query(query, a.TablePrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	
	return map[string]interface{}{
		"type":   "database",
		"driver": a.DriverName,
		"tables": tables,
	}, nil
}

// ExecuteTask executes a database operation
func (a *DBAdapter) ExecuteTask(action string, params map[string]interface{}) (map[string]interface{}, error) {
	switch action {
	case "query":
		return a.executeQuery(params)
	case "execute":
		return a.executeStatement(params)
	default:
		return nil, fmt.Errorf("unsupported action: %s", action)
	}
}

// executeQuery executes a SELECT query
func (a *DBAdapter) executeQuery(params map[string]interface{}) (map[string]interface{}, error) {
	queryStr, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required")
	}
	
	rows, err := a.DB.Query(queryStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	
	// Prepare result
	var results []map[string]interface{}
	
	// Prepare value holders
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}
	
	// Iterate over rows
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		
		// Create row map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		
		results = append(results, row)
	}
	
	return map[string]interface{}{
		"results": results,
	}, nil
}

// executeStatement executes a non-SELECT statement
func (a *DBAdapter) executeStatement(params map[string]interface{}) (map[string]interface{}, error) {
	stmtStr, ok := params["statement"].(string)
	if !ok {
		return nil, fmt.Errorf("statement parameter is required")
	}
	
	result, err := a.DB.Exec(stmtStr)
	if err != nil {
		return nil, err
	}
	
	// Get affected rows
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	
	// Get last insert ID if available
	lastInsertID, _ := result.LastInsertId()
	
	return map[string]interface{}{
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
	}, nil
}

// Close cleans up resources
func (a *DBAdapter) Close() error {
	if a.DB != nil {
		return a.DB.Close()
	}
	return nil
}
