import { defineStore } from 'pinia'
import { userApi, messageApi } from '@/api'

const tokenKey = 'tutoring_token'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem(tokenKey) || '',
    profile: null,
    loading: false,
    unreadMessages: 0
  }),
  getters: {
    isLogin: (state) => !!state.token && !!state.profile,
    // 会员未过期
    isMember: (state) => !!state.profile?.memberActive,
    memberText: (state) => state.profile?.memberTypeName || '免费试用',
    // 站点拥有者 / 管理员
    isStaff: (state) => !!state.profile?.isStaff,
    isOwner: (state) => state.profile?.role === 'owner',
    // 使用身份：teacher=老师 parent/personal=学员 org=机构（未登录默认老师口径）
    userRole: (state) => state.profile?.userRole || 'teacher'
  },
  actions: {
    setToken(token) {
      this.token = token
      if (token) {
        localStorage.setItem(tokenKey, token)
      } else {
        localStorage.removeItem(tokenKey)
      }
    },
    async login(payload) {
      const res = await userApi.login(payload)
      this.setToken(res.data.token)
      this.profile = res.data.user
      this.loadUnreadMessages()
      return res.data
    },
    async register(payload) {
      const res = await userApi.register(payload)
      this.setToken(res.data.token)
      this.profile = res.data.user
      this.loadUnreadMessages()
      return res.data
    },
    async fetchProfile() {
      if (!this.token) return null
      this.loading = true
      try {
        const res = await userApi.profile()
        this.profile = res.data
        this.loadUnreadMessages()
        return res.data
      } catch (e) {
        this.logout()
        return null
      } finally {
        this.loading = false
      }
    },
    // 未读站内信数量（头部角标）
    async loadUnreadMessages() {
      if (!this.token) {
        this.unreadMessages = 0
        return
      }
      try {
        const res = await messageApi.unread()
        this.unreadMessages = res.data?.count || 0
      } catch (e) {
        this.unreadMessages = 0
      }
    },
    async updateProfile(payload) {
      const res = await userApi.updateProfile(payload)
      this.profile = res.data
      return res.data
    },
    async activateMember(payload) {
      const res = await userApi.activateMember(payload)
      this.profile = res.data.user || res.data
      return res.data
    },
    logout() {
      this.setToken('')
      this.profile = null
    }
  }
})
