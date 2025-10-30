import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    name: 'home',
    component: () => import('@/views/HomeView.vue'),
    meta: {
      title: '主頁 - Ricky 遊戲小舖',
      description: '選擇創建房間或加入遊戲'
    }
  },
  {
    path: '/create',
    name: 'create-room',
    component: () => import('@/views/CreateRoomView.vue'),
    meta: {
      title: '創建房間 - Ricky 遊戲小舖',
      description: '創建新的遊戲房間'
    }
  },
  {
    path: '/join',
    name: 'join-room',
    component: () => import('@/views/JoinRoomView.vue'),
    meta: {
      title: '加入房間 - Ricky 遊戲小舖',
      description: '輸入房間代碼加入遊戲'
    }
  },
  {
    path: '/join/:roomId',
    name: 'join-room-direct',
    component: () => import('@/views/JoinRoomView.vue'),
    props: true,
    meta: {
      title: '加入房間 - Ricky 遊戲小舖',
      description: '直接加入指定房間'
    }
  },
  {
    path: '/lobby/:roomId',
    name: 'lobby',
    component: () => import('@/views/LobbyView.vue'),
    props: true,
    meta: {
      title: '等待大廳 - Ricky 遊戲小舖',
      description: '等待遊戲開始'
    }
  },
  {
    path: '/game/host/:roomId',
    name: 'game-host',
    component: () => import('@/views/GameHostView.vue'),
    props: true,
    meta: {
      title: '主持人視角 - Ricky 遊戲小舖',
      description: '遊戲主畫面控制台',
      fullscreen: true
    }
  },
  {
    path: '/game/player/:roomId',
    name: 'game-player',
    component: () => import('@/views/GamePlayerView.vue'),
    props: true,
    meta: {
      title: '玩家視角 - Ricky 遊戲小舖',
      description: '玩家答題介面',
      fullscreen: true
    }
  },
  {
    path: '/results/:roomId',
    name: 'results',
    component: () => import('@/views/ResultsView.vue'),
    props: true,
    meta: {
      title: '遊戲結果 - Ricky 遊戲小舖',
      description: '查看遊戲結果和排行榜'
    }
  },
  {
    path: '/about',
    name: 'about',
    component: () => import('@/views/AboutView.vue'),
    meta: {
      title: '關於 - Ricky 遊戲小舖',
      description: '關於這個遊戲的信息'
    }
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: {
      title: '頁面未找到 - Ricky 遊戲小舖',
      description: '您要找的頁面不存在'
    }
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    } else {
      return { top: 0 }
    }
  }
})

// 全域路由守衛
router.beforeEach((to, from, next) => {
  // 設置頁面標題
  if (to.meta?.title) {
    document.title = to.meta.title as string
  }
  
  // 設置頁面描述
  if (to.meta?.description) {
    const metaDescription = document.querySelector('meta[name="description"]')
    if (metaDescription) {
      metaDescription.setAttribute('content', to.meta.description as string)
    }
  }
  
  next()
})

export default router