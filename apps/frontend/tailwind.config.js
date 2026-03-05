/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        'bg-primary': 'var(--background-color)',
        'bg-secondary': 'var(--surface-color)',
        'bg-tertiary': 'var(--surface-secondary-color)',
        'bg-quaternary': 'var(--surface-tertiary-color)',
        'border-color': 'var(--border-color)',
        'text-primary': 'var(--text-color)',
        'text-secondary': 'var(--text-secondary-color)',
        'text-tertiary': 'var(--text-tertiary-color)',
        'accent-color': 'var(--theme-primary)',
        'hover-bg': 'var(--hover-background)',
        'selected-bg': 'var(--selected-background)',
      },
      dark: {
        colors: {
          'bg-primary': 'var(--background-color)',
          'bg-secondary': 'var(--surface-color)',
          'bg-tertiary': 'var(--surface-secondary-color)',
          'bg-quaternary': 'var(--surface-tertiary-color)',
          'border-color': 'var(--border-color)',
          'text-primary': 'var(--text-color)',
          'text-secondary': 'var(--text-secondary-color)',
          'text-tertiary': 'var(--text-tertiary-color)',
          'accent-color': 'var(--theme-primary)',
          'hover-bg': 'var(--hover-background)',
          'selected-bg': 'var(--selected-background)',
        },
      },
      fontSize: {
        xs: '0.75rem', // 12px
        sm: '0.875rem', // 14px
        base: '1rem', // 16px
        lg: '1.125rem', // 18px
        xl: '1.25rem', // 20px
        '2xl': '1.5rem', // 24px
        '3xl': '1.875rem', // 30px
        '4xl': '2.25rem', // 36px
        '5xl': '1.5rem', // 48px
      },
      spacing: {
        1: '0.25rem', // 4px
        2: '0.5rem', // 8px
        3: '0.75rem', // 12px
        4: '1rem', // 16px
        5: '1.25rem', // 20px
        6: '1.5rem', // 24px
        8: '2rem', // 32px
      },
    },
  },
  plugins: [],
};
