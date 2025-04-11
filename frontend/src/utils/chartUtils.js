/**
 * Format price with dollar sign and 2 decimal places
 * @param {Number} price - Price to format
 * @returns {String} Formatted price
 */
export const formatPrice = (price) => {
  return `$${price.toFixed(2)}`;
};

// Cache for timespan filter calculations
const filterCache = {
  timespan: null,
  timestamp: 0,
  result: [],
};

/**
 * Filter candle data based on selected timespan
 * @param {Array} allCandles - All candle data
 * @param {String} timespan - Selected timespan (1h, 1d, 1w, 1m, all)
 * @returns {Array} Filtered candle data
 */
export const filterCandlesByTimespan = (allCandles, timespan) => {
  if (!allCandles || allCandles.length === 0) return [];

  const now = Date.now();

  // Check if we can use cached results (if same timespan and < 5 sec old)
  if (timespan === filterCache.timespan && now - filterCache.timestamp < 5000) {
    return filterCache.result;
  }

  // Limit the number of candles to process for better performance
  const maxCandles = 5000;
  const candles =
    allCandles.length > maxCandles ? allCandles.slice(-maxCandles) : allCandles;

  let filteredCandles = [];

  // Apply filter based on timespan
  switch (timespan) {
    case "1h":
      // Last hour - use efficient filtering
      filteredCandles = filterByTimeThreshold(candles, now - 60 * 60 * 1000);
      break;
    case "1d":
      // Last day
      filteredCandles = filterByTimeThreshold(
        candles,
        now - 24 * 60 * 60 * 1000
      );
      break;
    case "1w":
      // Last week - sample data for performance
      filteredCandles = sampleDataForTimespan(
        filterByTimeThreshold(candles, now - 7 * 24 * 60 * 60 * 1000),
        timespan
      );
      break;
    case "1m":
      // Last month - sample data for performance
      filteredCandles = sampleDataForTimespan(
        filterByTimeThreshold(candles, now - 30 * 24 * 60 * 60 * 1000),
        timespan
      );
      break;
    case "all":
    default:
      // All data - sample for large datasets
      filteredCandles = sampleDataForTimespan(candles, timespan);
      break;
  }

  // Ensure we have at least some data to display
  if (filteredCandles.length === 0 && candles.length > 0) {
    // If no data in the timespan, show the most recent candles
    filteredCandles = candles.slice(-10);
  }

  // Update cache
  filterCache.timespan = timespan;
  filterCache.timestamp = now;
  filterCache.result = filteredCandles;

  return filteredCandles;
};

/**
 * Efficiently filter candles by timestamp threshold
 * @param {Array} candles - Candles to filter
 * @param {Number} threshold - Timestamp threshold
 * @returns {Array} Filtered candles
 */
function filterByTimeThreshold(candles, threshold) {
  // Use binary search to find starting index for better performance
  let start = 0;
  let end = candles.length - 1;
  let mid = 0;

  // Only do binary search if we have a sorted array with enough elements
  if (candles.length > 100) {
    while (start <= end) {
      mid = Math.floor((start + end) / 2);

      if (candles[mid].x < threshold) {
        start = mid + 1;
      } else {
        end = mid - 1;
      }
    }

    // Return slice from the found position
    return candles.slice(Math.max(0, start - 1));
  }

  // For smaller arrays, regular filter is fine
  return candles.filter((candle) => candle.x >= threshold);
}

/**
 * Sample data to reduce points for better performance
 * @param {Array} candles - Candles to sample
 * @param {String} timespan - Current timespan
 * @returns {Array} Sampled candles
 */
function sampleDataForTimespan(candles, timespan) {
  // For small datasets, no sampling needed
  if (candles.length < 100) return candles;

  // Determine sample rate based on timespan and data size
  let sampleRate = 1;

  if (timespan === "all") {
    if (candles.length > 1000) sampleRate = Math.floor(candles.length / 500);
  } else if (timespan === "1m") {
    if (candles.length > 500) sampleRate = Math.floor(candles.length / 300);
  } else if (timespan === "1w") {
    if (candles.length > 300) sampleRate = Math.floor(candles.length / 200);
  }

  // Always include the most recent candles
  const recentCount = 20;
  const recentCandles = candles.slice(-recentCount);

  // Sample older candles
  const olderCandles = candles.slice(0, -recentCount);
  const sampledOlderCandles =
    sampleRate > 1
      ? olderCandles.filter((_, i) => i % sampleRate === 0)
      : olderCandles;

  // Combine sampled older candles with recent candles
  return [...sampledOlderCandles, ...recentCandles];
}

/**
 * Get optimized chart options
 * @param {String} timespan - Selected timespan
 * @param {Array} data - Filtered data for the timespan
 * @returns {Object} Chart options
 */
export const getChartOptionsForTimespan = (timespan, data = []) => {
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
    title: {
      text: `Live Crypto Price Movement (${timespan.toUpperCase()})`,
      align: "center",
    },
    xaxis: {
      type: "datetime",
      labels: {
        formatter: getFormatterForTimespan(timespan),
        datetimeUTC: false,
      },
      tickAmount: getTickAmountForTimespan(timespan, data),
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
        format: "MMM dd, yyyy HH:mm",
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
    // Get appropriate time range
    const timeRange = getTimeRangeForData(data, timespan);
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
 * Get appropriate X-axis formatter for timespan
 * @param {String} timespan - Selected timespan
 * @returns {Function} Formatter function
 */
function getFormatterForTimespan(timespan) {
  switch (timespan) {
    case "1h":
      return function (val) {
        return new Date(val).toLocaleTimeString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
        });
      };
    case "1d":
      return function (val) {
        return new Date(val).toLocaleTimeString("en-US", {
          hour: "2-digit",
          minute: "2-digit",
        });
      };
    case "1w":
    case "1m":
      return function (val) {
        return new Date(val).toLocaleDateString("en-US", {
          month: "short",
          day: "numeric",
          hour: "2-digit",
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
 * Calculate appropriate tick amount based on timespan and data size
 * @param {String} timespan - Selected timespan
 * @param {Array} data - Chart data
 * @returns {Number} Tick amount
 */
function getTickAmountForTimespan(timespan, data) {
  // Default to 6 ticks
  let tickAmount = 6;

  // For large datasets, use fewer ticks
  if (data.length > 200) {
    return 5;
  }

  // For hourly view, use more ticks
  if (timespan === "1h") {
    return 8;
  }

  return tickAmount;
}

/**
 * Calculate appropriate time range for chart
 * @param {Array} data - Chart data
 * @param {String} timespan - Selected timespan
 * @returns {Object|null} Time range min and max
 */
function getTimeRangeForData(data, timespan) {
  if (!data || data.length === 0) return null;

  // Calculate min and max times
  const times = data.map((d) => d.x);
  let minTime = Math.min(...times);
  let maxTime = Math.max(...times);

  // Add padding based on timespan
  let padding = 0;
  switch (timespan) {
    case "1h":
      padding = 60 * 1000; // 1 minute
      break;
    case "1d":
      padding = 15 * 60 * 1000; // 15 minutes
      break;
    case "1w":
      padding = 6 * 60 * 60 * 1000; // 6 hours
      break;
    case "1m":
    case "all":
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

// Available timespan options
export const timespans = [
  { label: "1H", value: "1h" },
  { label: "1D", value: "1d" },
  { label: "1W", value: "1w" },
  { label: "1M", value: "1m" },
  { label: "All", value: "all" },
];
