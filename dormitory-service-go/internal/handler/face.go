package handler

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sims/campusvision/dormitory-service-go/internal/model/entity"
)

// FaceMatchRequest is the JSON body for POST /api/face/match.
type FaceMatchRequest struct {
	Embedding []float32 `json:"embedding"`
}

// FaceMatchResponse is the JSON response for the face match endpoint.
type FaceMatchResponse struct {
	Success bool       `json:"success"`
	Match   *FaceMatch `json:"match,omitempty"`
	Error   string     `json:"error,omitempty"`
}

// FaceMatch represents a matched face result.
type FaceMatch struct {
	Name       string  `json:"name"`
	StudentID  string  `json:"student_id"`
	Confidence float64 `json:"confidence"`
}

const matchThreshold = 0.65

// FaceMatch performs cosine similarity matching against stored face embeddings.
// POST /api/face/match
func (h *Handler) FaceMatch(c *gin.Context) {
	var req FaceMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Embedding) != 512 {
		c.JSON(http.StatusBadRequest, FaceMatchResponse{
			Success: false,
			Error:   "invalid embedding (must be 512 floats)",
		})
		return
	}

	rows, err := h.DB.QueryContext(c.Request.Context(), "SELECT id, name, student_id, embedding FROM face_embedding")
	if err != nil {
		c.JSON(http.StatusInternalServerError, FaceMatchResponse{
			Success: false,
			Error:   "db query failed",
		})
		return
	}
	defer rows.Close()

	var bestMatch *FaceMatch
	var bestScore float64

	for rows.Next() {
		var row entity.FaceEmbedding
		if err := rows.Scan(&row.ID, &row.Name, &row.StudentID, &row.Embedding); err != nil {
			continue
		}
		storedEmb := bytesToFloats(row.Embedding)
		if len(storedEmb) != 512 {
			continue
		}

		score := cosineSimilarity(req.Embedding, storedEmb)
		if score > bestScore && score >= matchThreshold {
			bestScore = score
			bestMatch = &FaceMatch{
				Name:       row.Name,
				StudentID:  row.StudentID,
				Confidence: math.Round(score*100) / 100,
			}
		}
	}

	c.JSON(http.StatusOK, FaceMatchResponse{
		Success: true,
		Match:   bestMatch,
	})
}

// cosineSimilarity computes the cosine similarity between two float32 vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// bytesToFloats converts a little-endian 32-bit float BLOB to []float32.
func bytesToFloats(data []byte) []float32 {
	if len(data)%4 != 0 {
		return nil
	}
	floats := make([]float32, len(data)/4)
	for i := range floats {
		bits := uint32(data[i*4]) | uint32(data[i*4+1])<<8 |
			uint32(data[i*4+2])<<16 | uint32(data[i*4+3])<<24
		floats[i] = math.Float32frombits(bits)
	}
	return floats
}
