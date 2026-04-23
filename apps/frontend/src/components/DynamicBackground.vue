<template>
  <div class="dynamic-background" ref="container">
    <canvas ref="canvas" class="background-canvas"></canvas>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useThemeStore } from '../stores/theme';
import { themeColors } from '../config/theme';

const themeStore = useThemeStore();
const canvas = ref<HTMLCanvasElement | null>(null);
const container = ref<HTMLDivElement | null>(null);

// ─── 网格 + 波浪 ───
const PITCH = 48;
const GAP = 4;
const CELL = PITCH - GAP;
const CORNER = 8;
const STROKE_RADIUS = 240;
const MIN_CELL_SIZE = 12;
const WAVE_FREQS = [0.0004, 0.00022, 0.00012];
const WAVE_SPATIAL_X = [0.9, 9, 0.8];
const WAVE_SPATIAL_Y = [0.5, 0.32, 0.12];
const WAVE_AMPS = [1, 0.8, 0.8];
// 每次加载随机相位，打破固定模式
const WAVE_PHASES = Array.from({ length: 3 }, () => Math.random() * Math.PI * 2);

// ─── HSL 工具 ───
const hexToHSL = (hex: string) => {
  const r = parseInt(hex.slice(1, 3), 16) / 255;
  const g = parseInt(hex.slice(3, 5), 16) / 255;
  const b = parseInt(hex.slice(5, 7), 16) / 255;
  const max = Math.max(r, g, b),
    min = Math.min(r, g, b);
  let h = 0,
    s = 0;
  const l = (max + min) / 2;
  if (max !== min) {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
    switch (max) {
      case r:
        h = ((g - b) / d + (g < b ? 6 : 0)) / 6;
        break;
      case g:
        h = ((b - r) / d + 2) / 6;
        break;
      case b:
        h = ((r - g) / d + 4) / 6;
        break;
    }
  }
  return { h: h * 360, s: s * 100, l: l * 100 };
};

const hslToRGB = (h: number, s: number, l: number): [number, number, number] => {
  h /= 360;
  s /= 100;
  l /= 100;
  if (s === 0) {
    const v = Math.round(l * 255);
    return [v, v, v];
  }
  const hue2rgb = (p: number, q: number, t: number) => {
    if (t < 0) t += 1;
    if (t > 1) t -= 1;
    if (t < 1 / 6) return p + (q - p) * 6 * t;
    if (t < 1 / 2) return q;
    if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
    return p;
  };
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
  const p = 2 * l - q;
  return [
    Math.round(hue2rgb(p, q, h + 1 / 3) * 255),
    Math.round(hue2rgb(p, q, h) * 255),
    Math.round(hue2rgb(p, q, h - 1 / 3) * 255),
  ];
};

// ─── 调色板（高斯多色点交融） ───
interface ColorStop {
  pos: number; // 0=peak, 1=valley — 不等距分布
  color: [number, number, number];
}

const palette = computed(() => {
  const isDark = themeStore.mode === 'dark';
  const { primary } = themeColors[themeStore.color];
  const { h, s } = hexToHSL(primary);
  const c = Math.min(s * 1.1, 45);
  // 深色模式下大幅压低饱和度，保持矿物感而非霓虹感
  const satScale = isDark ? 0.5 : 1.0;
  const satCap = isDark ? 28 : 55;

  // 5 个颜色点 — 前 4 个类似色域 ±40°，末尾 1 个补色方向跳色
  // 跳色饱和度压至极低，在矿物色域内产生「石中矿脉」般的视觉惊喜
  const stops: ColorStop[] = [
    { pos: 0.0, color: hslToRGB(h, c * 0.06, isDark ? 82 : 24) },
    {
      pos: 0.22,
      color: hslToRGB((h + 8) % 360, Math.min(c * 0.5 * satScale, satCap), isDark ? 62 : 42),
    },
    {
      pos: 0.5,
      color: hslToRGB((h + 18) % 360, Math.min(c * 0.9 * satScale, satCap), isDark ? 46 : 54),
    },
    {
      pos: 0.78,
      color: hslToRGB((h + 28) % 360, Math.min(s * 0.8 * satScale, satCap), isDark ? 34 : 60),
    },
    // 跳色 — 补色方向低饱和度矿物色
    {
      pos: 1.0,
      color: hslToRGB(
        (h + 168) % 360,
        Math.min(c * 0.4 * satScale, satCap * 0.6),
        isDark ? 50 : 48
      ),
    },
  ];

  return {
    bg: isDark ? '#111116' : '#F7F5F2',
    isDark,
    borderRGB: isDark ? [42, 42, 54] : ([214, 211, 206] as [number, number, number]),
    stops,
    bgAlpha: isDark ? 0.2 : 0.18,
    cellAlpha: isDark ? 0.55 : 0.45,
    strokeAlpha: isDark ? 0.9 : 0.85,
  };
});

