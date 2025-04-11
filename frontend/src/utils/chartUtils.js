/**
 * Format price with dollar sign and 2 decimal places
 * @param {Number} price - Price to format
 * @returns {String} Formatted price
 */
export const formatPrice = (price) => {
  return `$${price.toFixed(2)}`;
};

// Cache for candle data
const candleCache = {
  timeframe: null,
  timestamp: 0,
  result: [],
};

/**
 * Process candles from the backend, with optional light filtering for specific views
 * @param {Array} candles - Candles from the backend
 * @param {String} timeframe - Timeframe of the candles (1m, 5m, 15m, 1h, 4h, 1d)
 * @returns {Array} Processed candle data
 */
export const processCandles = (candles, timeframe) => {
  if (!candles || candles.length === 0) return [];

  const now = Date.now();

  // For large datasets, we might want to sample the data
  // This is a performance optimization only - the backend already has the right resolution
  let processedCandles = candles;

  if (candles.length > 100) {
    // Only sample for extremely large datasets
    processedCandles = sampleLargeDataset(candles);
  }

  // Update cache
  candleCache.timeframe = timeframe;
  candleCache.timestamp = now;
  candleCache.result = processedCandles;

  return processedCandles;
};

/**
 * Sample large datasets for better client-side performance
 * @param {Array} candles - Candles to sample
 * @returns {Array} Sampled candles
 */
function sampleLargeDataset(candles) {
  // Always include the most recent candles
  const recentCount = 100; // Keep more recent candles for accuracy
  const recentCandles = candles.slice(-recentCount);

  // Sample older candles if we have a very large dataset
  if (candles.length <= recentCount) return candles;

  const olderCandles = candles.slice(0, -recentCount);
  const sampleRate = Math.max(1, Math.floor(olderCandles.length / 500));

  const sampledOlderCandles =
    sampleRate > 1
      ? olderCandles.filter((_, i) => i % sampleRate === 0)
      : olderCandles;

  // Combine sampled older candles with recent candles
  return [...sampledOlderCandles, ...recentCandles];
}

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
        enabled: data.length < 100, // Only enable animations for small datasets
        easing: "linear",
        dynamicAnimation: {
          speed: 350,
        },
      },
      toolbar: {
        show: true,
        tools: {
          download: true,
          selection: true,
          zoom: true,
          zoomin: true,
          zoomout: true,
          pan: true,
          reset: true,
        },
      },
      zoom: {
        enabled: true,
      },
    },
    xaxis: {
      type: "datetime",
      labels: {
        formatter: getFormatterForTimeframe(timeframe),
        datetimeUTC: false,
      },
      tickAmount: getTickAmount(timeframe, data),
    },
    yaxis: {
      tooltip: { enabled: true },
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
      theme: "dark",
      x: {
        format: getTooltipFormat(timeframe),
      },
      y: {
        formatter: function (val) {
          return "$" + val.toFixed(2);
        },
      },
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
    const timeRange = getTimeRange(data, timeframe);
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
          month: "short",
          day: "numeric",
        });
      };
  }
}

/**
 * Get tooltip date format for timeframe
 * @param {String} timeframe - Selected timeframe
 * @returns {String} Date format string
 */
function getTooltipFormat(timeframe) {
  switch (timeframe) {
    case "1m":
    case "5m":
    case "15m":
      return "MMM dd, yyyy HH:mm:ss";
    case "1h":
    case "4h":
      return "MMM dd, yyyy HH:mm";
    case "1d":
      return "MMM dd, yyyy";
    default:
      return "MMM dd, yyyy";
  }
}

/**
 * Calculate appropriate tick amount based on timeframe and data size
 * @param {String} timeframe - Selected timeframe
 * @param {Array} data - Chart data
 * @returns {Number} Tick amount
 */
function getTickAmount(timeframe, data) {
  // Default to 6 ticks
  let tickAmount = 6;

  // For large datasets, use fewer ticks
  if (data.length > 200) {
    return 5;
  }

  // For minute-level timeframes, use more ticks
  if (timeframe === "1m" || timeframe === "5m") {
    return 8;
  }

  return tickAmount;
}

/**
 * Calculate appropriate time range for chart
 * @param {Array} data - Chart data
 * @param {String} timeframe - Selected timeframe
 * @returns {Object|null} Time range min and max
 */
function getTimeRange(data, timeframe) {
  if (!data || data.length === 0) return null;

  // Calculate min and max times
  const times = data.map((d) => d.x);
  let minTime = Math.min(...times);
  let maxTime = Math.max(...times);

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
