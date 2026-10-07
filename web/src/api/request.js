import axios from 'axios'
import { ElMessage } from 'element-plus'

const rawBase = import.meta.env.VITE_API_BASE || '/api'
// 0.0.0.0 是服务端监听地址，不能作为浏览器请求主机；若部署环境误配则回退相对路径
const baseURL = rawBase.includes('0.0.0.0') ? '/api' : rawBase

const request = axios.create({
  baseURL,
  timeout: 20000
})

request.interceptors.request.use((config) => {
  const token = localStorage.getItem('tutoring_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  (resp) => {
    const data = resp.data || {}
    if (data.code === 0 || data.code === undefined) {
      return data
    }
    // silent 请求（如角标轮询）失败不弹全局提示，由调用方自行处理
    if (!resp.config?.silent) {
      ElMessage.error(data.message || '请求失败')
    }
    return Promise.reject(new Error(data.message || '请求失败'))
  },
  (error) => {
    const status = error.response?.status
    const data = error.response?.data
    const silent = error.config?.silent
    if (status === 401) {
      if (!silent) {
        ElMessage.warning(data?.message || '登录已失效，请重新登录')
      }
      localStorage.removeItem('tutoring_token')
      setTimeout(() => {
        if (!location.hash.startsWith('#/login')) {
          location.href = '/'
        }
      }, 500)
    } else if (!silent) {
      ElMessage.error(data?.message || '网络异常，请稍后重试')
    }
    return Promise.reject(error)
  }
)

export default request
