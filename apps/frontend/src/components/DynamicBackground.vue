<template>
  <div class="dynamic-background" ref="container">
    <canvas ref="canvas" class="background-canvas"></canvas>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { themeColors } from '../config/theme';
import { useThemeStore } from '../stores/theme';

const themeStore = useThemeStore();
const canvas = ref<HTMLCanvasElement | null>(null);
const container = ref<HTMLDivElement | null>(null);

type RGB = [number, number, number];

interface Particle {
  x: number;
  y: number;
  size: number;
  speed: number;
  drift: number;
  alpha: number;
  phase: number;
}

interface TrailGlyph {
  x: number;
  y: number;
  symbol: string;
  bornAt: number;
  rolledAt: number;
  seed: number;
}

interface TrailMemory {
  symbol: string;
  rolledAt: number;
  seed: number;
}

interface ClickRipple {
  x: number;
  y: number;
  bornAt: number;
  maxRadius: number;
  activated: Set<string>;
}

const BASE_DPR = 1.5;
const PARTICLE_DENSITY = 18000;
const MAX_PARTICLES = 72;
const GRID_STEP = 56;
const TRAIL_TTL = 1450;
const TRAIL_MAX_GLYPHS = 420;
const TRAIL_MASK_WIDTH = 168;
const TRAIL_MASK_HEIGHT = 128;
const TRAIL_MASK_RADIUS = 34;
const TRAIL_ROLL_DURATION = 180;
const TRAIL_REROLL_COOLDOWN = 10000;
const RIPPLE_SPEED = 320;
const RIPPLE_BAND = 34;
const TRAIL_SYMBOLS = ['{', '}', '<', '>', '/', '*', '+', '=', '_', '#', '::', '[]'];
const TRAIL_ROLL_SYMBOLS = [
  '+',
  '-',
  '*',
  '/',
  '=',
  '<',
  '>',
  '{',
  '}',
  '[',
  ']',
  'x',
  'X',
  '#',
  '_',
];
const TRAIL_KAOMOJI = [
  '^_^',
  '>_<',
  'o_o',
  'owo',
  'uwu',
  '(^^)',
  '(._.)',
  '(=^.^=)',
  '(*^-^)',
  '( •_•)',
  '( ´ ▽ ` )',
  '(￣▽￣)',
  '(｀・ω・´)',
  '( ᵔ ᵕ ᵔ )',
];

let ctx: CanvasRenderingContext2D | null = null;
let textureCanvas: HTMLCanvasElement | null = null;
let textureCtx: CanvasRenderingContext2D | null = null;
let animationId: number | null = null;
let mediaQuery: MediaQueryList | null = null;
let prefersReducedMotion = false;
let canvasWidth = 0;
let canvasHeight = 0;
let dpr = 1;
let particles: Particle[] = [];
let trailGlyphs = new Map<string, TrailGlyph>();
let trailMemory = new Map<string, TrailMemory>();
let clickRipples: ClickRipple[] = [];
let lastTimestamp = 0;
let textureDirty = true;

const hexToRgb = (hex: string): RGB => [
  parseInt(hex.slice(1, 3), 16),
  parseInt(hex.slice(3, 5), 16),
  parseInt(hex.slice(5, 7), 16),
];

const mix = (a: number, b: number, t: number) => Math.round(a + (b - a) * t);

const mixRgb = (a: RGB, b: RGB, t: number): RGB => [
  mix(a[0], b[0], t),
  mix(a[1], b[1], t),
  mix(a[2], b[2], t),
];

const rgba = ([r, g, b]: RGB, alpha: number) => `rgba(${r}, ${g}, ${b}, ${alpha})`;

const fillRoundedRect = (
  target: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number
) => {
  const r = Math.min(radius, width / 2, height / 2);

  target.beginPath();
  target.moveTo(x + r, y);
  target.lineTo(x + width - r, y);
  target.quadraticCurveTo(x + width, y, x + width, y + r);
  target.lineTo(x + width, y + height - r);
  target.quadraticCurveTo(x + width, y + height, x + width - r, y + height);
  target.lineTo(x + r, y + height);
  target.quadraticCurveTo(x, y + height, x, y + height - r);
  target.lineTo(x, y + r);
  target.quadraticCurveTo(x, y, x + r, y);
  target.closePath();
  target.fill();
};

