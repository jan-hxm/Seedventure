<template>
  <div class="betting-component">
    <h1>Betting Component</h1>
    <p>Cash: ${{ formatMoney(cash) }}</p>
    <p>Current Price: ${{ formatMoney(currentPrice) }}</p>
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
          <span>Amount: ${{ formatMoney(bet.amount) }}</span>
          <span>Entry Price: ${{ formatMoney(bet.price) }}</span>
          <span :class="getProfitClass(calculateProfit(bet))">
            P/L: ${{ formatMoney(calculateProfit(bet)) }}
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

const currentPrice = ref(priceStore.state.currentPrice);
const cash = ref(100);
const betAmount = ref(0);
const message = ref("");
const activeBets = ref([]);

// Watch for price updates
watch(
  () => priceStore.state.currentPrice,
  (newPrice) => {
    currentPrice.value = Number(newPrice) || 0; // Ensure currentPrice is always a number
  }
);

const canBet = computed(() => {
  return betAmount.value > 0 && betAmount.value <= cash.value;
});

function placeBet(type) {
  if (betAmount.value > cash.value) {
    message.value = "Bet amount cannot exceed available cash.";
    return;
  }

  const newBet = {
    type: type,
    amount: Number(betAmount.value) || 0, // Ensure amount is a number
    price: Number(currentPrice.value) || 0, // Ensure price is a number
  };

  activeBets.value.push(newBet);
  cash.value -= newBet.amount;
  message.value = `Placed a ${type} bet of $${formatMoney(
    newBet.amount
  )} at $${formatMoney(newBet.price)}`;
  betAmount.value = 0;
}

function calculateProfit(bet) {
  const priceDiff = currentPrice.value - bet.price;
  if (bet.type === "buy") {
    return bet.amount * (priceDiff / bet.price);
  } else {
    return bet.amount * (-priceDiff / bet.price);
  }
}

function closeBet(index) {
  const bet = activeBets.value[index];
  const profit = calculateProfit(bet);
  cash.value += bet.amount + profit;

  message.value = `Closed ${bet.type} bet with ${
    profit >= 0 ? "profit" : "loss"
  } of $${formatMoney(Math.abs(profit))}`;
  activeBets.value.splice(index, 1);
}

function getProfitClass(profit) {
  return {
    "profit-positive": profit > 0,
    "profit-negative": profit < 0,
    "profit-neutral": profit === 0,
  };
}

function formatMoney(value) {
  // Convert to number if it's not already and handle invalid/null values
  const numValue = Number(value);
  if (isNaN(numValue)) {
    return "0.00";
  }
  return numValue.toFixed(2);
}
</script>

<style scoped>
.betting-component {
  font-family: Arial, sans-serif;
  max-width: 400px;
  margin: 0 auto;
  text-align: center;
  color: #333;
}

.buttons {
  margin-top: 10px;
}

button {
  margin: 5px;
  padding: 10px 20px;
  font-size: 16px;
  cursor: pointer;
}

button:disabled {
  background-color: #ccc;
  cursor: not-allowed;
}

.message {
  margin-top: 10px;
  font-weight: bold;
}

.active-bets {
  margin-top: 20px;
  padding: 10px;
}

.bet-card {
  border: 1px solid #ccc;
  border-radius: 4px;
  padding: 10px;
  margin: 10px 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.bet-info {
  display: flex;
  gap: 15px;
  align-items: center;
}

.bet-type {
  font-weight: bold;
}

.close-bet {
  background-color: #ff4444;
  color: white;
  border: none;
  padding: 5px 10px;
  border-radius: 4px;
}

.profit-positive {
  color: #4caf50;
}

.profit-negative {
  color: #f44336;
}

.profit-neutral {
  color: #888;
}
</style>
