import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import App from "./App.vue";
import SessionDetail from "./components/SessionDetail.vue";
import SessionsView from "./modules/sessions/SessionsView.vue";
import { createPinia } from "pinia";
import RecordersView from "./modules/recorders/RecordersView.vue";
/* import the fontawesome core */
import { library } from "@fortawesome/fontawesome-svg-core";

/* import font awesome icon component */
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";

/* import specific icons */
import { faTimes } from "@fortawesome/free-solid-svg-icons/faTimes";
import { faPlay } from "@fortawesome/free-solid-svg-icons/faPlay";
import { faPause } from "@fortawesome/free-solid-svg-icons/faPause";

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

/* add icons to the library */
library.add(faTimes, faPlay, faPause);

const app = createApp(App);

app.component("font-awesome-icon", FontAwesomeIcon);

app.use(createPinia());
app.use(router);

app.mount("#app");
