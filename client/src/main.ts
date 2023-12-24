import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "./App.vue";
import { createPinia } from "pinia";
import { library } from "@fortawesome/fontawesome-svg-core";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { faTimes } from "@fortawesome/free-solid-svg-icons/faTimes";
import { faPlay } from "@fortawesome/free-solid-svg-icons/faPlay";
import { faPause } from "@fortawesome/free-solid-svg-icons/faPause";
import RecordersIndexView from "./views/recorders/RecordersIndexView.vue";
import RecorderIndexView from "./views/recorders/:recorderId/RecorderIndexView.vue";
import SessionsIndexView from "./views/recorders/:recorderId/sessions/SessionsIndexView.vue";
import SessionView from "./views/recorders/:recorderId/sessions/:sessionId/SessionView.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/recorders",
      component: RecordersIndexView,
      children: [{
        path: ":recorderId",
        component: RecorderIndexView,
        children: [{
          path: "sessions",
          component: SessionsIndexView,
          children: [{
            path: ":sessionId",
            component: SessionView
          }]
        }]
      }]
    },

    { path: "/:pathMatch(.*)*", redirect: "/recorders" }
  ]
});

/* add icons to the library */
library.add(faTimes, faPlay, faPause);

const app = createApp(App);

app.component("font-awesome-icon", FontAwesomeIcon);

app.use(createPinia());
app.use(router);

app.mount("#app");
