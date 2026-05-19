package server

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"campusvision/test-env/internal/kafka"
	"campusvision/test-env/internal/simulation"
	"campusvision/test-env/internal/state"
)

type Handler struct {
	state    *state.State
	baseDir  string
}

func NewHandler(st *state.State, baseDir string) *Handler {
	return &Handler{state: st, baseDir: baseDir}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func jsonError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// ── GET /api/health ──────────────────────────────────────────────────────────

func (h *Handler) Health(c *gin.Context) {
	h.state.RLock()
	cameras := make(map[string]state.CameraDef, len(h.state.Cameras))
	for k, v := range h.state.Cameras {
		cameras[k] = v
	}
	cfg := h.state.Config
	startTime := h.state.StartTime
	kafkaOk := h.state.KafkaProducer != nil
	webcams := make(map[string]bool, len(h.state.Cameras))
	for cid := range h.state.Cameras {
		webcams[cid] = h.state.WebcamRunning[cid]
	}
	h.state.RUnlock()

	uptime := int(time.Since(startTime).Seconds())

	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"kafka":     kafkaOk,
		"cameras":   cameras,
		"uptime_sec": uptime,
		"config":    cfg,
		"webcams":   webcams,
	})
}

// ── GET /api/cameras ─────────────────────────────────────────────────────────

func (h *Handler) GetCameras(c *gin.Context) {
	h.state.RLock()
	cameras := make(map[string]state.CameraDef, len(h.state.Cameras))
	for k, v := range h.state.Cameras {
		cameras[k] = v
	}
	h.state.RUnlock()
	c.JSON(http.StatusOK, cameras)
}

// ── PUT /api/cameras/:id ─────────────────────────────────────────────────────

type upsertCameraBody struct {
	Building string `json:"building" binding:"required"`
	Label    string `json:"label" binding:"required"`
	Color    string `json:"color"`
}

func (h *Handler) UpsertCamera(c *gin.Context) {
	cameraID := c.Param("id")
	var body upsertCameraBody
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonError(c, http.StatusBadRequest, err.Error())
		return
	}

	color := body.Color
	if color == "" {
		color = "#555577"
	}

	h.state.Lock()
	h.state.Cameras[cameraID] = state.CameraDef{
		Building: body.Building,
		Label:    body.Label,
		Color:    color,
	}
	if _, exists := h.state.WebcamIndex[cameraID]; !exists {
		h.state.WebcamIndex[cameraID] = nil
	}
	h.state.Unlock()

	log.Printf("Camera upserted: %s", cameraID)
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"camera_id": cameraID,
		"config":    h.state.Cameras[cameraID],
	})
}

// ── DELETE /api/cameras/:id ──────────────────────────────────────────────────

func (h *Handler) DeleteCamera(c *gin.Context) {
	cameraID := c.Param("id")

	h.state.Lock()
	if _, ok := h.state.Cameras[cameraID]; !ok {
		h.state.Unlock()
		jsonError(c, http.StatusNotFound, fmt.Sprintf("Unknown camera: %s", cameraID))
		return
	}
	// Stop webcam if running
	if h.state.WebcamRunning[cameraID] {
		h.state.WebcamRunning[cameraID] = false
	}
	delete(h.state.Cameras, cameraID)
	delete(h.state.WebcamIndex, cameraID)
	delete(h.state.LatestFrames, cameraID)
	h.state.Unlock()

	log.Printf("Camera deleted: %s", cameraID)
	c.JSON(http.StatusOK, gin.H{"success": true, "camera_id": cameraID})
}

// ── GET /api/cameras/:id/status ──────────────────────────────────────────────

func (h *Handler) CameraStatus(c *gin.Context) {
	cameraID := c.Param("id")

	h.state.RLock()
	cam, ok := h.state.Cameras[cameraID]
	if !ok {
		h.state.RUnlock()
		jsonError(c, http.StatusNotFound, fmt.Sprintf("Unknown camera: %s", cameraID))
		return
	}
	frameData := h.state.LatestFrames[cameraID]
	frameSize := len(frameData)
	isWebcam := h.state.WebcamRunning[cameraID]
	webcamDev := h.state.WebcamIndex[cameraID]
	h.state.RUnlock()

	var devPtr *int
	if webcamDev != nil {
		val := *webcamDev
		devPtr = &val
	}

	c.JSON(http.StatusOK, gin.H{
		"camera_id":    cameraID,
		"building":     cam.Building,
		"label":        cam.Label,
		"has_frame":    frameData != nil,
		"frame_size":   frameSize,
		"is_webcam":    isWebcam,
		"webcam_device": devPtr,
	})
}

// ── GET /api/cameras/:id/frame.jpg ───────────────────────────────────────────

func (h *Handler) FrameJPEG(c *gin.Context) {
	cameraID := c.Param("id")

	h.state.RLock()
	_, ok := h.state.Cameras[cameraID]
	if !ok {
		h.state.RUnlock()
		jsonError(c, http.StatusNotFound, "camera not found")
		return
	}
	isWebcam := h.state.WebcamRunning[cameraID]
	h.state.RUnlock()

	frameData, hasFrame := h.state.GetLatestFrame(cameraID)

	if !hasFrame {
		if isWebcam {
			// Wait briefly for a webcam frame
			for i := 0; i < 10; i++ {
				time.Sleep(50 * time.Millisecond)
				frameData, hasFrame = h.state.GetLatestFrame(cameraID)
				if hasFrame {
					break
				}
			}
		}

		if !hasFrame {
			// Generate a simulated frame
			jpeg, err := simulation.GenerateFrame(h.state, cameraID, "", "idle")
			if err != nil {
				jsonError(c, http.StatusInternalServerError, err.Error())
				return
			}
			frameData = jpeg
			h.state.SetLatestFrame(cameraID, jpeg)
		}
	}

	if len(frameData) == 0 {
		jpeg, err := simulation.GenerateFrame(h.state, cameraID, "", "idle")
		if err != nil {
			jsonError(c, http.StatusInternalServerError, err.Error())
			return
		}
		frameData = jpeg
	}

	c.Data(http.StatusOK, "image/jpeg", frameData)
}