const palette = computed(() => {
  const isDark = themeStore.mode === 'dark';
  const primary = hexToRgb(themeColors[themeStore.color].primary);
  const secondary = hexToRgb(themeColors[themeStore.color].secondary);
  const base: RGB = isDark ? [17, 17, 22] : [247, 245, 242];
  const surface: RGB = isDark ? [26, 26, 34] : [255, 255, 255];
  const ink: RGB = isDark ? [236, 235, 232] : [28, 25, 23];

  return {
    isDark,
    base,
    surface,
    ink,
    primary,
    secondary,
    auraA: mixRgb(primary, surface, isDark ? 0.2 : 0.46),
    auraB: mixRgb(secondary, primary, isDark ? 0.28 : 0.12),
    grid: isDark ? ([42, 42, 54] as RGB) : ([214, 211, 206] as RGB),
    vignette: isDark ? ([0, 0, 0] as RGB) : ([120, 106, 88] as RGB),
  };
});

const resetParticles = () => {
  const area = canvasWidth * canvasHeight;
  const count = Math.min(MAX_PARTICLES, Math.max(20, Math.round(area / PARTICLE_DENSITY)));

  particles = Array.from({ length: count }, () => ({
    x: Math.random() * canvasWidth,
    y: Math.random() * canvasHeight,
    size: 1.4 + Math.random() * 3.2,
    speed: 12 + Math.random() * 20,
    drift: -8 + Math.random() * 16,
    alpha: 0.1 + Math.random() * 0.12,
    phase: Math.random() * Math.PI * 2,
  }));
};

const renderTexture = () => {
  if (!textureCtx) return;

  const p = palette.value;
  textureCtx.clearRect(0, 0, canvasWidth, canvasHeight);

  const baseGradient = textureCtx.createLinearGradient(0, 0, canvasWidth, canvasHeight);
  baseGradient.addColorStop(0, rgba(mixRgb(p.base, p.surface, p.isDark ? 0.03 : 0.25), 1));
  baseGradient.addColorStop(0.55, rgba(p.base, 1));
  baseGradient.addColorStop(1, rgba(mixRgb(p.base, p.primary, p.isDark ? 0.08 : 0.06), 1));
  textureCtx.fillStyle = baseGradient;
  textureCtx.fillRect(0, 0, canvasWidth, canvasHeight);

  const gridAlpha = p.isDark ? 0.18 : 0.28;
  textureCtx.lineWidth = 1;
  textureCtx.strokeStyle = rgba(p.grid, gridAlpha);
  textureCtx.beginPath();
  for (let x = -0.5; x <= canvasWidth + GRID_STEP; x += GRID_STEP) {
    textureCtx.moveTo(x, 0);
    textureCtx.lineTo(x, canvasHeight);
  }
  for (let y = -0.5; y <= canvasHeight + GRID_STEP; y += GRID_STEP) {
    textureCtx.moveTo(0, y);
    textureCtx.lineTo(canvasWidth, y);
  }
  textureCtx.stroke();

  const fineAlpha = p.isDark ? 0.05 : 0.08;
  textureCtx.strokeStyle = rgba(p.grid, fineAlpha);
  textureCtx.beginPath();
  for (let x = GRID_STEP / 2 - 0.5; x <= canvasWidth + GRID_STEP; x += GRID_STEP) {
    textureCtx.moveTo(x, 0);
    textureCtx.lineTo(x, canvasHeight);
  }
  for (let y = GRID_STEP / 2 - 0.5; y <= canvasHeight + GRID_STEP; y += GRID_STEP) {
    textureCtx.moveTo(0, y);
    textureCtx.lineTo(canvasWidth, y);
  }
  textureCtx.stroke();

  const vignette = textureCtx.createRadialGradient(
    canvasWidth * 0.52,
    canvasHeight * 0.42,
    Math.min(canvasWidth, canvasHeight) * 0.08,
    canvasWidth * 0.52,
    canvasHeight * 0.42,
    Math.max(canvasWidth, canvasHeight) * 0.78
  );
  vignette.addColorStop(0, rgba(p.vignette, 0));
  vignette.addColorStop(1, rgba(p.vignette, p.isDark ? 0.22 : 0.08));
  textureCtx.fillStyle = vignette;
  textureCtx.fillRect(0, 0, canvasWidth, canvasHeight);

  textureDirty = false;
};

