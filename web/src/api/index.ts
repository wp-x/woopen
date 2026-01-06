import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'

// 创建 axios 实例
const api = axios.create({
    baseURL: '/api',
    timeout: 30000,
    headers: {
        'Content-Type': 'application/json'
    }
})

// 请求拦截器
api.interceptors.request.use(
    (config) => {
        const authStore = useAuthStore()
        if (authStore.token) {
            config.headers.Authorization = `Bearer ${authStore.token}`
        }
        return config
    },
    (error) => {
        return Promise.reject(error)
    }
)

// 响应拦截器
api.interceptors.response.use(
    (response) => {
        const data = response.data
        if (data.code !== 0) {
            ElMessage.error(data.message || '请求失败')
            return Promise.reject(new Error(data.message))
        }
        return data
    },
    (error) => {
        if (error.response?.status === 401) {
            const authStore = useAuthStore()
            authStore.logout()
            window.location.href = '/admin/login'
        } else {
            ElMessage.error(error.response?.data?.message || error.message || '网络错误')
        }
        return Promise.reject(error)
    }
)

// 认证API
export const authApi = {
    login: (password: string) => api.post('/auth/login', { password }),
    logout: () => api.post('/auth/logout'),
    getMe: () => api.get('/auth/me')
}

// 文件API
export const fileApi = {
    list: (dirId?: string, page = 1, pageSize = 50) =>
        api.get('/files', { params: { dir_id: dirId, page, page_size: pageSize } }),
    link: (fid: string) => api.get('/files/link', { params: { fid } })
}

// 分享API
export const shareApi = {
    list: (page = 1, pageSize = 20) =>
        api.get('/shares', { params: { page, page_size: pageSize } }),
    create: (data: {
        target_type: 'file' | 'folder'
        target_id: string
        target_name: string
        target_path?: string
        share_code?: string
        password?: string
        expire_at?: string
        description?: string
    }) => api.post('/shares', data),
    batchCreate: (items: Array<{
        target_type: 'file' | 'folder'
        target_id: string
        target_name: string
        target_path?: string
        password?: string
        expire_at?: string
    }>) => api.post('/shares/batch', { items }),
    update: (id: number, data: any) => api.put(`/shares/${id}`, data),
    delete: (id: number) => api.delete(`/shares/${id}`),
    getQRCode: (id: number) => api.get(`/shares/${id}/qrcode`)
}

// 统计API
export const statsApi = {
    overview: () => api.get('/stats/overview'),
    trend: () => api.get('/stats/trend')
}

// 设置API
export const settingsApi = {
    get: () => api.get('/settings'),
    update: (data: any) => api.put('/settings', data),
    updatePassword: (oldPassword: string, newPassword: string) =>
        api.put('/settings/password', { old_password: oldPassword, new_password: newPassword }),
    testToken: (data: { refresh_token?: string; access_token?: string }) =>
        api.post('/settings/token/test', data)
}

// 公开访问API（不需要认证）
export const publicApi = {
    // 获取站点配置（公开）
    getSiteConfig: () => axios.get('/api/site-config').then(res => res.data),
    access: (code: string) => axios.get(`/s/${code}`).then(res => res.data),
    verify: (code: string, password: string) =>
        axios.post(`/s/${code}/verify`, { password }).then(res => res.data),
    getFiles: (code: string, dirId?: string, pwd?: string) => {
        const params: any = {}
        if (dirId) params.dir_id = dirId
        if (pwd) params.pwd = pwd
        return axios.get(`/s/${code}/files`, { params }).then(res => res.data)
    },
    getPreviewInfo: (code: string, fileId?: string) => {
        const params: any = {}
        if (fileId) params.file_id = fileId
        return axios.get(`/s/${code}/info`, { params }).then(res => res.data)
    },
    getDownloadUrl: (code: string, fileId?: string, pwd?: string) => {
        let url = `/s/${code}/download`
        if (fileId) url += `/${fileId}`
        if (pwd) url += `?pwd=${encodeURIComponent(pwd)}`
        return url
    },
    getPreviewUrl: (code: string, fileId?: string, pwd?: string) => {
        let url = `/s/${code}/preview`
        if (fileId) url += `/${fileId}`
        if (pwd) url += `?pwd=${encodeURIComponent(pwd)}`
        return url
    }
}

export default api
