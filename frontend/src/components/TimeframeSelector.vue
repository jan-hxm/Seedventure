<template>
  <div class="timespan-selector">
    <span>Timespan: </span>
    <button
      v-for="period in timeframes"
      :key="period.value"
      :class="['timespan-btn', { active: modelValue === period.value }]"
      @click="$emit('update:modelValue', period.value)"
    >
      {{ period.label }}
    </button>
  </div>
</template>

<script setup>
import { timeframes } from "../utils/chartUtils.js";

defineProps({
  modelValue: {
    type: String,
    required: true,
  },
});

defineEmits(["update:modelValue"]);
</script>

<style scoped>
.timespan-selector {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 10px 0;
  gap: 8px;
}

.timespan-btn {
  padding: 6px 12px;
  border: 1px solid #ddd;
  background-color: #f5f5f5;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
  color: #333;
}

.timespan-btn:hover {
  background-color: #e0e0e0;
}

.timespan-btn.active {
  background-color: #1976d2;
  color: white;
  border-color: #1976d2;
}
</style>
