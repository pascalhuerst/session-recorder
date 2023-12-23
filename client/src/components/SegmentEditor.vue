<template>
  <div class="modal-card" style="width: auto">
    <header class="modal-card-head">
      <p class="modal-card-title">Add Segment</p>
      <button type="button" class="delete" @click="$emit('close')" />
    </header>
    <section class="modal-card-body">
      <b-field label="Name">
        <b-input type="text" :value="name" placeholder="Segment Name" required>
        </b-input>
      </b-field>

      <b-field label="Start">
        <b-input type="text" :value="0" placeholder="Start Time" required>
        </b-input>
      </b-field>

      <b-field label="End">
        <b-input type="text" :value="10" placeholder="End Time" required>
        </b-input>
      </b-field>
    </section>
    <footer class="modal-card-foot">
      <b-button type="is-secundary" size="is-small" @click="$emit('close')"
        >Cancel</b-button
      >
      <b-button type="is-primary" size="is-small">Add Segment</b-button>
    </footer>
  </div>
</template>

<script>
export default {
  name: "SegmentEditor",
  props: ["name", "start", "end"],
};
</script>


<style scoped>
</style>
