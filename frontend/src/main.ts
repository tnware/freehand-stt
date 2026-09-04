import { mount } from 'svelte'
import AboutWindow from './AboutWindow.svelte'
import App from './App.svelte'
import SettingsWindow from './SettingsWindow.svelte'
import './app.css'

const Root = window.location.hash.startsWith('#settings')
  ? SettingsWindow
  : window.location.hash.startsWith('#about')
    ? AboutWindow
    : App

mount(Root, { target: document.getElementById('app')! })
