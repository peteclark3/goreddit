<template>
  <div class="container mx-auto px-4 py-8 bg-gray-100 min-h-screen">
    <h1 class="text-4xl font-bold mb-8">Reddit Firehose</h1>
    <div class="grid grid-cols-2 gap-8">
      <!-- Posts Column -->
      <div class="space-y-4">
        <div v-for="post in posts" :key="post.ID" class="bg-white p-4 rounded-lg shadow">
          <h2 class="text-xl font-semibold mb-2">
            <a :href="post.URL" target="_blank" class="text-blue-600 hover:text-blue-800">
              {{ post.Title || 'No Title' }}
            </a>
            <span class="ml-2 text-lg" :title="'Sentiment Score: ' + post.Sentiment">
              {{ getSentimentEmoji(post.Sentiment) }}
            </span>
          </h2>
          <div class="text-sm text-gray-600 mb-2">
            Posted in r/{{ post.Subreddit || '?' }} • Score: {{ post.Score || 0 }}
          </div>
          <div class="text-gray-700" v-if="post.Body">{{ post.Body }}</div>
          <div class="text-xs text-gray-500 mt-2">
            Topics: {{ post.Topics?.join(', ') }}
          </div>
        </div>
      </div>

      <!-- Topic Cloud Column -->
      <div class="bg-white p-4 rounded-lg shadow">
        <h2 class="text-2xl font-semibold mb-4">Topic Cloud</h2>
        <div class="h-[600px]">
          <word-cloud
            v-if="cloudWords.length > 0"
            :words="cloudWords"
            :width="600"
            :height="500"
          />
          <div v-else class="text-gray-500 text-center py-8">
            Waiting for topics...
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import WordCloud from './components/WordCloud.vue'

export default {
  components: {
    WordCloud
  },
  data() {
    return {
      posts: [],
      ws: null,
      topicFrequency: {},
    }
  },
  computed: {
    cloudWords() {
      console.log('Computing cloud words from frequencies:', this.topicFrequency)
      const words = Object.entries(this.topicFrequency)
        .sort(([,a], [,b]) => b - a)
        .slice(0, 50)
      console.log('Computed cloud words:', words)
      return words
    }
  },
  methods: {
    getSentimentEmoji(sentiment) {
      if (sentiment === 1.0) return '😊'
      if (sentiment === -1.0) return '😔'
      return '😐'
    },
    updateTopicFrequency(topics) {
      console.log('Updating topics:', topics)
      if (!Array.isArray(topics)) {
        console.warn('Topics is not an array:', topics)
        return
      }
      
      topics.forEach(topic => {
        this.topicFrequency[topic] = (this.topicFrequency[topic] || 0) + 1
      })
      console.log('Updated topic frequencies:', this.topicFrequency)

      // Keep only top 100 topics
      const sortedEntries = Object.entries(this.topicFrequency)
        .sort(([,a], [,b]) => b - a)
        .slice(0, 100)
      this.topicFrequency = Object.fromEntries(sortedEntries)
    },
    connectWebSocket() {
      console.log('Attempting to connect to WebSocket...')
      this.ws = new WebSocket('ws://localhost:8080/ws')
      
      this.ws.onopen = () => {
        console.log('WebSocket connected!')
      }

      this.ws.onmessage = (event) => {
        console.log('Raw WebSocket message:', event.data)
        let post
        try {
          post = JSON.parse(event.data)
          console.log('Parsed post:', post)
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
          return
        }
        
        // Add post to list
        this.posts.unshift(post)
        if (this.posts.length > 100) {
          this.posts.pop()
        }

        // Update topic frequencies
        if (post.Topics) {
          console.log('Found topics in post:', post.Topics)
          this.updateTopicFrequency(post.Topics)
        } else {
          console.warn('No topics in post:', post)
        }
      }

      this.ws.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason)
        setTimeout(() => {
          this.connectWebSocket()
        }, 1000)
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    }
  },
  mounted() {
    console.log('Component mounted, connecting WebSocket...')
    this.connectWebSocket()
  },
  beforeUnmount() {
    if (this.ws) {
      console.log('Closing WebSocket connection...')
      this.ws.close()
    }
  }
}
</script>

<style>
.vue-wordcloud {
  width: 100%;
  height: 400px;
}
</style>