const drawAura = (target: CanvasRenderingContext2D, timestamp: number) => {
  const p = palette.value;
  const t = timestamp * 0.00022;
  const maxSide = Math.max(canvasWidth, canvasHeight);
  const slowX = Math.sin(t) * canvasWidth * 0.16;
  const slowY = Math.cos(t * 0.82) * canvasHeight * 0.11;

  const primary = target.createRadialGradient(
    canvasWidth * 0.22 + slowX,
    canvasHeight * 0.2 + slowY,
    0,
    canvasWidth * 0.22 + slowX,
    canvasHeight * 0.2 + slowY,
    maxSide * 0.68
  );
  primary.addColorStop(0, rgba(p.auraA, p.isDark ? 0.46 : 0.48));
  primary.addColorStop(0.48, rgba(p.auraA, p.isDark ? 0.18 : 0.2));
  primary.addColorStop(1, rgba(p.auraA, 0));
  target.fillStyle = primary;
  target.fillRect(0, 0, canvasWidth, canvasHeight);

  const secondary = target.createRadialGradient(
    canvasWidth * 0.86 - slowX * 0.55,
    canvasHeight * 0.76 - slowY,
    0,
    canvasWidth * 0.86 - slowX * 0.55,
    canvasHeight * 0.76 - slowY,
    maxSide * 0.56
  );
  secondary.addColorStop(0, rgba(p.auraB, p.isDark ? 0.32 : 0.28));
  secondary.addColorStop(0.58, rgba(p.auraB, p.isDark ? 0.12 : 0.11));
  secondary.addColorStop(1, rgba(p.auraB, 0));
  target.fillStyle = secondary;
  target.fillRect(0, 0, canvasWidth, canvasHeight);
};

const drawParticles = (target: CanvasRenderingContext2D, dt: number, timestamp: number) => {
  const p = palette.value;
  const color = p.isDark ? mixRgb(p.primary, p.ink, 0.4) : mixRgb(p.primary, p.base, 0.2);

  for (const particle of particles) {
    if (!prefersReducedMotion) {
      particle.y -= particle.speed * (dt / 1000);
      particle.x += Math.sin(timestamp * 0.00062 + particle.phase) * particle.drift * (dt / 1000);
      if (particle.y < -12) {
        particle.y = canvasHeight + 12;
        particle.x = Math.random() * canvasWidth;
      }
    }

    const pulse = 0.72 + Math.sin(timestamp * 0.001 + particle.phase) * 0.28;
    target.fillStyle = rgba(color, particle.alpha * pulse);
    target.fillRect(particle.x, particle.y, particle.size, particle.size);
  }
};

const drawTrailGlyphs = (target: CanvasRenderingContext2D, timestamp: number) => {
  if (trailGlyphs.size === 0) return;

  const p = palette.value;
  const color = p.isDark ? mixRgb(p.primary, p.ink, 0.28) : mixRgb(p.primary, p.ink, 0.12);
  const fontSize = Math.max(13, Math.min(16, GRID_STEP * 0.28));
  const expired: string[] = [];

  target.save();
  target.font = `600 ${fontSize}px "Onest", "Noto Sans SC", ui-monospace, monospace`;
  target.textAlign = 'center';
  target.textBaseline = 'middle';

  for (const [key, glyph] of trailGlyphs) {
    const age = timestamp - glyph.bornAt;
    if (age >= TRAIL_TTL) {
      expired.push(key);
      continue;
    }

    const life = 1 - age / TRAIL_TTL;
    const pulse = 0.88 + Math.sin(timestamp * 0.006 + glyph.seed) * 0.12;
    const alpha = Math.pow(life, 1.18) * pulse;
    const size = 28 + life * 8;
    const rollAge = timestamp - glyph.rolledAt;
    const isRolling = rollAge >= 0 && rollAge < TRAIL_ROLL_DURATION;
    const rollIndex = Math.floor(rollAge / 30 + glyph.seed * 7) % TRAIL_ROLL_SYMBOLS.length;
    const displaySymbol = isRolling ? TRAIL_ROLL_SYMBOLS[rollIndex]! : glyph.symbol;

    target.fillStyle = rgba(p.primary, (p.isDark ? 0.08 : 0.07) * alpha);
    fillRoundedRect(target, glyph.x - size / 2, glyph.y - size / 2, size, size, 8);

    target.fillStyle = rgba(color, (p.isDark ? 0.86 : 0.72) * alpha);
    target.fillText(displaySymbol, glyph.x, glyph.y);
  }

  target.restore();

  for (const key of expired) {
    trailGlyphs.delete(key);
  }
};

