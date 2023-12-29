<template>
  <div class="sessions">
    <b-loading
      :is-full-page="true"
      :active="loading"
      :can-cancel="false"
      id="connectionSpinner"
    ></b-loading>
    <h1 class="title">Detail</h1>
    <div class="tags has-addons">
      <span class="tag">Session ID</span>
      <span class="tag is-primary">{{ sessionID }}</span>
    </div>
    <!--
    <div class="buttons">
      <section>
        <b-button size="is-small" type="is-primary" v-on:click="toggleOverview">Toggle Overview</b-button>
        <b-button size="is-small" type="is-primary" v-on:click="toggleZoomview">Toggle Zoomview</b-button>
      </section>
    </div>
    -->
    <div id="peaks-container">
      <h2 class="subtitle">Overview</h2>
      <div id="overview-waveform"></div>
      <div class="vspace"></div>
      <h2 class="subtitle">Zoom View</h2>

      <nav class="level">
        <!-- Left side -->
        <div class="level-left">
          <div class="level-item">
            <b-button size="is-small" type="is-primary" v-on:click="play">
              <i class="fas fa-play"></i>
            </b-button>
          </div>
          <div class="level-item">
            <b-button size="is-small" type="is-primary" v-on:click="zoomIn">
              <i class="fas fa-search-plus"></i>
            </b-button>
          </div>
          <div class="level-item">
            <b-button size="is-small" type="is-primary" v-on:click="zoomOut">
              <i class="fas fa-search-minus"></i>
            </b-button>
          </div>
        </div>

        <!-- Right side -->
        <div class="level-right">
          <div class="level-item">
            <span class="is-small">Create Segment</span>
          </div>
          <div class="level-item">
            <input
              v-model="segmentName"
              class="input is-small"
              type="text"
              placeholder="Name"
            />
          </div>
          <div class="level-item">
            <b-button
              :disabled="segmentName === ''"
              size="is-small"
              type="is-primary"
              v-on:click="addSegment"
            >
              <i class="fas fa-plus"></i>
            </b-button>
          </div>
        </div>
      </nav>
      <div id="zoomview-waveform"></div>
    </div>

    <div class="vspace"></div>

    <nav v-for="segment in segments" :key="segment.id" class="level">
      <!-- Left side -->
      <div class="level-left">
        <div class="level-item">
          <span class="is-small">Name</span>
        </div>
        <div class="level-item">
          <input
            v-model="segment.labelText"
            class="input is-small"
            type="text"
            placeholder="Name"
          />
        </div>
        <div class="level-item">
          <span class="is-small">Start Time</span>
        </div>
        <div class="level-item">
          <input
            v-model="segment.startTime"
            class="input is-small"
            type="text"
            placeholder="Start Time"
          />
        </div>
        <div class="level-item">
          <span class="is-small">End Time</span>
        </div>
        <div class="level-item">
          <input
            v-model="segment.endTime"
            class="input is-small"
            type="text"
            placeholder="End Time"
          />
        </div>
        <div class="level-item">
          <input
            v-model="segment.ID"
            class="input is-small"
            type="text"
            placeholder="End Time"
          />
        </div>
        <div class="level-item">
          <button class="delete" @click="removeSegment(segment.id)">
            {{ segment.id }}
          </button>
        </div>
      </div>

      <!-- Right side -->
      <div class="level-right">
        <div class="level-item"></div>
      </div>
    </nav>

    <div class="vspace"></div>

    <div class="vspace">
      <div class="buttons">
        <section>
          <button
            class="button is-danger is-outlined"
            is-pulled-left
            @click="$router.push('/')"
          >
            <span>Abort</span>
            <span class="icon is-small">
              <i class="fas fa-times"></i>
            </span>
          </button>
          <button
            class="button is-success"
            is-pulled-right
            @click="renderSegments()"
          >
            <span class="icon is-small">
              <i class="fas fa-check"></i>
            </span>
            <span>Render Segments</span>
          </button>
        </section>
      </div>
    </div>

    <audio>
      <source :src="this.mediaURL" type="audio/ogg" />
    </audio>
  </div>
</template>

<script>
import Peaks from "peaks.js";

const serverURL = "http://server.lan:8234/";
const primaryColor = "#0c86ed";
const secundaryColor = "#ed730c";
const backgroundColor = "#333333";
const foregroundColor = "#eeeeee55";

