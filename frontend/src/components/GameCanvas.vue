<script setup>
import { ref, onMounted, onUnmounted } from "vue";
import { Chart, registerables } from "chart.js";
Chart.register(...registerables);

const chartCanvas = ref(null);
const chart = ref(null);
const priceHistory = ref([]);
const updateInterval = ref(null);

const initChart = () => {
  console.log("Initializing chart with data:", priceHistory.value);
  const ctx = chartCanvas.value.getContext("2d");
  chart.value = new Chart(ctx, {
    type: "line",
    data: {
      labels: priceHistory.value.map((_, i) => i),
      datasets: [
        {
          label: "Stock Price",
          data: priceHistory.value,
          borderColor: "rgb(75, 192, 192)",
          tension: 0.1,
        },
      ],
    },
    options: {
      responsive: true,
      animation: false,
      scales: {
        y: {
          beginAtZero: false,
        },
      },
    },
  });
};

const updateChart = (newPrice) => {
  priceHistory.value.push(newPrice);
  chart.value.data.labels = priceHistory.value.map((_, i) => i);
  chart.value.data.datasets[0].data = priceHistory.value;
  chart.value.update();
};

const fetchHistoricalData = async () => {
  try {
    const response = await fetch("http://localhost:8080/history");
    const data = await response.json();
    priceHistory.value = data;
    initChart();
  } catch (error) {
    console.error("Error fetching historical data:", error);
  }
};

const startPriceUpdates = () => {
  updateInterval.value = setInterval(async () => {
    try {
      const response = await fetch("http://localhost:8080/current");
      const data = await response.json();
      updateChart(data.currentPrice);
    } catch (error) {
      console.error("Error fetching current price:", error);
    }
  }, 1000);
};

onMounted(async () => {
  await fetchHistoricalData();
  startPriceUpdates();
});

onUnmounted(() => {
  if (updateInterval.value) {
    clearInterval(updateInterval.value);
  }
  if (chart.value) {
    chart.value.destroy();
  }
});
</script>

<template>
  <div class="chart-container">
    <canvas ref="chartCanvas"></canvas>
  </div>
</template>

<style scoped>
.chart-container {
  width: 800px;
  height: 400px;
  margin: 0 auto;
}
</style>
