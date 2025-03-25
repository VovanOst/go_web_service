package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные

type Column struct {
	Name     string
	Type     string
	Nullable bool
	Key      string // например, "PRI" для первичного ключа
	Extra    string
}

type Table struct {
	Name    string
	Columns []Column
}

type DbExplorer struct {
	db     *sql.DB
	tables map[string]Table
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	ctx := context.Background()
	// Получаем выделенное соединение
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	// После завершения инициализации возвращаем соединение в пул
	defer conn.Close()

	explorer := &DbExplorer{
		db:     db,
		tables: make(map[string]Table),
	}

	// Сначала полностью читаем список таблиц
	rows, err := conn.QueryContext(ctx, "SHOW TABLES;")
	if err != nil {
		return nil, err
	}
	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			rows.Close() // закрываем, если ошибка
			return nil, err
		}
		tableNames = append(tableNames, tableName)
	}
	rows.Close() // закрываем результаты запроса

	// Теперь, для каждой таблицы, получаем метаданные, используя то же соединение
	for _, tableName := range tableNames {
		columns, err := getTableColumnsConn(conn, tableName)
		if err != nil {
			log.Printf("Ошибка получения структуры таблицы %s: %v", tableName, err)
			continue
		}
		explorer.tables[tableName] = Table{
			Name:    tableName,
			Columns: columns,
		}
	}

	return explorer, nil
}

func getTableColumnsConn(conn *sql.Conn, tableName string) ([]Column, error) {
	ctx := context.Background()
	query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", tableName)
	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var field, colType, nullable, key, extra string
		var tmp interface{}
		// SHOW FULL COLUMNS возвращает 9 столбцов: Field, Type, Collation, Null, Key, Default, Extra, Privileges, Comment.
		if err := rows.Scan(&field, &colType, &tmp, &nullable, &key, &tmp, &extra, &tmp, &tmp); err != nil {
			return nil, err
		}
		columns = append(columns, Column{
			Name:     field,
			Type:     colType,
			Nullable: nullable == "YES",
			Key:      key,
			Extra:    extra,
		})
	}
	return columns, nil
}

func (d *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		d.handleGet(w, r)
	case http.MethodPost:
		d.handlePost(w, r)
	case http.MethodPut:
		d.handlePut(w, r)
	case http.MethodDelete:
		d.handleDelete(w, r)
	default:
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}

func (d *DbExplorer) handleGet(w http.ResponseWriter, r *http.Request) {
	pathParts := splitPath(r.URL.Path)

	if len(pathParts) == 0 {
		// Возвращаем список таблиц
		tableNames := make([]string, 0, len(d.tables))
		for name := range d.tables {
			tableNames = append(tableNames, name)
		}
		sort.Strings(tableNames)
		response := map[string]interface{}{
			"response": map[string]interface{}{
				"tables": tableNames,
			},
		}
		jsonResponse(w, response)
		return
	}

	tableName := pathParts[0]

	// Проверяем, есть ли такая таблица
	if _, exists := d.tables[tableName]; !exists {
		// Возвращаем JSON с ошибкой вместо простого текста
		errorResponse(w, "unknown table", http.StatusNotFound)
		return
	}

	// Если запрашивается конкретная запись `GET /table/id`
	if len(pathParts) == 2 {
		d.handleGetRow(w, tableName, pathParts[1])
		return
	}

	// Обрабатываем `GET /table?limit=5&offset=0`
	d.handleGetTable(w, r, tableName)
}

