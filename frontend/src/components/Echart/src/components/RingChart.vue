<template>
  <Echart :options="pieOptions" :height="height" />
</template>

<script setup lang="ts">
import { ref, toRefs, watch } from 'vue';
import { Echart } from '../../index';


const props = defineProps({
  options: {
    type: Object,
    default: () => ({})
  },
  height: {
    type: Number,
    default: 300
  }
});

const {options} = toRefs(props);

const pieOptions = ref(options.value);

watch(options, (data)=>{
  pieOptions.value = data;
});

</script>

<style scoped>

</style>
