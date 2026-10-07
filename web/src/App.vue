<template>
  <MainLayout>
    <router-view v-slot="{ Component }">
      <component :is="Component" />
    </router-view>
  </MainLayout>
</template>

<script setup>
import MainLayout from '@/layouts/MainLayout.vue'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
if (userStore.token && !userStore.profile) {
  userStore.fetchProfile()
}
</script>
