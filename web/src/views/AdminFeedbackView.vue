<template>
  <div class="page">
    <section class="card">
      <div class="card-title">
        用户反馈
        <span class="sub">展示普通会员提交的问题，可答复与审核</span>
      </div>
      <FeedbackPanel />
    </section>
  </div>
</template>

<script setup>
import FeedbackPanel from '@/components/FeedbackPanel.vue'
</script>
