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
    <div v-else class="loading">Loading data from server...</div>
  </div>
</template>

<script>
// Use normal script setup to have better control over reactivity
import { ref, watch, nextTick, onMounted, onBeforeUnmount } from "vue";
import { getChartOptionsForTimespan } from "../utils/chartUtils.js";

export default {
  props: {
    candles: {
      type: Array,
      required: true,
    },
    timespan: {
      type: String,
      required: true,
    },
    isLoading: {
      type: Boolean,
      default: false,
    },
  },

  setup(props) {
    // Chart references
    const apexChart = ref(null);
    const chart = ref(null);

    // Use refs for data to control reactivity better
    const chartData = ref([]);
    const chartOptions = ref(getChartOptionsForTimespan(props.timespan));

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
        // No need to create a deep reactive chain, just transform the data
        chartData.value = props.candles.map((candle) => ({
          x: candle.x,
          y: candle.y,
        }));

        // Update chart options if chart is mounted
        if (chart.value) {
          updateChartOptions();
        }
      }, 50); // Small delay to batch updates
    };

    // Update chart options - use a dedicated function
    const updateChartOptions = () => {
      if (!chart.value || !chart.value.chart) return;

      const newOptions = getChartOptionsForTimespan(
        props.timespan,
        chartData.value
      );

      // Only update if we need to
      chart.value.updateOptions(newOptions, false, true, true);
    };

    // Watch for candles changes, but only if reference changes
    watch(
      () => props.candles,
      (newCandles) => {
        updateChartData();
      }
    );

    // Watch for timespan changes
    watch(
      () => props.timespan,
      (newTimespan) => {
        if (!chart.value) return;

        // Update options first
        chartOptions.value = getChartOptionsForTimespan(
          newTimespan,
          chartData.value
        );

        // Then update the chart
        nextTick(() => {
          updateChartOptions();
        });
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

    return {
      apexChart,
      chartData,
      chartOptions,
      onChartMounted,
    };
  },
};
</script>

<style scoped>
.chart-wrapper {
  margin: 20px 0;
  height: 450px;
  border-radius: 4px;
  overflow: hidden;
  background-color: #f9f9f9;
}

.loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
}
</style>
