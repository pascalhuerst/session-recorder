<script setup lang="ts"></script>

<template>
    <div class="visually-hidden">
        <slot />
    </div>
</template>

<style scoped>
.visually-hidden {
    clip: rect(0 0 0 0);
    position: absolute;
    clip-path: inset(50%);
    width: 1px;
    height: 1px;
    overflow: hidden;
    white-space: nowrap;
}
</style>
