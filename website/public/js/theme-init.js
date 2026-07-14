(function () {
  var stored = localStorage.getItem("theme");
  var prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  var isLight = stored === "light" || (!stored && !prefersDark);
  if (isLight) document.documentElement.classList.add("light");
})();
