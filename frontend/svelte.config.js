import adapter from '@sveltejs/adapter-node';

const config = {
  kit: {
    adapter: adapter()
  },
  compilerOptions: {
    warningFilter: (warning) => !warning.code.startsWith('a11y_')
  }
};

export default config;
