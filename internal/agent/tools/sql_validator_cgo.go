//go:build cgo

package tools

import (
	"fmt"
	"regexp"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// SQLSecurityValidator provides comprehensive SQL injection protection using PostgreSQL's official parser
type SQLSecurityValidator struct {
	allowedTables    map[string]bool
	allowedFunctions map[string]bool
	tenantID         uint64
}

// NewSQLSecurityValidator creates a new SQL security validator
func NewSQLSecurityValidator(tenantID uint64) *SQLSecurityValidator {
	return &SQLSecurityValidator{
		allowedTables: map[string]bool{
			"tenants":         true,
			"knowledge_bases": true,
			"knowledges":      true,
			"sessions":        true,
			"messages":        true,
			"chunks":          true,
			"embeddings":      true,
			"models":          true,
		},
		// Whitelist of allowed SQL functions (aggregates and safe functions only)
		allowedFunctions: map[string]bool{
			// Aggregate functions
			"count":            true,
			"sum":              true,
			"avg":              true,
			"min":              true,
			"max":              true,
			"array_agg":        true,
			"string_agg":       true,
			"bool_and":         true,
			"bool_or":          true,
			"json_agg":         true,
			"jsonb_agg":        true,
			"json_object_agg":  true,
			"jsonb_object_agg": true,
			// Safe scalar functions
			"coalesce":          true,
			"nullif":            true,
			"greatest":          true,
			"least":             true,
			"abs":               true,
			"ceil":              true,
			"floor":             true,
			"round":             true,
			"length":            true,
			"lower":             true,
			"upper":             true,
			"trim":              true,
			"ltrim":             true,
			"rtrim":             true,
			"substring":         true,
			"concat":            true,
			"concat_ws":         true,
			"replace":           true,
			"left":              true,
			"right":             true,
			"now":               true,
			"current_date":      true,
			"current_timestamp": true,
			"date_trunc":        true,
			"extract":           true,
			"to_char":           true,
			"to_date":           true,
			"to_timestamp":      true,
			"date_part":         true,
			"age":               true,
		},
		tenantID: tenantID,
	}
}

// ValidateAndSecure performs comprehensive SQL validation using PostgreSQL's official parser
func (v *SQLSecurityValidator) ValidateAndSecure(sqlQuery string) (string, error) {
	// Phase 1: Basic input validation
	if err := v.validateInput(sqlQuery); err != nil {
		return "", err
	}

	// Phase 2: Parse SQL using PostgreSQL's official parser
	result, err := pg_query.Parse(sqlQuery)
	if err != nil {
		return "", fmt.Errorf("SQL parse error: %v", err)
	}

	// Phase 3: Validate that we have exactly one statement
	if len(result.Stmts) == 0 {
		return "", fmt.Errorf("empty query")
	}
	if len(result.Stmts) > 1 {
		return "", fmt.Errorf("multiple statements are not allowed")
	}

	stmt := result.Stmts[0].Stmt

	// Phase 4: Ensure it's a SELECT statement
	selectStmt := stmt.GetSelectStmt()
	if selectStmt == nil {
		return "", fmt.Errorf("only SELECT queries are allowed")
	}

	// Phase 5: Validate the SELECT statement recursively
	tablesInQuery, err := v.validateSelectStmt(selectStmt)
	if err != nil {
		return "", err
	}

	// Phase 6: Normalize SQL (removes comments, standardizes format)
	normalizedSQL, err := pg_query.Deparse(result)
	if err != nil {
		return "", fmt.Errorf("failed to normalize SQL: %v", err)
	}

	// Phase 7: Inject tenant_id conditions
	securedSQL := v.injectTenantConditions(normalizedSQL, tablesInQuery)

	return securedSQL, nil
}

// validateInput performs basic input validation
func (v *SQLSecurityValidator) validateInput(sql string) error {
	// Check for null bytes
	if strings.Contains(sql, "\x00") {
		return fmt.Errorf("invalid character in SQL query")
	}

	// Check length limits
	if len(sql) < 6 {
		return fmt.Errorf("SQL query too short")
	}
	if len(sql) > 4096 {
		return fmt.Errorf("SQL query too long (max 4096 characters)")
	}

	return nil
}

// validateSelectStmt validates a SELECT statement and extracts table information
func (v *SQLSecurityValidator) validateSelectStmt(stmt *pg_query.SelectStmt) (map[string]string, error) {
	tablesInQuery := make(map[string]string) // table name -> alias

	// Check for UNION/INTERSECT/EXCEPT (compound queries)
	if stmt.Op != pg_query.SetOperation_SETOP_NONE {
		return nil, fmt.Errorf("compound queries (UNION/INTERSECT/EXCEPT) are not allowed")
	}

	// Check for WITH clause (CTEs) - could be used for complex attacks
	if stmt.WithClause != nil {
		return nil, fmt.Errorf("WITH clause (CTEs) is not allowed")
	}

	// Check for INTO clause (SELECT INTO)
	if stmt.IntoClause != nil {
		return nil, fmt.Errorf("SELECT INTO is not allowed")
	}

	// Check for LOCKING clause (FOR UPDATE, etc.)
	if len(stmt.LockingClause) > 0 {
		return nil, fmt.Errorf("locking clauses (FOR UPDATE, etc.) are not allowed")
	}

	// Validate FROM clause
	for _, fromItem := range stmt.FromClause {
		if err := v.validateFromItem(fromItem, tablesInQuery); err != nil {
			return nil, err
		}
	}

	// Validate target list (SELECT columns)
	for _, target := range stmt.TargetList {
		if err := v.validateNode(target); err != nil {
			return nil, err
		}
	}

	// Validate WHERE clause
	if stmt.WhereClause != nil {
		if err := v.validateNode(stmt.WhereClause); err != nil {
			return nil, err
		}
	}

	// Validate GROUP BY clause
	for _, groupBy := range stmt.GroupClause {
		if err := v.validateNode(groupBy); err != nil {
			return nil, err
		}
	}

	// Validate HAVING clause
	if stmt.HavingClause != nil {
		if err := v.validateNode(stmt.HavingClause); err != nil {
			return nil, err
		}
	}

	// Validate ORDER BY clause
	for _, sortBy := range stmt.SortClause {
		if err := v.validateNode(sortBy); err != nil {
			return nil, err
		}
	}

	// Ensure at least one valid table is referenced
	if len(tablesInQuery) == 0 {
		return nil, fmt.Errorf("no valid table found in query")
	}

	return tablesInQuery, nil
}

// validateFromItem validates a FROM clause item
func (v *SQLSecurityValidator) validateFromItem(node *pg_query.Node, tables map[string]string) error {
	if node == nil {
		return nil
	}

	// Handle RangeVar (simple table reference)
	if rv := node.GetRangeVar(); rv != nil {
		tableName := strings.ToLower(rv.Relname)

		// Check for schema qualification (e.g., pg_catalog.pg_class)
		if rv.Schemaname != "" {
			schemaName := strings.ToLower(rv.Schemaname)
			// Block all schema-qualified access except public
			if schemaName != "public" {
				return fmt.Errorf("access to schema '%s' is not allowed", rv.Schemaname)
			}
		}

		// Validate table name against whitelist
		if !v.allowedTables[tableName] {
			return fmt.Errorf("table not allowed: %s", rv.Relname)
		}

		// Get alias
		alias := tableName
		if rv.Alias != nil && rv.Alias.Aliasname != "" {
			alias = strings.ToLower(rv.Alias.Aliasname)
		}
		tables[tableName] = alias
		return nil
	}

	// Handle JoinExpr (JOIN)
	if je := node.GetJoinExpr(); je != nil {
		if err := v.validateFromItem(je.Larg, tables); err != nil {
			return err
		}
		if err := v.validateFromItem(je.Rarg, tables); err != nil {
			return err
		}
		if je.Quals != nil {
			if err := v.validateNode(je.Quals); err != nil {
				return err
			}
		}
		return nil
	}

	// Handle RangeSubselect (subquery in FROM) - NOT ALLOWED
	if node.GetRangeSubselect() != nil {
		return fmt.Errorf("subqueries in FROM clause are not allowed")
	}

	// Handle RangeFunction (function in FROM) - NOT ALLOWED
	if node.GetRangeFunction() != nil {
		return fmt.Errorf("functions in FROM clause are not allowed")
	}

	return nil
}

// validateNode recursively validates AST nodes for security issues
func (v *SQLSecurityValidator) validateNode(node *pg_query.Node) error {
	if node == nil {
		return nil
	}

	// Check for subqueries (SubLink)
	if sl := node.GetSubLink(); sl != nil {
		return fmt.Errorf("subqueries are not allowed")
	}

	// Check for function calls
	if fc := node.GetFuncCall(); fc != nil {
		return v.validateFuncCall(fc)
	}

	// Check for column references with schema
	if cr := node.GetColumnRef(); cr != nil {
		return v.validateColumnRef(cr)
	}

	// Check for type casts (could be used for attacks)
	if tc := node.GetTypeCast(); tc != nil {
		if err := v.validateNode(tc.Arg); err != nil {
			return err
		}
		// Validate the target type
		if tc.TypeName != nil {
			typeName := v.getTypeName(tc.TypeName)
			if strings.HasPrefix(strings.ToLower(typeName), "pg_") {
				return fmt.Errorf("casting to system type '%s' is not allowed", typeName)
			}
		}
	}

	// Recursively check A_Expr (expressions)
	if ae := node.GetAExpr(); ae != nil {
		if err := v.validateNode(ae.Lexpr); err != nil {
			return err
		}
		if err := v.validateNode(ae.Rexpr); err != nil {
			return err
		}
	}

	// Check BoolExpr (AND, OR, NOT)
	if be := node.GetBoolExpr(); be != nil {
		for _, arg := range be.Args {
			if err := v.validateNode(arg); err != nil {
				return err
			}
		}
	}

	// Check NullTest
	if nt := node.GetNullTest(); nt != nil {
		if err := v.validateNode(nt.Arg); err != nil {
			return err
		}
	}

	// Check CoalesceExpr
	if ce := node.GetCoalesceExpr(); ce != nil {
		for _, arg := range ce.Args {
			if err := v.validateNode(arg); err != nil {
				return err
			}
		}
	}

	// Check CaseExpr
	if caseExpr := node.GetCaseExpr(); caseExpr != nil {
		if err := v.validateNode(caseExpr.Arg); err != nil {
			return err
		}
		for _, when := range caseExpr.Args {
			if err := v.validateNode(when); err != nil {
				return err
			}
		}
		if err := v.validateNode(caseExpr.Defresult); err != nil {
			return err
		}
	}

	// Check CaseWhen
	if cw := node.GetCaseWhen(); cw != nil {
		if err := v.validateNode(cw.Expr); err != nil {
			return err
		}
		if err := v.validateNode(cw.Result); err != nil {
			return err
		}
	}

	// Check ResTarget (SELECT list items)
	if rt := node.GetResTarget(); rt != nil {
		if err := v.validateNode(rt.Val); err != nil {
			return err
		}
	}

	// Check SortBy (ORDER BY items)
	if sb := node.GetSortBy(); sb != nil {
		if err := v.validateNode(sb.Node); err != nil {
			return err
		}
	}

	// Check List
	if list := node.GetList(); list != nil {
		for _, item := range list.Items {
			if err := v.validateNode(item); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateFuncCall validates a function call against the whitelist
func (v *SQLSecurityValidator) validateFuncCall(fc *pg_query.FuncCall) error {
	// Get function name
	funcName := ""
	for _, namePart := range fc.Funcname {
		if s := namePart.GetString_(); s != nil {
			funcName = strings.ToLower(s.Sval)
		}
	}

	// Check for schema-qualified function calls
	if len(fc.Funcname) > 1 {
		// Get schema name
		schemaName := ""
		if s := fc.Funcname[0].GetString_(); s != nil {
			schemaName = strings.ToLower(s.Sval)
		}
		// Block all schema-qualified function calls except pg_catalog for basic functions
		if schemaName != "" && schemaName != "pg_catalog" {
			return fmt.Errorf("schema-qualified function calls are not allowed: %s", schemaName)
		}
	}

	// Block dangerous function prefixes
	dangerousPrefixes := []string{
		"pg_", "lo_", "dblink", "file_", "copy_",
	}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(funcName, prefix) {
			return fmt.Errorf("function '%s' is not allowed (dangerous prefix)", funcName)
		}
	}

	// Block specific dangerous functions
	dangerousFunctions := map[string]bool{
		"current_setting": true,
		"set_config":      true,
		"query_to_xml":    true,
		"xpath":           true,
		"xmlparse":        true,
		"txid_current":    true,
	}
	if dangerousFunctions[funcName] {
		return fmt.Errorf("function '%s' is not allowed", funcName)
	}

	// Check against whitelist
	if !v.allowedFunctions[funcName] {
		return fmt.Errorf("function not allowed: %s", funcName)
	}

	// Validate function arguments recursively
	for _, arg := range fc.Args {
		if err := v.validateNode(arg); err != nil {
			return err
		}
	}

	return nil
}

// validateColumnRef validates a column reference
func (v *SQLSecurityValidator) validateColumnRef(cr *pg_query.ColumnRef) error {
	// Check for system column access
	for _, field := range cr.Fields {
		if s := field.GetString_(); s != nil {
			colName := strings.ToLower(s.Sval)
			// Block access to system columns
			systemColumns := []string{"xmin", "xmax", "cmin", "cmax", "ctid", "tableoid"}
			for _, sysCol := range systemColumns {
				if colName == sysCol {
					return fmt.Errorf("access to system column '%s' is not allowed", colName)
				}
			}
			// Block pg_ prefixed identifiers
			if strings.HasPrefix(colName, "pg_") {
				return fmt.Errorf("access to '%s' is not allowed", colName)
			}
		}
	}
	return nil
}

// getTypeName extracts the type name from a TypeName node
func (v *SQLSecurityValidator) getTypeName(tn *pg_query.TypeName) string {
	var parts []string
	for _, name := range tn.Names {
		if s := name.GetString_(); s != nil {
			parts = append(parts, s.Sval)
		}
	}
	return strings.Join(parts, ".")
}

// injectTenantConditions adds tenant_id filtering to the query
func (v *SQLSecurityValidator) injectTenantConditions(sql string, tablesInQuery map[string]string) string {
	// Tables that require tenant_id filtering
	tablesWithTenantID := map[string]bool{
		"tenants":         true,
		"knowledge_bases": true,
		"knowledges":      true,
		"sessions":        true,
		"chunks":          true,
	}

	// Build tenant conditions
	var conditions []string
	for tableName, alias := range tablesInQuery {
		if tablesWithTenantID[tableName] {
			if tableName == "tenants" {
				conditions = append(conditions, fmt.Sprintf("%s.id = %d", alias, v.tenantID))
			} else {
				conditions = append(conditions, fmt.Sprintf("%s.tenant_id = %d", alias, v.tenantID))
			}
		}
	}

	if len(conditions) == 0 {
		return sql
	}

	tenantFilter := strings.Join(conditions, " AND ")

	// Check if WHERE clause exists
	wherePattern := regexp.MustCompile(`(?i)\bWHERE\b`)
	if wherePattern.MatchString(sql) {
		// Add to existing WHERE clause
		return wherePattern.ReplaceAllString(sql, fmt.Sprintf("WHERE %s AND ", tenantFilter))
	}

	// Add new WHERE clause before ORDER BY, GROUP BY, LIMIT, etc.
	clausePattern := regexp.MustCompile(`(?i)\b(GROUP BY|ORDER BY|LIMIT|OFFSET|HAVING|FETCH)\b`)
	if loc := clausePattern.FindStringIndex(sql); loc != nil {
		return sql[:loc[0]] + fmt.Sprintf(" WHERE %s ", tenantFilter) + sql[loc[0]:]
	}

	// Add WHERE clause at the end
	return fmt.Sprintf("%s WHERE %s", sql, tenantFilter)
}
