<template>
  <div class="chart-wrapper">
    <apexchart
      v-if="chartData.length > 0"
      ref="apexChart"
      type="candlestick"
      height="450"
      width="800"
      :options="chartOptions"
      :series="[{ data: chartData }]"
      @mounted="onChartMounted"
    ></apexchart>
    <div v-else class="loading">
      {{ isLoading ? "Loading data from server..." : "No data available" }}
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, onMounted, onBeforeUnmount } from "vue";
import { getChartOptions } from "../utils/chartUtils.js";

// Props definition
const props = defineProps({
  candles: {
    type: Array,
    required: true,
  },
  timeframe: {
    type: String,
    required: true,
  },
  isLoading: {
    type: Boolean,
    default: false,
  },
});

// Chart references
const apexChart = ref(null);
const chart = ref(null);

// Use refs for data to control reactivity better
const chartData = ref([]);
const chartOptions = ref(getChartOptions(props.timeframe));

// Debounce flag to prevent too frequent updates
let debounceTimer = null;

// Create chart data - only run this when necessary
const updateChartData = () => {
  // Cancel any pending updates
  if (debounceTimer) {
    clearTimeout(debounceTimer);
  }

  // Debounce the update
  debounceTimer = setTimeout(() => {
    try {
      chartData.value = props.candles;

      // Update chart options if chart is mounted
      if (chart.value) {
        updateChartOptions();
      }
    } catch (error) {
      console.error("Error updating chart data:", error);
    }
  }, 100); // Small delay to batch updates
};

// Update chart options - use a dedicated function
const updateChartOptions = async () => {
  try {
    if (!chart.value || !chart.value.chart) return;

    const newOptions = getChartOptions(props.timeframe, chartData.value);

    // Only update if we need to
    chart.value.updateOptions(newOptions, false, true, true);
  } catch (error) {
    console.error("Error updating chart options:", error);
  }
};

// Watch for candles changes, but only if reference changes
watch(
  () => props.candles,
  (newCandles) => {
    updateChartData();
  }
);

// Watch for timeframe changes
watch(
  () => props.timeframe,
  (newTimeframe) => {
    try {
      if (!chart.value) return;

      // Update options first
      chartOptions.value = getChartOptions(newTimeframe, chartData.value);

      // Then update the chart
      nextTick(() => {
        updateChartOptions();
      });
    } catch (error) {
      console.error("Error handling timeframe change:", error);
    }
  }
);

// Initialize chart data when mounted
onMounted(() => {
  updateChartData();
});

// Save chart reference when chart is mounted
const onChartMounted = (chartContext) => {
  chart.value = chartContext;

  // Initial update with proper data
  nextTick(() => {
    updateChartOptions();
  });
};

// Clean up when component is unmounted
onBeforeUnmount(() => {
  if (debounceTimer) {
    clearTimeout(debounceTimer);
    debounceTimer = null;
  }
});
</script>

<style scoped>
.chart-wrapper {
  margin: 20px 0;
  height: 450px;
  border-radius: 4px;
  overflow: hidden;
  background-color: #f9f9f9;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  font-size: 16px;
}
</style>