func (d *DbExplorer) handleGetTable(w http.ResponseWriter, r *http.Request, table string) {
	limit := 5
	offset := 0

	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		limit = l
	}

	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil {
		offset = o
	}

	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d OFFSET %d", table, limit, offset)
	rows, err := d.db.Query(query)
	if err != nil {
		errorResponse(w, "failed to query table", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		errorResponse(w, "failed to get columns", http.StatusInternalServerError)
		return
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		errorResponse(w, "failed to get column types", http.StatusInternalServerError)
		return
	}

	var records []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			errorResponse(w, "failed to scan row", http.StatusInternalServerError)
			return
		}

		record := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]

			// Обрабатываем NULL-значения
			if val == nil {
				record[colName] = nil
				continue
			}

			// Обрабатываем числовые и строковые значения
			switch columnTypes[i].DatabaseTypeName() {
			case "INT", "BIGINT", "TINYINT", "SMALLINT", "MEDIUMINT":
				// Если число, конвертируем в int
				if b, ok := val.([]byte); ok {
					num, err := strconv.Atoi(string(b))
					if err == nil {
						record[colName] = num
						continue
					}
				}
			case "FLOAT", "DOUBLE", "DECIMAL":
				// Если число с плавающей точкой, конвертируем в float64
				if b, ok := val.([]byte); ok {
					num, err := strconv.ParseFloat(string(b), 64)
					if err == nil {
						record[colName] = num
						continue
					}
				}
			default:
				// Оставляем строкой
				if b, ok := val.([]byte); ok {
					record[colName] = string(b)
				} else {
					record[colName] = val
				}
			}
		}

		records = append(records, record)
	}

	// Возвращаем JSON в правильном формате
	response := map[string]interface{}{
		"response": map[string]interface{}{
			"records": records,
		},
	}
	jsonResponse(w, response)
}

