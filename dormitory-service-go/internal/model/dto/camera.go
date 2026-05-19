package dto

// CameraCreateDTO is the request body for creating a new camera.
type CameraCreateDTO struct {
	CameraID   string `json:"camera_id" binding:"required"`
	Building   string `json:"building" binding:"required"`
	Name       string `json:"name" binding:"required"`
	RtspURL    string `json:"rtsp_url" binding:"required"`
	Direction  string `json:"direction"`
	Resolution string `json:"resolution"`
	Remark     string `json:"remark"`
	Type       string `json:"type"`
	Protocol   string `json:"protocol"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	Username   string `json:"username"`
}

// CameraUpdateDTO is the request body for updating an existing camera.
type CameraUpdateDTO struct {
	Name       string `json:"name"`
	Building   string `json:"building"`
	RtspURL    string `json:"rtsp_url"`
	Direction  string `json:"direction"`
	Resolution string `json:"resolution"`
	Enabled    *bool  `json:"enabled"`
	Status     string `json:"status"`
	Remark     string `json:"remark"`
	Type       string `json:"type"`
	Protocol   string `json:"protocol"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Path       string `json:"path"`
	Username   string `json:"username"`
}
