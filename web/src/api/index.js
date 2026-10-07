import request from './request'

export const userApi = {
  register: (data) => request.post('/user/register', data),
  login: (data) => request.post('/user/login', data),
  profile: () => request.get('/user/profile'),
  updateProfile: (data) => request.put('/user/profile', data),
  changePassword: (data) => request.put('/user/password', data),
  activateMember: (payload) => request.post('/user/member/activate', payload),
  payments: () => request.get('/user/payments'),
  upload: (formData) =>
    request.post('/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
}

export const orderApi = {
  list: (params) => request.get('/orders', { params }),
  options: () => request.get('/orders/options'),
  dashboard: (params) => request.get('/orders/dashboard', { params }),
  create: (data) => request.post('/orders', data),
  update: (id, data) => request.put(`/orders/${id}`, data),
  updateStatus: (id, status) => request.put(`/orders/${id}/status`, { status }),
  remove: (id) => request.delete(`/orders/${id}`)
}

export const scheduleApi = {
  week: (params) => request.get('/schedule', { params })
}

// 单次课程调整（改期 / 停课 / 改时间 / 临时加课）
export const lessonApi = {
  exceptions: (orderId) => request.get('/lessons/exception', { params: { orderId } }),
  adjust: (data) => request.post('/lessons/exception', data),
  removeException: (id) => request.delete(`/lessons/exception/${id}`),
  remove: (id) => request.delete(`/lessons/${id}`) // 删除一节已物化历史课次（老师未上课等）
}

export const feedbackApi = {
  mine: () => request.get('/feedback/mine'),
  create: (data) => request.post('/feedback', data)
}

export const adminApi = {
  users: (params) => request.get('/admin/users', { params }),
  createAdmin: (data) => request.post('/admin/users', data),
  stats: (params, opts) => request.get('/admin/stats', { params, ...opts }),
  freeze: (id, frozen) => request.put(`/admin/users/${id}/freeze`, { frozen }),
  renew: (id, payload) => request.put(`/admin/users/${id}/member`, payload),
  setRole: (id, role) => request.put(`/admin/users/${id}/role`, { role }),
  // password 留空则由服务端随机生成，回调数据里会带上明文密码
  resetPassword: (id, password) => request.put(`/admin/users/${id}/password`, { password }),
  removeUser: (id) => request.delete(`/admin/users/${id}`), // 删除账号（仅站长）
  feedbacks: (params, opts) => request.get('/admin/feedbacks', { params, ...opts }),
  replyFeedback: (id, payload) => request.put(`/admin/feedbacks/${id}`, payload),
  inviteCodes: (params) => request.get('/admin/invite-codes', { params }),
  createInviteCode: (data) => request.post('/admin/invite-codes', data),
  deleteInviteCode: (id) => request.delete(`/admin/invite-codes/${id}`),
  invalidateInviteCode: (id) => request.put(`/admin/invite-codes/${id}/invalidate`)
}

export const priceApi = {
  get: () => request.get('/price'),
  update: (data) => request.put('/admin/price', data),
  updateCard: (data) => request.post('/admin/price/card', data),
  schedules: () => request.get('/admin/price/schedules'),
  schedulesPending: () => request.get('/price/schedules/pending'),
  cancelSchedule: (id) => request.delete(`/admin/price/schedules/${id}`),
  impact: () => request.get('/admin/price/impact'),
  history: (params) => request.get('/admin/price/history', { params }),
  rollback: (id) => request.post(`/admin/price/history/${id}/rollback`)
}

export const messageApi = {
  list: (params) => request.get('/user/messages', { params }),
  unread: () => request.get('/user/messages/unread'),
  read: (id) => request.put(`/user/messages/${id}/read`),
  send: (data) => request.post('/admin/messages', data)
}
