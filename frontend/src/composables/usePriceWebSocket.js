import { useWebSocket } from "./useWebSocket.js";

/**
 * A composable for managing price data WebSocket connections
 * @returns {Object} Price WebSocket utilities
 */
export function usePriceWebSocket() {
  // Base WebSocket URL
  const WS_BASE_URL = "ws://localhost:8080/api/prices/live";

  // Create base WebSocket composable
  const websocket = useWebSocket(WS_BASE_URL);

  /**
   * Connect to the price WebSocket
   * @param {Object} handlers - Event handlers
   * @param {string} timeframe - Initial timeframe to connect with
   * @returns {Object} WebSocket connection
   */
  const connectPriceSocket = (handlers, timeframe = "1m") => {
    return websocket.connect(handlers, timeframe);
  };

  /**
   * Change the timeframe subscription
   * @param {string} timeframe - New timeframe
   * @returns {boolean} Success status
   */
  const changeTimeframe = (timeframe) => {
    // First try to send a timeframe change request
    const success = websocket.send({
      type: "changeSubscription",
      timeFrame: timeframe,
    });

    // If sending failed, try changing the path
    if (!success) {
      return websocket.changePath(timeframe);
    }

    return true;
  };

  return {
    // Re-export base WebSocket functionality
    connectionStatus: websocket.connectionStatus,
    error: websocket.error,
    isConnected: websocket.isConnected,
    disconnect: websocket.disconnect,

    // Price-specific functions
    connect: connectPriceSocket,
    changeTimeframe,
  };
}
