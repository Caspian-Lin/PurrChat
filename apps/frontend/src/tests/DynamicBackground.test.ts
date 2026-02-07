import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import DynamicBackground from '../components/DynamicBackground.vue';

describe('DynamicBackground Component', () => {
  let wrapper: ReturnType<typeof mount>;

  beforeEach(() => {
    // Mock canvas context
    const mockCanvasContext = {
      clearRect: vi.fn(),
      fillRect: vi.fn(),
      createLinearGradient: vi.fn(() => ({
        addColorStop: vi.fn(),
      })),
      save: vi.fn(),
      restore: vi.fn(),
      translate: vi.fn(),
      rotate: vi.fn(),
      scale: vi.fn(),
      beginPath: vi.fn(),
      arc: vi.fn(),
      fill: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      closePath: vi.fn(),
      stroke: vi.fn(),
    };

    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(mockCanvasContext as any);

    // Mock getBoundingClientRect
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue({
      width: 800,
      height: 600,
      top: 0,
      left: 0,
      right: 800,
      bottom: 600,
      x: 0,
      y: 0,
      toJSON: vi.fn(),
    });

    // Mock devicePixelRatio
    Object.defineProperty(window, 'devicePixelRatio', {
      writable: true,
      configurable: true,
      value: 1,
    });
  });

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount();
    }
  });

  it('should render correctly', () => {
    wrapper = mount(DynamicBackground);
    expect(wrapper.find('.dynamic-background').exists()).toBe(true);
    expect(wrapper.find('.background-canvas').exists()).toBe(true);
    expect(wrapper.find('.background-overlay').exists()).toBe(true);
  });

  it('should have correct CSS classes', () => {
    wrapper = mount(DynamicBackground);
    const bg = wrapper.find('.dynamic-background');
    expect(bg.classes()).toContain('dynamic-background');
  });

  it('should create canvas element', () => {
    wrapper = mount(DynamicBackground);
    const canvas = wrapper.find('.background-canvas');
    expect(canvas.exists()).toBe(true);
  });

  it('should create overlay element', () => {
    wrapper = mount(DynamicBackground);
    const overlay = wrapper.find('.background-overlay');
    expect(overlay.exists()).toBe(true);
  });
});