// ── GET /api/camera-frames ───────────────────────────────────────────────────

func (h *Handler) CameraFrames(c *gin.Context) {
	h.state.RLock()
	frames := make(map[string]string, len(h.state.LatestFrames))
	for cid, data := range h.state.LatestFrames {
		frames[cid] = base64.StdEncoding.EncodeToString(data)
	}
	h.state.RUnlock()
	c.JSON(http.StatusOK, frames)
}

// ── GET /api/config ──────────────────────────────────────────────────────────

func (h *Handler) GetConfig(c *gin.Context) {
	h.state.RLock()
	cfg := h.state.Config
	cameras := make(map[string]state.CameraDef, len(h.state.Cameras))
	for k, v := range h.state.Cameras {
		cameras[k] = v
	}
	webcamStatus := make(map[string]bool, len(h.state.Cameras))
	for cid := range h.state.Cameras {
		webcamStatus[cid] = h.state.WebcamRunning[cid]
	}
	webcamIndices := make(map[string]*int, len(h.state.WebcamIndex))
	for k, v := range h.state.WebcamIndex {
		if v != nil {
			val := *v
			webcamIndices[k] = &val
		} else {
			webcamIndices[k] = nil
		}
	}
	h.state.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"config":              cfg,
		"cameras":             cameras,
		"webcam_status":       webcamStatus,
		"camera_webcam_indices": webcamIndices,
	})
}

// ── PUT /api/config ──────────────────────────────────────────────────────────

type configUpdateBody struct {
	JPEGQuality          *int      `json:"jpeg_quality"`
	FrameWidth           *int      `json:"frame_width"`
	FrameHeight          *int      `json:"frame_height"`
	FPS                  *int      `json:"fps"`
	ConfidenceThreshold  *float64  `json:"confidence_threshold"`
	MinFaceSize          *int      `json:"min_face_size"`
	MatchThreshold       *float64  `json:"match_threshold"`
	CacheTTL             *int      `json:"cache_ttl"`
	RoiLineX             *float64  `json:"roi_line_x"`
	MinTrackPoints       *int      `json:"min_track_points"`
	DedupWindowSeconds   *int      `json:"dedup_window_seconds"`
	StrangerAlertEnabled *bool     `json:"stranger_alert_enabled"`
	StrangerAlertThresh  *float64  `json:"stranger_alert_threshold"`
	NightModeEnabled     *bool     `json:"night_mode_enabled"`
	NightModeStartHour   *int      `json:"night_mode_start_hour"`
	NightModeEndHour     *int      `json:"night_mode_end_hour"`
	MotionThreshold      *float64  `json:"motion_threshold"`
	DynamicExtraction    *bool     `json:"dynamic_extraction"`
	CameraSource         *string   `json:"camera_source"`
	WebcamDevice         *int      `json:"webcam_device"`
	TestPeople           *[]string `json:"test_people"`
}

func (h *Handler) UpdateConfig(c *gin.Context) {
	var body configUpdateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.state.Lock()
	updates := make(map[string]interface{})

	if body.JPEGQuality != nil {
		h.state.Config.JPEGQuality = *body.JPEGQuality
		updates["jpeg_quality"] = *body.JPEGQuality
	}
	if body.FrameWidth != nil {
		h.state.Config.FrameWidth = *body.FrameWidth
		updates["frame_width"] = *body.FrameWidth
	}
	if body.FrameHeight != nil {
		h.state.Config.FrameHeight = *body.FrameHeight
		updates["frame_height"] = *body.FrameHeight
	}
	if body.FPS != nil {
		h.state.Config.FPS = *body.FPS
		updates["fps"] = *body.FPS
	}
	if body.ConfidenceThreshold != nil {
		h.state.Config.ConfidenceThreshold = *body.ConfidenceThreshold
		updates["confidence_threshold"] = *body.ConfidenceThreshold
	}
	if body.MinFaceSize != nil {
		h.state.Config.MinFaceSize = *body.MinFaceSize
		updates["min_face_size"] = *body.MinFaceSize
	}
	if body.MatchThreshold != nil {
		h.state.Config.MatchThreshold = *body.MatchThreshold
		updates["match_threshold"] = *body.MatchThreshold
	}
	if body.CacheTTL != nil {
		h.state.Config.CacheTTL = *body.CacheTTL
		updates["cache_ttl"] = *body.CacheTTL
	}
	if body.RoiLineX != nil {
		h.state.Config.RoiLineX = *body.RoiLineX
		updates["roi_line_x"] = *body.RoiLineX
	}
	if body.MinTrackPoints != nil {
		h.state.Config.MinTrackPoints = *body.MinTrackPoints
		updates["min_track_points"] = *body.MinTrackPoints
	}
	if body.DedupWindowSeconds != nil {
		h.state.Config.DedupWindowSeconds = *body.DedupWindowSeconds
		updates["dedup_window_seconds"] = *body.DedupWindowSeconds
	}
	if body.StrangerAlertEnabled != nil {
		h.state.Config.StrangerAlertEnabled = *body.StrangerAlertEnabled
		updates["stranger_alert_enabled"] = *body.StrangerAlertEnabled
	}
	if body.StrangerAlertThresh != nil {
		h.state.Config.StrangerAlertThresh = *body.StrangerAlertThresh
		updates["stranger_alert_threshold"] = *body.StrangerAlertThresh
	}
	if body.NightModeEnabled != nil {
		h.state.Config.NightModeEnabled = *body.NightModeEnabled
		updates["night_mode_enabled"] = *body.NightModeEnabled
	}
	if body.NightModeStartHour != nil {
		h.state.Config.NightModeStartHour = *body.NightModeStartHour
		updates["night_mode_start_hour"] = *body.NightModeStartHour
	}
	if body.NightModeEndHour != nil {
		h.state.Config.NightModeEndHour = *body.NightModeEndHour
		updates["night_mode_end_hour"] = *body.NightModeEndHour
	}
	if body.MotionThreshold != nil {
		h.state.Config.MotionThreshold = *body.MotionThreshold
		updates["motion_threshold"] = *body.MotionThreshold
	}
	if body.DynamicExtraction != nil {
		h.state.Config.DynamicExtraction = *body.DynamicExtraction
		updates["dynamic_extraction"] = *body.DynamicExtraction
	}
	if body.CameraSource != nil {
		h.state.Config.CameraSource = *body.CameraSource
		updates["camera_source"] = *body.CameraSource
	}
	if body.WebcamDevice != nil {
		h.state.Config.WebcamDevice = *body.WebcamDevice
		updates["webcam_device"] = *body.WebcamDevice
	}
	if body.TestPeople != nil {
		h.state.Config.TestPeople = *body.TestPeople
		updates["test_people"] = *body.TestPeople
	}

	cfg := h.state.Config
	h.state.Unlock()

	log.Printf("Config updated: %v", updates)
	c.JSON(http.StatusOK, gin.H{"success": true, "updated": updates, "config": cfg})
}

