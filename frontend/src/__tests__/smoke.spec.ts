import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import HelloWorld from './fixtures/HelloWorld.vue';

describe('smoke test', () => {
  it('renders a Vue component with props', () => {
    const wrapper = mount(HelloWorld, {
      props: { msg: 'Hello Vitest!' },
    });
    expect(wrapper.text()).toContain('Hello Vitest!');
  });
});