func (d *DbExplorer) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Ожидаем путь вида /table/id
	pathParts := splitPath(r.URL.Path)
	if len(pathParts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	tableName := pathParts[0]
	recordID := pathParts[1]

	// Проверяем, существует ли такая таблица
	if _, exists := d.tables[tableName]; !exists {
		errorResponse(w, "unknown table", http.StatusNotFound)
		return
	}

	// Формируем и выполняем DELETE-запрос
	query := fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", tableName)
	result, err := d.db.Exec(query, recordID)
	if err != nil {
		http.Error(w, "database error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "error retrieving affected rows: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем ответ в виде JSON
	response := map[string]interface{}{
		"response": map[string]interface{}{
			"deleted": rowsAffected,
		},
	}
	jsonResponse(w, response)
}

func (d *DbExplorer) handlePut(w http.ResponseWriter, r *http.Request) {
	// Извлекаем имя таблицы из пути, например, "/users/" → "users"
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.Error(w, `{"error": "invalid path"}`, http.StatusBadRequest)
		return
	}
	table := parts[0]

	tableMeta, ok := d.tables[table]
	if !ok {
		http.Error(w, `{"error": "unknown table"}`, http.StatusNotFound)
		return
	}

	// Определяем первичный ключ и автоинкрементное свойство
	pk := ""
	isAutoIncrement := false
	for _, col := range tableMeta.Columns {
		if col.Key == "PRI" {
			pk = col.Name
			if strings.Contains(strings.ToLower(col.Extra), "auto_increment") {
				isAutoIncrement = true
			}
			break
		}
	}
	if pk == "" {
		http.Error(w, `{"error": "table has no primary key"}`, http.StatusInternalServerError)
		return
	}

	// Распаковываем JSON-тело запроса в map
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, `{"error": "cant unpack json"}`, http.StatusBadRequest)
		return
	}

	// Если первичный ключ является автоинкрементным, удаляем его из данных
	if isAutoIncrement {
		delete(data, pk)
	}
	// Если для неавтоинкрементного pk значение передано, его оставляем (так как ожидается, что для таблицы users pk = user_id передаётся)

	// Формируем списки для INSERT, обходя все столбцы из метаданных
	var cols []string
	var placeholders []string
	var values []interface{}

	// Для каждого столбца, кроме автоинкрементного pk, если значение передано — используем его,
	// иначе, если поле НЕ NULL, подставляем значение по умолчанию, зависящее от типа.
	for _, col := range tableMeta.Columns {
		// Пропускаем автоинкрементный pk
		if isAutoIncrement && col.Name == pk {
			continue
		}
		cols = append(cols, fmt.Sprintf("`%s`", col.Name))
		if val, exists := data[col.Name]; exists {
			// Если значение передано, проверяем тип
			if val == nil {
				// Если поле не допускает NULL — ошибка
				if !col.Nullable {
					http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, col.Name), http.StatusBadRequest)
					return
				}
				placeholders = append(placeholders, "NULL")
			} else {
				if !isValidType(val, col.Type) {
					http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, col.Name), http.StatusBadRequest)
					return
				}
				placeholders = append(placeholders, "?")
				values = append(values, val)
			}
		} else {
			// Значение не передано в JSON.
			// Если поле допускает NULL, можно подставить NULL.
			// Если поле НЕ NULL, подставляем значение по умолчанию:
			// для текстовых типов — пустая строка, для числовых — 0.
			if col.Nullable {
				placeholders = append(placeholders, "NULL")
			} else {
				colTypeLower := strings.ToLower(col.Type)
				if strings.Contains(colTypeLower, "char") ||
					strings.Contains(colTypeLower, "text") ||
					strings.Contains(colTypeLower, "varchar") {
					placeholders = append(placeholders, "?")
					values = append(values, "")
				} else if strings.Contains(colTypeLower, "int") ||
					strings.Contains(colTypeLower, "float") ||
					strings.Contains(colTypeLower, "double") ||
					strings.Contains(colTypeLower, "decimal") {
					placeholders = append(placeholders, "?")
					values = append(values, 0)
				} else {
					// Если тип не распознан, подставляем пустую строку
					placeholders = append(placeholders, "?")
					values = append(values, "")
				}
			}
		}
	}

	// Если никаких полей для вставки не осталось — ошибка
	if len(cols) == 0 {
		http.Error(w, `{"error": "no valid fields provided"}`, http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", table, strings.Join(cols, ", "), strings.Join(placeholders, ", "))
	result, err := d.db.Exec(query, values...)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "database error: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	var insertedID int64
	if isAutoIncrement {
		insertedID, err = result.LastInsertId()
		if err != nil {
			http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
			return
		}
	} else {
		// Если pk не автоинкрементный, берем его значение из запроса
		if idVal, ok := data[pk]; ok {
			switch v := idVal.(type) {
			case float64:
				insertedID = int64(v)
			default:
				insertedID = 0
			}
		}
	}

	jsonResponse(w, map[string]interface{}{
		"response": map[string]interface{}{pk: insertedID},
	})
}

func (d *DbExplorer) handlePost(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.Error(w, `{"error": "invalid path"}`, http.StatusBadRequest)
		return
	}

	table := parts[0]
	id := parts[1]

	tableMeta, ok := d.tables[table]
	if !ok {
		http.Error(w, `{"error": "unknown table"}`, http.StatusNotFound)
		return
	}

	pk := "id"
	for _, col := range tableMeta.Columns {
		if col.Key == "PRI" {
			pk = col.Name
			break
		}
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, `{"error": "cant unpack json"}`, http.StatusBadRequest)
		return
	}

	if _, exists := data[pk]; exists {
		http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, pk), http.StatusBadRequest)
		return
	}

	setParts := []string{}
	values := []interface{}{}

	for key, val := range data {
		colType := ""
		isNullable := false
		for _, col := range tableMeta.Columns {
			if col.Name == key {
				colType = col.Type
				isNullable = col.Nullable
				break
			}
		}

		if colType == "" {
			http.Error(w, fmt.Sprintf(`{"error": "unknown field %s"}`, key), http.StatusBadRequest)
			return
		}

		if val == nil {
			if !isNullable {
				http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, key), http.StatusBadRequest)
				return
			}
			setParts = append(setParts, fmt.Sprintf("`%s` = NULL", key))
		} else {
			if !isValidType(val, colType) {
				http.Error(w, fmt.Sprintf(`{"error": "field %s have invalid type"}`, key), http.StatusBadRequest)
				return
			}
			setParts = append(setParts, fmt.Sprintf("`%s` = ?", key))
			values = append(values, val)
		}
	}

	values = append(values, id)
	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s` = ?", table, strings.Join(setParts, ", "), pk)

	result, err := d.db.Exec(query, values...)
	if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"response": map[string]int{"updated": int(rowsAffected)},
	})
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": message,
	})
}

// Функция получения структуры таблицы
func getTableColumns(db *sql.DB, tableName string) ([]Column, error) {
	rows, err := db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`;", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var field, colType, nullable string
		var tmp interface{} // временная переменная для пропуска лишних столбцов
		// SHOW FULL COLUMNS возвращает 9 столбцов: Field, Type, Collation, Null, Key, Default, Extra, Privileges, Comment.
		if err := rows.Scan(&field, &colType, &tmp, &nullable, &tmp, &tmp, &tmp, &tmp, &tmp); err != nil {
			return nil, err
		}

		columns = append(columns, Column{
			Name:     field,
			Type:     colType,
			Nullable: nullable == "YES",
		})
	}

	return columns, nil
}

