package server

import (
	"strings"

	"github.com/gin-gonic/gin"
	"campusvision/test-env/internal/state"
)

// SetupRouter creates and configures the Gin router with all API routes
// and static file serving for the Vue frontend.
func SetupRouter(st *state.State, baseDir string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	h := NewHandler(st, baseDir)

	// ── API routes ──────────────────────────────────────────────────────────

	// Health
	r.GET("/api/health", h.Health)

	// SSE event stream
	r.GET("/api/events/stream", h.SSEHandler)

	// Cameras
	r.GET("/api/cameras", h.GetCameras)
	r.PUT("/api/cameras/:id", h.UpsertCamera)
	r.DELETE("/api/cameras/:id", h.DeleteCamera)
	r.GET("/api/cameras/:id/status", h.CameraStatus)
	r.GET("/api/cameras/:id/frame.jpg", h.FrameJPEG)
	r.GET("/api/camera-frames", h.CameraFrames)

	// Config
	r.GET("/api/config", h.GetConfig)
	r.PUT("/api/config", h.UpdateConfig)
	r.PUT("/api/config/reset", h.ResetConfig)

	// People
	r.GET("/api/people", h.GetPeople)
	r.POST("/api/people", h.AddPerson)
	r.DELETE("/api/people", h.RemovePerson)
	r.POST("/api/people/import-csv", h.ImportPeopleCSV)

	// Faces
	r.GET("/api/faces", h.ListFaces)
	r.POST("/api/faces", h.EnrollFace)
	r.DELETE("/api/faces/:name", h.DeleteFace)
	r.GET("/api/faces/:name/image", h.FaceImage)

	// Recognition
	r.GET("/api/recognition/status", h.RecognitionStatus)
	r.GET("/api/recognition/results", h.GetRecognitionResults)
	r.POST("/api/toggle-fake-data", h.ToggleFakeData)

	// Behavior
	r.GET("/api/behavior/status", h.BehaviorStatus)

	// Simulation
	r.POST("/api/cameras/:cameraId/simulate", h.Simulate)

	// Events
	r.GET("/api/events", h.GetEvents)
	r.DELETE("/api/events", h.ClearEvents)

	// Stats
	r.GET("/api/stats", h.GetStats)

	// Scenarios
	r.GET("/api/scenarios/random", h.ScenarioRandom)
	r.POST("/api/scenarios/preset", h.ScenarioPreset)

	// Webcam
	r.POST("/api/webcam/start", h.WebcamStart)
	r.POST("/api/webcam/stop", h.WebcamStop)
	r.POST("/api/webcam/start-all", h.WebcamStartAll)
	r.POST("/api/webcam/stop-all", h.WebcamStopAll)
	r.GET("/api/webcam/status", h.WebcamStatusGET)
	r.POST("/api/webcam/status", h.WebcamStatus)

	// ── Static file serving (Vue frontend) ───────────────────────────────────

	// Serve /assets/* from the static directory
	staticDir := baseDir + "/frontend/dist"
	r.Static("/assets", staticDir+"/assets")

	// SPA fallback: for any non-API route, serve index.html
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Return JSON 404 for unknown API routes
		if strings.HasPrefix(path, "/api/") {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}
		// SPA fallback: serve index.html for all other routes
		c.File(staticDir + "/index.html")
	})

	return r
}
