package handler_test

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sims/campusvision/dormitory-service-go/internal/handler"
)

// floatsToBytes converts a float32 slice to little-endian byte slice.
// This is the inverse of handler.bytesToFloats.
func floatsToBytes(floats []float32) []byte {
	data := make([]byte, len(floats)*4)
	for i, f := range floats {
		bits := math.Float32bits(f)
		data[i*4] = byte(bits)
		data[i*4+1] = byte(bits >> 8)
		data[i*4+2] = byte(bits >> 16)
		data[i*4+3] = byte(bits >> 24)
	}
	return data
}

// setupFaceMatchTest creates a Handler with a sqlmock-backed *sqlx.DB for FaceMatch tests.
func setupFaceMatchTest(t *testing.T, faceMatchKey string, threshold float64) (sqlmock.Sqlmock, *handler.Handler, *httptest.ResponseRecorder) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "mysql")
	h := &handler.Handler{
		DB:                 sqlxDB,
		FaceMatchKey:       faceMatchKey,
		FaceMatchThreshold: threshold,
	}
	w := httptest.NewRecorder()
	t.Cleanup(func() { _ = db.Close() })
	return mock, h, w
}

// ginTestContext creates a gin.Context for testing with the given request.
func ginTestContext(w *httptest.ResponseRecorder, method, url, body string) *gin.Context {
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, url, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

// ---------- FaceMatch tests ----------

func TestFaceMatch_Success(t *testing.T) {
	mock, h, w := setupFaceMatchTest(t, "", 0.65) // no auth required

	// Create a 512-dim embedding with identical input and stored values
	emb := make([]float32, 512)
	for i := range emb {
		emb[i] = 0.1
	}
	embBytes := floatsToBytes(emb)

	// Mock DB: one row with the same embedding
	rows := sqlmock.NewRows([]string{"id", "name", "student_id", "embedding"}).
		AddRow(int64(1), "Alice", "S001", embBytes)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, student_id, embedding FROM face_embedding ORDER BY id LIMIT ? OFFSET ?")).
		WithArgs(100, 0).
		WillReturnRows(rows)

	reqBody, err := json.Marshal(map[string]any{"embedding": emb})
	require.NoError(t, err)

	c := ginTestContext(w, "POST", "/api/face/match", string(reqBody))
	h.FaceMatch(c)

	var resp handler.FaceMatchResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	require.NotNil(t, resp.Match)
	assert.Equal(t, "Alice", resp.Match.Name)
	assert.Equal(t, "S001", resp.Match.StudentID)
	assert.Equal(t, 1.0, resp.Match.Confidence) // identical vectors → cosine similarity = 1.0
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFaceMatch_NoMatchFound(t *testing.T) {
	mock, h, w := setupFaceMatchTest(t, "", 0.65)

	// Stored embedding: all 0.1
	storedEmb := make([]float32, 512)
	for i := range storedEmb {
		storedEmb[i] = 0.1
	}
	storedBytes := floatsToBytes(storedEmb)

	// Request embedding: all -0.1 → cosine similarity = -1.0, well below threshold
	reqEmb := make([]float32, 512)
	for i := range reqEmb {
		reqEmb[i] = -0.1
	}

	rows := sqlmock.NewRows([]string{"id", "name", "student_id", "embedding"}).
		AddRow(int64(1), "Alice", "S001", storedBytes)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, student_id, embedding FROM face_embedding ORDER BY id LIMIT ? OFFSET ?")).
		WithArgs(100, 0).
		WillReturnRows(rows)

	reqBody, err := json.Marshal(map[string]any{"embedding": reqEmb})
	require.NoError(t, err)

	c := ginTestContext(w, "POST", "/api/face/match", string(reqBody))
	h.FaceMatch(c)

	var resp handler.FaceMatchResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success) // response is still success=true
	assert.Nil(t, resp.Match)    // but match is nil since no one passed threshold
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFaceMatch_Unauthorized(t *testing.T) {
	_, h, w := setupFaceMatchTest(t, "test-key", 0.65) // auth required

	// Send request without X-API-Key header
	c := ginTestContext(w, "POST", "/api/face/match", `{"embedding": [0.1, 0.2]}`)
	h.FaceMatch(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp handler.FaceMatchResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "invalid or missing API key", resp.Error)
}

func TestFaceMatch_WithValidAPIKey(t *testing.T) {
	mock, h, w := setupFaceMatchTest(t, "test-key", 0.65) // auth required

	emb := make([]float32, 512)
	for i := range emb {
		emb[i] = 0.1
	}
	embBytes := floatsToBytes(emb)

	rows := sqlmock.NewRows([]string{"id", "name", "student_id", "embedding"}).
		AddRow(int64(1), "Bob", "S002", embBytes)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, student_id, embedding FROM face_embedding ORDER BY id LIMIT ? OFFSET ?")).
		WithArgs(100, 0).
		WillReturnRows(rows)

	reqBody, err := json.Marshal(map[string]any{"embedding": emb})
	require.NoError(t, err)

	c := ginTestContext(w, "POST", "/api/face/match", string(reqBody))
	c.Request.Header.Set("Authorization", "Bearer test-key")
	h.FaceMatch(c)

	var resp handler.FaceMatchResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp.Success)
	require.NotNil(t, resp.Match)
	assert.Equal(t, "Bob", resp.Match.Name)
	assert.Equal(t, "S002", resp.Match.StudentID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFaceMatch_InvalidEmbedding(t *testing.T) {
	_, h, w := setupFaceMatchTest(t, "", 0.65)

	// Only 10 floats instead of the required 512
	c := ginTestContext(w, "POST", "/api/face/match",
		`{"embedding": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0]}`)
	h.FaceMatch(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp handler.FaceMatchResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "invalid embedding (must be 512 floats)", resp.Error)
}