const render = (timestamp: number) => {
  if (!ctx) return;

  const dt = lastTimestamp ? Math.min(timestamp - lastTimestamp, 48) : 16;
  lastTimestamp = timestamp;

  if (textureDirty) renderTexture();
  updateClickRipples(timestamp);

  ctx.clearRect(0, 0, canvasWidth, canvasHeight);
  if (textureCanvas) {
    ctx.drawImage(textureCanvas, 0, 0, canvasWidth, canvasHeight);
  }
  drawAura(ctx, timestamp);
  drawTrailGlyphs(ctx, timestamp);
  drawParticles(ctx, dt, timestamp);

  if (prefersReducedMotion) {
    animationId = null;
    return;
  }

  animationId = requestAnimationFrame(render);
};

const queueRender = () => {
  if (animationId) cancelAnimationFrame(animationId);
  lastTimestamp = 0;
  animationId = requestAnimationFrame(render);
};

const resizeCanvas = () => {
  if (!canvas.value || !container.value) return;

  const rect = container.value.getBoundingClientRect();
  dpr = Math.min(window.devicePixelRatio || 1, BASE_DPR);
  canvasWidth = Math.max(1, rect.width);
  canvasHeight = Math.max(1, rect.height);

  canvas.value.width = Math.round(canvasWidth * dpr);
  canvas.value.height = Math.round(canvasHeight * dpr);
  canvas.value.style.width = `${canvasWidth}px`;
  canvas.value.style.height = `${canvasHeight}px`;

  ctx = canvas.value.getContext('2d');
  if (ctx) ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  textureCanvas = document.createElement('canvas');
  textureCanvas.width = canvas.value.width;
  textureCanvas.height = canvas.value.height;
  textureCtx = textureCanvas.getContext('2d');
  if (textureCtx) textureCtx.setTransform(dpr, 0, 0, dpr, 0, 0);

  textureDirty = true;
  resetParticles();
  queueRender();
};

const handleMotionChange = (event: MediaQueryListEvent) => {
  prefersReducedMotion = event.matches;
  queueRender();
};

const pointInRoundedMask = (dx: number, dy: number) => {
  const halfW = TRAIL_MASK_WIDTH / 2;
  const halfH = TRAIL_MASK_HEIGHT / 2;
  const innerX = halfW - TRAIL_MASK_RADIUS;
  const innerY = halfH - TRAIL_MASK_RADIUS;
  const absX = Math.abs(dx);
  const absY = Math.abs(dy);

  if (absX > halfW || absY > halfH) return false;
  if (absX <= innerX || absY <= innerY) return true;

  return (absX - innerX) ** 2 + (absY - innerY) ** 2 <= TRAIL_MASK_RADIUS ** 2;
};

const pickTrailSymbol = () => {
  if (Math.random() < 0.16) {
    return TRAIL_KAOMOJI[Math.floor(Math.random() * TRAIL_KAOMOJI.length)]!;
  }

  return TRAIL_SYMBOLS[Math.floor(Math.random() * TRAIL_SYMBOLS.length)]!;
};

const trimTrailGlyphs = () => {
  if (trailGlyphs.size <= TRAIL_MAX_GLYPHS) return;

  const overflow = trailGlyphs.size - TRAIL_MAX_GLYPHS;
  const keys = trailGlyphs.keys();
  for (let i = 0; i < overflow; i++) {
    const oldestKey = keys.next().value;
    if (oldestKey) trailGlyphs.delete(oldestKey);
  }
};

const activateGridGlyph = (col: number, row: number, now: number, forceReroll = false) => {
  const key = `${col}:${row}`;
  const glyphX = col * GRID_STEP;
  const glyphY = row * GRID_STEP;
  const existing = trailGlyphs.get(key);
  const memory = trailMemory.get(key);
  const shouldReroll = forceReroll || !memory || now - memory.rolledAt >= TRAIL_REROLL_COOLDOWN;
  const symbol = shouldReroll ? pickTrailSymbol() : memory.symbol;
  const rolledAt = shouldReroll ? now : memory.rolledAt;
  const seed = shouldReroll ? Math.random() * Math.PI * 2 : memory.seed;

  trailMemory.set(key, { symbol, rolledAt, seed });

  trailGlyphs.set(key, {
    x: glyphX,
    y: glyphY,
    symbol: shouldReroll ? symbol : (existing?.symbol ?? symbol),
    bornAt: now,
    rolledAt: shouldReroll ? rolledAt : (existing?.rolledAt ?? rolledAt),
    seed: shouldReroll ? seed : (existing?.seed ?? seed),
  });
};

