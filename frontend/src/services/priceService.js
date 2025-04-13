/**
 * API service for price data
 */

// API URL constants
const API_BASE_URL = "http://localhost:8080/api";
const PRICES_ENDPOINT = `${API_BASE_URL}/prices`;

/**
 * Fetch historical price data
 * @param {Function} onSuccess - Success callback
 * @param {Function} onError - Error callback
 * @param {string} timeframe - Timeframe to fetch
 * @returns {Promise} Fetch promise
 */
export async function fetchHistoricalData(
  onSuccess,
  onError,
  timeframe = "1m"
) {
  try {
    const response = await fetch(
      `${PRICES_ENDPOINT}/history?timeframe=${timeframe}`
    );

    if (!response.ok) {
      throw new Error(
        `Failed to fetch data: ${response.status} ${response.statusText}`
      );
    }

    const data = await response.json();

    // Format the data for chart display
    const formattedData = data.candles.map((candle) => ({
      x: candle.x,
      y: candle.y,
    }));

    // Call success callback with formatted data
    if (onSuccess) {
      onSuccess(formattedData);
    }

    return formattedData;
  } catch (error) {
    console.error("Error fetching historical data:", error);

    // Call error callback
    if (onError) {
      onError(error);
    }

    return [];
  }
}

/**
 * Fetch available timeframes
 * @returns {Promise<string[]>} Available timeframes
 */
export async function fetchAvailableTimeframes() {
  try {
    const response = await fetch(`${PRICES_ENDPOINT}/timeframes`);

    if (!response.ok) {
      throw new Error(
        `Failed to fetch timeframes: ${response.status} ${response.statusText}`
      );
    }

    return await response.json();
  } catch (error) {
    console.error("Error fetching timeframes:", error);
    throw error;
  }
}
