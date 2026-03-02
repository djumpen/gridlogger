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
      <div>
        <p class="sub">Оберіть будинок зі списку, щоб переглянути стан електропостачання.</p>
      </div>
    </header>

    <p class="landing-note">Увійдіть через Telegram, щоб додати власний проєкт.</p>

    <p v-if="projectsLoading">Завантаження…</p>
    <p v-else-if="projectsError" class="error">{{ projectsError }}</p>
    <ul v-else-if="projects.length" class="project-list">
      <li v-for="project in projects" :key="project.id">
        <a :href="`/${project.slug}`" class="project-link">
          <span class="project-name">{{ project.name }}</span>
          <span class="project-city" v-if="project.city">м. {{ project.city }}</span>
        </a>
      </li>
    </ul>
    <p v-else class="sub">Поки що проєктів немає.</p>
  </section>
</template>
