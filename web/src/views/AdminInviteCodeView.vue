<template>
  <div class="page">
    <section class="card">
      <div class="card-title">
        邀请码管理
        <span class="sub">为申请人（手机号）生成邀请码，一个手机号（微信）仅可申请一个；注册时需与手机号匹配</span>
      </div>
      <InviteCodePanel />
    </section>
  </div>
</template>

<script setup>
import InviteCodePanel from '@/components/InviteCodePanel.vue'
</script>
