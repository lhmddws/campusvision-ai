<template>
  <div
    ref="containerRef"
    class="frame-canvas"
    :class="{ 'frame-canvas--offline': offline }"
  >
    <canvas
      ref="canvasRef"
      v-show="!!frameData && !offline"
      class="frame-canvas__element"
    />

    <!-- Empty state: no frame data -->
    <div v-if="!frameData && !offline" class="frame-canvas__overlay">
      <el-icon :size="48" class="frame-canvas__overlay-icon">
        <Clock />
      </el-icon>
      <span class="frame-canvas__overlay-text">等待画面...</span>
    </div>

    <!-- Offline state -->
    <div v-if="offline" class="frame-canvas__overlay frame-canvas__overlay--dark">
      <el-icon :size="48" class="frame-canvas__offline-icon">
        <VideoPause />
      </el-icon>
      <span class="frame-canvas__offline-text">摄像头离线</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue';
import { Clock, VideoPause } from '@element-plus/icons-vue';

/** Bounding box from face-recognition pipeline */
export interface BBox {
  x1: number;
  y1: number;
  x2: number;
  y2: number;
  name: string | null;
  confidence: number;
  frame_sequence: number;
}

interface Props {
  cameraId: string;
  frameData: string;
  bboxes?: BBox[];
  offline?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  bboxes: () => [],
  offline: false,
});

const ORIGINAL_WIDTH = 1280;
const ORIGINAL_HEIGHT = 720;

const containerRef = ref<HTMLDivElement | null>(null);
const canvasRef = ref<HTMLCanvasElement | null>(null);

let isCancelled = false;
let resizeObserver: ResizeObserver | null = null;

/** Ensure the base64 string is usable as a data URL */
function toDataUrl(base64: string): string {
  if (base64.startsWith('data:')) return base64;
  return `data:image/jpeg;base64,${base64}`;
}

/** Get canvas-to-original scale factors */
function getScaleFactors(
  canvas: HTMLCanvasElement,
): { scaleX: number; scaleY: number } {
  return {
    scaleX: canvas.width / ORIGINAL_WIDTH,
    scaleY: canvas.height / ORIGINAL_HEIGHT,
  };
}

/** Confidence to stroke colour */
function confidenceColor(confidence: number): string {
  if (confidence >= 0.8) return '#67c23a';
  if (confidence >= 0.6) return '#e6a23c';
  return '#f56c6c';
}

/** Draw all bbox rectangles and labels onto the canvas */
function drawBboxes(
  ctx: CanvasRenderingContext2D,
  canvas: HTMLCanvasElement,
): void {
  const { scaleX, scaleY } = getScaleFactors(canvas);
  const labelHeight = 20;
  const labelPadding = 4;

  ctx.save();
  ctx.lineWidth = 2;
  ctx.textBaseline = 'top';

  for (const bbox of props.bboxes) {
    const x = bbox.x1 * scaleX;
    const y = bbox.y1 * scaleY;
    const w = (bbox.x2 - bbox.x1) * scaleX;
    const h = (bbox.y2 - bbox.y1) * scaleY;

    if (w <= 0 || h <= 0) continue;

    const color = confidenceColor(bbox.confidence);

    // Draw rect
    ctx.strokeStyle = color;
    ctx.strokeRect(x, y, w, h);

    // Draw label
    const label =
      bbox.name != null
        ? `${bbox.name} ${(bbox.confidence * 100).toFixed(1)}%`
        : `陌生人 ${(bbox.confidence * 100).toFixed(1)}%`;

    ctx.font = '12px "PingFang SC", "Microsoft YaHei", sans-serif';
    const textWidth = ctx.measureText(label).width;
    const bgWidth = textWidth + labelPadding * 2;

    // Determine label y: below bbox if room, otherwise above
    let labelY = y + h;
    if (labelY + labelHeight > canvas.height) {
      labelY = y - labelHeight;
    }
    labelY = Math.max(0, Math.min(labelY, canvas.height - labelHeight));

    // Clamp x so label stays on-screen
    const labelX = Math.max(0, Math.min(x, canvas.width - bgWidth));

    // Background pill
    ctx.fillStyle = color;
    const radius = 3;
    ctx.beginPath();
    ctx.moveTo(labelX + radius, labelY);
    ctx.lineTo(labelX + bgWidth - radius, labelY);
    ctx.quadraticCurveTo(labelX + bgWidth, labelY, labelX + bgWidth, labelY + radius);
    ctx.lineTo(labelX + bgWidth, labelY + labelHeight - radius);
    ctx.quadraticCurveTo(
      labelX + bgWidth,
      labelY + labelHeight,
      labelX + bgWidth - radius,
      labelY + labelHeight,
    );
    ctx.lineTo(labelX + radius, labelY + labelHeight);
    ctx.quadraticCurveTo(labelX, labelY + labelHeight, labelX, labelY + labelHeight - radius);
    ctx.lineTo(labelX, labelY + radius);
    ctx.quadraticCurveTo(labelX, labelY, labelX + radius, labelY);
    ctx.closePath();
    ctx.fill();

    // Text
    ctx.fillStyle = '#ffffff';
    ctx.fillText(label, labelX + labelPadding, labelY + labelPadding);
  }

  ctx.restore();
}