// ── PUT /api/config/reset ────────────────────────────────────────────────────

func (h *Handler) ResetConfig(c *gin.Context) {
	defaultPeople := state.DefaultConfig.TestPeople
	peopleCopy := make([]string, len(defaultPeople))
	copy(peopleCopy, defaultPeople)

	h.state.Lock()
	// Keep test_people from current config
	currentPeople := h.state.Config.TestPeople
	h.state.Config = state.DefaultConfig
	h.state.Config.TestPeople = currentPeople
	cfg := h.state.Config
	h.state.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "config": cfg})
}

// ── GET /api/people ──────────────────────────────────────────────────────────

func (h *Handler) GetPeople(c *gin.Context) {
	h.state.RLock()
	people := make([]string, len(h.state.Config.TestPeople))
	copy(people, h.state.Config.TestPeople)
	h.state.RUnlock()
	c.JSON(http.StatusOK, gin.H{"people": people})
}

// ── POST /api/people ─────────────────────────────────────────────────────────

type addPersonBody struct {
	Name string `json:"name"`
}

func (h *Handler) AddPerson(c *gin.Context) {
	var body addPersonBody
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonError(c, http.StatusBadRequest, err.Error())
		return
	}

	name := strings.TrimSpace(body.Name)
	h.state.Lock()
	if name != "" {
		found := false
		for _, p := range h.state.Config.TestPeople {
			if p == name {
				found = true
				break
			}
		}
		if !found {
			h.state.Config.TestPeople = append(h.state.Config.TestPeople, name)
		}
	}
	people := make([]string, len(h.state.Config.TestPeople))
	copy(people, h.state.Config.TestPeople)
	h.state.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "people": people})
}

// ── DELETE /api/people ───────────────────────────────────────────────────────

func (h *Handler) RemovePerson(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		jsonError(c, http.StatusBadRequest, "name query parameter required")
		return
	}

	h.state.Lock()
	newList := make([]string, 0, len(h.state.Config.TestPeople))
	for _, p := range h.state.Config.TestPeople {
		if p != name {
			newList = append(newList, p)
		}
	}
	h.state.Config.TestPeople = newList
	people := make([]string, len(newList))
	copy(people, newList)
	h.state.Unlock()

	c.JSON(http.StatusOK, gin.H{"success": true, "people": people})
}

// ── POST /api/people/import-csv ──────────────────────────────────────────────

func (h *Handler) ImportPeopleCSV(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		jsonError(c, http.StatusBadRequest, "file required")
		return
	}

	if !strings.HasSuffix(file.Filename, ".csv") {
		jsonError(c, http.StatusBadRequest, "请上传 .csv 文件")
		return
	}

	f, err := file.Open()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer f.Close()

	// Read all content
	content, err := io.ReadAll(f)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Try UTF-8 with BOM first, then GBK
	text := string(content)
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		text = string(content[3:]) // strip BOM
	}

	reader := csv.NewReader(strings.NewReader(text))
	headers, err := reader.Read()
	if err != nil {
		jsonError(c, http.StatusBadRequest, "CSV 文件为空或格式错误")
		return
	}

	// Normalize column names
	nameIdx := -1
	studentIDIdx := -1
	for i, hdr := range headers {
		hdrLower := strings.ToLower(strings.TrimSpace(hdr))
		switch hdrLower {
		case "name", "姓名", "名字", "学生姓名":
			nameIdx = i
		case "student_id", "学号", "id", "编号", "studentid":
			studentIDIdx = i
		}
	}
	if nameIdx == -1 {
		jsonError(c, http.StatusBadRequest, "CSV缺少姓名列 (name/姓名)")
		return
	}

	imported := 0
	h.state.Lock()
	for {
		row, err := reader.Read()
		if err != nil {
			break
		}
		if nameIdx >= len(row) {
			continue
		}
		raw := strings.TrimSpace(row[nameIdx])
		if raw == "" {
			continue
		}
		studentID := ""
		if studentIDIdx >= 0 && studentIDIdx < len(row) {
			studentID = strings.TrimSpace(row[studentIDIdx])
		}
		display := raw
		if studentID != "" {
			display = fmt.Sprintf("%s (%s)", raw, studentID)
		}

		found := false
		for _, p := range h.state.Config.TestPeople {
			if p == display {
				found = true
				break
			}
		}
		if !found {
			h.state.Config.TestPeople = append(h.state.Config.TestPeople, display)
			imported++
		}
	}
	people := make([]string, len(h.state.Config.TestPeople))
	copy(people, h.state.Config.TestPeople)
	h.state.Unlock()

	h.state.LogEvent("system", "csv_import", fmt.Sprintf("CSV导入 %d 人 (%s)", imported, file.Filename))

	c.JSON(http.StatusOK, gin.H{"success": true, "imported": imported, "people": people})
}

