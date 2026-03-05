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
    -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
  background: var(--background-color);
  color: var(--text-color);
}

#app {
  width: 100%;
  height: 100%;
}
</style>
