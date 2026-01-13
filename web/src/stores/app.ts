import { defineStore } from 'pinia'
import { ref } from 'vue'

const TERMS_ACCEPTED_KEY = 'container-registry-terms-accepted'

export const useAppStore = defineStore('app', () => {
  const version = ref('')
  const loading = ref(false)
  const termsAccepted = ref(localStorage.getItem(TERMS_ACCEPTED_KEY) === 'true')

  function setVersion(v: string) {
    version.value = v
  }

  function setLoading(l: boolean) {
    loading.value = l
  }

  function acceptTerms() {
    localStorage.setItem(TERMS_ACCEPTED_KEY, 'true')
    termsAccepted.value = true
  }

  function checkTermsAccepted(): boolean {
    return localStorage.getItem(TERMS_ACCEPTED_KEY) === 'true'
  }

  return {
    version,
    loading,
    termsAccepted,
    setVersion,
    setLoading,
    acceptTerms,
    checkTermsAccepted
  }
})
