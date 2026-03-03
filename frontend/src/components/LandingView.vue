<script setup>
defineProps({
  projects: {
    type: Array,
    required: true
  },
  projectsLoading: {
    type: Boolean,
    required: true
  },
  projectsError: {
    type: String,
    required: true
  }
})
</script>

<template>
  <section class="landing">
    <header class="hero">
      <div class="landing-intro">
        <ul class="landing-bullets">
          <li>Оберіть адресу зі списку, щоб переглянути стан електропостачання</li>
          <li>Увійдіть через Telegram, щоб додати свою адресу</li>
          <li>Підпишіться на оновлення щоб отримувати повідомлення про зміну статусу в Telegram</li>
        </ul>
      </div>
    </header>

    <p v-if="projectsLoading">Завантаження…</p>
    <p v-else-if="projectsError" class="error">{{ projectsError }}</p>
    <template v-else-if="projects.length">
      <h2 class="landing-list-title">Доступні адреси</h2>
      <ul class="project-list">
      <li v-for="project in projects" :key="project.id">
        <a :href="`/${project.slug}`" class="project-link">
          <span class="project-name">{{ project.name }}</span>
          <span class="project-city" v-if="project.city">м. {{ project.city }}</span>
        </a>
      </li>
      </ul>
    </template>
    <p v-else class="sub">Поки що проєктів немає.</p>
  </section>
</template>
