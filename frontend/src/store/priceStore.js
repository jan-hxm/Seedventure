import { reactive, readonly, shallowRef } from "vue";
import {
  fetchHistoricalData,
  connectWebSocket,
  disconnectWebSocket,
} from "../services/priceService.js";
import {
  filterCandlesByTimespan,
  calculatePriceChange,
} from "../utils/chartUtils.js";

// Use shallowRef for large arrays to prevent deep reactivity
const allCandles = shallowRef([]);
const filteredCandles = shallowRef([]);

// Only make UI state reactive, not the entire data set
const state = reactive({
  selectedTimespan: "1h",
  connectionStatus: "Disconnected",
  dataInfo: "No data loaded",
  currentPrice: "$0.00",
  priceChange: "$0.00",
  isPositiveChange: false,
  isLoading: true,
  error: null,
});

// Use a non-reactive variable for websocket reference
let ws = null;

// Actions
const actions = {
  /**
   * Set the selected timespan and filter data
   */
  setTimespan(timespan) {
    state.selectedTimespan = timespan;

    // Use setTimeout to avoid blocking the UI thread
    setTimeout(() => {
      filteredCandles.value = filterCandlesByTimespan(
        allCandles.value,
        timespan
      );
    }, 0);
  },

  /**
   * Get the filtered candles (for components)
   */
  getFilteredCandles() {
    return filteredCandles.value;
  },

  /**
   * Load historical data from API
   */
  async loadHistoricalData() {
    state.isLoading = true;
    state.dataInfo = "Fetching historical data...";
    state.error = null;

    try {
      const data = await fetchHistoricalData(
        // Success callback
        (formattedData) => {
          // Use batch updates to reduce reactivity overhead
          allCandles.value = formattedData;

          // Apply timespan filter - non-blocking
          setTimeout(() => {
            filteredCandles.value = filterCandlesByTimespan(
              formattedData,
              state.selectedTimespan
            );

            // Update UI state
            state.isLoading = false;
            state.dataInfo = `Loaded ${formattedData.length} historical candles`;

            // Update price info based on the latest candle if available
            if (formattedData.length > 0) {
              const latestCandle = formattedData[formattedData.length - 1];
              const priceInfo = calculatePriceChange(latestCandle);

              state.currentPrice = priceInfo.currentPrice;
              state.priceChange = priceInfo.priceChange;
              state.isPositiveChange = priceInfo.isPositive;
            }
          }, 0);
        },
        // Error callback
        (error) => {
          state.error = error.message || "Failed to load historical data";
          state.dataInfo = "Error loading historical data";
          state.isLoading = false;
        }
      );

      return data;
    } catch (error) {
      state.error = error.message || "Failed to load historical data";
      state.dataInfo = "Error loading historical data";
      state.isLoading = false;
      return [];
    }
  },

  /**
   * Connect to WebSocket for live updates
   */
  connectToLiveData() {
    state.connectionStatus = "Connecting...";

    ws = connectWebSocket({
      onOpen: () => {
        state.connectionStatus = "Connected";
      },
      onMessage: (message) => {
        const { type, candle } = message;

        // Format the candle data for ApexCharts
        const formattedCandle = {
          x: candle.x,
          y: candle.y,
        };

        // Create a new array only when necessary, using non-reactive operations
        const currentCandles = [...allCandles.value];

        // Update an existing candle or add a new one
        if (type === "update") {
          const existingIndex = currentCandles.findIndex(
            (item) => item.x === candle.x
          );

          if (existingIndex >= 0) {
            // Update existing candle
            currentCandles[existingIndex] = formattedCandle;
          } else {
            // Add new candle if not found
            currentCandles.push(formattedCandle);
          }
        } else if (type === "new") {
          // Add new candle
          currentCandles.push(formattedCandle);
        }

        // Batch update the allCandles reference to reduce reactivity overhead
        allCandles.value = currentCandles;

        // Re-apply timespan filter in a non-blocking way
        setTimeout(() => {
          filteredCandles.value = filterCandlesByTimespan(
            currentCandles,
            state.selectedTimespan
          );
        }, 0);

        // Update price info with the latest candle
        const priceInfo = calculatePriceChange(formattedCandle);
        state.currentPrice = priceInfo.currentPrice;
        state.priceChange = priceInfo.priceChange;
        state.isPositiveChange = priceInfo.isPositive;

        // Update data info
        state.dataInfo = `Last update: ${new Date().toLocaleTimeString()}`;
      },
      onClose: () => {
        state.connectionStatus = "Disconnected";
      },
      onError: (error) => {
        state.error = error.message || "WebSocket connection error";
      },
    });
  },

  /**
   * Disconnect from WebSocket
   */
  disconnect() {
    disconnectWebSocket();
    state.connectionStatus = "Disconnected";
  },
};

// Create and export the store
export default {
  state: readonly(state),
  ...actions,
};
