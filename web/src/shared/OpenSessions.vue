<template>
  <section class="section">
    <div class="sessions">
      <div class="tabs  is-boxed">
        <ul v-for="(recorder, key) in sessionsList"
            :key="key">
          <li :class="selectedRecorderID === key ? 'is-active' : ''"><a @click="selectedRecorderID=key"> {{ key }} </a>
          </li>
        </ul>
      </div>


      <div v-if="selectedRecorderID !== undefined">
        <section class="hero" v-for="(session, index) in sessionsList[selectedRecorderID].open_sessions" :key="index">
          <div class="hero-body">
            <h1 class="title">Session #{{ index }}</h1>

            <nav class="level">
              <!-- Left side -->
              <div class="level-left">

                <div class="level-item">
                  <div class="tags has-addons">
                    <span class="tag">Lifetime</span>
                    <span class="tag is-primary">{{ formatHourToLive(session.hours_to_live) }} Hours</span>
                  </div>
                </div>
                <div class="level-item">
                  <div class="tags has-addons">
                    <span class="tag">Created</span>
                    <span class="tag is-primary">{{ new Date(session.timestamp).toLocaleString() }}</span>
                  </div>
                </div>
              </div>

              <!-- Right side -->
              <div class="level-right">
                <div class="level-item">
                  <span class="tag is-danger">
                      Delete Session 
                    <button class="delete is-small" @click="deleteSession(selectedRecorderID, session.id)"></button>
                  </span>
                </div>
                <div class="level-item">
                  <a :href="backendServerURL +  selectedRecorderID + '/' + session.id + '/' + 'data.wav'">
                    <span class="tag is-primary">
                      <i class="fas fa-download"></i>
                    </span>
                  </a>
                </div>

                <!-- Flagged Sessions don't get autodeleted -->
                <b-switch v-model="session.flagged"
                          @input="flagSession(selectedRecorderID, session.id, session.flagged)">
                  {{ session.flagged == true ? 'Keep Session' : 'Audo Delete' }}
                </b-switch>

                <!-- Playback cmds send to fileserver
                <div v-for="recorderID in availableRecorderIDs" :key="recorderID" class="level-item">
                    <span class="tags has-addons">
                      <span class="tag" >
                        <a @click="playSession(recorderID, session.id)">
                        <i class="fas fa-play"></i>
                        </a>
                      </span>
                      <span class="tag  is-primary">{{ recorderID }}</span>
                    </span>
                </div>
                -->
              </div>
            </nav>

            <a v-bind:href="serverURL + 'detail/' + selectedRecorderID + '/' + session.id">
              <img
                  class="waveform"
                  :src="backendServerURL +  selectedRecorderID + '/' + session.id + '/' + 'overview.png'"
              />
            </a>
          </div>
        </section>
      </div>
    </div>
  </section>
</template>

<script>

export default {
  name: "OpenSessions",
  components: {},
  data() {
    return {
      serverURL: "http://server.lan/",
      backendServerURL: "http://server.lan:8234/",
      sessionsList: {},
      selectedAudioFileURL: '',
      selectedArrayBufferURL: '',
      selectedRecorderID: undefined,
      availableRecorderIDs: [],
    };
  },
  created() {
    console.log('backendServerURL: ' + this.backendServerURL)
    this.getSessionsList();
  },
  methods: {
    getSessionsList() {
      fetch(this.backendServerURL + 'introspect')
          .then((response) => response.json())
          .then((data) => {
            this.sessionsList = data;

            this.availableRecorderIDs = []
            for (let key in this.sessionsList) {
              this.availableRecorderIDs.push(key)
            }

            console.log(JSON.stringify(this.sessionsList))

            /*
                      for (let key in this.sessionsList) {
                        this.selectedRecorderID = key
                        break;
                      }
            */
          });
    },
    formatHourToLive(hours) {
      return Math.floor(hours)
    },
    deleteSession(recorderID, sessionID) {
      const requestOptions = {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({
          recorderID: recorderID,
          sessionID: sessionID,
        })
      };

      fetch(this.backendServerURL + 'delete', requestOptions)
          .then(response => {
            console.log(JSON.stringify(response));
            this.getSessionsList();
          });
    },
    flagSession(recorderID, sessionID, isFlagged) {

      console.log('isFlagged: ' + isFlagged)

      const requestOptions = {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({
          recorderID: recorderID,
          sessionID: sessionID,
          flagged: isFlagged,
        })
      };

      fetch(this.backendServerURL + 'flag', requestOptions)
          .then(response => {
            console.log(JSON.stringify(response));
            //this.getSessionsList();
          });
    },
    playSession(recorderID, sessionID) {

      console.log("playSession")

      const requestOptions = {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({
          recorderID: recorderID,
          sessionID: sessionID,
          seek: 0,
        })
      };

      fetch(this.backendServerURL + 'play', requestOptions)
          .then(response => {
            console.log(JSON.stringify(response));
          });
    }
  },
};
</script>


<style scoped>
.waveform {
  height: 200px;
  width: 100%;
}
</style>
