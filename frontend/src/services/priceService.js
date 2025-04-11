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
 */
export const fetchHistoricalData = async (onSuccess, onError) => {
  try {
    const response = await fetch(`${API_BASE_URL}/prices/history`);

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data = await response.json();

    // Format the data for ApexCharts
    const formattedData = data.map((candle) => ({
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
 * Connect to WebSocket for live price updates
 * @param {Object} handlers - Object containing event handlers
 */
export const connectWebSocket = (handlers) => {
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

  // Create new WebSocket connection
  ws = new WebSocket(WS_URL);

  // Handle connection open
  ws.onopen = () => {
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
    if (onClose) onClose();

    // Try to reconnect after a delay
    reconnectTimeout = setTimeout(() => {
      connectWebSocket(handlers);
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
