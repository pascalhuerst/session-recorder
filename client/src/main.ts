import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "./App.vue";

import OpenSessions from "./components/OpenSessions.vue";
import SessionDetail from "./components/SessionDetail.vue";

const router = createRouter({
  history: createWebHistory(),
  mode: "history",
  routes: [
    { path: "/", component: OpenSessions },
    { path: "/", component: OpenSessions },
    {
      path: "/detail/:recorderID/:sessionID",
      component: SessionDetail,
      props: true
    }
  ]
});

const app = createApp(App);

app.use(router);

app.mount("#app");
