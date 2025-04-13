<template>
  <div class="betting-component">
    <h1>Bets</h1>
    <p>Cash: ${{ formatNumber(cash) }}</p>
    <p>Current Price: {{ currentPrice }}</p>
    <p v-if="message" class="message">{{ message }}</p>

    <div>
      <label for="betAmount">Bet Amount:</label>
      <input
        type="number"
        id="betAmount"
        v-model.number="betAmount"
        :max="cash"
        placeholder="Enter your bet"
      />
    </div>

    <div class="buttons">
      <button @click="placeBet('buy')" :disabled="!canBet">Buy</button>
      <button @click="placeBet('call')" :disabled="!canBet">Call</button>
    </div>

    <!-- Active Bets Section -->
    <div v-if="activeBets.length > 0" class="active-bets">
      <h2>Active Bets</h2>
      <div v-for="(bet, index) in activeBets" :key="index" class="bet-card">
        <div class="bet-info">
          <span class="bet-type">{{ bet.type.toUpperCase() }}</span>
          <span>Amount: ${{ formatNumber(bet.amount) }}</span>
          <span>Entry Price: {{ bet.price }}</span>
          <span :class="getProfitClass(calculateProfit(bet))">
            P/L: ${{ calculateProfit(bet) }}
          </span>
        </div>
        <button @click="closeBet(index)" class="close-bet">Close Bet</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from "vue";
import priceStore from "../store/priceStore.js";

// Extract current price from store
const rawCurrentPrice = computed(() => priceStore.state.currentPrice);
// Parse numeric value from currentPrice string (which may include "$" sign)
const currentPrice = computed(() => {
  try {
    // Handle price as string (e.g., "$123.45")
    if (typeof rawCurrentPrice.value === "string") {
      const parsed = rawCurrentPrice.value.replace(/[^0-9.-]+/g, "");
      if (parsed === "" || parsed === "0" || parsed === "0.00") {
        console.error(
          "CRITICAL ERROR: Current price parsed to zero or empty string",
          {
            raw: rawCurrentPrice.value,
            parsed,
          }
        );
      }
      return parsed;
    }
    // Handle price as number
    if (rawCurrentPrice.value === 0 || rawCurrentPrice.value === 0.0) {
      console.error("CRITICAL ERROR: Current price is zero", {
        price: rawCurrentPrice.value,
        type: typeof rawCurrentPrice.value,
      });
    }
    return rawCurrentPrice.value;
  } catch (error) {
    console.error("CRITICAL ERROR: Failed to parse current price", {
      raw: rawCurrentPrice.value,
      error,
    });
    return 0;
  }
});

// Player's cash amount
const cash = ref(100);
// Bet amount input
const betAmount = ref(0);
// Status/notification message
const message = ref("");
// List of active bets
const activeBets = ref([]);

// Computed property to check if a bet can be placed
const canBet = computed(() => {
  return betAmount.value > 0 && betAmount.value <= cash.value;
});

// Format number to 2 decimal places
function formatNumber(num) {
  // Handle various input types
  if (num === null || num === undefined) {
    console.error(
      "CRITICAL ERROR: Attempted to format null or undefined number",
      {
        value: num,
        type: typeof num,
        stack: new Error().stack,
      }
    );
    return "0.00";
  }

  // Convert to number if it's a string
  const numValue = typeof num === "string" ? parseFloat(num) : num;

  // Check if it's a valid number
  if (isNaN(numValue)) {
    console.error("CRITICAL ERROR: Attempted to format NaN value", {
      original: num,
      parsed: numValue,
      type: typeof num,
      stack: new Error().stack,
    });
    return "0.00";
  }

  // Check for zero values (might be a critical error in some contexts)
  if (numValue === 0 || numValue === 0.0) {
    console.warn("Potential issue: Formatting zero value", {
      original: num,
      context: new Error().stack,
    });
  }

  // Format with 2 decimal places
  return numValue.toFixed(2);
}

// Parse price value from string if needed
function parsePrice(price) {
  if (price === null || price === undefined) {
    console.error(
      "CRITICAL ERROR: Attempted to parse null or undefined price",
      {
        value: price,
        stack: new Error().stack,
      }
    );
    return 0;
  }

  if (typeof price === "string") {
    // Remove any non-numeric characters except decimal point
    const parsed = parseFloat(price.replace(/[^0-9.-]+/g, ""));

    if (isNaN(parsed)) {
      console.error("CRITICAL ERROR: Price parsed to NaN", {
        original: price,
        stack: new Error().stack,
      });
      return 0;
    }

    if (parsed === 0 || parsed === 0.0) {
      console.error("CRITICAL ERROR: Price parsed to zero", {
        original: price,
        stack: new Error().stack,
      });
    }

    return parsed;
  }

  const result = parseFloat(price);

  if (isNaN(result)) {
    console.error("CRITICAL ERROR: Price conversion resulted in NaN", {
      original: price,
      type: typeof price,
      stack: new Error().stack,
    });
    return 0;
  }

  if (result === 0 || result === 0.0) {
    console.error("CRITICAL ERROR: Price is zero", {
      original: price,
      stack: new Error().stack,
    });
  }

  return result;
}