// ─── 颜色查找表（LUT） ───
// 主题变化时预计算 256 级颜色，渲染时直接索引替代 5×exp()
const COLOR_LUT_SIZE = 256;
let colorLUT: Uint8Array = new Uint8Array(COLOR_LUT_SIZE * 3);

const rebuildColorLUT = () => {
  const { stops } = palette.value;
  colorLUT = new Uint8Array(COLOR_LUT_SIZE * 3);
  for (let i = 0; i < COLOR_LUT_SIZE; i++) {
    const cw = i / (COLOR_LUT_SIZE - 1);
    const t = 1 - cw;
    let totalW = 0;
    let r = 0;
    let g = 0;
    let b = 0;
    for (const stop of stops) {
      const dist = (t - stop.pos) / BLEND_SIGMA;
      const w = Math.exp(-(dist * dist));
      totalW += w;
      r += stop.color[0] * w;
      g += stop.color[1] * w;
      b += stop.color[2] * w;
    }
    const off = i * 3;
    colorLUT[off] = Math.round(r / totalW);
    colorLUT[off + 1] = Math.round(g / totalW);
    colorLUT[off + 2] = Math.round(b / totalW);
  }
};

// 主题变化时重建颜色 LUT + 重绘背景
watch(palette, () => {
  rebuildColorLUT();
  bgDirty = true;
});

// ─── 格子 ───
interface LitCell {
  col: number;
  row: number;
  opacity: number;
  alive: boolean;
}

// ─── 有机体 ───
interface Organism {
  cells: Set<string>;
  cellData: Map<string, LitCell>;
  dirCol: number;
  dirRow: number;
  stepTimer: number;
  stepInterval: number;
  stepsUntilTurn: number;
  maxCells: number;
  alive: boolean;
}

// ─── 状态 ───
let organisms: Organism[] = [];
let gridCols = 0;
let gridRows = 0;
let gridOffsetX = 0;
let gridOffsetY = 0;
let ctx: CanvasRenderingContext2D | null = null;
let animationId: number | null = null;
let canvasWidth = 0;
let canvasHeight = 0;
let mouseX = -9999;
let mouseY = -9999;
let prefersReducedMotion = false;
let lastTimestamp = 0;
let occupied = new Set<string>();
let bgDirty = true; // 背景网格需要重绘
let bgFrameSkip = 0; // 背景帧跳过计数器
const BG_REDRAW_INTERVAL = 3; // 每 3 帧重绘一次背景
let offBg: HTMLCanvasElement | null = null;
let offBgCtx: CanvasRenderingContext2D | null = null;
const cellKey = (c: number, r: number) => `${c},${r}`;
const parseKey = (key: string): [number, number] => {
  const i = key.indexOf(',');
  return [parseInt(key.slice(0, i)), parseInt(key.slice(i + 1))];
};

const WAVE_AMP_SUM = WAVE_AMPS.reduce((a, b) => a + b, 0);

// 简易格子伪噪声，打破纯正弦的规则感
const cellNoise = (col: number, row: number): number => {
  const n = Math.sin(col * 12.9898 + row * 78.233) * 43758.5453;
  return n - Math.floor(n);
};

