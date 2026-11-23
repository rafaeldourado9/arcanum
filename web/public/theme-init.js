// Aplica o tema antes do React carregar, para nao piscar branco no modo escuro.
(function () {
  try {
    var saved = localStorage.getItem("arcanum.theme");
    var dark = saved ? saved === "dark" : window.matchMedia("(prefers-color-scheme: dark)").matches;
    if (dark) document.documentElement.classList.add("dark");
  } catch (e) {
    /* sem localStorage: fica o tema claro */
  }
})();
