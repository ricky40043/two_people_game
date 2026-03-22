<script setup lang="ts">
defineProps<{
  visible: boolean
  roomId: string
  playerName: string
  hostName: string
  roomStatus: string
  isHost: boolean
}>()

const emit = defineEmits<{
  rejoin: []
  leave: []
}>()

const statusText = (status: string) => {
  switch (status) {
    case 'waiting': return '等待中'
    case 'playing':
    case 'question_display':
    case 'answering':
    case 'show_result': return '遊戲進行中'
    case 'finished': return '已結束'
    default: return status
  }
}
</script>

<template>
  <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4">
    <div class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-2xl">
      <h2 class="mb-2 text-center text-xl font-bold text-gray-800">偵測到進行中的遊戲</h2>

      <div class="mb-4 rounded-lg bg-gray-50 p-4 text-sm text-gray-600">
        <p>房間代碼：<span class="font-bold text-indigo-600">{{ roomId }}</span></p>
        <p>你的名字：<span class="font-semibold">{{ playerName }}</span></p>
        <p>主持人：<span class="font-semibold">{{ hostName }}</span></p>
        <p>狀態：<span class="font-semibold" :class="{
          'text-green-600': roomStatus === 'waiting',
          'text-orange-600': ['playing', 'question_display', 'answering', 'show_result'].includes(roomStatus),
          'text-gray-400': roomStatus === 'finished'
        }">{{ statusText(roomStatus) }}</span></p>
        <p v-if="isHost" class="mt-1 font-semibold text-indigo-600">你是主持人</p>
      </div>

      <div class="flex gap-3">
        <button
          @click="emit('leave')"
          class="flex-1 rounded-xl border-2 border-gray-300 bg-white px-4 py-3 font-semibold text-gray-700 transition hover:bg-gray-100"
        >
          離開，回首頁
        </button>
        <button
          @click="emit('rejoin')"
          class="flex-1 rounded-xl bg-indigo-600 px-4 py-3 font-semibold text-white transition hover:bg-indigo-700"
        >
          返回遊戲
        </button>
      </div>
    </div>
  </div>
</template>