const waveWeight = (col: number, row: number, t: number): number => {
  let sum = 0;
  for (let i = 0; i < WAVE_FREQS.length; i++) {
    sum +=
      WAVE_AMPS[i]! *
      Math.sin(
        t * WAVE_FREQS[i]! + col * WAVE_SPATIAL_X[i]! + row * WAVE_SPATIAL_Y[i]! + WAVE_PHASES[i]!
      );
  }
  const raw = (sum / WAVE_AMP_SUM + 1) / 2;
  // 叠加格子噪声增加随机性
  const noisy = Math.min(1, Math.max(0, raw + (cellNoise(col, row) - 0.5) * 0.25));
  return Math.pow(noisy, 0.55);
};

// 颜色波浪 — 双波干涉 + 噪声，产生非线性多方向色域
// 两束不同角度的慢波叠加，形成 2D 干涉图样而非单向线性渐变
const colorWeight = (col: number, row: number, t: number): number => {
  const w1 = Math.sin(t * 0.00006 + col * 0.18 + row * 0.05 + WAVE_PHASES[0]! * 2);
  const w2 = Math.sin(t * 0.00004 - col * 0.04 + row * 0.2 + WAVE_PHASES[1]! * 3);
  const noise = (cellNoise(col, row) - 0.5) * 0.12;
  const combined = (w1 + w2) / 2 + noise;
  return Math.min(1, Math.max(0, (combined + 1) / 2));
};

// 高斯交融的 sigma 参数（LUT 构建使用）
const BLEND_SIGMA = 0.18;

// LUT 查表：O(1) 替代 5×exp()
const lutColor = (cw: number): [number, number, number] => {
  const idx = Math.min(Math.round(cw * (COLOR_LUT_SIZE - 1)), COLOR_LUT_SIZE - 1);
  const off = idx * 3;
  return [colorLUT[off]!, colorLUT[off + 1]!, colorLUT[off + 2]!];
};

// ─── 方向 ───
const DIRS: [number, number][] = [
  [1, 0],
  [-1, 0],
  [0, 1],
  [0, -1],
];
const pickDir = (): [number, number] => DIRS[Math.floor(Math.random() * 4)]!;

const pickTurn = (dc: number, dr: number): [number, number] => {
  const r = Math.random();
  if (r < 0.35) return [dr, -dc];
  if (r < 0.7) return [-dr, dc];
  return [-dc, -dr] as [number, number];
};

// ─── 形状演化 ───
const centroid = (org: Organism): [number, number] => {
  let sc = 0,
    sr = 0;
  for (const key of org.cells) {
    const [c, r] = parseKey(key);
    sc += c;
    sr += r;
  }
  const n = org.cells.size;
  return [sc / n, sr / n];
};

const isRemovable = (org: Organism, candidate: string): boolean => {
  if (org.cells.size <= 1) return false;

  let start = '';
  for (const key of org.cells) {
    if (key !== candidate) {
      start = key;
      break;
    }
  }

  const visited = new Set<string>([start]);
  const stack = [start];
  while (stack.length > 0) {
    const current = stack.pop()!;
    const [col, row] = parseKey(current);
    for (const [dc, dr] of DIRS) {
      const nk = cellKey(col + dc, row + dr);
      if (nk !== candidate && org.cells.has(nk) && !visited.has(nk)) {
        visited.add(nk);
        stack.push(nk);
      }
    }
  }

  return visited.size === org.cells.size - 1;
};

const tryGrow = (org: Organism): void => {
  const boundary = new Set<string>();
  for (const key of org.cells) {
    const [col, row] = parseKey(key);
    for (const [dc, dr] of DIRS) {
      const nc = col + dc;
      const nr = row + dr;
      if (nc < 0 || nc >= gridCols || nr < 0 || nr >= gridRows) continue;
      const nk = cellKey(nc, nr);
      if (!org.cells.has(nk) && !occupied.has(nk)) {
        boundary.add(nk);
      }
    }
  }

  if (boundary.size === 0) return;

  const arr = [...boundary];
  const target = arr[Math.floor(Math.random() * arr.length)]!;
  const [tc, tr] = parseKey(target);

  org.cells.add(target);
  org.cellData.set(target, { col: tc, row: tr, opacity: 0, alive: true });
  occupied.add(target);
};

