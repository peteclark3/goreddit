/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{vue,js,ts,jsx,tsx}"],
  theme: {
    extend: {
      gridTemplateColumns: {
        2: "3fr 2fr",
      },
    },
  },
  plugins: [],
};