// ── GET /api/faces ───────────────────────────────────────────────────────────

func (h *Handler) ListFaces(c *gin.Context) {
	metaFile := filepath.Join(h.baseDir, "face_data", "metadata.json")
	faces := make([]state.FaceEntry, 0)

	data, err := os.ReadFile(metaFile)
	if err == nil {
		json.Unmarshal(data, &faces)
	}

	// Ensure image_url for each
	for i, f := range faces {
		imgPath := filepath.Join(h.baseDir, "face_data", "images", f.Name+".jpg")
		if _, err := os.Stat(imgPath); err == nil {
			faces[i].ImageURL = fmt.Sprintf("/api/faces/%s/image", f.Name)
		}
	}

	c.JSON(http.StatusOK, gin.H{"faces": faces, "total": len(faces)})
}

// ── POST /api/faces ──────────────────────────────────────────────────────────

func (h *Handler) EnrollFace(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	studentID := strings.TrimSpace(c.PostForm("student_id"))

	if name == "" {
		jsonError(c, http.StatusBadRequest, "姓名不能为空")
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		jsonError(c, http.StatusBadRequest, "image file required")
		return
	}

	// Save image
	imgDir := filepath.Join(h.baseDir, "face_data", "images")
	os.MkdirAll(imgDir, 0755)
	imgPath := filepath.Join(imgDir, name+".jpg")

	src, err := file.Open()
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer src.Close()

	dst, err := os.Create(imgPath)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	defer dst.Close()
	io.Copy(dst, src)

	// Update metadata
	metaFile := filepath.Join(h.baseDir, "face_data", "metadata.json")
	faces := make([]state.FaceEntry, 0)
	if data, err := os.ReadFile(metaFile); err == nil {
		json.Unmarshal(data, &faces)
	}

	// Remove existing entry for same name
	filtered := make([]state.FaceEntry, 0, len(faces))
	for _, f := range faces {
		if f.Name != name {
			filtered = append(filtered, f)
		}
	}

	entry := state.FaceEntry{
		Name:       name,
		StudentID:  studentID,
		EnrolledAt: time.Now().Format(time.RFC3339),
		ImageURL:   fmt.Sprintf("/api/faces/%s/image", name),
	}
	filtered = append(filtered, entry)

	metaData, _ := json.MarshalIndent(filtered, "", "  ")
	os.WriteFile(metaFile, metaData, 0644)

	h.state.LogEvent("system", "face_enroll", fmt.Sprintf("人脸录入: %s (%s)", name, studentID))

	c.JSON(http.StatusOK, gin.H{"success": true, "face": entry})
}

// ── DELETE /api/faces/:name ──────────────────────────────────────────────────

func (h *Handler) DeleteFace(c *gin.Context) {
	name := c.Param("name")

	// Remove image
	imgPath := filepath.Join(h.baseDir, "face_data", "images", name+".jpg")
	os.Remove(imgPath)

	// Update metadata
	metaFile := filepath.Join(h.baseDir, "face_data", "metadata.json")
	faces := make([]state.FaceEntry, 0)
	if data, err := os.ReadFile(metaFile); err == nil {
		json.Unmarshal(data, &faces)
	}
	filtered := make([]state.FaceEntry, 0, len(faces))
	for _, f := range faces {
		if f.Name != name {
			filtered = append(filtered, f)
		}
	}
	metaData, _ := json.MarshalIndent(filtered, "", "  ")
	os.WriteFile(metaFile, metaData, 0644)

	h.state.LogEvent("system", "face_delete", fmt.Sprintf("删除人脸: %s", name))

	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": name})
}

// ── GET /api/faces/:name/image ───────────────────────────────────────────────

func (h *Handler) FaceImage(c *gin.Context) {
	name := c.Param("name")
	imgPath := filepath.Join(h.baseDir, "face_data", "images", name+".jpg")

	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		jsonError(c, http.StatusNotFound, "Face image not found")
		return
	}

	c.File(imgPath)
}

// ── GET /api/recognition/status ──────────────────────────────────────────────

func (h *Handler) RecognitionStatus(c *gin.Context) {
	metaFile := filepath.Join(h.baseDir, "face_data", "metadata.json")
	enrolled := make([]state.FaceEntry, 0)
	if data, err := os.ReadFile(metaFile); err == nil {
		json.Unmarshal(data, &enrolled)
	}

	h.state.RLock()
	cfg := h.state.Config
	h.state.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"enrolled_count":      len(enrolled),
		"enrolled_faces":      enrolled,
		"confidence_threshold": cfg.ConfidenceThreshold,
		"match_threshold":     cfg.MatchThreshold,
		"cache_ttl":           cfg.CacheTTL,
	})
}

// ── GET /api/behavior/status ─────────────────────────────────────────────────

