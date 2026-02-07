<template>
  <div class="dynamic-background" ref="container">
    <canvas ref="canvas" class="background-canvas"></canvas>
    <div class="background-overlay"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useThemeStore } from '../stores/theme';
import { themeColors } from '../config/theme';

const themeStore = useThemeStore();
const canvas = ref<HTMLCanvasElement | null>(null);
const container = ref<HTMLDivElement | null>(null);

// 颜色转换函数
const hexToHSL = (hex: string): { h: number; s: number; l: number } => {
  let r = 0,
    g = 0,
    b = 0;
  if (hex.length === 4) {
    r = parseInt(hex[1]! + hex[1]!, 16);
    g = parseInt(hex[2]! + hex[2]!, 16);
    b = parseInt(hex[3]! + hex[3]!, 16);
  } else if (hex.length === 7) {
    r = parseInt(hex.slice(1, 3), 16);
    g = parseInt(hex.slice(3, 5), 16);
    b = parseInt(hex.slice(5, 7), 16);
  }
  r /= 255;
  g /= 255;
  b /= 255;
  const max = Math.max(r, g, b),
    min = Math.min(r, g, b);
  let h = 0,
    s = 0,
    l = (max + min) / 2;
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

const hslToHex = (h: number, s: number, l: number): string => {
  h /= 360;
  s /= 100;
  l /= 100;
  let r, g, b;
  if (s === 0) {
    r = g = b = l;
  } else {
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
    r = hue2rgb(p, q, h + 1 / 3);
    g = hue2rgb(p, q, h);
    b = hue2rgb(p, q, h - 1 / 3);
  }
  const toHex = (x: number) => {
    const hex = Math.round(x * 255).toString(16);
    return hex.length === 1 ? '0' + hex : hex;
  };
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
};

// 生成次主题色
const generateSecondaryColors = (primaryHex: string) => {
  const hsl = hexToHSL(primaryHex);

  // 次主题色1: 偏移30度色相
  const color1 = hslToHex((hsl.h + 30) % 360, hsl.s, hsl.l);

  // 次主题色2: 增加30亮度
  const color2 = hslToHex(hsl.h, hsl.s, Math.min(hsl.l + 30, 100));

  // 次主题色3: 偏移50度色相
  const color3 = hslToHex((hsl.h + 50) % 360, hsl.s, hsl.l);

  return [primaryHex, color1, color2, color3];
};

// 获取当前主题色
const currentColors = computed(() => {
  const primaryColor = themeColors[themeStore.color].primary;
  return generateSecondaryColors(primaryColor);
});

// 几何图形类
class GeometricShape {
  x: number;
  y: number;
  size: number;
  speedX: number;
  speedY: number;
  rotation: number;
  rotationSpeed: number;
  colorIndex: number;
  type: 'circle' | 'triangle' | 'square' | 'hexagon';
  opacity: number;
  targetOpacity: number;

  constructor(canvasWidth: number, canvasHeight: number, colors: string[]) {
    this.x = Math.random() * canvasWidth;
    this.y = Math.random() * canvasHeight;
    this.size = Math.random() * 200 + 20;
    this.speedX = (Math.random() - 0.5) * 0.03;
    this.speedY = (Math.random() - 0.5) * 0.03;
    this.rotation = Math.random() * Math.PI * 0.02;
    this.rotationSpeed = (Math.random() - 0.5) * 0.002;
    this.colorIndex = Math.floor(Math.random() * colors.length);
    this.type = ['circle', 'triangle', 'square', 'hexagon'][Math.floor(Math.random() * 4)] as any;
    // this.opacity = Math.random() * 0.3 + 0.1;
    this.opacity = 0.3;
    this.targetOpacity = this.opacity;
  }

  update(canvasWidth: number, canvasHeight: number) {
    this.x += this.speedX;
    this.y += this.speedY;
    this.rotation += this.rotationSpeed;

    // 边界检测
    if (this.x < -this.size) this.x = canvasWidth + this.size;
    if (this.x > canvasWidth + this.size) this.x = -this.size;
    if (this.y < -this.size) this.y = canvasHeight + this.size;
    if (this.y > canvasHeight + this.size) this.y = -this.size;

    // // 透明度渐变
    // this.opacity += (this.targetOpacity - this.opacity) * 0.02;
    // if (Math.random() < 0.005) {
    //   this.targetOpacity = Math.random() * 0.3 + 0.1;
    // }
  }

  draw(ctx: CanvasRenderingContext2D, colors: string[]) {
    ctx.save();
    ctx.translate(this.x, this.y);
    ctx.rotate(this.rotation);
    ctx.globalAlpha = this.opacity;
    ctx.fillStyle = colors[this.colorIndex] ?? colors[0];

    switch (this.type) {
      case 'circle':
        ctx.beginPath();
        ctx.arc(0, 0, this.size / 2, 0, Math.PI * 2);
        ctx.fill();
        break;
      case 'triangle':
        ctx.beginPath();
        ctx.moveTo(0, -this.size / 2);
        ctx.lineTo(this.size / 2, this.size / 2);
        ctx.lineTo(-this.size / 2, this.size / 2);
        ctx.closePath();
        ctx.fill();
        break;
      case 'square':
        ctx.fillRect(-this.size / 2, -this.size / 2, this.size, this.size);
        break;
      case 'hexagon':
        ctx.beginPath();
        for (let i = 0; i < 6; i++) {
          const angle = (Math.PI / 3) * i;
          const x = (Math.cos(angle) * this.size) / 2;
          const y = (Math.sin(angle) * this.size) / 2;
          if (i === 0) ctx.moveTo(x, y);
          else ctx.lineTo(x, y);
        }
        ctx.closePath();
        ctx.fill();
        break;
    }

    ctx.restore();
  }
}

// 连接线类
class ConnectionLine {
  shape1: GeometricShape;
  shape2: GeometricShape;

  constructor(shape1: GeometricShape, shape2: GeometricShape) {
    this.shape1 = shape1;
    this.shape2 = shape2;
  }

  draw(ctx: CanvasRenderingContext2D, colors: string[]) {
    const dx = this.shape2.x - this.shape1.x;
    const dy = this.shape2.y - this.shape1.y;
    const distance = Math.sqrt(dx * dx + dy * dy);

    if (distance < 200) {
      ctx.save();
      ctx.beginPath();
      ctx.moveTo(this.shape1.x, this.shape1.y);
      ctx.lineTo(this.shape2.x, this.shape2.y);
      ctx.strokeStyle = colors[0] || '#000000';
      ctx.globalAlpha = (1 - distance / 200) * 0.2;
      ctx.lineWidth = 1;
      ctx.stroke();
      ctx.restore();
    }
  }
}

let shapes: GeometricShape[] = [];
let animationId: number | null = null;
let ctx: CanvasRenderingContext2D | null = null;
let canvasWidth = 0;
let canvasHeight = 0;

const initShapes = (width: number, height: number) => {
  shapes = [];
  const numShapes = Math.floor((width * height) / 30000);
  for (let i = 0; i < numShapes; i++) {
    shapes.push(new GeometricShape(width, height, currentColors.value));
  }
};

const animate = () => {
  if (!canvas.value || !ctx) return;

  const width = canvasWidth;
  const height = canvasHeight;

  ctx.clearRect(0, 0, width, height);

  // 绘制渐变背景
  const gradient = ctx.createLinearGradient(0, 0, width, height);
  gradient.addColorStop(0, currentColors.value[0] + '20');
  gradient.addColorStop(0.5, currentColors.value[1] + '15');
  gradient.addColorStop(1, currentColors.value[2] + '20');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, width, height);

  // 更新和绘制形状
  shapes.forEach((shape) => {
    shape.update(width, height);
    shape.draw(ctx!, currentColors.value);
  });

  // 绘制连接线
  for (let i = 0; i < shapes.length; i++) {
    for (let j = i + 1; j < shapes.length; j++) {
      const shape1 = shapes[i];
      const shape2 = shapes[j];
      if (shape1 && shape2) {
        const line = new ConnectionLine(shape1, shape2);
        line.draw(ctx!, currentColors.value);
      }
    }
  }

  animationId = requestAnimationFrame(animate);
};