/** Main render: decode frame → draw → overlay bboxes */
async function renderFrame(): Promise<void> {
  if (isCancelled) return;

  const canvas = canvasRef.value;
  const container = containerRef.value;
  if (!canvas || !container) return;

  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  // Set canvas pixel dimensions to match CSS layout
  const { width: containerWidth } = container.getBoundingClientRect();
  if (containerWidth <= 0) return;

  const canvasWidth = Math.floor(containerWidth);
  const canvasHeight = Math.floor(canvasWidth * (ORIGINAL_HEIGHT / ORIGINAL_WIDTH));

  // Only resize when needed to preserve canvas state
  if (canvas.width !== canvasWidth || canvas.height !== canvasHeight) {
    canvas.width = canvasWidth;
    canvas.height = canvasHeight;
  }

  ctx.clearRect(0, 0, canvas.width, canvas.height);

  if (!props.frameData || props.offline) return;

  // Draw frame
  try {
    // Primary: createImageBitmap (non-blocking decode)
    const blob = await fetch(toDataUrl(props.frameData)).then((r) => r.blob());
    if (isCancelled) return;

    const bitmap = await createImageBitmap(blob);
    if (isCancelled) {
      bitmap.close();
      return;
    }

    ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
    bitmap.close();
  } catch {
    if (isCancelled) return;

    // Fallback: Image element
    try {
      const img = new Image();
      img.src = toDataUrl(props.frameData);
      await new Promise<void>((resolve, reject) => {
        img.onload = () => resolve();
        img.onerror = () => reject(new Error('Image decode failed'));
      });
      if (isCancelled) return;
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    } catch {
      return; // Silent fail on decode error
    }
  }

  // Overlay bboxes
  drawBboxes(ctx, canvas);
}

// ── Watchers ────────────────────────────────────────────────

watch(
  () => [props.frameData, props.bboxes, props.offline],
  () => {
    nextTick(() => renderFrame());
  },
  { deep: true },
);

// ── Lifecycle ───────────────────────────────────────────────

onMounted(() => {
  if (containerRef.value) {
    resizeObserver = new ResizeObserver(() => {
      if (props.frameData && !props.offline && canvasRef.value) {
        nextTick(() => renderFrame());
      }
    });
    resizeObserver.observe(containerRef.value);
  }
});

onUnmounted(() => {
  isCancelled = true;
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
});
</script>

<style scoped lang="scss">
$text-secondary: #8c8c8c;

.frame-canvas {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  background: #000;
  border-radius: 8px;
  overflow: hidden;

  &--offline {
    opacity: 0.6;
  }

  &__element {
    display: block;
    width: 100%;
    height: 100%;
  }

  &__overlay {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;

    &--dark {
      background: rgba(0, 0, 0, 0.55);
    }
  }

  &__overlay-icon {
    color: $text-secondary;
  }

  &__overlay-text {
    font-size: 14px;
    color: $text-secondary;
    user-select: none;
  }

  &__offline-icon {
    color: #ffffff;
  }

  &__offline-text {
    font-size: 14px;
    color: #ffffff;
    user-select: none;
  }
}
</style>