func (h *Handler) BehaviorStatus(c *gin.Context) {
	h.state.RLock()
	cfg := h.state.Config
	h.state.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"enabled":               true,
		"roi_line_x":            cfg.RoiLineX,
		"min_track_points":      cfg.MinTrackPoints,
		"motion_threshold":      cfg.MotionThreshold,
		"dynamic_extraction":    cfg.DynamicExtraction,
		"night_mode_enabled":    cfg.NightModeEnabled,
		"night_mode_start_hour": cfg.NightModeStartHour,
		"night_mode_end_hour":   cfg.NightModeEndHour,
		"stranger_alert_enabled": cfg.StrangerAlertEnabled,
		"stranger_alert_threshold": cfg.StrangerAlertThresh,
		"dedup_window_seconds":  cfg.DedupWindowSeconds,
	})
}

// ── POST /api/cameras/:cameraId/simulate ─────────────────────────────────────

type simulateBody struct {
	Action string `json:"action"`
	Person string `json:"person"`
}

func (h *Handler) Simulate(c *gin.Context) {
	cameraID := c.Param("cameraId")

	var body simulateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		body = simulateBody{Action: "entry"}
	}

	h.state.RLock()
	cam, ok := h.state.Cameras[cameraID]
	if !ok {
		h.state.RUnlock()
		jsonError(c, http.StatusNotFound, fmt.Sprintf("Unknown camera: %s", cameraID))
		return
	}
	building := cam.Building
	label := cam.Label
	isWebcam := h.state.WebcamRunning[cameraID]
	cfg := h.state.Config
	h.state.RUnlock()

	action := body.Action
	switch action {
	case "entry", "exit", "idle":
	default:
		action = "idle"
	}

	// Pick person
	person := body.Person
	if person == "" && action != "idle" {
		person = h.state.PickRandomPerson()
	}

	// Generate frame (if webcam not active)
	frameData := []byte{}
	if !isWebcam {
		jpeg, err := simulation.GenerateFrame(h.state, cameraID, person, action)
		if err == nil {
			frameData = jpeg
			h.state.SetLatestFrame(cameraID, jpeg)
		}
	} else {
		frameData, _ = h.state.GetLatestFrame(cameraID)
	}

	// Push to Kafka t_dorm_frame
	seq := time.Now().UnixMilli()
	quality := cfg.JPEGQuality

	frameMsg := map[string]interface{}{
		"camera_id":      cameraID,
		"building":       building,
		"timestamp":      seq,
		"frame_sequence": seq,
		"frame_data":     base64.StdEncoding.EncodeToString(frameData),
		"frame_width":    cfg.FrameWidth,
		"frame_height":   cfg.FrameHeight,
		"jpeg_quality":   quality,
		"is_dynamic":     action != "idle",
	}

	kafkaFrameOK := false
	if err := kafka.SendMessage(h.state.KafkaProducer, building, frameMsg); err == nil {
		kafkaFrameOK = true
	}

	// Push event to t_dorm_event
	kafkaEventOK := false
	if action != "idle" && h.state.KafkaEventPub != nil {
		studentID := ""
		name := person
		if person != "" {
			if idx := strings.Index(person, "("); idx > 0 {
				name = strings.TrimSpace(person[:idx])
				studentID = strings.TrimSpace(person[idx+1 : len(person)-1])
			}
		}

		eventMsg := map[string]interface{}{
			"camera_id":       cameraID,
			"building":        building,
			"event_type":      action,
			"student_id":      studentID,
			"name":            name,
			"confidence":      cfg.ConfidenceThreshold,
			"timestamp":       seq,
			"frame_sequence":  seq,
			"is_stranger":     false,
			"snapshot_path":   "",
			"direction_method": "roi_line",
		}

		if err := kafka.SendMessage(h.state.KafkaEventPub, building, eventMsg); err == nil {
			kafkaEventOK = true
		}
	}

	detail := fmt.Sprintf("%s %s", label, action)
	if person != "" {
		detail += fmt.Sprintf(" [%s]", person)
	}
	entry := h.state.LogEvent(cameraID, action, detail)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"camera_id":   cameraID,
		"action":      action,
		"kafka":       kafkaFrameOK,
		"kafka_event": kafkaEventOK,
		"event":       entry,
		"frame_bytes": len(frameData),
	})
}

// ── GET /api/events ──────────────────────────────────────────────────────────

func (h *Handler) GetEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 50
	}

	events := h.state.GetEvents(limit)
	c.JSON(http.StatusOK, events)
}

// ── DELETE /api/events ───────────────────────────────────────────────────────

