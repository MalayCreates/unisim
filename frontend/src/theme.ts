import { createTheme } from '@mantine/core';

// Ops-dashboard styling: tight spacing, monospace accents, blue primary.
export const theme = createTheme({
  primaryColor: 'blue',
  fontFamilyMonospace: 'ui-monospace, SFMono-Regular, Menlo, monospace',
  defaultRadius: 'sm',
  components: {
    Paper: {
      defaultProps: { withBorder: true },
    },
  },
});
