import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import { describe, expect, it } from 'vitest';
import BotPermissionReview from '../components/home/panel/bots/BotPermissionReview.vue';

describe('BotPermissionReview', () => {
  it('keeps core capabilities enabled and requires acknowledgement for selected sensitive access', async () => {
    const wrapper = mount(BotPermissionReview, {
      props: {
        botName: '翻译助手',
        targetLabel: '项目群',
        requestedCapabilities: ['messages:read_trigger', 'messages:send', 'network:external'],
      },
    });

    const checkboxes = wrapper.findAll<HTMLInputElement>('input[type="checkbox"]');
    expect(checkboxes).toHaveLength(4);
    expect(checkboxes[0].element.disabled).toBe(true);
    expect(checkboxes[1].element.disabled).toBe(true);
    expect(wrapper.emitted('change')?.at(-1)).toEqual([
      ['messages:read_trigger', 'messages:send', 'network:external'],
      false,
    ]);

    await checkboxes[3].setValue(true);
    await nextTick();
    expect(wrapper.emitted('change')?.at(-1)).toEqual([
      ['messages:read_trigger', 'messages:send', 'network:external'],
      true,
    ]);
  });

  it('allows optional capabilities to be removed without sensitive acknowledgement', async () => {
    const wrapper = mount(BotPermissionReview, {
      props: {
        botName: '翻译助手',
        targetLabel: '我的私聊',
        requestedCapabilities: ['messages:read_trigger', 'messages:send', 'network:external'],
      },
    });

    const externalCapability = wrapper.findAll<HTMLInputElement>('input[type="checkbox"]')[2];
    await externalCapability.setValue(false);
    await nextTick();

    expect(wrapper.text()).not.toContain('我了解：');
    expect(wrapper.emitted('change')?.at(-1)).toEqual([
      ['messages:read_trigger', 'messages:send'],
      true,
    ]);
  });
});
