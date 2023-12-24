import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "./App.vue";
import { createPinia } from "pinia";
import { library } from "@fortawesome/fontawesome-svg-core";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { faTimes } from "@fortawesome/free-solid-svg-icons/faTimes";
import { faPlay } from "@fortawesome/free-solid-svg-icons/faPlay";
import { faPause } from "@fortawesome/free-solid-svg-icons/faPause";
import RecordersView from "./views/recorders/RecordersView.vue";
import RecordersIndexView from "./views/recorders/index/RecordersIndexView.vue";
import RecorderView from "./views/recorders/:recorderId/RecorderView.vue";
import SessionsView from "./views/recorders/:recorderId/sessions/SessionsView.vue";
import SessionsIndexView from "./views/recorders/:recorderId/sessions/index/SessionsIndexView.vue";
import SessionView from "./views/recorders/:recorderId/sessions/:sessionId/SessionView.vue";
import { faWaveSquare } from "@fortawesome/free-solid-svg-icons/faWaveSquare";
import { faMicrochip } from "@fortawesome/free-solid-svg-icons/faMicrochip";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/recorders",
      component: RecordersView,
      children: [
        {
          path: "",
          component: RecordersIndexView
        },
        {
          path: ":recorderId",
          component: RecorderView,
          children: [
            {
              path: "",
              component: RecordersIndexView
            }, {
              path: "sessions",
              component: SessionsView,
              children: [
                {
                  path: "",
                  component: SessionsIndexView
                }, {
                  path: ":sessionId",
                  component: SessionView
                }
              ]
            }]
        }]
    },

    { path: "/:pathMatch(.*)*", redirect: "/recorders" }
  ]
});

/* add icons to the library */
library.add(faTimes, faPlay, faPause, faMicrochip, faWaveSquare);

const app = createApp(App);

app.component("font-awesome-icon", FontAwesomeIcon);

app.use(createPinia());
app.use(router);

app.mount("#app");
