import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUIStore = defineStore('ui', () => {
    const toast = ref({
        show: false,
        type: 'success' as 'success' | 'error',
        title: '',
        message: ''
    })

    let toastTimeout: any = null

    function showToast(type: 'success' | 'error', title: string, message: string) {
        // Reset
        toast.value.show = false
        clearTimeout(toastTimeout)

        // Set new content
        setTimeout(() => {
            toast.value = {
                show: true,
                type,
                title,
                message
            }

            // Auto hide
            toastTimeout = setTimeout(() => {
                toast.value.show = false
            }, 3000)
        }, 10)
    }

    return {
        toast,
        showToast
    }
})
