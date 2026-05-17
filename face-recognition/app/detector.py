from dataclasses import dataclass
from typing import Optional

import numpy as np
import cv2


@dataclass
class Face:
    x1: float
    y1: float
    x2: float
    y2: float
    confidence: float
    landmarks: Optional[list] = None


def _nms(boxes: np.ndarray, scores: np.ndarray, iou_threshold: float = 0.5) -> list:
    """Non-maximum suppression — remove overlapping bounding boxes.

    Args:
        boxes: (N, 4) array of (x1, y1, x2, y2) bounding boxes.
        scores: (N,) array of confidence scores.
        iou_threshold: IoU threshold for suppression.

    Returns:
        List of indices of kept boxes (sorted by descending score).
    """
    if len(boxes) == 0:
        return []

    x1 = boxes[:, 0]
    y1 = boxes[:, 1]
    x2 = boxes[:, 2]
    y2 = boxes[:, 3]

    areas = (x2 - x1) * (y2 - y1)
    order = scores.argsort()[::-1]

    keep = []
    while len(order) > 0:
        i = order[0]
        keep.append(int(i))
        if len(order) == 1:
            break

        xx1 = np.maximum(x1[i], x1[order[1:]])
        yy1 = np.maximum(y1[i], y1[order[1:]])
        xx2 = np.minimum(x2[i], x2[order[1:]])
        yy2 = np.minimum(y2[i], y2[order[1:]])

        w = np.maximum(0.0, xx2 - xx1)
        h = np.maximum(0.0, yy2 - yy1)
        inter = w * h
        ovr = inter / (areas[i] + areas[order[1:]] - inter + 1e-10)

        inds = np.where(ovr <= iou_threshold)[0]
        order = order[inds + 1]

    return keep


