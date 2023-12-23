import Vue from 'vue';
import VueRouter from 'vue-router';
import App from './App.vue';

import OpenSessions from './components/OpenSessions.vue';
import SessionDetail from './components/SessionDetail.vue';

import store from './store';

import Buefy from 'buefy';
import 'buefy/dist/buefy.css';
import './assets/css/domestic-recorder.scss';


Vue.use(VueRouter);
Vue.use(Buefy);
Vue.config.productionTip = false;

const router = new VueRouter({
  mode: 'history',

  routes: [
    { path: '/', component: OpenSessions },
    { path: '/detail/:recorderID/:sessionID', component: SessionDetail, props: true }
  ]
});

store.dispatch('connectStreams');

new Vue({
  router,
  store,
  render: h => h(App),
}).$mount('#app');
