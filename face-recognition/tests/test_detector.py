import cv2
import numpy as np
import pytest

from app.detector import Face, FaceDetector, _nms


# ---------------------------------------------------------------------------
# NMS tests
# ---------------------------------------------------------------------------

class TestNMS:
    def test_removes_overlapping(self):
        boxes = np.array([[10, 10, 100, 100], [20, 20, 110, 110]], dtype=np.float32)
        scores = np.array([0.9, 0.8], dtype=np.float32)
        keep = _nms(boxes, scores, iou_threshold=0.5)
        assert len(keep) == 1
        assert keep[0] == 0  # higher-score box kept

    def test_keeps_non_overlapping(self):
        boxes = np.array(
            [[10, 10, 50, 50], [100, 100, 150, 150]], dtype=np.float32
        )
        scores = np.array([0.9, 0.8], dtype=np.float32)
        keep = _nms(boxes, scores, iou_threshold=0.5)
        assert len(keep) == 2

    def test_empty_boxes(self):
        boxes = np.empty((0, 4), dtype=np.float32)
        scores = np.empty((0,), dtype=np.float32)
        keep = _nms(boxes, scores)
        assert keep == []

    def test_single_box(self):
        boxes = np.array([[10, 10, 100, 100]], dtype=np.float32)
        scores = np.array([0.9], dtype=np.float32)
        keep = _nms(boxes, scores)
        assert keep == [0]

    def test_iou_threshold_zero(self):
        """With iou_threshold=0 any overlap causes suppression."""
        boxes = np.array([[10, 10, 100, 100], [15, 15, 95, 95]], dtype=np.float32)
        scores = np.array([0.9, 0.8], dtype=np.float32)
        keep = _nms(boxes, scores, iou_threshold=0.0)
        assert len(keep) == 1
        assert keep[0] == 0

    def test_low_score_first(self):
        """Box with lower score but no overlap is still kept."""
        boxes = np.array(
            [[10, 10, 100, 100], [200, 200, 300, 300]], dtype=np.float32
        )
        scores = np.array([0.9, 0.3], dtype=np.float32)
        keep = _nms(boxes, scores, iou_threshold=0.5)
        assert len(keep) == 2


# ---------------------------------------------------------------------------
# Quality filter tests
# ---------------------------------------------------------------------------

class TestQualityFilter:
    def test_rejects_blurry(self, face_detector, synthetic_image, blurry_image):
        sharp_face = Face(x1=10, y1=10, x2=90, y2=90, confidence=0.9)
        blurry_face = Face(x1=10, y1=10, x2=90, y2=90, confidence=0.9)

        filtered_sharp = face_detector._quality_filter(synthetic_image, [sharp_face])
        filtered_blurry = face_detector._quality_filter(blurry_image, [blurry_face])

        assert len(filtered_sharp) == 1  # noise is sharp → kept
        assert len(filtered_blurry) == 0  # uniform is blurry → rejected

    def test_threshold_zero_passes_all(self, no_blur_detector, blurry_image):
        face = Face(x1=10, y1=10, x2=110, y2=110, confidence=0.9)
        filtered = no_blur_detector._quality_filter(blurry_image, [face])
        assert len(filtered) == 1

    def test_rejects_extreme_aspect_ratio(self, no_blur_detector, blurry_image):
        wide = Face(x1=10, y1=80, x2=110, y2=90, confidence=0.9)   # w/h=10
        tall = Face(x1=80, y1=10, x2=90, y2=110, confidence=0.9)   # w/h=0.1
        normal = Face(x1=10, y1=10, x2=110, y2=110, confidence=0.9)  # w/h=1.0

        filtered = no_blur_detector._quality_filter(
            blurry_image, [wide, tall, normal]
        )
        assert len(filtered) == 1
        assert filtered[0] == normal

    def test_empty_faces_returns_empty(self, face_detector, synthetic_image):
        filtered = face_detector._quality_filter(synthetic_image, [])
        assert filtered == []

    def test_keeps_multiple_good_faces(self, face_detector, synthetic_image):
        faces = [
            Face(x1=10, y1=10, x2=90, y2=90, confidence=0.9),
            Face(x1=110, y1=10, x2=190, y2=90, confidence=0.8),
        ]
        filtered = face_detector._quality_filter(synthetic_image, faces)
        assert len(filtered) == 2

    def test_face_roi_clamped_to_image_bounds(self, face_detector, synthetic_image):
        """Face extending beyond image edge is clamped, not crashed."""
        face = Face(x1=-10, y1=-10, x2=50, y2=50, confidence=0.9)
        filtered = face_detector._quality_filter(synthetic_image, [face])
        assert isinstance(filtered, list)


# ---------------------------------------------------------------------------
# Fallback detection tests
# ---------------------------------------------------------------------------

class TestFallbackDetect:
    def test_returns_list_for_any_image(self, face_detector, synthetic_image):
        result = face_detector._fallback_detect(synthetic_image)
        assert isinstance(result, list)
        if len(result) > 0:
            assert isinstance(result[0], Face)

    def test_detect_without_model_uses_fallback(self, face_detector, synthetic_image):
        """detect() with model_path=None should not crash."""
        result = face_detector.detect(synthetic_image)
        assert isinstance(result, list)

    def test_cascade_file_exists(self):
        path = cv2.data.haarcascades + "haarcascade_frontalface_default.xml"
        import os
        assert os.path.isfile(path), "Haar Cascade XML file not found"


# ---------------------------------------------------------------------------
# Postprocess tests
# ---------------------------------------------------------------------------

class TestPostprocess:
    def test_empty_outputs_returns_empty_list(self):
        detector = FaceDetector(
            model_path="",
            conf_threshold=0.5,
            input_size=(640, 640),
            min_face_size=80,
        )
        result = detector._postprocess([], (640, 480))
        assert result == []

    def test_none_outputs_returns_empty_list(self):
        detector = FaceDetector(
            model_path="",
            conf_threshold=0.5,
            input_size=(640, 640),
            min_face_size=80,
        )
        result = detector._postprocess(None, (640, 480))
        assert result == []


# ---------------------------------------------------------------------------
# Face dataclass tests
# ---------------------------------------------------------------------------

class TestFaceDataclass:
    def test_construct_without_landmarks(self):
        face = Face(x1=10.0, y1=20.0, x2=100.0, y2=120.0, confidence=0.95)
        assert face.x1 == 10.0
        assert face.y1 == 20.0
        assert face.x2 == 100.0
        assert face.y2 == 120.0
        assert face.confidence == 0.95
        assert face.landmarks is None

    def test_construct_with_landmarks(self):
        landmarks = [(30, 40), (70, 40), (50, 60), (30, 80), (70, 80)]
        face = Face(
            x1=10.0, y1=20.0, x2=100.0, y2=120.0,
            confidence=0.95, landmarks=landmarks,
        )
        assert face.landmarks == landmarks
        assert len(face.landmarks) == 5

    def test_face_equality_by_value(self):
        f1 = Face(x1=0, y1=0, x2=10, y2=10, confidence=0.9)
        f2 = Face(x1=0, y1=0, x2=10, y2=10, confidence=0.9)
        # dataclass default eq compares by value
        assert f1 == f2

    def test_face_repr(self):
        face = Face(x1=0, y1=0, x2=10, y2=10, confidence=0.9)
        r = repr(face)
        assert "Face(" in r
        assert "confidence=0.9" in r
