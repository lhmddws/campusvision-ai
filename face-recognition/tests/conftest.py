import numpy as np
import pytest

from app.detector import Face, FaceDetector


@pytest.fixture(scope="session")
def face_detector():
    """FaceDetector instance using Haar Cascade fallback (no ONNX model)."""
    return FaceDetector(
        model_path="",
        conf_threshold=0.5,
        input_size=(640, 640),
        min_face_size=80,
        blur_threshold=100.0,
    )


@pytest.fixture(scope="session")
def no_blur_detector():
    """FaceDetector with blur check disabled."""
    return FaceDetector(
        model_path="",
        conf_threshold=0.5,
        input_size=(640, 640),
        min_face_size=80,
        blur_threshold=0.0,
    )


@pytest.fixture
def synthetic_image():
    """200x200 random noise BGR image (sharp, high Laplacian variance)."""
    return np.random.randint(0, 256, (200, 200, 3), dtype=np.uint8)


@pytest.fixture
def blurry_image():
    """200x200 uniform-grey BGR image (zero Laplacian variance)."""
    return np.ones((200, 200, 3), dtype=np.uint8) * 128


def make_face(x1=10, y1=10, x2=110, y2=110, confidence=0.9, landmarks=None):
    """Helper to create a Face instance with defaults."""
    return Face(x1=x1, y1=y1, x2=x2, y2=y2, confidence=confidence, landmarks=landmarks)
