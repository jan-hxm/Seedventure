<template>
  <div class="candlestick-chart-container">
    <h2>Live Price Chart</h2>

    <!-- Connection Status Component -->
    <ConnectionStatus
      :status="priceStore.state.connectionStatus"
      :dataInfo="priceStore.state.dataInfo"
    />

    <!-- Timeframe Selector Component -->
    <TimeframeSelector v-model="selectedTimeframe" />

    <!-- Chart Container Component -->
    <ChartContainer
      :candles="candles"
      :timeframe="selectedTimeframe"
      :isLoading="priceStore.state.isLoading"
    />

    <!-- Price Info Component -->
    <PriceInfo
      :currentPrice="priceStore.state.currentPrice"
      :priceChange="priceStore.state.priceChange"
      :isPositive="priceStore.state.isPositiveChange"
    />
  </div>
</template>

<script>
// Use normal script tag instead of setup for better control
import {
  ref,
  computed,
  onMounted,
  onBeforeUnmount,
  watch,
  onUnmounted,
} from "vue";
import ConnectionStatus from "./ConnectionStatus.vue";
import TimeframeSelector from "./TimeframeSelector.vue";
import ChartContainer from "./ChartContainer.vue";
import PriceInfo from "./PriceInfo.vue";
import priceStore from "../store/priceStore.js";

export default {
  components: {
    ConnectionStatus,
    TimeframeSelector,
    ChartContainer,
    PriceInfo,
  },

  setup() {
    // Local reference to timeframe for v-model binding
    const selectedTimeframe = ref(priceStore.state.selectedTimeframe);
    let timeframeWatcher = null;

    // Create a computed property for candles to ensure reactivity
    const candles = computed(() => {
      try {
        return priceStore.getCandles() || [];
      } catch (error) {
        console.error("Error getting candles:", error);
        return [];
      }
    });

    // Watch for local timeframe changes and update store
    onMounted(() => {
      // Set up watcher with a debounce
      let debounceTimer = null;

      timeframeWatcher = watch(selectedTimeframe, (newValue) => {
        if (debounceTimer) clearTimeout(debounceTimer);

        debounceTimer = setTimeout(async () => {
          try {
            // Set loading state
            priceStore.state.isLoading = true;

            // Change timeframe and wait for data
            await priceStore.setTimeframe(newValue);
          } catch (error) {
            console.error("Error changing timeframe:", error);
          }
        }, 100);
      });

      // Initial setup
      initializeData();
    });

    // Initialize data
    const initializeData = async () => {
      try {
        // Load available timeframes first
        await priceStore.loadAvailableTimeframes();

        // Then load historical data
        await priceStore.loadHistoricalData();

        // Finally connect to WebSocket for real-time updates
        priceStore.connectToLiveData();
      } catch (error) {
        console.error("Error initializing data:", error);
      }
    };

    // Clean up when component is unmounted
    onBeforeUnmount(() => {
      priceStore.disconnect();
    });

    // Remove watchers when component is unmounted
    onUnmounted(() => {
      if (timeframeWatcher) {
        timeframeWatcher(); // Stop the watcher
      }
    });

    return {
      selectedTimeframe,
      candles,
      priceStore,
    };
  },
};
</script>

<style scoped>
.candlestick-chart-container {
  max-width: 1000px;
  margin: 0 auto;
  padding: 20px;
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

h2 {
  font-size: 24px;
  margin-bottom: 20px;
  text-align: center;
  color: #333;
}

@media (max-width: 768px) {
  .candlestick-chart-container {
    padding: 15px;
    margin: 0 10px;
  }

  h2 {
    font-size: 20px;
    margin-bottom: 15px;
  }
}
</style>
