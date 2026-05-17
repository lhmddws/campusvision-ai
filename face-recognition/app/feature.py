import numpy as np
import cv2

# ArcFace 112×112 alignment: left_eye, right_eye, nose, left_mouth, right_mouth
ARCFACE_112_TARGETS = np.array([
    [38.2946, 51.6963],
    [73.5318, 51.5014],
    [56.0252, 71.7366],
    [41.5493, 92.3655],
    [70.7299, 92.2041],
], dtype=np.float32)


def _warp_face(image: np.ndarray, src_points: np.ndarray,
               dst_points: np.ndarray, output_size: tuple) -> np.ndarray:
    transform, _ = cv2.estimateAffinePartial2D(src_points, dst_points)
    if transform is None:
        return cv2.resize(image, output_size)
    return cv2.warpAffine(image, transform, output_size, flags=cv2.INTER_LINEAR)


class FeatureExtractor:
    def __init__(self, model_path: str, embedding_size: int):
        self.embedding_size = embedding_size
        self.session = None
        if model_path:
            import onnxruntime as ort
            self.session = ort.InferenceSession(model_path)

    def extract(self, image: np.ndarray, face) -> np.ndarray:
        aligned = self._align_face(image, face)
        if self.session:
            return self._onnx_extract(aligned)
        return self._fallback_embedding(aligned)

    def _align_face(self, image: np.ndarray, face) -> np.ndarray:
        landmarks = getattr(face, 'landmarks', None)
        if landmarks is not None and len(landmarks) >= 5:
            src_pts = np.array(landmarks[:5], dtype=np.float32)
            return _warp_face(image, src_pts, ARCFACE_112_TARGETS, (112, 112))
        x1, y1, x2, y2 = map(int, (face.x1, face.y1, face.x2, face.y2))
        face_img = image[y1:y2, x1:x2]
        if face_img.size == 0:
            return np.zeros((112, 112, 3), dtype=np.uint8)
        return cv2.resize(face_img, (112, 112))

    def _onnx_extract(self, face_img: np.ndarray) -> np.ndarray:
        assert self.session is not None  # guarded by extract()
        img = face_img.astype(np.float32) / 255.0
        img = np.transpose(img, (2, 0, 1))[None, ...]
        inputs = {self.session.get_inputs()[0].name: img}
        outputs = self.session.run(None, inputs)
        embedding = outputs[0].flatten()
        norm = np.linalg.norm(embedding)
        return embedding / norm if norm > 0 else embedding

    def _fallback_embedding(self, face_img: np.ndarray) -> np.ndarray:
        # Placeholder: resize + flatten + normalize for dev testing
        feat = cv2.resize(face_img, (32, 32)).flatten().astype(np.float32)
        norm = np.linalg.norm(feat)
        return feat / norm if norm > 0 else feat
