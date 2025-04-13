/**
 * Format price with dollar sign and 2 decimal places
 * @param {Number} price - Price to format
 * @returns {String} Formatted price
 */
export const formatPrice = (price) => {
  return `$${price.toFixed(2)}`;
};

/**
 * Get chart options for the selected timeframe
 * @param {String} timeframe - Selected timeframe (1m, 5m, 15m, 1h, 4h, 1d)
 * @param {Array} data - Candle data
 * @returns {Object} Chart options
 */
export const getChartOptions = (timeframe, data = []) => {
  // Base chart options - use minimal animations for better performance
  const baseOptions = {
    chart: {
      type: "candlestick",
      height: 450,
      fontFamily: "Helvetica, Arial, sans-serif",
      background: "#f8f9fa",
      animations: {
        enabled: true,
        easing: "linear",
        dynamicAnimation: {
          speed: 350,
        },
      },
      toolbar: {
        show: false,
      },
      zoom: {
        enabled: false,
      },
    },
    xaxis: {
      type: "datetime",
      labels: {
        formatter: getFormatterForTimeframe(timeframe),
        datetimeUTC: false,
      },
      tickAmount: 5,
    },
    yaxis: {
      tooltip: {
        enabled: false,
      },
      labels: {
        formatter: function (val) {
          return "$" + val.toFixed(2);
        },
      },
      forceNiceScale: true,
    },
    plotOptions: {
      candlestick: {
        colors: {
          upward: "#26a69a",
          downward: "#ef5350",
        },
        wick: { useFillColor: true },
      },
    },
    tooltip: {
      enabled: false,
    },
    grid: {
      borderColor: "#e0e0e0",
      strokeDashArray: 5,
    },
    responsive: [
      {
        breakpoint: 1000,
        options: {
          chart: {
            width: "100%",
          },
        },
      },
    ],
    // Add performance optimizations
    dataLabels: {
      enabled: false,
    },
    fill: {
      opacity: 1,
    },
  };

  // Only add time range if we have data
  if (data.length > 0) {
    // Calculate appropriate time range with padding
    const minMaxTimes = getMinMaxTimes(data);
    const timeRange = getTimeRange(minMaxTimes, timeframe);
    if (timeRange) {
      baseOptions.xaxis = {
        ...baseOptions.xaxis,
        min: timeRange.min,
        max: timeRange.max,
      };
    }
  }

  return baseOptions;
};

/**
 * Get min and max times from data efficiently
 * @param {Array} data - Candle data
 * @returns {Object} Min and max times
 */
function getMinMaxTimes(data) {
  if (!data || data.length === 0) {
    return { min: 0, max: 0 };
  }

  let min = data[0].x;
  let max = data[0].x;

  // Manual iteration is more efficient than using Math.min/max with spread
  for (let i = 1; i < data.length; i++) {
    const timestamp = data[i].x;
    if (timestamp < min) min = timestamp;
    if (timestamp > max) max = timestamp;
  }

  return { min, max };
}

/**
 * Get appropriate X-axis formatter for timeframe
 * @param {String} timeframe - Selected timeframe
 * @returns {Function} Formatter function
 */
function getFormatterForTimeframe(timeframe) {
  switch (timeframe) {
    case "1m":
    case "5m":
      return function (val) {
        return new Date(val).toLocaleTimeString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
        });
      };
    case "15m":
    case "1h":
      return function (val) {
        return new Date(val).toLocaleTimeString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
        });
      };
    case "4h":
      return function (val) {
        return new Date(val).toLocaleDateString("en-US", {
          month: "short",
          day: "numeric",
          hour: "2-digit",
        });
      };
    case "1d":
      return function (val) {
        return new Date(val).toLocaleDateString("en-US", {
          month: "short",
          day: "numeric",
        });
      };
    default:
      return function (val) {
        return new Date(val).toLocaleDateString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
        });
      };
  }
}

/**
 * Calculate appropriate time range for chart
 * @param {Object} minMaxTimes - Min and max times
 * @param {String} timeframe - Selected timeframe
 * @returns {Object|null} Time range min and max
 */
function getTimeRange(minMaxTimes, timeframe) {
  if (!minMaxTimes || minMaxTimes.min === 0) return null;

  let minTime = minMaxTimes.min;
  let maxTime = minMaxTimes.max;

  // Add padding based on timeframe
  let padding = 0;
  switch (timeframe) {
    case "1m":
      padding = 60 * 1000; // 1 minute
      break;
    case "5m":
      padding = 5 * 60 * 1000; // 5 minutes
      break;
    case "15m":
      padding = 15 * 60 * 1000; // 15 minutes
      break;
    case "1h":
      padding = 60 * 60 * 1000; // 1 hour
      break;
    case "4h":
      padding = 4 * 60 * 60 * 1000; // 4 hours
      break;
    case "1d":
      padding = 24 * 60 * 60 * 1000; // 1 day
      break;
  }

  // Add padding to max (future)
  maxTime += padding;

  return { min: minTime, max: maxTime };
}

/**
 * Calculate price change information from a candle
 * @param {Object} candle - Candle data
 * @returns {Object} Price change information
 */
export const calculatePriceChange = (candle) => {
  if (!candle || !candle.y) {
    return {
      currentPrice: "$0.00",
      priceChange: "$0.00 (0.00%)",
      isPositive: false,
    };
  }

  const closePrice = candle.y[3];
  const openPrice = candle.y[0];

  // Calculate and format price change
  const change = closePrice - openPrice;
  const changePercent = (change / openPrice) * 100;
  let priceChangeText;

  if (Math.abs(change) > 0.001) {
    const sign = change > 0 ? "+" : "";
    priceChangeText = `${sign}$${change.toFixed(
      2
    )} (${sign}${changePercent.toFixed(2)}%)`;
  } else {
    priceChangeText = "$0.00 (0.00%)";
  }

  return {
    currentPrice: formatPrice(closePrice),
    priceChange: priceChangeText,
    isPositive: change > 0,
  };
};

// Available timeframes directly from the backend
export const timeframes = [
  { label: "1 Min", value: "1m" },
  { label: "5 Min", value: "5m" },
  { label: "15 Min", value: "15m" },
  { label: "1 Hour", value: "1h" },
  { label: "4 Hour", value: "4h" },
  { label: "1 Day", value: "1d" },
];
