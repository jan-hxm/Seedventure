import { reactive, shallowRef } from "vue";
import {
  fetchHistoricalData,
  fetchAvailableTimeframes,
} from "../services/priceService.js";
import { calculatePriceChange, timeframes } from "../utils/chartUtils.js";
import { usePriceWebSocket } from "../composables/usePriceWebSocket.js";

// Core configuration
const MAX_CANDLES = 100;
const DEBOUNCE_TIME = 100;

// Create the price websocket composable
const priceWebSocket = usePriceWebSocket();

// Data storage
const candles = shallowRef([]);

// UI state
const state = reactive({
  selectedTimeframe: "1m",
  availableTimeframes: [],
  connectionStatus: "Disconnected",
  dataInfo: "No data loaded",
  currentPrice: "0.00",
  priceChange: "0.00",
  isPositiveChange: false,
  isLoading: true,
  error: null,
});

// Debounce timer
let updateTimer = null;

// Helper functions
function addCandle(newCandle) {
  const updatedCandles = [...candles.value, newCandle];

  // Enforce maximum limit
  if (updatedCandles.length > MAX_CANDLES) {
    updatedCandles.splice(0, updatedCandles.length - MAX_CANDLES);
  }

  candles.value = updatedCandles;
}

function debounceUIUpdate(isTimeframeData = false, timeFrame = null) {
  if (updateTimer) clearTimeout(updateTimer);

  updateTimer = setTimeout(() => {
    // Update price information if candles exist
    if (candles.value.length > 0) {
      const latestCandle = candles.value[candles.value.length - 1];
      const { currentPrice, priceChange, isPositive } =
        calculatePriceChange(latestCandle);

      state.currentPrice = currentPrice;
      state.priceChange = priceChange;
      state.isPositiveChange = isPositive;
    }

    // Update data info
    if (isTimeframeData) {
      state.dataInfo = `Loaded ${candles.value.length} candles for timeframe ${timeFrame}`;
      state.isLoading = false;
    } else {
      state.dataInfo = `Last update: ${new Date().toLocaleTimeString()} (${
        state.selectedTimeframe
      })`;
    }
  }, DEBOUNCE_TIME);
}

// Process WebSocket messages
function processWebSocketMessage(message) {
  // Handle candle updates
  if (message.type && message.timeFrame === state.selectedTimeframe) {
    const { type, candle } = message;
    const formattedCandle = { x: candle.x, y: candle.y };

    // Update or add candle
    if (type === "update") {
      const index = candles.value.findIndex((item) => item.x === candle.x);

      if (index >= 0) {
        // Update existing candle
        const updatedCandles = [...candles.value];
        updatedCandles[index] = formattedCandle;
        candles.value = updatedCandles;
      } else {
        // Add as new candle
        addCandle(formattedCandle);
      }
    } else if (type === "new") {
      addCandle(formattedCandle);
    }

    // Debounce UI updates
    debounceUIUpdate();
  }
  // Handle timeframe data response
  else if (message.timeFrame === state.selectedTimeframe && message.candles) {
    let formattedCandles = message.candles.map((candle) => ({
      x: candle.x,
      y: candle.y,
    }));

    // Limit candle count
    if (formattedCandles.length > MAX_CANDLES) {
      formattedCandles = formattedCandles.slice(-MAX_CANDLES);
    }

    // Update candles
    candles.value = formattedCandles;

    // Debounce UI updates
    debounceUIUpdate(true, message.timeFrame);
  }
}

// Store actions
const actions = {
  // Set timeframe and reload data
  async setTimeframe(timeframe) {
    if (timeframe === state.selectedTimeframe) return;

    state.isLoading = true;
    state.selectedTimeframe = timeframe;

    // Update WebSocket connection if needed
    if (priceWebSocket.isConnected()) {
      if (!priceWebSocket.changeTimeframe(timeframe)) {
        // If changing timeframe failed, reconnect
        priceWebSocket.disconnect();
        setTimeout(actions.connectToLiveData, 300);
      }
    }

    await actions.loadHistoricalData();
  },

  // Load available timeframes
  async loadAvailableTimeframes() {
    try {
      state.availableTimeframes = await fetchAvailableTimeframes();
    } catch (error) {
      console.error("Failed to load timeframes:", error);
      state.availableTimeframes = timeframes.map((tf) => tf.value);
    }
    return state.availableTimeframes;
  },

  // Load historical price data
  async loadHistoricalData() {
    state.isLoading = true;
    state.dataInfo = "Fetching historical data...";
    state.error = null;

    try {
      await fetchHistoricalData(
        // Success callback
        (data) => {
          // Limit size and update data
          candles.value =
            data.length > MAX_CANDLES ? data.slice(-MAX_CANDLES) : data;

          // Update UI state
          state.isLoading = false;
          state.dataInfo = `Loaded ${candles.value.length} candles (${state.selectedTimeframe})`;

          // Update price info if data exists
          if (candles.value.length > 0) {
            const latestCandle = candles.value[candles.value.length - 1];
            const { currentPrice, priceChange, isPositive } =
              calculatePriceChange(latestCandle);

            state.currentPrice = currentPrice;
            state.priceChange = priceChange;
            state.isPositiveChange = isPositive;
          }
        },
        // Error callback
        (error) => {
          state.error = error.message || "Failed to load data";
          state.dataInfo = "Error loading data";
          state.isLoading = false;
        },
        state.selectedTimeframe
      );
    } catch (error) {
      state.error = error.message || "Failed to load data";
      state.dataInfo = "Error loading data";
      state.isLoading = false;
    }
  },

  // Connect to WebSocket for live updates
  connectToLiveData() {
    state.connectionStatus = "Connecting...";

    // Set up connection state syncing
    state.connectionStatus = priceWebSocket.connectionStatus.value;

    // Watch for connection status changes
    const unwatch = (priceWebSocket.connectionStatus.value = (newStatus) => {
      state.connectionStatus = newStatus;
    });

    // Connect to WebSocket
    priceWebSocket.connect(
      {
        onOpen: () => {
          state.connectionStatus = "Connected";
        },
        onMessage: processWebSocketMessage,
        onClose: () => {
          state.connectionStatus = "Disconnected";
        },
        onError: (error) => {
          state.error = error.message || "WebSocket error";
        },
      },
      state.selectedTimeframe
    );
  },

  // Disconnect WebSocket
  disconnect() {
    priceWebSocket.disconnect();
    state.connectionStatus = "Disconnected";

    if (updateTimer) {
      clearTimeout(updateTimer);
      updateTimer = null;
    }
  },
};

// Initialize
actions.loadAvailableTimeframes();

// Export store
export default {
  state,
  getCandles: () => candles.value,
  ...actions,
};
