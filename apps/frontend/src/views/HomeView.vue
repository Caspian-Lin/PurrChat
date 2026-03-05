<template>
  <div class="flex h-screen">
    <!-- 左侧导航栏 -->
    <SideNavbar
      :current-user="currentUser"
      @show-profile="handleShowProfile"
    />

    <!-- 路由视图 - 显示不同的panel -->
    <div class="flex-1">
      <router-view />
    </div>

    <!-- 个人资料弹窗 -->
    <UserProfileModal
      :show="showProfile"
      :user="currentUser"
      @update:show="showProfile = $event"
      @logout="handleLogout"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useAuthController } from '../controllers/authController';
import SideNavbar from '../components/home/SideNavbar.vue';
import UserProfileModal from '../components/home/UserProfileModal.vue';

// Auth
const auth = useAuthController();
const { currentUser, handleLogout } = auth;

// Profile modal state
const showProfile = ref(false);

// Handlers
const handleShowProfile = () => {
  showProfile.value = true;
};

// Lifecycle
onMounted(async () => {
  await auth.checkAuth();
});
</script>

<style scoped></style>
