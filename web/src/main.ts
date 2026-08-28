import { mount } from "svelte";
import App from "./App.svelte";
import "./styles.css";

export { socket } from "./ws";

const target = document.getElementById("root");
if (!target) throw new Error("#root not found");

export default mount(App, { target });