func (d *DbExplorer) handleGetRow(w http.ResponseWriter, table, id string) {
	// Получаем метаданные таблицы
	tableMeta, ok := d.tables[table]
	if !ok || len(tableMeta.Columns) == 0 {
		http.Error(w, "Error getting table structure", http.StatusInternalServerError)
		return
	}

	// Определяем имя первичного ключа
	pk := getPrimaryKey(tableMeta)
	if pk == "" {
		http.Error(w, "Primary key not found", http.StatusInternalServerError)
		return
	}

	// Формируем SQL-запрос
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE `%s` = ? LIMIT 1", table, pk)
	row := d.db.QueryRow(query, id)

	// Подготавливаем слайсы для сканирования
	values := make([]interface{}, len(tableMeta.Columns))
	valuePtrs := make([]interface{}, len(tableMeta.Columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	err := row.Scan(valuePtrs...)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "record not found"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, `{"error": "database error"}`, http.StatusInternalServerError)
		return
	}

	// Формируем JSON-ответ
	record := make(map[string]interface{})
	for i, col := range tableMeta.Columns {
		val := values[i]
		if b, ok := val.([]byte); ok {
			if strings.HasPrefix(strings.ToLower(col.Type), "int") {
				if num, err := strconv.Atoi(string(b)); err == nil {
					record[col.Name] = num
					continue
				}
			}
			record[col.Name] = string(b)
		} else {
			record[col.Name] = val
		}
	}

	response := map[string]interface{}{
		"response": map[string]interface{}{
			"record": record,
		},
	}
	jsonResponse(w, response)
}

func getPrimaryKey(tableMeta Table) string {
	for _, col := range tableMeta.Columns {
		if col.Key == "PRI" {
			return col.Name
		}
	}
	return ""
}

func (d *DbExplorer) getTableColumns(table string) ([]string, error) {
	if tableInfo, exists := d.tables[table]; exists {
		columnNames := make([]string, len(tableInfo.Columns))
		for i, col := range tableInfo.Columns {
			columnNames[i] = col.Name
		}
		return columnNames, nil
	}

	query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`", table)
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var field, colType, colNull, colKey, colDefault, extra string
		if err := rows.Scan(&field, &colType, &colNull, &colKey, &colDefault, &extra); err != nil {
			return nil, err
		}
		columns = append(columns, Column{Name: field, Type: colType})
	}

	d.tables[table] = Table{Name: table, Columns: columns}

	// Конвертируем []Column в []string перед возвратом
	columnNames := make([]string, len(columns))
	for i, col := range columns {
		columnNames[i] = col.Name
	}

	return columnNames, nil
}

func isValidType(value interface{}, sqlType string) bool {
	switch value.(type) {
	case float64:
		return strings.Contains(strings.ToLower(sqlType), "int") ||
			strings.Contains(strings.ToLower(sqlType), "float") ||
			strings.Contains(strings.ToLower(sqlType), "double") ||
			strings.Contains(strings.ToLower(sqlType), "decimal")
	case string:
		return strings.Contains(strings.ToLower(sqlType), "char") ||
			strings.Contains(strings.ToLower(sqlType), "text") ||
			strings.Contains(strings.ToLower(sqlType), "varchar")
	case nil:
		return true
	default:
		return false
	}
}
