<template>
  <div class="flex flex-col min-h-screen">
    <Header :isDark="isDark" :isContact="showContact" @toggleDark="toggleDark" @goContact="showContact = true" @goHome="showContact = false" />
    <HomeView v-if="!showContact" />
    <ContactView v-else />
    <Footer />
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import Header from './components/common/Header.vue'
import HomeView from './views/HomeView.vue'
import ContactView from './views/ContactView.vue'
import Footer from './components/common/Footer.vue'

const isDark = ref(false)
const showContact = ref(false)

onMounted(() => {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
})

watch(isDark, (newVal) => {
  if (newVal) {
    document.documentElement.classList.add('dark')
    localStorage.setItem('theme', 'dark')
  } else {
    document.documentElement.classList.remove('dark')
    localStorage.setItem('theme', 'light')
  }
})

const toggleDark = () => {
  isDark.value = !isDark.value
}
</script>
