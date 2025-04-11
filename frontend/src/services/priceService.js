// API and WebSocket configuration
const API_BASE_URL = "http://localhost:8080/api";
const WS_URL = "ws://localhost:8080/api/prices/live";

// WebSocket connection
let ws = null;
let reconnectTimeout = null;

/**
 * Fetch historical price data from the server
 * @param {Function} onSuccess - Callback for successful data fetch
 * @param {Function} onError - Callback for errors
 * @param {String} timeframe - Timeframe for candles (1m, 5m, 15m, 1h, 4h, 1d)
 * @param {Number} limit - Maximum number of candles to fetch
 * @param {Number} from - Start timestamp (optional)
 * @param {Number} to - End timestamp (optional)
 */
export const fetchHistoricalData = async (
  onSuccess,
  onError,
  timeframe = "1d",
  limit = 100,
  from = null,
  to = null
) => {
  try {
    // Build query parameters
    const params = new URLSearchParams();
    params.append("timeframe", timeframe);
    params.append("limit", limit.toString());

    if (from !== null) {
      params.append("from", from.toString());
    }

    if (to !== null) {
      params.append("to", to.toString());
    }

    const response = await fetch(
      `${API_BASE_URL}/prices/history?${params.toString()}`
    );

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data = await response.json();

    // Format the data for ApexCharts
    // Note: The API now returns a different structure with timeFrameData
    const formattedData = data.candles.map((candle) => ({
      x: candle.x,
      y: candle.y,
    }));

    console.log("Formatted historical data:", formattedData);
    onSuccess(formattedData);

    return formattedData;
  } catch (error) {
    console.error("Error fetching historical data:", error);
    if (onError) onError(error);
    return [];
  }
};

/**
 * Fetch available timeframes from the server
 * @returns {Promise<Array>} Array of available timeframes
 */
export const fetchAvailableTimeframes = async () => {
  try {
    const response = await fetch(`${API_BASE_URL}/prices/timeframes`);

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    return await response.json();
  } catch (error) {
    console.error("Error fetching available timeframes:", error);
    return ["1m", "5m", "15m", "1h", "4h", "1d"]; // Default fallback
  }
};

/**
 * Connect to WebSocket for live price updates
 * @param {Object} handlers - Object containing event handlers
 * @param {String} timeframe - Timeframe for candles (1m, 5m, 15m, 1h, 4h, 1d)
 */
export const connectWebSocket = (handlers, timeframe = "5m") => {
  const { onOpen, onMessage, onClose, onError } = handlers;

  // Clear any existing reconnect timeouts
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  // Close existing connection if any
  if (
    ws &&
    (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)
  ) {
    ws.close();
  }

  // Create new WebSocket connection with timeframe in the path
  ws = new WebSocket(`${WS_URL}/${timeframe}`);

  // Handle connection open
  ws.onopen = () => {
    console.log(`WebSocket connected (${timeframe})`);
    if (onOpen) onOpen();
  };

  // Handle incoming messages
  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      if (onMessage) onMessage(message);
    } catch (error) {
      console.error("Error processing WebSocket message:", error);
    }
  };

  // Handle connection close
  ws.onclose = () => {
    console.log("WebSocket disconnected");
    if (onClose) onClose();

    // Try to reconnect after a delay
    reconnectTimeout = setTimeout(() => {
      connectWebSocket(handlers, timeframe);
    }, 5000);
  };

  // Handle connection errors
  ws.onerror = (error) => {
    console.error("WebSocket error:", error);
    if (onError) onError(error);
  };

  return ws;
};

/**
 * Change the timeframe for an existing WebSocket connection
 * @param {String} timeframe - New timeframe to subscribe to
 */
export const changeTimeframe = (timeframe) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    // Send request to change timeframe
    ws.send(
      JSON.stringify({
        timeFrame: timeframe,
      })
    );
    console.log(`Changed WebSocket timeframe to ${timeframe}`);
    return true;
  }
  return false;
};

/**
 * Disconnect WebSocket
 */
export const disconnectWebSocket = () => {
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }

  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.close();
  }
};
