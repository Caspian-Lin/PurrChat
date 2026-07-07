import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import DynamicBackground from '../components/DynamicBackground.vue';

describe('DynamicBackground Component', () => {
  let wrapper: ReturnType<typeof mount>;

  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();

    // Mock getContext to return a minimal mock canvas context
    const mockCtx = {
      fillRect: vi.fn(),
      clearRect: vi.fn(),
      createRadialGradient: vi.fn(() => ({
        addColorStop: vi.fn(),
      })),
      save: vi.fn(),
      restore: vi.fn(),
      setTransform: vi.fn(),
      scale: vi.fn(),
      drawImage: vi.fn(),
      fill: vi.fn(),
      stroke: vi.fn(),
      beginPath: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      quadraticCurveTo: vi.fn(),
      closePath: vi.fn(),
    };
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(
      mockCtx as unknown as CanvasRenderingContext2D
    );

    // Mock getBoundingClientRect for container sizing
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      width: 1000,
      height: 800,
      left: 0,
      top: 0,
      right: 1000,
      bottom: 800,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect);

    // Mock devicePixelRatio
    vi.spyOn(window, 'devicePixelRatio', 'get').mockReturnValue(1);

    // Mock matchMedia
    vi.spyOn(window, 'matchMedia').mockReturnValue({
      matches: true,
      media: '',
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    });
  });

  afterEach(() => {
    if (wrapper) wrapper.unmount();
    vi.restoreAllMocks();
  });

  it('should render the container element with canvas', () => {
    wrapper = mount(DynamicBackground);
    const container = wrapper.find('.dynamic-background');
    expect(container.exists()).toBe(true);
    expect(container.find('canvas').exists()).toBe(true);
  });

  it('should render canvas with background-canvas class', () => {
    wrapper = mount(DynamicBackground);
    const canvas = wrapper.find('.background-canvas');
    expect(canvas.exists()).toBe(true);
    expect(canvas.element.tagName.toLowerCase()).toBe('canvas');
  });

  it('should register resize event listener on mount', () => {
    const resizeSpy = vi.spyOn(window, 'addEventListener');

    wrapper = mount(DynamicBackground);

    expect(resizeSpy).toHaveBeenCalledWith('resize', expect.any(Function));
  });

  it('should clean up event listeners on unmount', () => {
    const resizeSpy = vi.spyOn(window, 'removeEventListener');

    wrapper = mount(DynamicBackground);
    wrapper.unmount();

    expect(resizeSpy).toHaveBeenCalledWith('resize', expect.any(Function));
  });
});
