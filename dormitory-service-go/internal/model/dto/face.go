package dto

// FaceCreateDTO is the request body for creating a new face record.
type FaceCreateDTO struct {
	StudentID  string `json:"student_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	RoomNumber string `json:"room_number"`
}

// FaceUpdateDTO is the request body for updating an existing face record.
type FaceUpdateDTO struct {
	Name       string `json:"name"`
	RoomNumber string `json:"room_number"`
}

// FaceBatchImportDTO is the request body for batch importing face records.
type FaceBatchImportDTO struct {
	Students []FaceCreateDTO `json:"students" binding:"required,min=1"`
}

// FaceEnrollDTO holds enrollment data decoded from multipart form.
type FaceEnrollDTO struct {
	StudentID  string
	Name       string
	RoomNumber string
	PhotoBytes []byte
}