const resizeCanvas = () => {
  if (!canvas.value || !container.value) return;

  const rect = container.value.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;

  // 设置 canvas 的实际像素尺寸
  canvas.value.width = rect.width * dpr;
  canvas.value.height = rect.height * dpr;

  // 设置 canvas 的 CSS 尺寸
  canvas.value.style.width = rect.width + 'px';
  canvas.value.style.height = rect.height + 'px';

  // 获取 context 并缩放以适应设备像素比
  ctx = canvas.value.getContext('2d');
  if (ctx) {
    ctx.scale(dpr, dpr);
  }

  // 保存 CSS 尺寸用于动画
  canvasWidth = rect.width;
  canvasHeight = rect.height;

  initShapes(canvasWidth, canvasHeight);
};

onMounted(() => {
  if (canvas.value) {
    ctx = canvas.value.getContext('2d');
    resizeCanvas();
    animate();
  }

  window.addEventListener('resize', resizeCanvas);
});

onUnmounted(() => {
  if (animationId) {
    cancelAnimationFrame(animationId);
  }
  window.removeEventListener('resize', resizeCanvas);
});

// 监听主题变化
watch(
  () => themeStore.color,
  () => {
    shapes.forEach((shape) => {
      shape.colorIndex = Math.floor(Math.random() * currentColors.value.length);
    });
  }
);
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
  image-rendering: -webkit-optimize-contrast;
  image-rendering: crisp-edges;
  image-rendering: pixelated;
}

.background-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, rgba(0, 0, 0, 0.1) 0%, rgba(0, 0, 0, 0.05) 100%);
  pointer-events: none;
}
</style>
