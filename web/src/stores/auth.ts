import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
    const token = ref<string | null>(localStorage.getItem('woopen_token'))
    const darkMode = ref<boolean>(localStorage.getItem('woopen_dark') === 'true')

    const isLoggedIn = computed(() => !!token.value)

    function setToken(newToken: string) {
        token.value = newToken
        localStorage.setItem('woopen_token', newToken)
    }

    function logout() {
        token.value = null
        localStorage.removeItem('woopen_token')
    }

    function toggleDarkMode() {
        darkMode.value = !darkMode.value
        localStorage.setItem('woopen_dark', String(darkMode.value))
        applyDarkMode()
    }

    function applyDarkMode() {
        if (darkMode.value) {
            document.documentElement.classList.add('dark')
        } else {
            document.documentElement.classList.remove('dark')
        }
    }

    // 初始化暗黑模式
    applyDarkMode()

    return {
        token,
        darkMode,
        isLoggedIn,
        setToken,
        logout,
        toggleDarkMode
    }
})