const tryShrink = (org: Organism): void => {
  if (org.cells.size <= 1) return;

  const removable: string[] = [];
  for (const key of org.cells) {
    if (isRemovable(org, key)) {
      removable.push(key);
    }
  }

  if (removable.length === 0) return;

  const target = removable[Math.floor(Math.random() * removable.length)]!;
  const cell = org.cellData.get(target);
  if (cell) cell.alive = false;
  org.cells.delete(target);
};

const tryMove = (org: Organism): void => {
  const canMoveAll = (dc: number, dr: number): boolean => {
    for (const key of org.cells) {
      const [col, row] = parseKey(key);
      const nc = col + dc;
      const nr = row + dr;
      if (nc < 0 || nc >= gridCols || nr < 0 || nr >= gridRows) return false;
      const nk = cellKey(nc, nr);
      if (!org.cells.has(nk) && occupied.has(nk)) return false;
    }
    return true;
  };

  let dc = org.dirCol;
  let dr = org.dirRow;

  if (!canMoveAll(dc, dr)) {
    const shuffled = [...DIRS].sort(() => Math.random() - 0.5);
    let found = false;
    for (const [tdc, tdr] of shuffled) {
      if (canMoveAll(tdc, tdr)) {
        dc = tdc;
        dr = tdr;
        org.dirCol = tdc;
        org.dirRow = tdr;
        found = true;
        break;
      }
    }
    if (!found) return;
  }

  const newCells = new Set<string>();
  for (const key of org.cells) {
    const [col, row] = parseKey(key);
    newCells.add(cellKey(col + dc, row + dr));
  }

  for (const key of org.cells) {
    if (!newCells.has(key)) {
      const cell = org.cellData.get(key);
      if (cell) cell.alive = false;
    }
  }

  for (const nk of newCells) {
    if (!org.cells.has(nk)) {
      const [nc, nr] = parseKey(nk);
      org.cellData.set(nk, { col: nc, row: nr, opacity: 0, alive: true });
      occupied.add(nk);
    }
  }

  org.cells = newCells;
};

// ─── 杀死有机体并在新位置重生 ───
const killAndRespawn = (org: Organism) => {
  // 标记所有格子死亡（触发淡出）
  for (const cell of org.cellData.values()) {
    cell.alive = false;
  }
  org.cells.clear();
  org.alive = false;

  // 在随机空闲位置重生
  for (let attempt = 0; attempt < 20; attempt++) {
    const col = Math.floor(Math.random() * gridCols);
    const row = Math.floor(Math.random() * gridRows);
    const key = cellKey(col, row);
    if (occupied.has(key)) continue;

    occupied.add(key);
    const [dc, dr] = pickDir();
    org.cells = new Set([key]);
    org.cellData = new Map([[key, { col, row, opacity: 0, alive: true }]]);
    org.alive = true;
    org.dirCol = dc;
    org.dirRow = dr;
    org.stepTimer = 0;
    org.stepInterval = 8000 + Math.random() * 12000;
    org.stepsUntilTurn = 2 + Math.floor(Math.random() * 5);
    org.maxCells = 3 + Math.floor(Math.random() * 3); // 3-5 格
    return;
  }
};

