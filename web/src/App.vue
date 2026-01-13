<script setup lang="ts">
import { ref } from 'vue'
import Layout from '@/components/Layout.vue'
import TermsDialog from '@/components/TermsDialog.vue'

const termsAccepted = ref(false)

const handleTermsAccepted = () => {
  termsAccepted.value = true
}
</script>

<template>
  <TermsDialog @accepted="handleTermsAccepted" />
  <Layout v-if="termsAccepted">
    <router-view />
  </Layout>
</template>

<style>
#app {
  width: 100%;
  min-height: 100vh;
}
</style>