export default {
  name: "SessionDetail",
  components: {},
  data() {
    return {
      backendServerURL: "http://server.lan:8234/",
      sessionsList: [],
      arrayBufferURL: "",
      mediaURL: "",
      isSegmentEditorActive: false,
      segments: {},
      segmentName: "",
      playState: "Play",
      instance: undefined,
      loading: true,
    };
  },
  props: ["recorderID", "sessionID"],
  created() {
    this.arrayBufferURL =
      serverURL + this.recorderID + "/" + this.sessionID + "/waveform.dat";
    this.mediaURL =
      serverURL + this.recorderID + "/" + this.sessionID + "/data.ogg";
    console.log("Waveform " + this.arrayBufferURL);
    console.log("Media: " + this.mediaURL);
  },
  mounted() {
    this.getWaveformData();
  },
  methods: {
    getWaveformData() {
      const options = {
        containers: {
          overview: document.getElementById("overview-waveform"),
          zoomview: document.getElementById("zoomview-waveform"),
        },
        mediaElement: document.querySelector("audio"),
        mediaUrl: this.mediaURL,
        dataUri: {
          arraybuffer: this.arrayBufferURL,
        },
        // Color for segment start marker handles
        segmentStartMarkerColor: secundaryColor,

        // Color for segment end marker handles
        segmentEndMarkerColor: secundaryColor,

        // Color for the zoomable waveform
        zoomWaveformColor: primaryColor,

        // Color for the overview waveform
        overviewWaveformColor: secundaryColor,

        // Color for the overview waveform rectangle
        // that shows what the zoomable view shows
        overviewHighlightColor: foregroundColor,

        // Color for segments on the waveform
        segmentColor: backgroundColor,

        // Color of the play head
        playheadColor: foregroundColor,

        // Color of the play head text
        playheadTextColor: foregroundColor,

        // Precision of time label of play head and point/segment markers
        timeLabelPrecision: 2,

        // Show current time next to the play head
        // (zoom view only)
        showPlayheadTime: true,

        // the color of a point marker
        pointMarkerColor: secundaryColor,

        // Color of the axis gridlines
        axisGridlineColor: foregroundColor,

        // Color of the axis labels
        axisLabelColor: foregroundColor,

        // Array of zoom levels in samples per pixel (big >> small)
        zoomLevels: [
          512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072, 262144,
        ],
        /*
        segments: [{
          startTime: 1,
          endTime: 10,
          editable: true,
          color: "#ffffff99",
          labelText: "Cutmarks"
        }],
        */
      };

      this.instance = Peaks.init(options, (err, inst) => {
        if (err) {
          console.error("Failed to initialize Peaks instance: " + err.message);
          return;
        }
        this.loading = false;

        inst.on("segments.add", this.segmentAdded);
        inst.on("segments.remove", this.segmentRemoved);
      });
    },
    segmentAdded(segments) {
      console.log("segmentAdded: ");
      for (var i = 0; i < segments.length; i++) {
        console.log(":" + segments[i].labelText);
      }
    },
    segmentRemoved(segments) {
      console.log("segmentRemoved: ");
      for (var i = 0; i < segments.length; i++) {
        console.log(":" + segments[i].labelText);
      }
    },
    play() {
      if (this.instance !== undefined) {
        if (this.playState == "Play") {
          this.instance.player.play();
          this.playState = "Pause";
        } else {
          this.instance.player.pause();
          this.playState = "Play";
        }
      }
    },
    removeSegment(id) {
      console.log("Removing Segment: " + id);
      this.instance.segments.removeById(id);
    },
    addSegment() {
      this.instance.segments.add({
        startTime: this.instance.player.getCurrentTime(),
        endTime: this.instance.player.getCurrentTime() + 10,
        labelText: this.segmentName,
        editable: true,
        color: "#ffffff99",
      });
      this.segmentName = "";
    },
    syncSegments() {
      var segments = this.instance.segments.getSegments();

      for (var i = 0; i < segments.length; i++) {
        var segment = segments[i];

        this.$set(this.segments, segment.id, {
          labelText: segment.labelText,
          startTime: segment.startTime,
          endTime: segment.endTime,
          id: segment.id,
          filetypes: ["mp3", "ogg", "wav", "flac"],
        });
      }
    },
    toggleZoomview() {
      if (this.instance !== undefined) {
        var container = document.getElementById("zoomview-waveform");
        var zoomview = this.instance.views.getView("zoomview");

        if (zoomview) {
          this.instance.views.destroyZoomview();
          container.style.display = "none";
        } else {
          container.style.display = "block";
          this.instance.views.createZoomview(container);
        }
      }
    },
    toggleOverview() {
      if (this.instance !== undefined) {
        var container = document.getElementById("overview-waveform");
        var overview = this.instance.views.getView("overview");

        if (overview) {
          this.instance.views.destroyOverview();
          container.style.display = "none";
        } else {
          container.style.display = "block";
          this.instance.views.createOverview(container);
        }
      }
    },
    zoomIn() {
      if (this.instance !== undefined) {
        this.instance.zoom.zoomIn();
      }
    },
    zoomOut() {
      this.instance.zoom.zoomOut();
    },
    renderSegments() {
      this.syncSegments();

      const requestOptions = {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          sessionID: this.sessionID,
          recorderID: this.recorderID,
          segments: this.segments,
        }),
      };

      fetch(this.backendServerURL + "render", requestOptions).then(
        (response) => {
          console.log(JSON.stringify(response));
        },
      );

      console.log(JSON.stringify(this.segments));

      //this.$router.push('/');
    },
  },
};
</script>

<style scoped>
.vspace {
  padding-top: 50px;
}
</style>