func (h *Handler) ClearEvents(c *gin.Context) {
	h.state.ClearEvents()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ── GET /api/stats ───────────────────────────────────────────────────────────

func (h *Handler) GetStats(c *gin.Context) {
	events := h.state.GetEvents(9999)
	h.state.RLock()
	framesCount := len(h.state.LatestFrames)
	camerasCount := len(h.state.Cameras)
	kafkaConnected := h.state.KafkaProducer != nil
	uptime := int(time.Since(h.state.StartTime).Seconds())
	h.state.RUnlock()

	totalEvents := len(events)
	eventTypeCounts := map[string]int{"entry": 0, "exit": 0, "idle": 0}
	buildingStats := make(map[string]int)
	cameraEventCounts := make(map[string]int)

	for _, ev := range events {
		etype := ev.EventType
		if _, ok := eventTypeCounts[etype]; ok {
			eventTypeCounts[etype]++
		}
		cameraEventCounts[ev.CameraID]++
		if ev.Building != "" {
			buildingStats[ev.Building]++
		}
	}

	// events per minute is 0 since event log doesn't store unix timestamps (matching Python behavior)
	eventsPerMin := 0

	c.JSON(http.StatusOK, gin.H{
		"frames_generated":    framesCount,
		"events_total":        totalEvents,
		"event_type_counts":   eventTypeCounts,
		"building_stats":      buildingStats,
		"camera_event_counts": cameraEventCounts,
		"events_per_min":      eventsPerMin,
		"peak_events_per_min": eventsPerMin,
		"active_cameras":      camerasCount,
		"kafka_connected":     kafkaConnected,
		"kafka_frames_sent":   framesCount,
		"kafka_events_sent":   totalEvents,
		"uptime_sec":          uptime,
	})
}

// ── GET /api/scenarios/random ────────────────────────────────────────────────

func (h *Handler) ScenarioRandom(c *gin.Context) {
	countStr := c.DefaultQuery("count", "5")
	count, err := strconv.Atoi(countStr)
	if err != nil || count < 1 {
		count = 5
	}

	results := make([]gin.H, 0, count)
	for i := 0; i < count; i++ {
		cids := make([]string, 0)
		h.state.RLock()
		for cid := range h.state.Cameras {
			cids = append(cids, cid)
		}
		h.state.RUnlock()

		if len(cids) == 0 {
			break
		}
		cid := cids[rand.Intn(len(cids))]
		actions := []string{"entry", "exit"}
		act := actions[rand.Intn(2)]
		person := h.state.PickRandomPerson()

		h.state.RLock()
		cam, _ := h.state.Cameras[cid]
		cfg := h.state.Config
		h.state.RUnlock()

		// Generate frame
		jpeg, err := simulation.GenerateFrame(h.state, cid, person, act)
		var frameBytes int
		if err == nil {
			frameBytes = len(jpeg)
			h.state.SetLatestFrame(cid, jpeg)
		}

		// Kafka
		seq := time.Now().UnixMilli()
		frameMsg := map[string]interface{}{
			"camera_id":      cid,
			"building":       cam.Building,
			"timestamp":      seq,
			"frame_sequence": seq,
			"frame_data":     base64.StdEncoding.EncodeToString(jpeg),
			"frame_width":    cfg.FrameWidth,
			"frame_height":   cfg.FrameHeight,
			"jpeg_quality":   cfg.JPEGQuality,
			"is_dynamic":     true,
		}
		kafkaOK := kafka.SendMessage(h.state.KafkaProducer, cam.Building, frameMsg) == nil

		detail := fmt.Sprintf("%s %s", cam.Label, act)
		if person != "" {
			detail += fmt.Sprintf(" [%s]", person)
		}
		entry := h.state.LogEvent(cid, act, detail)

		results = append(results, gin.H{
			"success":     true,
			"camera_id":   cid,
			"action":      act,
			"kafka":       kafkaOK,
			"event":       entry,
			"frame_bytes": frameBytes,
		})

		time.Sleep(50 * time.Millisecond)
	}

	c.JSON(http.StatusOK, gin.H{"generated": count, "results": results})
}

// ── POST /api/scenarios/preset ───────────────────────────────────────────────

type presetBody struct {
	Preset string `json:"preset"`
}

func (h *Handler) ScenarioPreset(c *gin.Context) {
	var body presetBody
	if err := c.ShouldBindJSON(&body); err != nil {
		body.Preset = "rush_hour"
	}

	results := make([]gin.H, 0)

	switch body.Preset {
	case "rush_hour":
		for i := 0; i < 20; i++ {
			cids := make([]string, 0)
			h.state.RLock()
			for cid := range h.state.Cameras {
				cids = append(cids, cid)
			}
			h.state.RUnlock()

			if len(cids) == 0 {
				break
			}
			cid := cids[rand.Intn(len(cids))]
			actions := []string{"entry", "exit"}
			act := actions[rand.Intn(2)]
			person := h.state.PickRandomPerson()

			res := h.doSimulate(cid, act, person)
			results = append(results, res)
			time.Sleep(50 * time.Millisecond)
		}

	case "night_mode":
		h.state.Lock()
		h.state.Config.NightModeEnabled = true
		h.state.Unlock()

		for i := 0; i < 8; i++ {
			cids := make([]string, 0)
			h.state.RLock()
			for cid := range h.state.Cameras {
				cids = append(cids, cid)
			}
			h.state.RUnlock()

			if len(cids) == 0 {
				break
			}
			cid := cids[rand.Intn(len(cids))]
			res := h.doSimulate(cid, "idle", "")
			results = append(results, res)
			time.Sleep(50 * time.Millisecond)
		}

	case "stranger":
		h.state.Lock()
		oldThreshold := h.state.Config.ConfidenceThreshold
		h.state.Config.ConfidenceThreshold = 0.9
		h.state.Unlock()

		for i := 0; i < 6; i++ {
			cids := make([]string, 0)
			h.state.RLock()
			for cid := range h.state.Cameras {
				cids = append(cids, cid)
			}
			h.state.RUnlock()

			if len(cids) == 0 {
				break
			}
			cid := cids[rand.Intn(len(cids))]
			actions := []string{"entry", "exit"}
			act := actions[rand.Intn(2)]
			person := h.state.PickRandomPerson()

			res := h.doSimulate(cid, act, person)
			results = append(results, res)
			time.Sleep(50 * time.Millisecond)
		}

		h.state.Lock()
		h.state.Config.ConfidenceThreshold = oldThreshold
		h.state.Unlock()

	case "all_entry":
		h.state.RLock()
		cids := make([]string, 0, len(h.state.Cameras))
		for cid := range h.state.Cameras {
			cids = append(cids, cid)
		}
		h.state.RUnlock()

		for _, cid := range cids {
			person := h.state.PickRandomPerson()
			res := h.doSimulate(cid, "entry", person)
			results = append(results, res)
			time.Sleep(100 * time.Millisecond)
		}

	case "all_exit":
		h.state.RLock()
		cids := make([]string, 0, len(h.state.Cameras))
		for cid := range h.state.Cameras {
			cids = append(cids, cid)
		}
		h.state.RUnlock()

		for _, cid := range cids {
			person := h.state.PickRandomPerson()
			res := h.doSimulate(cid, "exit", person)
			results = append(results, res)
			time.Sleep(100 * time.Millisecond)
		}

	case "clear_log":
		h.state.ClearEvents()
		c.JSON(http.StatusOK, gin.H{"success": true, "preset": "clear_log", "events_cleared": true})
		return

	default:
		jsonError(c, http.StatusBadRequest, fmt.Sprintf("Unknown preset: %s", body.Preset))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"preset":    body.Preset,
		"generated": len(results),
		"results":   results,
	})
}