const addTrailGlyph = (x: number, y: number) => {
  const centerCol = Math.round(x / GRID_STEP);
  const centerRow = Math.round(y / GRID_STEP);
  const colRadius = Math.ceil(TRAIL_MASK_WIDTH / GRID_STEP / 2);
  const rowRadius = Math.ceil(TRAIL_MASK_HEIGHT / GRID_STEP / 2);
  const bornAt = performance.now();

  for (let row = centerRow - rowRadius; row <= centerRow + rowRadius; row++) {
    for (let col = centerCol - colRadius; col <= centerCol + colRadius; col++) {
      const glyphX = col * GRID_STEP;
      const glyphY = row * GRID_STEP;
      if (!pointInRoundedMask(glyphX - x, glyphY - y)) continue;

      activateGridGlyph(col, row, bornAt);
    }
  }

  trimTrailGlyphs();

  if (prefersReducedMotion || !animationId) {
    queueRender();
  }
};

const updateClickRipples = (timestamp: number) => {
  if (clickRipples.length === 0) return;

  const activeRipples: ClickRipple[] = [];
  const maxCol = Math.ceil(canvasWidth / GRID_STEP);
  const maxRow = Math.ceil(canvasHeight / GRID_STEP);

  for (const ripple of clickRipples) {
    const radius = (timestamp - ripple.bornAt) * 0.001 * RIPPLE_SPEED;
    if (radius > ripple.maxRadius + RIPPLE_BAND) continue;

    const minCol = Math.max(0, Math.floor((ripple.x - radius - RIPPLE_BAND) / GRID_STEP));
    const maxRingCol = Math.min(maxCol, Math.ceil((ripple.x + radius + RIPPLE_BAND) / GRID_STEP));
    const minRow = Math.max(0, Math.floor((ripple.y - radius - RIPPLE_BAND) / GRID_STEP));
    const maxRingRow = Math.min(maxRow, Math.ceil((ripple.y + radius + RIPPLE_BAND) / GRID_STEP));

    for (let row = minRow; row <= maxRingRow; row++) {
      for (let col = minCol; col <= maxRingCol; col++) {
        const key = `${col}:${row}`;
        if (ripple.activated.has(key)) continue;

        const glyphX = col * GRID_STEP;
        const glyphY = row * GRID_STEP;
        const distance = Math.hypot(glyphX - ripple.x, glyphY - ripple.y);
        if (Math.abs(distance - radius) > RIPPLE_BAND) continue;

        ripple.activated.add(key);
        activateGridGlyph(col, row, timestamp, true);
      }
    }

    activeRipples.push(ripple);
  }

  clickRipples = activeRipples;
  trimTrailGlyphs();
};

const handlePointerMove = (event: PointerEvent) => {
  if (!container.value) return;

  const rect = container.value.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;

  if (x < 0 || y < 0 || x > rect.width || y > rect.height) return;
  addTrailGlyph(x, y);
};

const handlePointerDown = (event: PointerEvent) => {
  if (!container.value) return;

  const rect = container.value.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;

  if (x < 0 || y < 0 || x > rect.width || y > rect.height) return;

  clickRipples.push({
    x,
    y,
    bornAt: performance.now(),
    maxRadius: Math.hypot(Math.max(x, rect.width - x), Math.max(y, rect.height - y)),
    activated: new Set(),
  });

  if (prefersReducedMotion || !animationId) {
    queueRender();
  }
};

watch(palette, () => {
  textureDirty = true;
  queueRender();
});

onMounted(() => {
  mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
  prefersReducedMotion = mediaQuery.matches;

  resizeCanvas();
  window.addEventListener('resize', resizeCanvas);
  document.addEventListener('pointermove', handlePointerMove, { passive: true });
  document.addEventListener('pointerdown', handlePointerDown, { passive: true });
  mediaQuery.addEventListener('change', handleMotionChange);
});

onUnmounted(() => {
  if (animationId) cancelAnimationFrame(animationId);
  window.removeEventListener('resize', resizeCanvas);
  document.removeEventListener('pointermove', handlePointerMove);
  document.removeEventListener('pointerdown', handlePointerDown);
  mediaQuery?.removeEventListener('change', handleMotionChange);
});
</script>

<style scoped>
.dynamic-background {
  position: absolute;
  inset: 0;
  overflow: hidden;
  z-index: 0;
  background: var(--background-color);
}

.background-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}
</style>
