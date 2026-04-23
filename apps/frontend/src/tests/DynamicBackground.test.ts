import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import DynamicBackground from '../components/DynamicBackground.vue';

describe('DynamicBackground Component', () => {
  let wrapper: ReturnType<typeof mount>;

  beforeEach(() => {
    setActivePinia(createPinia());
  });

  afterEach(() => {
    if (wrapper) wrapper.unmount();
  });

  it('should render the container element with aria-hidden', () => {
    wrapper = mount(DynamicBackground);
    const el = wrapper.find('.dynamic-bg');
    expect(el.exists()).toBe(true);
    expect(el.attributes('aria-hidden')).toBe('true');
  });

  it('should render the tinted surface layer', () => {
    wrapper = mount(DynamicBackground);
    expect(wrapper.find('.surface').exists()).toBe(true);
  });

  it('should render the SVG dot grid with pattern definition', () => {
    wrapper = mount(DynamicBackground);
    const svg = wrapper.find('.dot-grid');
    expect(svg.exists()).toBe(true);
    expect(svg.element.tagName.toLowerCase()).toBe('svg');
    expect(svg.find('pattern').exists()).toBe(true);
    expect(svg.find('circle').exists()).toBe(true);
  });

  it('should render five cutout shapes with varied shape classes', () => {
    wrapper = mount(DynamicBackground);
    const cutouts = wrapper.findAll('.cutout');
    expect(cutouts.length).toBe(5);
    // DOM order follows parallax layer grouping: deep(1,3), mid(2,5), near(4)
    const classSet = new Set(cutouts.flatMap((c) => c.classes()));
    for (let i = 1; i <= 5; i++) {
      expect(classSet.has(`cutout-${i}`)).toBe(true);
    }
  });

  it('should render three parallax depth layers', () => {
    wrapper = mount(DynamicBackground);
    const layers = wrapper.findAll('.parallax');
    expect(layers.length).toBe(3);
    expect(layers[0].classes()).toContain('parallax--deep');
    expect(layers[1].classes()).toContain('parallax--mid');
    expect(layers[2].classes()).toContain('parallax--near');
  });

  it('should apply theme-reactive CSS custom properties', () => {
    wrapper = mount(DynamicBackground);
    const style = wrapper.find('.dynamic-bg').attributes('style');
    expect(style).toContain('--surface-color');
    expect(style).toContain('--grid-color');
    expect(style).toContain('--mx');
    expect(style).toContain('--my');
  });

  it('should use rgba format for surface and grid colors', () => {
    wrapper = mount(DynamicBackground);
    const style = wrapper.find('.dynamic-bg').attributes('style') ?? '';
    const surfaceMatch = style.match(/--surface-color:\s*rgba\([^)]+\)/);
    expect(surfaceMatch).not.toBeNull();
    const gridMatch = style.match(/--grid-color:\s*rgba\([^)]+\)/);
    expect(gridMatch).not.toBeNull();
  });

  it('should initialize parallax vars at zero', () => {
    wrapper = mount(DynamicBackground);
    const style = wrapper.find('.dynamic-bg').attributes('style') ?? '';
    expect(style).toContain('--mx: 0');
    expect(style).toContain('--my: 0');
  });

  it('should update parallax vars in response to mouse movement', async () => {
    wrapper = mount(DynamicBackground);
    const container = wrapper.find('.dynamic-bg');
    const el = container.element as HTMLElement;

    // Mock getBoundingClientRect for predictable calculations
    const mockRect = {
      width: 1000, height: 800,
      left: 0, top: 0, right: 1000, bottom: 800,
      x: 0, y: 0, toJSON: () => ({}),
    };
    vi.spyOn(el, 'getBoundingClientRect').mockReturnValue(mockRect as DOMRect);

    // Initially parallax is zero
    let style = container.attributes('style') ?? '';
    expect(style).toContain('--mx: 0');

    // Move mouse — mx/my should change from zero
    await container.trigger('mousemove', { clientX: 750, clientY: 600 });
    style = container.attributes('style') ?? '';

    // Verify values changed (no longer zero)
    const mxMatch = style.match(/--mx:\s*([^;]+)/);
    const myMatch = style.match(/--my:\s*([^;]+)/);
    expect(mxMatch).not.toBeNull();
    expect(myMatch).not.toBeNull();
    expect(parseFloat(mxMatch![1])).not.toBe(0);
    expect(parseFloat(myMatch![1])).not.toBe(0);
  });

  it('should reset parallax vars to zero on mouse leave', async () => {
    wrapper = mount(DynamicBackground);
    const container = wrapper.find('.dynamic-bg');
    const el = container.element as HTMLElement;

    vi.spyOn(el, 'getBoundingClientRect').mockReturnValue({
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

    // Move mouse away from center
    await container.trigger('mousemove', { clientX: 800, clientY: 200 });

    // Leave — should reset to 0
    await container.trigger('mouseleave');

    const style = container.attributes('style') ?? '';
    expect(style).toContain('--mx: 0');
    expect(style).toContain('--my: 0');
  });

  it('should not add document-level event listeners (uses element-level handlers)', () => {
    const docSpy = vi.spyOn(document, 'addEventListener');
    const winSpy = vi.spyOn(window, 'addEventListener');

    wrapper = mount(DynamicBackground);

    expect(docSpy).not.toHaveBeenCalledWith('mousemove', expect.any(Function));
    expect(docSpy).not.toHaveBeenCalledWith('mouseleave', expect.any(Function));

    wrapper.unmount();

    docSpy.mockRestore();
    winSpy.mockRestore();
  });
});
