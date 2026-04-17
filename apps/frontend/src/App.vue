<template>
  <div id="app">
    <router-view />
    <MessageContainer />
  </div>
</template>

<script setup lang="ts">
import MessageContainer from './components/common/MessageContainer.vue';
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family:
    'Onest',
    'Noto Sans SC',
    -apple-system,
    BlinkMacSystemFont,
    'Segoe UI',
    Roboto,
    sans-serif;
  background: var(--background-color);
  color: var(--text-color);
}

#app {
  width: 100%;
  height: 100%;
}
</style>
