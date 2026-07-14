const themeToggle = document.getElementById("theme-toggle");
if (themeToggle) {
  const lightIcon = themeToggle.querySelector(".theme-icon-light");
  const darkIcon = themeToggle.querySelector(".theme-icon-dark");

  function applyTheme() {
    const isLight = document.documentElement.classList.contains("light");
    if (lightIcon) lightIcon.classList.toggle("hidden", !isLight);
    if (darkIcon) darkIcon.classList.toggle("hidden", isLight);
    themeToggle.setAttribute("aria-pressed", String(isLight));
  }

  applyTheme();

  themeToggle.addEventListener("click", () => {
    const isLight = document.documentElement.classList.toggle("light");
    localStorage.setItem("theme", isLight ? "light" : "dark");
    applyTheme();
  });

  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (e) => {
    if (!localStorage.getItem("theme")) {
      const isLight = !e.matches;
      document.documentElement.classList.toggle("light", isLight);
      applyTheme();
    }
  });
}

const toggle = document.getElementById("nav-toggle");
const navLinks = document.getElementById("nav-links");
if (toggle && navLinks) {
  const menuIcon = toggle.querySelector(".menu-icon");
  const closeIcon = toggle.querySelector(".close-icon");

  toggle.addEventListener("click", function () {
    const isOpen = navLinks.classList.toggle("open");
    if (menuIcon) menuIcon.classList.toggle("hidden", isOpen);
    if (closeIcon) closeIcon.classList.toggle("hidden", !isOpen);
    toggle.setAttribute("aria-expanded", String(isOpen));
  });

  navLinks.querySelectorAll(".nav-link").forEach((link) => {
    link.addEventListener("click", () => {
      navLinks.classList.remove("open");
      if (menuIcon) menuIcon.classList.remove("hidden");
      if (closeIcon) closeIcon.classList.add("hidden");
      toggle.setAttribute("aria-expanded", "false");
    });
  });
}
