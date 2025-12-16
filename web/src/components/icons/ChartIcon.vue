<template>
  <svg
    class="icon"
    viewBox="0 0 1024 1024"
    version="1.1"
    xmlns="http://www.w3.org/2000/svg"
    p-id="25870"
    xmlns:xlink="http://www.w3.org/1999/xlink"
    :width="size"
    :height="size"
  >
    <path
      d="M976 944H48c-26.496 0-48-21.44-48-48V128c0-26.496 21.504-48 48-48s48 21.504 48 48v720h880c26.496 0 48 21.504 48 48 0 26.56-21.504 48-48 48zM864 800h-96c-26.496 0-48-21.504-48-48V416c0-26.496 21.504-48 48-48h96c26.496 0 48 21.504 48 48v336c0 26.56-21.504 48-48 48z m-272 0h-96c-26.496 0-48-21.44-48-48V224c0-26.496 21.504-48 48-48h96c26.496 0 48 21.504 48 48v528c0 26.56-21.504 48-48 48z m-272 0h-96c-26.496 0-48-21.504-48-48v-96c0-26.496 21.504-48 48-48h96c26.496 0 48 21.504 48 48v96c0 26.56-21.504 48-48 48z"
      :fill="color"
      p-id="25871"
    ></path>
  </svg>
</template>

<script setup lang="ts">
interface Props {
  size?: number | string
  color?: string
}

withDefaults(defineProps<Props>(), {
  size: 16,
  color: '#9377E0'
})
</script>

<style scoped>
.icon {
  display: inline-block;
  vertical-align: middle;
}
</style>

