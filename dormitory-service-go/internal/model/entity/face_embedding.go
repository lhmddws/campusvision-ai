package entity

import (
	"database/sql"
	"time"
)

// FaceEmbedding maps to the face_embedding table.
// Stores 512-dim float32 face embeddings as BLOBs for face recognition.
type FaceEmbedding struct {
	ID        int64          `db:"id" json:"id"`
	Name      string         `db:"name" json:"name"`
	StudentID string         `db:"student_id" json:"student_id"`
	Embedding []byte         `db:"embedding" json:"-"`
	ImagePath sql.NullString `db:"image_path" json:"image_path"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
}
