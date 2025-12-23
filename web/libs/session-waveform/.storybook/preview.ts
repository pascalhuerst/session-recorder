import type { Preview } from '@storybook/vue3';
import { setup as setupLib } from '../src/setup';
import { createPinia, setActivePinia } from 'pinia';
import './styles.css';

setupLib();

const preview: Preview = {
  decorators: [
    (story) => {
      const pinia = createPinia();
      setActivePinia(pinia);
      return story();
    },
  ],
};

export default preview;