// doSimulate is an internal helper used by ScenarioPreset.
func (h *Handler) doSimulate(cameraID, action, person string) gin.H {
	h.state.RLock()
	cam, ok := h.state.Cameras[cameraID]
	if !ok {
		h.state.RUnlock()
		return gin.H{"success": false, "error": "unknown camera"}
	}
	building := cam.Building
	label := cam.Label
	cfg := h.state.Config
	h.state.RUnlock()

	if person == "" && action != "idle" {
		person = h.state.PickRandomPerson()
	}

	jpeg, err := simulation.GenerateFrame(h.state, cameraID, person, action)
	var frameBytes int
	if err == nil {
		frameBytes = len(jpeg)
		h.state.SetLatestFrame(cameraID, jpeg)
	}

	seq := time.Now().UnixMilli()
	frameMsg := map[string]interface{}{
		"camera_id":      cameraID,
		"building":       building,
		"timestamp":      seq,
		"frame_sequence": seq,
		"frame_data":     base64.StdEncoding.EncodeToString(jpeg),
		"frame_width":    cfg.FrameWidth,
		"frame_height":   cfg.FrameHeight,
		"jpeg_quality":   cfg.JPEGQuality,
		"is_dynamic":     action != "idle",
	}
	kafkaOK := kafka.SendMessage(h.state.KafkaProducer, building, frameMsg) == nil

	detail := fmt.Sprintf("%s %s", label, action)
	if person != "" {
		detail += fmt.Sprintf(" [%s]", person)
	}
	entry := h.state.LogEvent(cameraID, action, detail)

	return gin.H{
		"success":     true,
		"camera_id":   cameraID,
		"action":      action,
		"kafka":       kafkaOK,
		"event":       entry,
		"frame_bytes": frameBytes,
	}
}

// ── Webcam endpoints ─────────────────────────────────────────────────────────

type webcamStartBody struct {
	CameraID    string `json:"camera_id" binding:"required"`
	DeviceIndex *int   `json:"device_index"`
}

type webcamStopBody struct {
	CameraID string `json:"camera_id" binding:"required"`
}

func (h *Handler) WebcamStart(c *gin.Context) {
	var body webcamStartBody
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonError(c, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		jsonError(c, http.StatusBadRequest, "ffmpeg not found — webcam capture requires ffmpeg installed")
		return
	}

	cid := body.CameraID
	h.state.RLock()
	_, ok := h.state.Cameras[cid]
	if !ok {
		h.state.RUnlock()
		jsonError(c, http.StatusNotFound, fmt.Sprintf("Unknown camera: %s", cid))
		return
	}
	h.state.RUnlock()

	// Stop existing capture if any
	h.state.Lock()
	if h.state.WebcamRunning[cid] {
		h.state.WebcamRunning[cid] = false
	}
	h.state.Config.CameraSource = "webcam"

	deviceIdx := 0
	if body.DeviceIndex != nil {
		deviceIdx = *body.DeviceIndex
	} else {
		deviceIdx = h.state.Config.WebcamDevice
	}
	h.state.WebcamIndex[cid] = &deviceIdx
	h.state.WebcamRunning[cid] = true
	h.state.Unlock()

	// Start webcam worker in goroutine
	go h.webcamWorker(cid, deviceIdx)

	time.Sleep(300 * time.Millisecond)

	h.state.RLock()
	running := h.state.WebcamRunning[cid]
	h.state.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"camera_id":    cid,
		"device_index": deviceIdx,
		"running":      running,
	})
}

func (h *Handler) WebcamStop(c *gin.Context) {
	var body webcamStopBody
	if err := c.ShouldBindJSON(&body); err != nil {
		jsonError(c, http.StatusBadRequest, err.Error())
		return
	}

	cid := body.CameraID
	h.state.RLock()
	_, ok := h.state.Cameras[cid]
	if !ok {
		h.state.RUnlock()
		jsonError(c, http.StatusNotFound, fmt.Sprintf("Unknown camera: %s", cid))
		return
	}
	h.state.RUnlock()

	h.state.Lock()
	h.state.WebcamRunning[cid] = false
	h.state.WebcamIndex[cid] = nil
	// If no cameras are using webcam, revert to simulated
	anyWebcam := false
	for _, v := range h.state.WebcamIndex {
		if v != nil {
			anyWebcam = true
			break
		}
	}
	if !anyWebcam {
		h.state.Config.CameraSource = "simulated"
	}
	h.state.Unlock()

	time.Sleep(300 * time.Millisecond)

	h.state.RLock()
	running := h.state.WebcamRunning[cid]
	h.state.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"camera_id": cid,
		"running":   running,
	})
}

func (h *Handler) WebcamStartAll(c *gin.Context) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		jsonError(c, http.StatusBadRequest, "ffmpeg not found — webcam capture requires ffmpeg installed")
		return
	}

	h.state.RLock()
	cids := make([]string, 0, len(h.state.Cameras))
	for cid := range h.state.Cameras {
		cids = append(cids, cid)
	}
	h.state.RUnlock()

	results := make(map[string]gin.H)
	for i, cid := range cids {
		h.state.Lock()
		if h.state.WebcamRunning[cid] {
			h.state.WebcamRunning[cid] = false
		}
		h.state.Config.CameraSource = "webcam"
		h.state.WebcamIndex[cid] = &[]int{i}[0]
		h.state.WebcamRunning[cid] = true
		h.state.Unlock()

		go h.webcamWorker(cid, i)
		time.Sleep(200 * time.Millisecond)
		results[cid] = gin.H{"device_index": i, "running": true}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "results": results})
}

