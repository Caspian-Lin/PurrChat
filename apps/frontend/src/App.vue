<template>
  <div id="app">
    <router-view />
    <NotificationList />
  </div>
</template>

<script setup lang="ts">
import NotificationList from './components/common/NotificationList.vue';
</script>
