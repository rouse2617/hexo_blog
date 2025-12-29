<template>
  <div class="chat-page">
    <ChatWindow />
  </div>
</template>

<script setup lang="ts">
import ChatWindow from '@/components/chat/ChatWindow.vue'
</script>

<style scoped>
.chat-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
</style>