func (h *Handler) WebcamStopAll(c *gin.Context) {
	h.state.Lock()
	for cid := range h.state.WebcamRunning {
		h.state.WebcamRunning[cid] = false
	}
	stopped := make([]string, 0, len(h.state.WebcamRunning))
	for cid := range h.state.WebcamRunning {
		stopped = append(stopped, cid)
	}
	for cid := range h.state.WebcamIndex {
		h.state.WebcamIndex[cid] = nil
	}
	h.state.Config.CameraSource = "simulated"
	h.state.Unlock()

	time.Sleep(300 * time.Millisecond)

	c.JSON(http.StatusOK, gin.H{"success": true, "stopped": stopped})
}

func (h *Handler) WebcamStatus(c *gin.Context) {
	h.state.RLock()
	cameras := make(map[string]gin.H)
	for cid := range h.state.Cameras {
		running := h.state.WebcamRunning[cid]
		_, hasFrame := h.state.LatestFrames[cid]
		var devPtr *int
		if h.state.WebcamIndex[cid] != nil {
			val := *h.state.WebcamIndex[cid]
			devPtr = &val
		}
		cameras[cid] = gin.H{
			"running":      running,
			"device_index": devPtr,
			"has_frame":    hasFrame,
		}
	}
	cameraSource := h.state.Config.CameraSource
	h.state.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"active_captures": cameras,
		"camera_source":   cameraSource,
	})
}

func (h *Handler) WebcamStatusGET(c *gin.Context) {
	h.state.RLock()
	cameras := make(map[string]gin.H)
	for cid := range h.state.Cameras {
		_, hasFrame := h.state.LatestFrames[cid]
		var devPtr *int
		if h.state.WebcamIndex[cid] != nil {
			val := *h.state.WebcamIndex[cid]
			devPtr = &val
		}
		cameras[cid] = gin.H{
			"device_index": devPtr,
			"has_frame":    hasFrame,
		}
	}
	cameraSource := h.state.Config.CameraSource
	h.state.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"active_captures": cameras,
		"camera_source":   cameraSource,
	})
}

// webcamWorker runs ffmpeg as a subprocess to capture webcam video.
func (h *Handler) webcamWorker(cameraID string, deviceIdx int) {
	log.Printf("Webcam %s: starting worker (device %d)", cameraID, deviceIdx)

	h.state.RLock()
	w := h.state.Config.FrameWidth
	ht := h.state.Config.FrameHeight
	fps := h.state.Config.FPS
	if fps < 1 {
		fps = 5
	}
	qRaw := h.state.Config.JPEGQuality
	h.state.RUnlock()

	q := qRaw
	if q < 10 {
		q = 10
	}
	if q > 100 {
		q = 100
	}
	qFfmpeg := 27 - q*25/100
	if qFfmpeg < 2 {
		qFfmpeg = 2
	}
	if qFfmpeg > 25 {
		qFfmpeg = 25
	}

	args := []string{
		"-loglevel", "error",
		"-f", "avfoundation",
		"-video_device_index", strconv.Itoa(deviceIdx),
		"-video_size", "640x480",
		"-r", "30",
		"-i", "",
		"-vf", fmt.Sprintf("scale=%d:%d,fps=%d", w, ht, fps),
		"-f", "mjpeg",
		"-q:v", strconv.Itoa(qFfmpeg),
		"-",
	}

	cmd := exec.Command("ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Webcam %s: stdout pipe error: %v", cameraID, err)
		h.state.Lock()
		h.state.WebcamRunning[cameraID] = false
		h.state.Unlock()
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Webcam %s: stderr pipe error: %v", cameraID, err)
		stdout.Close()
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Webcam %s: start error: %v", cameraID, err)
		h.state.Lock()
		h.state.WebcamRunning[cameraID] = false
		h.state.Unlock()
		return
	}

	log.Printf("Webcam %s: started ffmpeg PID %d", cameraID, cmd.Process.Pid)

	// Read stderr in background to prevent blocking
	go io.Copy(io.Discard, stderr)

	// MJPEG parsing state
	buf := make([]byte, 0)
	readBuf := make([]byte, 65536)
	soi := []byte{0xFF, 0xD8}
	eoi := []byte{0xFF, 0xD9}

	running := true
	for running {
		n, err := stdout.Read(readBuf)
		if err != nil {
			break
		}
		buf = append(buf, readBuf[:n]...)

		// Parse complete JPEG frames from buffer
		for {
			start := bytes.Index(buf, soi)
			if start == -1 {
				buf = buf[:0]
				break
			}

			afterSOI := buf[start+2:]
			eoiPos := bytes.Index(afterSOI, eoi)
			if eoiPos == -1 {
				// Incomplete frame; trim before SOI if needed
				if start > 0 {
					buf = buf[start:]
				}
				break
			}

			// Full JPEG frame: SOI ... EOI
			frameEnd := start + 2 + eoiPos + 2
			jpeg := make([]byte, frameEnd-start)
			copy(jpeg, buf[start:frameEnd])
			buf = buf[frameEnd:]

			h.state.SetLatestFrame(cameraID, jpeg)
		}

		h.state.RLock()
		running = h.state.WebcamRunning[cameraID]
		h.state.RUnlock()
	}

	// Cleanup
	cmd.Process.Kill()
	cmd.Wait()

	h.state.Lock()
	h.state.WebcamRunning[cameraID] = false
	h.state.WebcamIndex[cameraID] = nil
	h.state.Unlock()

	log.Printf("Webcam %s: stopped", cameraID)
}