// Place a new bet
function placeBet(type) {
  if (betAmount.value > cash.value) {
    message.value = "Bet amount cannot exceed available cash.";
    return;
  }

  // Get current price as a number
  const priceValue = parsePrice(currentPrice.value);

  if (priceValue === 0) {
    console.error("CRITICAL ERROR: Placing bet with zero price", {
      currentPrice: currentPrice.value,
      parsed: priceValue,
      betType: type,
      betAmount: betAmount.value,
    });
    message.value = "Error: Cannot place bet with invalid price.";
    return;
  }

  if (betAmount.value === 0) {
    console.error("CRITICAL ERROR: Placing bet with zero amount", {
      amount: betAmount.value,
      betType: type,
    });
    message.value = "Error: Bet amount must be greater than zero.";
    return;
  }

  // Create new bet object
  const newBet = {
    type: type,
    amount: betAmount.value,
    price: priceValue,
  };

  console.log("Placing bet", {
    type: newBet.type,
    amount: newBet.amount,
    price: newBet.price,
    currentCash: cash.value,
  });

  // Add to active bets, deduct from cash
  activeBets.value.push(newBet);
  cash.value = Number(cash.value) - Number(newBet.amount);

  // Show confirmation message
  message.value = `Placed a ${type} bet of $${formatNumber(
    newBet.amount
  )} at $${formatNumber(newBet.price)}`;
  betAmount.value = 0;
}

// Calculate current profit for a bet
function calculateProfit(bet) {
  try {
    // Validate bet object
    if (!bet || typeof bet !== "object") {
      console.error(
        "CRITICAL ERROR: Invalid bet object passed to calculateProfit",
        {
          bet,
          type: typeof bet,
          stack: new Error().stack,
        }
      );
      return "0.00";
    }

    // Parse current price and bet price to ensure they're numbers
    const currentPriceValue = parsePrice(currentPrice.value);
    const betPrice = parseFloat(bet.price);
    const betAmount = parseFloat(bet.amount);

    // Check for zero or invalid values
    if (betPrice === 0) {
      console.error("CRITICAL ERROR: Bet price is zero in profit calculation", {
        bet,
        parsedPrice: betPrice,
        stack: new Error().stack,
      });
      return "0.00";
    }

    if (isNaN(betPrice) || isNaN(betAmount) || isNaN(currentPriceValue)) {
      console.error("CRITICAL ERROR: NaN values in profit calculation", {
        currentPrice: currentPriceValue,
        betPrice,
        betAmount,
        original: {
          currentPrice: currentPrice.value,
          bet,
        },
        stack: new Error().stack,
      });
      return "0.00";
    }

    // Calculate the difference
    const priceDiff = currentPriceValue - betPrice;
    let profit = 0;

    // Calculate profit based on bet type
    if (bet.type === "buy") {
      profit = betAmount * (priceDiff / betPrice);
    } else {
      // call
      profit = betAmount * (-priceDiff / betPrice);
    }

    return formatNumber(profit);
  } catch (error) {
    console.error("CRITICAL ERROR: Exception in calculateProfit", {
      error,
      bet,
      currentPrice: currentPrice.value,
      stack: error.stack,
    });
    return "0.00";
  }
}

