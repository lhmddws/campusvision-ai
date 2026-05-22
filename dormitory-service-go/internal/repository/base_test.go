package repository_test

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/sims/campusvision/dormitory-service-go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEntity is a simple struct with db tags used to test BaseRepository CRUD methods.
type testEntity struct {
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	Building string `db:"building"`
}

// noTagEntity has no db tags — used to test the getDBColumns error path via Create.
type noTagEntity struct {
	ID   int64
	Name string
}

// newMockRepo creates a go-sqlmock backed BaseRepository[testEntity] for testing.
func newMockRepo(t *testing.T) (sqlmock.Sqlmock, *repository.BaseRepository[testEntity]) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "mysql")
	repo := repository.NewBaseRepository[testEntity](sqlxDB, "test_table")
	t.Cleanup(func() { _ = db.Close() })
	return mock, repo
}

// ---------- getDBColumns (indirect via Create) ----------

func TestBase_Create_ReturnsErrorWhenNoDbTags(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "mysql")
	repo := repository.NewBaseRepository[noTagEntity](sqlxDB, "no_tag_table")

	var e noTagEntity
	_, err = repo.Create(&e)
	assert.ErrorContains(t, err, "no columns found from db tags")
}

func TestBase_Create_BuildsInsertQueryFromDbTags(t *testing.T) {
	mock, repo := newMockRepo(t)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO test_table (id, name, building) VALUES (?, ?, ?)")).
		WithArgs(int64(0), "Alice", "A").
		WillReturnResult(sqlmock.NewResult(1, 1))

	entity := &testEntity{Name: "Alice", Building: "A"}
	id, err := repo.Create(entity)
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- FindByID ----------

func TestBase_FindByID_ReturnsEntityWhenFound(t *testing.T) {
	mock, repo := newMockRepo(t)

	rows := sqlmock.NewRows([]string{"id", "name", "building"}).
		AddRow(int64(42), "Alice", "A")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM test_table WHERE id = ? LIMIT 1")).
		WithArgs(int64(42)).
		WillReturnRows(rows)

	entity, err := repo.FindByID(42)
	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, int64(42), entity.ID)
	assert.Equal(t, "Alice", entity.Name)
	assert.Equal(t, "A", entity.Building)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBase_FindByID_ReturnsErrorWhenNotFound(t *testing.T) {
	mock, repo := newMockRepo(t)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM test_table WHERE id = ? LIMIT 1")).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	entity, err := repo.FindByID(999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Nil(t, entity)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---------- FindAll ----------

func TestBase_FindAll_NoOrderBy(t *testing.T) {
	mock, repo := newMockRepo(t)

	rows := sqlmock.NewRows([]string{"id", "name", "building"}).
		AddRow(int64(1), "Alice", "A").
		AddRow(int64(2), "Bob", "B")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM test_table")).
		WillReturnRows(rows)

	entities, err := repo.FindAll()
	require.NoError(t, err)
	assert.Len(t, entities, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBase_FindAll_WithOrderBy(t *testing.T) {
	mock, repo := newMockRepo(t)

	rows := sqlmock.NewRows([]string{"id", "name", "building"}).
		AddRow(int64(2), "Bob", "B").
		AddRow(int64(1), "Alice", "A")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM test_table ORDER BY name DESC")).
		WillReturnRows(rows)

	entities, err := repo.FindAll("name DESC")
	require.NoError(t, err)
	assert.Len(t, entities, 2)
	assert.Equal(t, "Bob", entities[0].Name)
	assert.Equal(t, "Alice", entities[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBase_FindAll_WithEmptyOrderByDefaultsToNoOrder(t *testing.T) {
	mock, repo := newMockRepo(t)

	rows := sqlmock.NewRows([]string{"id", "name", "building"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM test_table")).
		WillReturnRows(rows)

	entities, err := repo.FindAll("")
	require.NoError(t, err)
	assert.Len(t, entities, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}
