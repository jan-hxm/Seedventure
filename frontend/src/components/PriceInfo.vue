<template>
  <div class="price-info">
    <div class="current-price">
      Current Price: <span :class="priceClass">${{ currentPrice }}</span>
    </div>
    <div class="price-change">
      Change: <span :class="priceClass">${{ priceChange }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  currentPrice: {
    type: String,
    required: true,
  },
  priceChange: {
    type: String,
    required: true,
  },
  isPositive: {
    type: Boolean,
    default: false,
  },
});

// Computed class for price change indicator
const priceClass = computed(() => {
  if (props.isPositive) return "price-up";
  if (!props.isPositive && props.priceChange !== "$0.00 (0.00%)")
    return "price-down";
  return "";
});
</script>

<style scoped>
.price-info {
  display: flex;
  justify-content: space-between;
  padding: 15px;
  background-color: #f5f5f5;
  border-radius: 4px;
  font-size: 16px;
}

.current-price {
  color: #333;
  font-weight: bold;
}

.price-change {
  color: #333;
}

.price-up {
  color: #26a69a;
}

.price-down {
  color: #ef5350;
}
</style>