// ─── 前进一步（有机形状演化） ───
const stepOrganism = (org: Organism) => {
  if (!org.alive) return;

  // 转向
  org.stepsUntilTurn--;
  if (org.stepsUntilTurn <= 0) {
    [org.dirCol, org.dirRow] = pickTurn(org.dirCol, org.dirRow);
    org.stepsUntilTurn = 2 + Math.floor(Math.random() * 5);
  }

  // 鼠标微弱吸引
  const [cCol, cRow] = centroid(org);
  const headPx = gridOffsetX + cCol * PITCH;
  const headPy = gridOffsetY + cRow * PITCH;
  const mdx = mouseX - headPx;
  const mdy = mouseY - headPy;
  const mDist = Math.sqrt(mdx * mdx + mdy * mdy);
  if (mDist < STROKE_RADIUS * 1.5 && mDist > 1 && Math.random() < 0.12) {
    if (Math.abs(mdx) > Math.abs(mdy)) {
      org.dirCol = Math.sign(mdx);
      org.dirRow = 0;
    } else {
      org.dirCol = 0;
      org.dirRow = Math.sign(mdy);
    }
  }

  // 随机停留
  if (Math.random() < 0.25) return;

  // 1 格时有概率死亡 → 重生到新位置
  if (org.cells.size <= 1 && Math.random() < 0.15) {
    killAndRespawn(org);
    return;
  }

  // 根据尺寸选择动作
  let growW = 0.4,
    shrinkW = 0.2;
  if (org.cells.size >= org.maxCells) {
    growW = 0;
    shrinkW = 0.4;
  } else if (org.cells.size <= 1) {
    growW = 0.5;
    shrinkW = 0;
  }

  const r = Math.random();
  if (r < growW) {
    tryGrow(org);
  } else if (r < growW + shrinkW) {
    tryShrink(org);
  } else {
    tryMove(org);
  }
};

// ─── 初始化 ───
const initWorld = (w: number, h: number) => {
  gridCols = Math.ceil(w / PITCH) + 2;
  gridRows = Math.ceil(h / PITCH) + 2;
  gridOffsetX = -PITCH * 0.5;
  gridOffsetY = -PITCH * 0.5;

  const count = Math.min(180, Math.max(24, Math.floor((w * h) / 2000)));
  organisms = [];
  occupied = new Set();

  const sectorCols = Math.max(1, Math.ceil(Math.sqrt((count * gridCols) / gridRows)));
  const sectorRows = Math.max(1, Math.ceil(count / sectorCols));
  const colPerSector = gridCols / sectorCols;
  const rowPerSector = gridRows / sectorRows;

  let placed = 0;
  for (let sr = 0; sr < sectorRows && placed < count; sr++) {
    for (let sc = 0; sc < sectorCols && placed < count; sc++) {
      const col = Math.max(
        0,
        Math.min(gridCols - 1, Math.floor(sc * colPerSector + Math.random() * colPerSector))
      );
      const row = Math.max(
        0,
        Math.min(gridRows - 1, Math.floor(sr * rowPerSector + Math.random() * rowPerSector))
      );

      const key = cellKey(col, row);
      if (occupied.has(key)) continue;

      occupied.add(key);
      const [dc, dr] = pickDir();

      organisms.push({
        cells: new Set([key]),
        cellData: new Map([[key, { col, row, opacity: 0, alive: true }]]),
        dirCol: dc,
        dirRow: dr,
        stepTimer: Math.random() * 8000,
        stepInterval: 8000 + Math.random() * 12000,
        stepsUntilTurn: 2 + Math.floor(Math.random() * 5),
        maxCells: 3 + Math.floor(Math.random() * 3),
        alive: true,
      });
      placed++;
    }
  }
};

// ─── 绘制辅助 ───
function fillRoundedRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number
) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + w - r, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + r);
  ctx.lineTo(x + w, y + h - r);
  ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
  ctx.lineTo(x + r, y + h);
  ctx.quadraticCurveTo(x, y + h, x, y + h - r);
  ctx.lineTo(x, y + r);
  ctx.quadraticCurveTo(x, y, x + r, y);
  ctx.closePath();
  ctx.fill();
}

function strokeRoundedRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number
) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + w - r, y);
  ctx.quadraticCurveTo(x + w, y, x + w, y + r);
  ctx.lineTo(x + w, y + h - r);
  ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h);
  ctx.lineTo(x + r, y + h);
  ctx.quadraticCurveTo(x, y + h, x, y + h - r);
  ctx.lineTo(x, y + r);
  ctx.quadraticCurveTo(x, y, x + r, y);
  ctx.closePath();
  ctx.stroke();
}

