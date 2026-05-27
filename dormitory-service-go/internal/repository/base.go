package repository

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// defaultQueryContext returns a context with 30s timeout for DB queries.
// Used as fallback when no external context is provided.
func defaultQueryContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// BaseRepository provides common CRUD operations for any entity.
// T is the entity type, constrained by any (Go 1.18+).
type BaseRepository[T any] struct {
	DB        *sqlx.DB
	TableName string
}

// NewBaseRepository creates a new BaseRepository.
func NewBaseRepository[T any](db *sqlx.DB, tableName string) *BaseRepository[T] {
	return &BaseRepository[T]{
		DB:        db,
		TableName: tableName,
	}
}

// FindByID retrieves an entity by its primary key.
func (r *BaseRepository[T]) FindByID(ctx context.Context, id int64) (*T, error) {
	var entity T
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", r.TableName)
	err := r.DB.GetContext(ctx, &entity, query, id)
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

var allowedOrderBy = map[string]bool{
	"id": true, "created_at": true, "updated_at": true,
	"student_id": true, "building": true, "room": true,
	"camera_id": true, "event_time": true, "timestamp": true,
	"status": true, "severity": true, "detected_time": true,
	"config_key": true, "config_group": true, "sync_version": true,
	"name": true, "occurred_at": true, "report_date": true,
}

func sanitizeOrderBy(orderBy string) string {
	if orderBy == "" {
		return "created_at DESC"
	}
	field := strings.TrimSuffix(strings.TrimSuffix(orderBy, " ASC"), " DESC")
	field = strings.TrimSpace(field)
	if !allowedOrderBy[field] {
		return "created_at DESC"
	}
	return orderBy
}

// FindAll retrieves all entities with optional ordering.
func (r *BaseRepository[T]) FindAll(ctx context.Context, orderBy ...string) ([]T, error) {
	var entities []T
	query := fmt.Sprintf("SELECT * FROM %s", r.TableName)
	if len(orderBy) > 0 && orderBy[0] != "" {
		query += " ORDER BY " + sanitizeOrderBy(orderBy[0])
	}
	err := r.DB.SelectContext(ctx, &entities, query)
	if err != nil {
		return nil, err
	}
	return entities, nil
}

// getDBColumns extracts column names from db struct tags of entity type T.
// Returns all columns and columns excluding "id" (for UPDATE SET clause).
func getDBColumns[T any]() (all []string, withoutPK []string) {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return all, withoutPK
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		all = append(all, tag)
		if tag != "id" {
			withoutPK = append(withoutPK, tag)
		}
	}
	return
}

// Create inserts a new entity and returns the last insert ID.
// The entity must be a pointer to a struct with db tags.
func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) (int64, error) {
	columns, _ := getDBColumns[T]()
	if len(columns) == 0 {
		return 0, fmt.Errorf("insert %s: no columns found from db tags", r.TableName)
	}

	// Build: INSERT INTO table (col1, col2, col3) VALUES (:col1, :col2, :col3)
	colList := strings.Join(columns, ", ")
	paramList := make([]string, len(columns))
	for i, col := range columns {
		paramList[i] = ":" + col
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", r.TableName, colList, strings.Join(paramList, ", "))

	result, err := r.DB.NamedExecContext(ctx, query, entity)
	if err != nil {
		return 0, fmt.Errorf("insert %s: %w", r.TableName, err)
	}
	return result.LastInsertId()
}

// Update updates an entity by its ID.
func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) error {
	_, withoutPK := getDBColumns[T]()
	if len(withoutPK) == 0 {
		return fmt.Errorf("update %s: no columns found from db tags", r.TableName)
	}

	// Build: UPDATE table SET col1 = :col1, col2 = :col2 WHERE id = :id
	setClauses := make([]string, len(withoutPK))
	for i, col := range withoutPK {
		setClauses[i] = fmt.Sprintf("%s = :%s", col, col)
	}
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = :id", r.TableName, strings.Join(setClauses, ", "))

	_, err := r.DB.NamedExecContext(ctx, query, entity)
	if err != nil {
		return fmt.Errorf("update %s: %w", r.TableName, err)
	}
	return nil
}

// Delete removes an entity by its ID.
func (r *BaseRepository[T]) Delete(ctx context.Context, id int64) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", r.TableName)
	_, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete %s: %w", r.TableName, err)
	}
	return nil
}

// Count returns the total number of rows matching an optional WHERE clause.
func (r *BaseRepository[T]) Count(ctx context.Context, where string, args ...interface{}) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", r.TableName)
	if where != "" {
		query += " WHERE " + where
	}
	var count int64
	err := r.DB.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// FindWithPagination retrieves a page of entities with total count.
// whereClause is the WHERE part (without "WHERE"), e.g., "building = ? AND event_type = ?".
// args are the parameter values for the where clause.
// orderBy is the ORDER BY clause (without "ORDER BY"), e.g., "created_at DESC".
func (r *BaseRepository[T]) FindWithPagination(
	ctx context.Context,
	whereClause string,
	args []interface{},
	orderBy string,
	page int,
	size int,
) ([]T, int64, error) {
	if page < 1 {
		page = 1
	}
	const maxPageSize = 100
	if size <= 0 {
		size = 20
	} else if size > maxPageSize {
		size = maxPageSize
	}

	// Count total matching records
	total, err := r.Count(ctx, whereClause, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count %s: %w", r.TableName, err)
	}

	// Build the query
	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf("SELECT * FROM %s", r.TableName))
	if whereClause != "" {
		queryBuilder.WriteString(" WHERE " + whereClause)
	}
	if orderBy != "" {
		queryBuilder.WriteString(" ORDER BY " + sanitizeOrderBy(orderBy))
	} else {
		queryBuilder.WriteString(" ORDER BY id DESC")
	}

	offset := (page - 1) * size
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d", size, offset))

	var entities []T
	err = r.DB.SelectContext(ctx, &entities, queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select %s: %w", r.TableName, err)
	}

	return entities, total, nil
}

// TotalPages calculates the total number of pages.
func TotalPages(total int64, size int) int {
	if size <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(size)))
}
