(function () {
  "use strict";

  var HTMX_TIMEOUT_MS = 10000;
  var STREAM_RETRY_INITIAL_MS = 3000;
  var STREAM_RETRY_MAX_MS = 30000;
  var TOAST_DISPLAY_MS = 2500;
  var TOAST_FADE_MS = 300;
  var CONSECUTIVE_ERROR_THRESHOLD = 3;

  htmx.config.timeout = HTMX_TIMEOUT_MS;
  htmx.config.allowEval = false;

  var pendingRequests = new Set();
  var consecutiveErrors = 0;
  var streamRetryDelay = STREAM_RETRY_INITIAL_MS;

  document.body.addEventListener("doAction", function (e) {
    var url = e.detail.url;
    if (!url || url.indexOf("/api/") !== 0) return;

    htmx.ajax("POST", url, {
      target: "#status-panel",
      swap: "outerHTML",
    });
  });

  function getPTZAxis(path) {
    if (path.indexOf("/api/ptz/") !== 0) return null;
    return path.split("/").pop();
  }

  function getSlider(axis) {
    return axis ? document.getElementById("slider-" + axis) : null;
  }

  document.addEventListener("htmx:configRequest", function (e) {
    var pathInfo = e.detail.pathInfo;
    if (!pathInfo) return;
    var path = pathInfo.requestPath;
    var axis = getPTZAxis(path);
    if (axis) {
      var elt = e.detail.elt;
      if (elt && elt.classList.contains("ptz-slider")) {
        e.detail.parameters.value = elt.value;
      }
    }
  });

  document.addEventListener("input", function (e) {
    if (!e.target.classList.contains("ptz-slider")) return;
    var axis = e.target.id.replace("slider-", "");
    var valEl = document.getElementById("val-" + axis);
    if (!valEl) return;
    var suffix = axis === "zoom" ? "x" : "\u00b0";
    valEl.textContent = e.target.value + suffix;
  });

  function showToast(msg, type) {
    type = type || "success";
    var container = document.getElementById("toast-container");
    var toast = document.createElement("div");
    toast.className = "toast toast-" + type + " show";
    toast.textContent = msg;
    container.appendChild(toast);
    setTimeout(function () {
      toast.classList.remove("show");
      setTimeout(function () {
        toast.remove();
      }, TOAST_FADE_MS);
    }, TOAST_DISPLAY_MS);
  }

  function showOfflineBanner() {
    var panel = document.getElementById("status-panel");
    if (!panel || panel.querySelector(".offline-banner")) return;
    var banner = document.createElement("div");
    banner.className = "error-banner offline-banner";
    var dot = document.createElement("span");
    dot.className = "offline-dot";
    banner.appendChild(dot);
    banner.appendChild(
      document.createTextNode(
        " Daemon unreachable \u2014 reconnecting\u2026",
      ),
    );
    panel.insertBefore(banner, panel.firstChild);
  }

  document.addEventListener("htmx:beforeRequest", function (e) {
    var path = e.detail.pathInfo && e.detail.pathInfo.requestPath;
    if (path === "/panel" && document.visibilityState !== "visible") {
      e.detail.xhr.abort();
      return;
    }
    if (path && pendingRequests.has(path)) {
      e.detail.xhr.abort();
      return;
    }
    if (path) pendingRequests.add(path);
    var axis = getPTZAxis(path);
    var slider = getSlider(axis);
    if (slider) slider.classList.add("sending");
  });

  document.addEventListener("htmx:afterRequest", function (e) {
    var path = e.detail.pathInfo && e.detail.pathInfo.requestPath;

    if (path) {
      pendingRequests.delete(path);
      var axis = getPTZAxis(path);
      var slider = getSlider(axis);
      if (slider) slider.classList.remove("sending");
    }

    if (e.detail.failed) {
      consecutiveErrors++;
      var errAxis = getPTZAxis(path);
      var errSlider = getSlider(errAxis);
      if (errSlider && errSlider.dataset.lastGood !== undefined) {
        errSlider.value = errSlider.dataset.lastGood;
        var valEl = document.getElementById("val-" + errAxis);
        if (valEl) {
          var suffix = errAxis === "zoom" ? "x" : "\u00b0";
          valEl.textContent = errSlider.dataset.lastGood + suffix;
        }
      }
      if (consecutiveErrors >= CONSECUTIVE_ERROR_THRESHOLD) {
        showOfflineBanner();
      }
      showToast(
        consecutiveErrors >= CONSECUTIVE_ERROR_THRESHOLD
          ? "Connection lost \u2014 retrying"
          : "Request failed",
        "error",
      );
      return;
    }

    consecutiveErrors = 0;
    streamRetryDelay = STREAM_RETRY_INITIAL_MS;
    var offlineBanner = document.querySelector(".offline-banner");
    if (offlineBanner) offlineBanner.remove();

    var okAxis = getPTZAxis(path);
    var okSlider = getSlider(okAxis);
    if (okSlider) okSlider.dataset.lastGood = okSlider.value;
  });

  document.addEventListener("htmx:responseError", function (e) {
    var panel = document.getElementById("status-panel");
    if (!panel || panel.querySelector(".error-banner:not(.offline-banner)")) return;
    var banner = document.createElement("div");
    banner.className = "error-banner";
    banner.textContent = "Connection error \u2014 will retry automatically";
    panel.insertBefore(banner, panel.firstChild);
  });

  document.addEventListener("htmx:timeout", function () {
    showToast("Request timed out", "error");
  });

  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") {
      streamRetryDelay = STREAM_RETRY_INITIAL_MS;
    }
  });

  document.addEventListener("keydown", function (e) {
    if (
      e.target.tagName === "INPUT" ||
      e.target.tagName === "TEXTAREA" ||
      e.target.tagName === "SELECT"
    )
      return;
    var actionMap = {
      t: "/api/track",
      i: "/api/idle",
      p: "/api/privacy",
      c: "/api/center",
    };
    var url = actionMap[e.key.toLowerCase()];
    if (url) {
      e.preventDefault();
      var badge = document.querySelector(".header-badge");
      if (badge && badge.textContent === "Offline") {
        showToast("Camera offline", "error");
        return;
      }
      htmx.trigger(document.body, "doAction", { url: url });
      return;
    }
    var ptzStep = { pan: 5, tilt: 5, zoom: 10 };
    var ptzAction = null;
    switch (e.key) {
      case "ArrowLeft":
        ptzAction = { axis: "pan", delta: -ptzStep.pan };
        break;
      case "ArrowRight":
        ptzAction = { axis: "pan", delta: ptzStep.pan };
        break;
      case "ArrowUp":
        ptzAction = { axis: "tilt", delta: ptzStep.tilt };
        break;
      case "ArrowDown":
        ptzAction = { axis: "tilt", delta: -ptzStep.tilt };
        break;
      case "+":
      case "=":
        ptzAction = { axis: "zoom", delta: ptzStep.zoom };
        break;
      case "-":
      case "_":
        ptzAction = { axis: "zoom", delta: -ptzStep.zoom };
        break;
    }
    if (!ptzAction) return;
    e.preventDefault();
    var slider = document.getElementById("slider-" + ptzAction.axis);
    if (!slider) return;
    var current = parseInt(slider.value, 10) || 0;
    var next = current + ptzAction.delta;
    slider.value = next;
    var suffix = ptzAction.axis === "zoom" ? "x" : "\u00b0";
    var valEl = document.getElementById("val-" + ptzAction.axis);
    if (valEl) valEl.textContent = next + suffix;
    htmx.trigger(document.body, "doAction", {
      url: "/api/ptz/" + ptzAction.axis,
    });
  });

  (function () {
    var img = document.getElementById("preview-img");
    if (!img) return;
    var retryTimer = null;
    img.addEventListener("error", function () {
      if (retryTimer) return;
      this.style.display = "none";
      var fallback = document.getElementById("preview-fallback");
      if (fallback) {
        fallback.style.display = "flex";
        var label = fallback.querySelector("div:last-child");
        if (label) label.textContent = "Reconnecting\u2026";
      }
      var delay = streamRetryDelay;
      retryTimer = setTimeout(function () {
        retryTimer = null;
        img.src = "/api/stream?" + Date.now();
        img.style.display = "";
        if (fallback) fallback.style.display = "none";
      }, delay);
      streamRetryDelay = Math.min(
        streamRetryDelay * 2,
        STREAM_RETRY_MAX_MS,
      );
    });
  })();
})();