// ─── 绘制背景网格到离屏 canvas ───
const renderBackground = (timestamp: number) => {
  if (!offBgCtx) return;
  const p = palette.value;

  offBgCtx.clearRect(0, 0, canvasWidth, canvasHeight);
  offBgCtx.fillStyle = p.bg;
  offBgCtx.fillRect(0, 0, canvasWidth, canvasHeight);

  // 矿物质感参数：凸面高光强度 + 边缘暗化比
  const hlBoost = p.isDark ? 8 : 12;
  const edgeDim = p.isDark ? 0.88 : 0.9;

  for (let ri = 0; ri < gridRows; ri++) {
    for (let ci = 0; ci < gridCols; ci++) {
      const ww = waveWeight(ci, ri, timestamp);
      const sz = MIN_CELL_SIZE + (CELL - MIN_CELL_SIZE) * ww;
      const off = (CELL - sz) / 2;
      const cw = colorWeight(ci, ri, timestamp);
      const [lr, lg, lb] = lutColor(cw);
      // 中性色高可见 + 跳色区增强，让补色格子不被透明度吞没
      const bandAlpha = p.bgAlpha * (0.55 + (1 - cw) * 0.45);
      const cornerR = Math.min(sz / 2, CORNER * (2 - ww));
      const cx = gridOffsetX + ci * PITCH + off;
      const cy = gridOffsetY + ri * PITCH + off;

      // 径向渐变模拟凸面光泽 — 过小格子跳过（视觉不可见）
      if (sz < 18) {
        offBgCtx.fillStyle = `rgba(${lr}, ${lg}, ${lb}, ${bandAlpha})`;
      } else {
        const grad = offBgCtx.createRadialGradient(
          cx + sz * 0.38,
          cy + sz * 0.38,
          sz * 0.05,
          cx + sz * 0.5,
          cy + sz * 0.5,
          sz * 0.6
        );
        grad.addColorStop(
          0,
          `rgba(${Math.min(255, lr + hlBoost)}, ${Math.min(255, lg + hlBoost)}, ${Math.min(255, lb + hlBoost)}, ${bandAlpha})`
        );
        grad.addColorStop(1, `rgba(${lr}, ${lg}, ${lb}, ${bandAlpha * edgeDim})`);
        offBgCtx.fillStyle = grad;
      }
      fillRoundedRect(offBgCtx, cx, cy, sz, sz, cornerR);
    }
  }

  // 全局漫射光 — 模拟左上方光源，增加空间纵深
  const lightGrad = offBgCtx.createRadialGradient(
    canvasWidth * 0.25,
    canvasHeight * 0.2,
    0,
    canvasWidth * 0.5,
    canvasHeight * 0.5,
    Math.max(canvasWidth, canvasHeight) * 0.65
  );
  lightGrad.addColorStop(0, p.isDark ? 'rgba(255, 255, 255, 0.02)' : 'rgba(255, 255, 255, 0.035)');
  lightGrad.addColorStop(1, p.isDark ? 'rgba(0, 0, 0, 0.015)' : 'rgba(0, 0, 0, 0.01)');
  offBgCtx.fillStyle = lightGrad;
  offBgCtx.fillRect(0, 0, canvasWidth, canvasHeight);

  bgDirty = false;
};