class FaceDetector:
    def __init__(self, model_path: str, conf_threshold: float,
                 input_size: tuple, min_face_size: int,
                 blur_threshold: float = 100.0,
                 nms_iou_threshold: float = 0.5):
        self.conf_threshold = conf_threshold
        self.input_size = input_size
        self.min_face_size = min_face_size
        self.blur_threshold = blur_threshold
        self.nms_iou_threshold = nms_iou_threshold
        self.session = None
        if model_path:
            import onnxruntime as ort
            self.session = ort.InferenceSession(model_path)

    def detect(self, image: np.ndarray) -> list[Face]:
        if self.session is None:
            faces = self._fallback_detect(image)
        else:
            faces = self._onnx_detect(image)
        faces = self._quality_filter(image, faces)
        return faces

    def _quality_filter(self, image: np.ndarray, faces: list[Face]) -> list[Face]:
        """Filter faces by Laplacian blur and aspect-ratio heuristics.
        No-op when blur_threshold == 0 (blur check disabled).
        """
        if not faces:
            return []

        filtered = []
        for face in faces:
            # 1. Blur detection via Laplacian variance (disabled when threshold == 0)
            if self.blur_threshold > 0:
                x1, y1, x2, y2 = map(int, (face.x1, face.y1, face.x2, face.y2))
                x1, y1 = max(0, x1), max(0, y1)
                x2, y2 = min(image.shape[1], x2), min(image.shape[0], y2)
                if x2 > x1 and y2 > y1:
                    face_roi = image[y1:y2, x1:x2]
                    gray_roi = cv2.cvtColor(face_roi, cv2.COLOR_BGR2GRAY)
                    laplacian_var = cv2.Laplacian(gray_roi, cv2.CV_64F).var()
                    if laplacian_var < self.blur_threshold:
                        continue

            # 2. Aspect ratio heuristic: reject extremes
            w = face.x2 - face.x1
            h = face.y2 - face.y1
            if w > 0 and h > 0 and (w / h > 1.5 or w / h < 0.5):
                continue

            filtered.append(face)

        return filtered

    def _onnx_detect(self, image: np.ndarray) -> list[Face]:
        # ONNX RetinaFace inference
        assert self.session is not None  # guarded by detect()
        img = cv2.resize(image, self.input_size)
        img = img.astype(np.float32) / 255.0
        img = np.transpose(img, (2, 0, 1))[None, ...]

        inputs = {self.session.get_inputs()[0].name: img}
        outputs = self.session.run(None, inputs)

        return self._postprocess(outputs, image.shape)

    def _postprocess(self, outputs: list, orig_shape: tuple) -> list[Face]:
        """Decode RetinaFace ONNX outputs into Face objects with NMS.

        RetinaFace-ResNet50 produces 3 concatenated output tensors across
        3 FPN levels (strides 8, 16, 32) with 2 anchors per grid cell.

        Args:
            outputs: ONNX model outputs:
                outputs[0]: bbox predictions, shape (1, N, 4) — [dx, dy, dw, dh]
                outputs[1]: confidence scores, shape (1, N, 2) — [bg, face]
                outputs[2]: landmark predictions, shape (1, N, 10) — 5 pts × (dx, dy)
            orig_shape: original image shape (H, W, C)

        Returns:
            List of Face objects with decoded bboxes and landmarks
        """
        if self.session is None or len(outputs) < 3:
            return []

        bbox_preds = outputs[0][0]   # (N, 4)
        conf_preds = outputs[1][0]   # (N, 2)
        landm_preds = outputs[2][0]  # (N, 10)
        N = bbox_preds.shape[0]
        if N == 0:
            return []

        # 1. Generate prior boxes (anchors) for 3 RetinaFace FPN levels
        input_h, input_w = self.input_size
        strides = [8, 16, 32]
        min_sizes = [[16, 32], [64, 128], [256, 512]]
        priors = self._generate_priors(input_h, input_w, strides, min_sizes)

        # 2. Decode bbox predictions → normalized [x1, y1, x2, y2]
        # Decoding follows RetinaFace: cx = prior_cx + dx * v[0] * prior_w
        variance = (0.1, 0.2)
        boxes = np.empty_like(bbox_preds)
        boxes[:, :2] = priors[:, :2] + bbox_preds[:, :2] * variance[0] * priors[:, 2:]
        boxes[:, 2:] = priors[:, 2:] * np.exp(bbox_preds[:, 2:] * variance[1])
        boxes[:, :2] -= boxes[:, 2:] / 2  # cx,cy,w,h → x1,y1,x2,y2
        boxes[:, 2:] += boxes[:, :2]

        # 3. Decode landmark predictions (5 pts × dx,dy)
        landm_parts = []
        for k in range(5):
            c = 2 * k
            part = priors[:, :2] + landm_preds[:, c:c+2] * variance[0] * priors[:, 2:]
            landm_parts.append(part)
        landms = np.concatenate(landm_parts, axis=1)

        # 4. Confidence threshold filtering (idx 1 = face confidence)
        scores = conf_preds[:, 1]
        keep = scores >= self.conf_threshold
        boxes = boxes[keep]
        landms = landms[keep]
        scores = scores[keep]
        if len(scores) == 0:
            return []

        # 5. Scale from normalized [0,1] to original image coordinates
        orig_h, orig_w = orig_shape[:2]
        boxes *= np.array([orig_w, orig_h, orig_w, orig_h], dtype=np.float32)
        landms *= np.array([orig_w, orig_h] * 5, dtype=np.float32)
        boxes[:, 0] = np.clip(boxes[:, 0], 0, orig_w)
        boxes[:, 1] = np.clip(boxes[:, 1], 0, orig_h)
        boxes[:, 2] = np.clip(boxes[:, 2], 0, orig_w)
        boxes[:, 3] = np.clip(boxes[:, 3], 0, orig_h)

        # 6. min_face_size filter
        face_size = np.maximum(boxes[:, 2] - boxes[:, 0],
                               boxes[:, 3] - boxes[:, 1])
        size_keep = face_size >= self.min_face_size
        boxes = boxes[size_keep]
        landms = landms[size_keep]
        scores = scores[size_keep]
        if len(scores) == 0:
            return []

        # 7. Non-Maximum Suppression
        keep_idx = _nms(boxes, scores, self.nms_iou_threshold)
        boxes = boxes[keep_idx]
        landms = landms[keep_idx]
        scores = scores[keep_idx]

        # 8. Build Face objects with landmarks
        faces = []
        for i in range(len(scores)):
            landmarks = [(float(landms[i, 2 * k]), float(landms[i, 2 * k + 1]))
                         for k in range(5)]
            faces.append(Face(
                x1=float(boxes[i, 0]),
                y1=float(boxes[i, 1]),
                x2=float(boxes[i, 2]),
                y2=float(boxes[i, 3]),
                confidence=float(scores[i]),
                landmarks=landmarks,
            ))
        return faces

    @staticmethod
    def _generate_priors(input_h: int, input_w: int,
                         strides: list, min_sizes: list) -> np.ndarray:
        """Generate RetinaFace prior boxes (anchors) for all FPN levels.

        Each FPN level has a feature map of size (ceil(H/stride), ceil(W/stride))
        with 2 anchor sizes per grid cell. Priors are normalized to [0, 1].

        Returns:
            ndarray of shape (N, 4) with columns [cx, cy, w, h] in [0, 1]
        """
        priors = []
        for stride, sizes in zip(strides, min_sizes):
            fm_h = int(np.ceil(input_h / stride))
            fm_w = int(np.ceil(input_w / stride))
            for i in range(fm_h):
                for j in range(fm_w):
                    for size in sizes:
                        cx = (j + 0.5) * stride / input_w
                        cy = (i + 0.5) * stride / input_h
                        w = size / input_w
                        h = size / input_h
                        priors.append([cx, cy, w, h])
        return np.array(priors, dtype=np.float32).reshape(-1, 4)

    def _fallback_detect(self, image: np.ndarray) -> list[Face]:
        # Haar Cascade fallback for dev/testing without ONNX model
        gray = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)
        cascade = cv2.CascadeClassifier(
            "/opt/homebrew/Cellar/opencv/4.13.0_8/share/opencv4/haarcascades/haarcascade_frontalface_default.xml"  # type: ignore[attr-defined]  # cv2.data added dynamically by opencv-python
        )
        rects = cascade.detectMultiScale(gray, 1.1, 5, minSize=(self.min_face_size, self.min_face_size))
        faces = []
        for x, y, w, h in rects:
            faces.append(Face(
                x1=float(x), y1=float(y),
                x2=float(x + w), y2=float(y + h),
                confidence=0.9
            ))
        return faces
