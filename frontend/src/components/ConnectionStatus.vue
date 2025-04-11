<template>
  <div class="status-bar">
    <div class="connection-status" :class="statusClass">
      {{ status }}
    </div>
    <div class="data-info">{{ dataInfo }}</div>
  </div>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  status: {
    type: String,
    required: true,
  },
  dataInfo: {
    type: String,
    required: true,
  },
});

// Computed class for connection status indicator
const statusClass = computed(() => {
  switch (props.status) {
    case "Connected":
      return "connected";
    case "Disconnected":
      return "disconnected";
    case "Connecting...":
      return "connecting";
    default:
      return "";
  }
});
</script>

<style scoped>
.status-bar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 15px;
  font-size: 14px;
}

.connection-status {
  padding: 4px 10px;
  border-radius: 12px;
  display: flex;
  align-items: center;
}

.connection-status:before {
  content: "";
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
}

.connected {
  color: #2e7d32;
  background-color: rgba(46, 125, 50, 0.1);
}

.connected:before {
  background-color: #2e7d32;
}

.disconnected {
  color: #c62828;
  background-color: rgba(198, 40, 40, 0.1);
}

.disconnected:before {
  background-color: #c62828;
}

.connecting {
  color: #f57c00;
  background-color: rgba(245, 124, 0, 0.1);
}

.connecting:before {
  background-color: #f57c00;
}

.data-info {
  color: #666;
}
</style>