// ─── 动画 ───
const animate = (timestamp: number) => {
  if (!ctx) return;
  const dt = lastTimestamp ? timestamp - lastTimestamp : 16;
  lastTimestamp = timestamp;

  const p = palette.value;

  // ── 1. 背景网格（离屏缓存，每 N 帧更新） ──
  bgFrameSkip++;
  if (bgDirty || bgFrameSkip >= BG_REDRAW_INTERVAL) {
    renderBackground(timestamp);
    bgFrameSkip = 0;
  }
  // 将离屏 canvas 像素对像素绘制到主 canvas
  // 需要先重置 transform 避免 DPR 双重缩放
  if (offBg) {
    ctx.save();
    ctx.setTransform(1, 0, 0, 1, 0, 0);
    ctx.drawImage(offBg, 0, 0);
    ctx.restore();
  }

  const reduced = prefersReducedMotion;

  // ── 2. 更新 & 绘制有机体 ──
  const fadeSpeed = 0.01;
  const cellAlpha = p.cellAlpha;
  const breathCycle = timestamp * 0.0004;
  const breathAmp = 0.05;

  for (const org of organisms) {
    if (!reduced) {
      org.stepTimer += dt;
      if (org.stepTimer >= org.stepInterval) {
        stepOrganism(org);
        org.stepTimer -= org.stepInterval;
      }
    }

    const toDelete: string[] = [];
    for (const [key, cell] of org.cellData) {
      if (cell.alive) {
        if (cell.opacity < cellAlpha) {
          cell.opacity = Math.min(cell.opacity + fadeSpeed * (dt / 16), cellAlpha);
        }
      } else {
        cell.opacity -= fadeSpeed * (dt / 16);
        if (cell.opacity <= 0) {
          occupied.delete(key);
          toDelete.push(key);
          continue;
        }
      }

      const phase = (cell.col * 0.7 + cell.row * 1.1) * 0.5;
      const breath = Math.sin(breathCycle + phase) * breathAmp;

      const ww = waveWeight(cell.col, cell.row, timestamp);
      const cw = colorWeight(cell.col, cell.row, timestamp);
      const sz = MIN_CELL_SIZE + (CELL - MIN_CELL_SIZE) * ww;
      const off = (CELL - sz) / 2;
      const cellX = gridOffsetX + cell.col * PITCH + off;
      const cellY = gridOffsetY + cell.row * PITCH + off;
      const cornerR = Math.min(sz / 2, CORNER * (2 - ww));
      const [cr, cg, cb] = lutColor(cw);
      const clampedOpacity = Math.min(Math.max(cell.opacity + breath, 0), 0.85);

      // 投影层 — 模拟有机体浮于背景表面之上
      const shadowMult = p.isDark ? 0.12 : 0.2;
      ctx.fillStyle = `rgba(0, 0, 0, ${clampedOpacity * shadowMult})`;
      fillRoundedRect(ctx, cellX + 0.8, cellY + 0.8, sz, sz, cornerR);

      // 主体 — 径向渐变模拟凸面高光
      const orgHl = p.isDark ? 14 : 18;
      if (sz < 18) {
        ctx.fillStyle = `rgba(${cr}, ${cg}, ${cb}, ${clampedOpacity})`;
      } else {
        const grad = ctx.createRadialGradient(
          cellX + sz * 0.35,
          cellY + sz * 0.35,
          0,
          cellX + sz * 0.5,
          cellY + sz * 0.5,
          sz * 0.58
        );
        grad.addColorStop(
          0,
          `rgba(${Math.min(255, cr + orgHl)}, ${Math.min(255, cg + orgHl)}, ${Math.min(255, cb + orgHl)}, ${clampedOpacity})`
        );
        grad.addColorStop(1, `rgba(${cr}, ${cg}, ${cb}, ${clampedOpacity})`);
        ctx.fillStyle = grad;
      }
      fillRoundedRect(ctx, cellX, cellY, sz, sz, cornerR);
    }
    for (const key of toDelete) {
      org.cellData.delete(key);
    }
  }

  // ── 3. 鼠标附近描边 ──
  if (mouseX > -9000) {
    const cR = Math.ceil(STROKE_RADIUS / PITCH) + 1;
    const mouseCol = Math.round((mouseX - gridOffsetX) / PITCH);
    const mouseRow = Math.round((mouseY - gridOffsetY) / PITCH);
    ctx.lineWidth = 2;
    for (let dri = -cR; dri <= cR; dri++) {
      for (let dci = -cR; dci <= cR; dci++) {
        const col = mouseCol + dci;
        const row = mouseRow + dri;
        if (col < 0 || col >= gridCols || row < 0 || row >= gridRows) continue;

        const px = gridOffsetX + col * PITCH;
        const py = gridOffsetY + row * PITCH;
        const cx = px + CELL / 2;
        const cy = py + CELL / 2;
        const dist = Math.sqrt((cx - mouseX) ** 2 + (cy - mouseY) ** 2);

        if (dist < STROKE_RADIUS) {
          const strength = Math.pow(1 - dist / STROKE_RADIUS, 2.5);
          const ww = waveWeight(col, row, timestamp);
          const cw = colorWeight(col, row, timestamp);
          const sz = MIN_CELL_SIZE + (CELL - MIN_CELL_SIZE) * ww;
          const off = (CELL - sz) / 2;
          const cornerR = Math.min(sz / 2, CORNER * (2 - ww));
          const [sr, sg, sb] = lutColor(cw);
          const strokeOpacity = Math.min(p.strokeAlpha * strength, 1.0);
          ctx.strokeStyle = `rgba(${sr}, ${sg}, ${sb}, ${strokeOpacity})`;
          strokeRoundedRect(ctx, px + off, py + off, sz, sz, cornerR);
        }
      }
    }
  }

  // reduced-motion 时只渲染一帧静态图，不持续循环
  if (prefersReducedMotion) {
    animationId = null;
    return;
  }
  animationId = requestAnimationFrame(animate);
};

