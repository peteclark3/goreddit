<template>
  <div ref="cloudContainer" class="word-cloud">
    <svg :width="width" :height="height">
      <g :transform="`translate(${width/2},${height/2})`">
        <text
          v-for="word in computedWords"
          :key="word.text"
          :style="{ fontSize: word.size + 'px' }"
          :fill="word.color"
          :transform="`translate(${word.x},${word.y})rotate(${word.rotate})`"
          text-anchor="middle"
          :class="'word-' + word.text"
        >
          {{ word.text }}
        </text>
      </g>
    </svg>
  </div>
</template>

<script>
import * as d3 from 'd3-scale'
import cloud from 'd3-cloud'

export default {
  name: 'WordCloud',
  props: {
    words: {
      type: Array,
      required: true
    },
    width: {
      type: Number,
      default: 600
    },
    height: {
      type: Number,
      default: 400
    }
  },
  data() {
    return {
      computedWords: []
    }
  },
  watch: {
    words: {
      handler: 'updateCloud',
      deep: true
    }
  },
  methods: {
    updateCloud() {
      if (!this.words.length) return

      const maxFreq = Math.max(...this.words.map(w => w[1]))
      const fontSize = d3.scaleLinear()
        .domain([0, maxFreq])
        .range([12, 60])

      const colorScale = d3.scaleLinear()
        .domain([0, maxFreq])
        .range(['#bfdbfe', '#1d4ed8']) // Light blue to dark blue

      cloud()
        .size([this.width, this.height])
        .words(this.words.map(([text, value]) => ({
          text,
          size: fontSize(value),
          value,
          color: colorScale(value)
        })))
        .padding(5)
        .rotate(0)
        .font('Impact')
        .fontSize(d => d.size)
        .on('end', words => {
          this.computedWords = words
        })
        .start()
    }
  },
  mounted() {
    this.updateCloud()
  }
}
</script>

<style scoped>
.word-cloud {
  width: 100%;
  height: 100%;
}
</style> 