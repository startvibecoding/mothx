import './style.css';
import './lib/preferences.js';
import { mount } from 'svelte';
import App from './App.svelte';

const app = mount(App, {
  target: document.getElementById('app')
});

export default app;