// ─── Canvas ───
const resizeCanvas = () => {
  if (!canvas.value || !container.value) return;
  const rect = container.value.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;

  canvas.value.width = rect.width * dpr;
  canvas.value.height = rect.height * dpr;
  canvas.value.style.width = rect.width + 'px';
  canvas.value.style.height = rect.height + 'px';

  ctx = canvas.value.getContext('2d');
  if (ctx) ctx.scale(dpr, dpr);

  canvasWidth = rect.width;
  canvasHeight = rect.height;

  // 同步离屏 canvas 尺寸
  offBg = document.createElement('canvas');
  offBg.width = canvas.value.width;
  offBg.height = canvas.value.height;
  offBgCtx = offBg.getContext('2d');
  if (offBgCtx) offBgCtx.scale(dpr, dpr);

  bgDirty = true;
  rebuildColorLUT();
  initWorld(canvasWidth, canvasHeight);

  // resize 后在 reduced 模式下重绘一帧
  if (prefersReducedMotion && !animationId) {
    requestAnimationFrame(animate);
  }
};

// ─── 鼠标 ───
const onMouseMove = (e: MouseEvent) => {
  if (!container.value) return;
  const rect = container.value.getBoundingClientRect();
  mouseX = e.clientX - rect.left;
  mouseY = e.clientY - rect.top;
};
const onMouseLeave = () => {
  mouseX = -9999;
  mouseY = -9999;
};

// ─── 生命周期 ───
onMounted(() => {
  prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  if (canvas.value) {
    ctx = canvas.value.getContext('2d');
    resizeCanvas();
    animationId = requestAnimationFrame(animate);
  }

  window.addEventListener('resize', resizeCanvas);
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mouseleave', onMouseLeave);

  window.matchMedia('(prefers-reduced-motion: reduce)').addEventListener('change', (e) => {
    prefersReducedMotion = e.matches;
    if (e.matches) {
      // 进入 reduced：渲染一帧静态图后停止
      bgDirty = true;
      requestAnimationFrame(animate);
    } else if (!animationId) {
      // 恢复动画
      lastTimestamp = 0;
      animationId = requestAnimationFrame(animate);
    }
  });
});

onUnmounted(() => {
  if (animationId) cancelAnimationFrame(animationId);
  window.removeEventListener('resize', resizeCanvas);
  document.removeEventListener('mousemove', onMouseMove);
  document.removeEventListener('mouseleave', onMouseLeave);
});
</script>

<style scoped>
.dynamic-background {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  overflow: hidden;
  z-index: 0;
}

.background-canvas {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}
</style>
