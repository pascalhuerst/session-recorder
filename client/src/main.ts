import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "./App.vue";
import SessionDetail from "./components/SessionDetail.vue";
import SessionsView from "./modules/sessions/SessionsView.vue";
import { createPinia } from "pinia";
import RecordersView from "./modules/recorders/RecordersView.vue";

const router = createRouter({
  history: createWebHistory(),
  mode: "history",
  routes: [
    { path: "/recorders", component: RecordersView },
    { path: "/recorders/:recorderId/sessions", component: SessionsView },
    {
      path: "/detail/:recorderID/:sessionID",
      component: SessionDetail,
      props: true
    },
    { path: "/:pathMatch(.*)*", redirect: "/recorders" }
  ]
});

const app = createApp(App);

app.use(createPinia());
app.use(router);

app.mount("#app");
