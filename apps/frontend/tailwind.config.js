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
        'border-subtle': 'var(--border-subtle-color)',
        'text-primary': 'var(--text-color)',
        'text-secondary': 'var(--text-secondary-color)',
        'text-tertiary': 'var(--text-tertiary-color)',
        'accent-color': 'var(--theme-primary)',
        'accent-secondary': 'var(--theme-secondary)',
        'hover-bg': 'var(--hover-background)',
        'selected-bg': 'var(--selected-background)',
        'msg-sent': 'var(--message-sent-background)',
        'msg-received': 'var(--message-received-background)',
        success: 'var(--color-success)',
        'success-bg': 'var(--color-success-bg)',
        'warning-color': 'var(--color-warning)',
        'warning-bg': 'var(--color-warning-bg)',
        'error-color': 'var(--color-error)',
        'error-bg': 'var(--color-error-bg)',
        'info-color': 'var(--color-info)',
        'info-bg': 'var(--color-info-bg)',
      },
      dark: {
        colors: {
          'bg-primary': 'var(--background-color)',
          'bg-secondary': 'var(--surface-color)',
          'bg-strong': 'var(--strong-background-color)',
          'bg-tertiary': 'var(--surface-secondary-color)',
          'bg-quaternary': 'var(--surface-tertiary-color)',
          'border-color': 'var(--border-color)',
          'border-subtle': 'var(--border-subtle-color)',
          'text-primary': 'var(--text-color)',
          'text-secondary': 'var(--text-secondary-color)',
          'text-tertiary': 'var(--text-tertiary-color)',
          'accent-color': 'var(--theme-primary)',
          'accent-secondary': 'var(--theme-secondary)',
          'hover-bg': 'var(--hover-background)',
          'selected-bg': 'var(--selected-background)',
        },
      },
      fontSize: {
        xs: '0.75rem', // 12px - micro/caption
        sm: '0.875rem', // 14px - body-sm
        base: '0.9375rem', // 15px - body
        lg: '1.125rem', // 18px - h3
        xl: '1.25rem', // 20px - h2
        '2xl': '1.5rem', // 24px - h1
        '3xl': '1.875rem', // 30px
        '4xl': '2.25rem', // 36px
        '5xl': '2rem', // 32px - display
      },
      fontFamily: {
        body: ["'Onest'", "'Noto Sans SC'", 'sans-serif'],
        display: ["'Bricolage Grotesque'", "'Onest'", 'sans-serif'],
      },
      borderRadius: {
        'purr-xs': 'var(--radius-xs)',
        'purr-sm': 'var(--radius-sm)',
        'purr-md': 'var(--radius-md)',
        'purr-lg': 'var(--radius-lg)',
        'purr-xl': 'var(--radius-xl)',
      },
      boxShadow: {
        'purr-xs': 'var(--shadow-xs)',
        'purr-sm': 'var(--shadow-sm)',
        'purr-md': 'var(--shadow-md)',
        'purr-lg': 'var(--shadow-lg)',
        'purr-xl': 'var(--shadow-xl)',
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
      transitionTimingFunction: {
        purr: 'cubic-bezier(0.25, 1, 0.5, 1)',
        'purr-out': 'cubic-bezier(0.16, 1, 0.3, 1)',
      },
      transitionDuration: {
        instant: '100ms',
        fast: '200ms',
        normal: '300ms',
        slow: '500ms',
      },
    },
  },
  plugins: [],
};
