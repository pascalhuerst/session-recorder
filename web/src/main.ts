import { createApp } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import App from './App.vue';
import { createPinia } from 'pinia';
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome';
import RecordersView from './views/recorders/RecordersView.vue';
import RecordersIndexView from './views/recorders/views/index/RecordersIndexView.vue';
import RecorderView from './views/recorders/views/_recorderId/RecorderView.vue';
import SessionsView from './views/recorders/views/_recorderId/views/sessions/SessionsView.vue';
import SessionsIndexView from './views/recorders/views/_recorderId/views/sessions/views/index/SessionsIndexView.vue';
import { setup } from '@session-recorder/session-waveform';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/recorders',
      component: RecordersView,
      children: [
        {
          path: '',
          component: RecordersIndexView,
        },
        {
          path: ':recorderId',
          component: RecorderView,
          children: [
            {
              path: '',
              component: RecordersIndexView,
            },
            {
              path: 'sessions',
              component: SessionsView,
              children: [
                {
                  path: '',
                  component: SessionsIndexView,
                },
              ],
            },
          ],
        },
      ],
    },

    { path: '/:pathMatch(.*)*', redirect: '/recorders' },
  ],
});

const app = createApp(App);

app.component('font-awesome-icon', FontAwesomeIcon);

app.use(createPinia());
app.use(router);

setup();

app.mount('#app');
