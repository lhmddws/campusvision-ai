import cv2, numpy as np, onnxruntime as ort, os, sys
os.environ['PYTHONWARNINGS'] = 'ignore'

# Load fixture
i = cv2.imread(r'C:\Users\lhmddws\campusvision-ai\face-recognition\tests\fixtures\face.jpg')
if i is None:
    print("Cannot load fixture")
    sys.exit(1)
print(f'Fixture: shape={i.shape} mean={i.mean():.1f}')

# Preprocess
r = cv2.resize(i, (640, 640))
x = r.astype(np.float32) - np.array([104.0, 117.0, 123.0], dtype=np.float32)
x = x.transpose(2, 0, 1)[None, ...]

# Model
s = ort.InferenceSession(r'C:\Users\lhmddws\campusvision-ai\face-recognition\app\models\retinaface_mv2.onnx',
                          providers=['CPUExecutionProvider'])
o = s.run(None, {s.get_inputs()[0].name: x})
sc = o[1][0][:, 1]
print(f'Conf: max={sc.max():.6f} >=0.5:{(sc>=0.5).sum()} >=0.3:{(sc>=0.3).sum()} >=0.1:{(sc>=0.1).sum()} >=0.01:{(sc>=0.01).sum()}')
top10 = np.sort(sc)[::-1][:10].tolist()
print(f'Top10: {[f"{s:.6f}" for s in top10]}')
