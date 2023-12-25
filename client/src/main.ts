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
import RecordersIndexView from "./views/recorders/views/index/RecordersIndexView.vue";
import RecorderView from "./views/recorders/views/:recorderId/RecorderView.vue";
import SessionsView from "./views/recorders/views/:recorderId/views/sessions/SessionsView.vue";
import SessionsIndexView from "./views/recorders/views/:recorderId/views/sessions/views/index/SessionsIndexView.vue";
import SessionView from "./views/recorders/views/:recorderId/views/sessions/views/:sessionId/SessionView.vue";
import { faWaveSquare } from "@fortawesome/free-solid-svg-icons/faWaveSquare";
import { faMicrochip } from "@fortawesome/free-solid-svg-icons/faMicrochip";
import { faHeart } from "@fortawesome/free-solid-svg-icons/faHeart";
import { faTrash } from "@fortawesome/free-solid-svg-icons/faTrash";
import { faThumbtack } from "@fortawesome/free-solid-svg-icons/faThumbtack";
import { faArrowLeft } from "@fortawesome/free-solid-svg-icons/faArrowLeft";
import { faPlus } from "@fortawesome/free-solid-svg-icons/faPlus";
import { faMinus } from "@fortawesome/free-solid-svg-icons/faMinus";
import { faMagnifyingGlassChart } from "@fortawesome/free-solid-svg-icons/faMagnifyingGlassChart";
import { faArrowUpRightDots } from "@fortawesome/free-solid-svg-icons/faArrowUpRightDots";
import { faStopwatch } from "@fortawesome/free-solid-svg-icons/faStopwatch";

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
library.add(faTimes, faPlay, faPause, faMicrochip, faWaveSquare, faTrash, faHeart, faThumbtack, faArrowLeft, faPlus, faMinus, faMagnifyingGlassChart, faArrowUpRightDots, faStopwatch);

const app = createApp(App);

app.component("font-awesome-icon", FontAwesomeIcon);

app.use(createPinia());
app.use(router);

app.mount("#app");
