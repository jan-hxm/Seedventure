import { ref, onUnmounted } from "vue";

/**
 * A composable for managing WebSocket connections
 * @param {string} baseUrl - The base WebSocket URL
 * @returns {Object} WebSocket utilities
 */
export function useWebSocket(baseUrl) {
  const connectionStatus = ref("disconnected");
  const error = ref(null);
  let socket = null;
  let handlers = {};

  /**
   * Connect to a WebSocket with the given path and callbacks
   * @param {Object} options - Connection options
   * @param {Function} options.onOpen - Called when connection opens
   * @param {Function} options.onMessage - Called with each message
   * @param {Function} options.onClose - Called when connection closes
   * @param {Function} options.onError - Called on error
   * @param {string} path - URL path to connect to
   * @returns {WebSocket} The WebSocket instance
   */
  const connect = (options, path = "") => {
    // Close existing connection if any
    disconnect();

    // Reset error
    error.value = null;

    try {
      // Construct WebSocket URL
      const url = path ? `${baseUrl}/${path}` : baseUrl;

      // Create WebSocket
      socket = new WebSocket(url);
      connectionStatus.value = "connecting";

      // Save handlers for potential reconnection
      handlers = { ...options };

      // Set up event handlers
      socket.onopen = () => {
        connectionStatus.value = "connected";
        if (options.onOpen) options.onOpen();
      };

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (options.onMessage) options.onMessage(data);
        } catch (err) {
          console.error("Error parsing WebSocket message:", err);
          if (options.onError) options.onError(err);
        }
      };

      socket.onclose = (event) => {
        connectionStatus.value = "disconnected";
        if (options.onClose) options.onClose(event);
      };

      socket.onerror = (err) => {
        error.value = err;
        connectionStatus.value = "error";
        if (options.onError) options.onError(err);
      };

      return socket;
    } catch (err) {
      error.value = err;
      connectionStatus.value = "error";
      if (options.onError) options.onError(err);
      return null;
    }
  };

  /**
   * Send data through the WebSocket
   * @param {Object|string} data - Data to send
   * @returns {boolean} Success status
   */
  const send = (data) => {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return false;
    }

    try {
      const message = typeof data === "string" ? data : JSON.stringify(data);
      socket.send(message);
      return true;
    } catch (err) {
      console.error("Error sending WebSocket message:", err);
      error.value = err;
      return false;
    }
  };

  /**
   * Change the WebSocket path without full reconnection when possible
   * @param {string} newPath - New path to connect to
   * @returns {boolean} Whether the change was successful
   */
  const changePath = (newPath) => {
    // If we can just send a message to change state, do that
    if (socket && socket.readyState === WebSocket.OPEN) {
      try {
        // Try to send a request to change the subscription
        send({ type: "changeSubscription", path: newPath });
        return true;
      } catch (err) {
        // If sending fails, reconnect instead
        connect(handlers, newPath);
        return false;
      }
    } else {
      // If not connected, reconnect with new path
      connect(handlers, newPath);
      return false;
    }
  };

  /**
   * Disconnect the WebSocket
   */
  const disconnect = () => {
    if (socket) {
      // Only try to close if not already closed
      if (socket.readyState !== WebSocket.CLOSED) {
        socket.close();
      }
      socket = null;
    }
    connectionStatus.value = "disconnected";
  };

  /**
   * Check if the WebSocket is currently connected
   * @returns {boolean} Connection status
   */
  const isConnected = () => {
    return socket && socket.readyState === WebSocket.OPEN;
  };

  // Clean up on component unmount
  onUnmounted(() => {
    disconnect();
  });

  return {
    connectionStatus,
    error,
    connect,
    disconnect,
    send,
    changePath,
    isConnected,
  };
}
