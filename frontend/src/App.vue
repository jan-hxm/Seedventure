<script setup>
import BettingComponent from "./components/BettingComponent.vue";
import GameCanvas from "./components/GameCanvas.vue";
</script>

<template>
  <h1>Seedventure</h1>
  <div class="main-container">
    <GameCanvas></GameCanvas>
    <BettingComponent></BettingComponent>
  </div>
</template>

<style scoped>
.logo {
  height: 6em;
  padding: 1.5em;
  will-change: filter;
  transition: filter 300ms;
}
.logo:hover {
  filter: drop-shadow(0 0 2em #646cffaa);
}
.logo.vue:hover {
  filter: drop-shadow(0 0 2em #42b883aa);
}
.main-container {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 2em;
}
</style>
