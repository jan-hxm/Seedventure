import { reactive, shallowRef } from "vue";
import {
  fetchHistoricalData,
  connectWebSocket,
  disconnectWebSocket,
  changeTimeframe as wsChangeTimeframe,
  fetchAvailableTimeframes,
} from "../services/priceService.js";
import {
  processCandles,
  calculatePriceChange,
  timeframes,
} from "../utils/chartUtils.js";

// Use shallowRef for large arrays to prevent deep reactivity
const candles = shallowRef([]);

// Only make UI state reactive, not the entire data set
const state = reactive({
  selectedTimeframe: "1m", // Default to 1-minute candles
  availableTimeframes: [],
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
   * Set the selected timeframe and load data
   */
  async setTimeframe(timeframe) {
    // Only change if different
    if (timeframe === state.selectedTimeframe) return;

    const oldTimeframe = state.selectedTimeframe;
    state.selectedTimeframe = timeframe;

    // Update current WebSocket if connected
    if (state.connectionStatus === "Connected") {
      const success = wsChangeTimeframe(timeframe);
      if (!success) {
        // If changing timeframe failed, reconnect with new timeframe
        actions.disconnect();
        actions.connectToLiveData();
      }
    }

    // Always reload data when timeframe changes
    await actions.loadHistoricalData();
  },

  /**
   * Load available timeframes from API
   */
  async loadAvailableTimeframes() {
    try {
      const availableTimeframes = await fetchAvailableTimeframes();
      state.availableTimeframes = availableTimeframes;
      return availableTimeframes;
    } catch (error) {
      console.error("Failed to load available timeframes:", error);
      // Fall back to default timeframes from chartUtils
      state.availableTimeframes = timeframes.map((tf) => tf.value);
      return state.availableTimeframes;
    }
  },

  /**
   * Get the candles for components
   */
  getCandles() {
    return candles.value;
  },

  /**
   * Load historical data from API
   */
  async loadHistoricalData() {
    state.isLoading = true;
    state.dataInfo = "Fetching historical data...";
    state.error = null;

    try {
      // Calculate appropriate limit based on timeframe
      const limit = calculateLimitForTimeframe(state.selectedTimeframe);

      const data = await fetchHistoricalData(
        // Success callback
        (formattedData) => {
          // Use shallowRef for better performance
          candles.value = formattedData;

          // Update UI state
          state.isLoading = false;
          state.dataInfo = `Loaded ${formattedData.length} candles (${state.selectedTimeframe})`;

          // Update price info based on the latest candle if available
          if (formattedData.length > 0) {
            const latestCandle = formattedData[formattedData.length - 1];
            const priceInfo = calculatePriceChange(latestCandle);

            state.currentPrice = priceInfo.currentPrice;
            state.priceChange = priceInfo.priceChange;
            state.isPositiveChange = priceInfo.isPositive;
          }
        },
        // Error callback
        (error) => {
          state.error = error.message || "Failed to load historical data";
          state.dataInfo = "Error loading historical data";
          state.isLoading = false;
        },
        state.selectedTimeframe,
        limit
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

    ws = connectWebSocket(
      {
        onOpen: () => {
          state.connectionStatus = "Connected";
        },
        onMessage: (message) => {
          // The message can now be either an UpdateMessage or a TimeFrameData response

          // Check if this is a candle update
          if (
            message.type &&
            (message.type === "new" || message.type === "update")
          ) {
            const { type, candle, timeFrame } = message;

            // Only process updates for the selected timeframe
            if (timeFrame === state.selectedTimeframe) {
              // Format the candle data for ApexCharts
              const formattedCandle = {
                x: candle.x,
                y: candle.y,
              };

              // Create a new array only when necessary
              const currentCandles = [...candles.value];

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

              // Batch update the candles reference to reduce reactivity overhead
              candles.value = currentCandles;

              // Update price info with the latest candle
              const priceInfo = calculatePriceChange(formattedCandle);
              state.currentPrice = priceInfo.currentPrice;
              state.priceChange = priceInfo.priceChange;
              state.isPositiveChange = priceInfo.isPositive;

              // Update data info
              state.dataInfo = `Last update: ${new Date().toLocaleTimeString()} (${
                state.selectedTimeframe
              })`;
            }
          }
          // Check if this is a timeframe data response (from changing timeframes)
          else if (message.timeFrame && message.candles) {
            // This is a response to changing timeframes
            const { timeFrame, candles: newCandles } = message;

            // Only process if this matches our current timeframe
            if (timeFrame === state.selectedTimeframe) {
              // Format the candles for ApexCharts
              const formattedCandles = newCandles.map((candle) => ({
                x: candle.x,
                y: candle.y,
              }));

              // Update candles
              candles.value = formattedCandles;

              // Update UI with latest candle if available
              if (formattedCandles.length > 0) {
                const latestCandle =
                  formattedCandles[formattedCandles.length - 1];
                const priceInfo = calculatePriceChange(latestCandle);
                state.currentPrice = priceInfo.currentPrice;
                state.priceChange = priceInfo.priceChange;
                state.isPositiveChange = priceInfo.isPositive;
              }

              state.dataInfo = `Loaded ${formattedCandles.length} candles for timeframe ${timeFrame}`;
            }
          }
        },
        onClose: () => {
          state.connectionStatus = "Disconnected";
        },
        onError: (error) => {
          state.error = error.message || "WebSocket connection error";
        },
      },
      state.selectedTimeframe
    );
  },

  /**
   * Disconnect from WebSocket
   */
  disconnect() {
    disconnectWebSocket();
    state.connectionStatus = "Disconnected";
  },

  /**
   * Process candles for display (with optimizations)
   */
  processCandles() {
    return processCandles(candles.value, state.selectedTimeframe);
  },
};

/**
 * Calculate appropriate limit based on timeframe
 * @param {String} timeframe - Selected timeframe
 * @returns {Number} Limit value
 */
function calculateLimitForTimeframe(timeframe) {
  switch (timeframe) {
    case "1m":
      return 120; // 2 hours of 1-minute candles
    case "5m":
      return 144; // 12 hours of 5-minute candles
    case "15m":
      return 192; // 2 days of 15-minute candles
    case "1h":
      return 168; // 7 days of hourly candles
    case "4h":
      return 180; // 30 days of 4-hour candles
    case "1d":
      return 365; // 1 year of daily candles
    default:
      return 100; // Default limit
  }
}

// Initialize the store when it's created
actions.loadAvailableTimeframes();

// Create and export the store
export default {
  state: state,
  ...actions,
};