// Close a bet and settle profit/loss
function closeBet(index) {
  try {
    console.log("Closing bet", {
      index,
      activeBets: activeBets.value,
      currentCash: cash.value,
    });

    // Get the bet to close
    const bet = activeBets.value[index];

    if (!bet) {
      console.error("CRITICAL ERROR: Attempted to close non-existent bet", {
        index,
        activeBets: activeBets.value,
        stack: new Error().stack,
      });
      message.value = "Error: Bet not found.";
      return;
    }

    // Calculate profit as number (parse from formatted string)
    const profitFormatted = calculateProfit(bet);
    const profit = parseFloat(profitFormatted);

    if (isNaN(profit)) {
      console.error("CRITICAL ERROR: Profit is NaN when closing bet", {
        bet,
        profitFormatted,
        parsedProfit: profit,
        stack: new Error().stack,
      });
      message.value = "Error: Invalid profit calculation.";
      return;
    }

    // Calculate total return (bet amount + profit)
    const betAmount = parseFloat(bet.amount);

    if (isNaN(betAmount)) {
      console.error("CRITICAL ERROR: Bet amount is NaN when closing bet", {
        bet,
        parsedAmount: betAmount,
        stack: new Error().stack,
      });
      message.value = "Error: Invalid bet amount.";
      return;
    }

    const returnAmount = betAmount + profit;

    console.log("Bet return calculation", {
      betAmount,
      profit,
      returnAmount,
      currentCash: cash.value,
    });

    // Update cash balance (ensure numeric operation)
    const oldCash = Number(cash.value);
    cash.value = oldCash + returnAmount;

    console.log("Cash updated after closing bet", {
      oldCash,
      returnAmount,
      newCash: cash.value,
    });

    if (isNaN(cash.value)) {
      console.error("CRITICAL ERROR: Cash became NaN after closing bet", {
        oldCash,
        returnAmount,
        newCash: cash.value,
        stack: new Error().stack,
      });
      // Try to recover
      cash.value = oldCash;
      message.value = "Error: Invalid calculation when closing bet.";
      return;
    }

    if (cash.value === 0 || cash.value === 0.0) {
      console.warn("Cash is zero after closing bet", {
        oldCash,
        returnAmount,
        bet,
        stack: new Error().stack,
      });
    }

    // Show result message
    message.value = `Closed ${bet.type} bet with ${
      profit >= 0 ? "profit" : "loss"
    } of $${formatNumber(Math.abs(profit))}`;

    // Remove bet from active list
    activeBets.value.splice(index, 1);
  } catch (error) {
    console.error("CRITICAL ERROR: Exception when closing bet", {
      error,
      index,
      bet: activeBets.value[index],
      stack: error.stack,
    });
    message.value = "Error closing bet. See console for details.";
  }
}

// Get CSS class based on profit value
function getProfitClass(profitStr) {
  const profit = parseFloat(profitStr);
  return {
    "profit-positive": profit > 0,
    "profit-negative": profit < 0,
    "profit-neutral": profit === 0 || isNaN(profit),
  };
}

// Debug watcher to log current price changes
watch(currentPrice, (newVal, oldVal) => {
  if (newVal === "0" || newVal === "0.00" || newVal === 0 || newVal === 0.0) {
    console.error("CRITICAL ERROR: Current price changed to zero", {
      oldValue: oldVal,
      newValue: newVal,
      raw: rawCurrentPrice.value,
      stack: new Error().stack,
    });
  }
});

// Monitor cash value for debugging
watch(cash, (newVal, oldVal) => {
  if (newVal === 0 || newVal === 0.0) {
    console.error("CRITICAL ERROR: Cash value became zero", {
      oldValue: oldVal,
      newValue: newVal,
      stack: new Error().stack,
    });
  }

  if (isNaN(newVal)) {
    console.error("CRITICAL ERROR: Cash value became NaN", {
      oldValue: oldVal,
      newValue: newVal,
      stack: new Error().stack,
    });
    // Try to recover
    cash.value = oldVal || 100;
  }
});
</script>

<style scoped>
.betting-component {
  font-family: Arial, sans-serif;
  max-width: 400px;
  margin: 0 auto;
  text-align: center;
  padding: 20px;
  background-color: #f8f9fa;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  color: #333;
}

.buttons {
  margin-top: 15px;
  display: flex;
  justify-content: center;
  gap: 10px;
}

button {
  margin: 5px;
  padding: 10px 20px;
  font-size: 16px;
  cursor: pointer;
  background-color: #4caf50;
  color: white;
  border: none;
  border-radius: 4px;
  transition: background-color 0.2s;
}

button:hover:not(:disabled) {
  background-color: #45a049;
}

button:disabled {
  background-color: #ccc;
  cursor: not-allowed;
}

.message {
  margin-top: 10px;
  font-weight: bold;
  padding: 8px;
  background-color: #e8f5e9;
  border-radius: 4px;
  border-left: 4px solid #4caf50;
}

.active-bets {
  margin-top: 20px;
  padding: 10px;
  border-top: 1px solid #ddd;
}

.bet-card {
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 15px;
  margin: 12px 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: white;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.bet-info {
  display: flex;
  flex-wrap: wrap;
  gap: 15px;
  align-items: center;
}

.bet-type {
  font-weight: bold;
  background-color: #2c3e50;
  color: white;
  padding: 3px 8px;
  border-radius: 4px;
}

.close-bet {
  background-color: #f44336;
  color: white;
  border: none;
  padding: 5px 10px;
  border-radius: 4px;
  margin: 0;
}

.close-bet:hover {
  background-color: #d32f2f;
}

.profit-positive {
  color: #4caf50;
  font-weight: bold;
}

.profit-negative {
  color: #f44336;
  font-weight: bold;
}

.profit-neutral {
  color: #888;
}

input[type="number"] {
  padding: 8px;
  width: 100px;
  border: 1px solid #ddd;
  border-radius: 4px;
  margin: 0 10px;
}

label {
  font-weight: bold;
}
</style>
