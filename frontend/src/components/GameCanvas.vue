<template>
  <div class="candlestick-chart-container">
    <h2>Live Crypto Price Chart</h2>

    <!-- Connection Status Component -->
    <ConnectionStatus
      :status="priceStore.state.connectionStatus"
      :dataInfo="priceStore.state.dataInfo"
    />

    <!-- Timespan Selector Component -->
    <TimespanSelector v-model="selectedTimespan" />

    <!-- Chart Container Component -->
    <ChartContainer
      :candles="priceStore.getFilteredCandles()"
      :timespan="selectedTimespan"
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
import { ref, onMounted, onBeforeUnmount, watch, onUnmounted } from "vue";
import ConnectionStatus from "./ConnectionStatus.vue";
import TimespanSelector from "./TimespanSelector.vue";
import ChartContainer from "./ChartContainer.vue";
import PriceInfo from "./PriceInfo.vue";
import priceStore from "../store/priceStore.js";

export default {
  components: {
    ConnectionStatus,
    TimespanSelector,
    ChartContainer,
    PriceInfo,
  },

  setup() {
    // Local reference to timespan for v-model binding
    const selectedTimespan = ref(priceStore.state.selectedTimespan);
    let timespanWatcher = null;

    // Watch for local timespan changes and update store
    onMounted(() => {
      // Set up watcher with a debounce
      let debounceTimer = null;

      timespanWatcher = watch(selectedTimespan, (newValue) => {
        if (debounceTimer) clearTimeout(debounceTimer);

        debounceTimer = setTimeout(() => {
          priceStore.setTimespan(newValue);
        }, 100);
      });

      // Load historical data first
      priceStore.loadHistoricalData().then(() => {
        // Then connect to WebSocket for real-time updates
        priceStore.connectToLiveData();
      });
    });

    // Clean up when component is unmounted
    onBeforeUnmount(() => {
      priceStore.disconnect();
    });

    // Remove watchers when component is unmounted
    onUnmounted(() => {
      if (timespanWatcher) {
        timespanWatcher(); // Stop the watcher
      }
    });

    return {
      selectedTimespan,
      priceStore,
    };
  },
};
</script>

<style scoped>
.candlestick-chart-container {
  max-width: 900px;
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
</style>
