<template>
  <div class="chart-container">
    <!-- Timeframe selection -->
    <div class="timeframe-selector">
      <button
        v-for="tf in availableTimeframes"
        :key="tf.value"
        :class="{ active: selectedTimeframe === tf.value }"
        @click="changeTimeframe(tf.value)"
      >
        {{ tf.label }}
      </button>
    </div>

    <!-- Status indicator -->
    <div class="connection-status" :class="connectionStatus.toLowerCase()">
      {{ connectionStatus }}
    </div>

    <!-- Chart element -->
    <div ref="chartRef" class="chart"></div>

    <!-- Loading overlay -->
    <div v-if="isLoading" class="loading-overlay">
      <div class="spinner"></div>
      <div>Loading...</div>
    </div>
  </div>
</template>

<script setup>
import { ref, shallowRef, onMounted, onUnmounted, watch, nextTick } from "vue";
import ApexCharts from "apexcharts";
import priceStore from "../store/priceStore";
import { getChartOptions, timeframes } from "../utils/chartUtils";

// Constants
const MAX_CANDLES = 100;

// Reference to chart DOM element and ApexCharts instance
const chartRef = ref(null);
const chart = shallowRef(null); // Use shallowRef for non-reactive objects

// Reactive references to store state (direct access, no props)
const selectedTimeframe = ref(priceStore.state.selectedTimeframe);
const connectionStatus = ref(priceStore.state.connectionStatus);
const currentPrice = ref(priceStore.state.currentPrice);
const priceChange = ref(priceStore.state.priceChange);
const isPositiveChange = ref(priceStore.state.isPositiveChange);
const isLoading = ref(priceStore.state.isLoading);
const availableTimeframes = ref(timeframes);

// Track if component is mounted to prevent updates after unmount
let isMounted = false;
let updateTimer = null;

// Change timeframe and reload data
const changeTimeframe = (timeframe) => {
  if (selectedTimeframe.value === timeframe) return;

  selectedTimeframe.value = timeframe;
  priceStore.setTimeframe(timeframe);
};

// Initialize the chart with current data
const initializeChart = () => {
  if (!chartRef.value || !isMounted) return;

  try {
    // Get limited candles directly from store (no separate function call to format)
    const storeCandles = priceStore.getCandles() || [];
    const candles = storeCandles.slice(-MAX_CANDLES).map((candle) => ({
      x: new Date(candle.x),
      y: candle.y,
    }));

    // Setup chart with initial options
    const options = {
      ...getChartOptions(selectedTimeframe.value, candles),
      series: [
        {
          name: "Price",
          data: candles,
        },
      ],
    };

    // Create chart instance
    chart.value = new ApexCharts(chartRef.value, options);
    chart.value.render();

    // Connect to live data
    if (connectionStatus.value !== "Connected") {
      priceStore.connectToLiveData();
    }
  } catch (err) {
    console.error("Error initializing chart:", err);
  }
};

// Update chart data (debounced to prevent too many updates)
const updateChartData = () => {
  if (updateTimer) {
    clearTimeout(updateTimer);
  }

  updateTimer = setTimeout(() => {
    if (!chart.value || !isMounted) return;

    try {
      // Get limited candles directly from store
      const storeCandles = priceStore.getCandles() || [];
      const candles = storeCandles.slice(-MAX_CANDLES).map((candle) => ({
        x: new Date(candle.x),
        y: candle.y,
      }));

      // Only update if we have data
      if (candles.length > 0) {
        chart.value.updateSeries([
          {
            name: "Price",
            data: candles,
          },
        ]);
      }
    } catch (err) {
      console.error("Error updating chart data:", err);
    }
  }, 100); // 100ms debounce
};

// Update chart options when timeframe changes
const updateChartOptions = () => {
  if (!chart.value || !isMounted) return;

  try {
    // Get limited data directly from store
    const storeCandles = priceStore.getCandles() || [];
    const candles = storeCandles.slice(-MAX_CANDLES).map((candle) => ({
      x: new Date(candle.x),
      y: candle.y,
    }));

    // Update options
    const options = getChartOptions(selectedTimeframe.value, candles);
    chart.value.updateOptions(options, false, true);
  } catch (err) {
    console.error("Error updating chart options:", err);
  }
};

// Set up watchers

// Create a simple update trigger to watch for store changes
const updateTrigger = ref(0);

// Set up an interval to check for store changes
let storeCheckInterval = null;
const startStoreWatcher = () => {
  // Check for changes every 1000ms (1 second)
  storeCheckInterval = setInterval(() => {
    updateTrigger.value++;
  }, 1000);
};

// Watch the update trigger to refresh the chart
watch(updateTrigger, () => {
  updateChartData();

  // Update UI state values
  currentPrice.value = priceStore.state.currentPrice;
  priceChange.value = priceStore.state.priceChange;
  isPositiveChange.value = priceStore.state.isPositiveChange;
  connectionStatus.value = priceStore.state.connectionStatus;
  isLoading.value = priceStore.state.isLoading;
});

// Watch for timeframe changes
watch(
  () => selectedTimeframe.value,
  () => {
    updateChartOptions();
  }
);

// Watch connection status for display updates
watch(
  () => priceStore.state.connectionStatus,
  (newStatus) => {
    connectionStatus.value = newStatus;
  }
);

// Watch loading state
watch(
  () => priceStore.state.isLoading,
  (newState) => {
    isLoading.value = newState;
  }
);

// Lifecycle hooks
onMounted(() => {
  isMounted = true;

  // Load available timeframes
  priceStore.loadAvailableTimeframes().then((result) => {
    if (result && result.length > 0) {
      availableTimeframes.value = result.map((tf) => {
        const existing = timeframes.find((item) => item.value === tf);
        return existing || { label: tf, value: tf };
      });
    }
  });

  // Initialize chart on next tick
  nextTick(() => {
    initializeChart();
    startStoreWatcher();
  });
});

onUnmounted(() => {
  isMounted = false;

  // Clean up resources
  if (updateTimer) {
    clearTimeout(updateTimer);
    updateTimer = null;
  }

  if (storeCheckInterval) {
    clearInterval(storeCheckInterval);
    storeCheckInterval = null;
  }

  if (chart.value) {
    chart.value.destroy();
    chart.value = null;
  }

  // Disconnect from WebSocket
  priceStore.disconnect();
});
</script>

<style scoped>
.chart-container {
  position: relative;
  width: 95%;
  height: 500px;
  background-color: #f8f9fa;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  padding: 16px;
  overflow: hidden;
}

.chart {
  width: 100%;
  height: calc(100% - 80px);
  margin-top: 12px;
}

.timeframe-selector {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.timeframe-selector button {
  padding: 6px 12px;
  border: 1px solid #ddd;
  background-color: #f8f9fa;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}

.timeframe-selector button.active {
  background-color: #2c3e50;
  color: white;
  border-color: #2c3e50;
}

.connection-status {
  position: absolute;
  top: 16px;
  right: 16px;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: bold;
}

.connection-status.connected {
  background-color: rgba(38, 166, 154, 0.2);
  color: #26a69a;
}

.connection-status.connecting {
  background-color: rgba(255, 193, 7, 0.2);
  color: #ffc107;
}

.connection-status.disconnected {
  background-color: rgba(239, 83, 80, 0.2);
  color: #ef5350;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(255, 255, 255, 0.8);
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  z-index: 10;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 4px solid rgba(0, 0, 0, 0.1);
  border-radius: 50%;
  border-left-color: #2c3e50;
  animation: spin 1s linear infinite;
  margin-bottom: 12px;
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
</style>
