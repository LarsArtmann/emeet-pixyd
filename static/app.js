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

  /* -------------------------------------------------------------------- */
  /*  Action dispatch (keyboard shortcuts → HTMX POST)                     */
  /* -------------------------------------------------------------------- */

  document.body.addEventListener("doAction", function (e) {
    var url = e.detail.url;
    if (!url || url.indexOf("/api/") !== 0) return;

    htmx.ajax("POST", url, {
      target: "#status-panel",
      swap: "outerHTML",
    });
  });

  /* -------------------------------------------------------------------- */
  /*  PTZ helpers                                                          */
  /* -------------------------------------------------------------------- */

  function getPTZAxis(path) {
    if (path.indexOf("/api/ptz/") !== 0) return null;
    return path.split("/").pop();
  }

  function getSlider(axis) {
    return axis ? document.getElementById("slider-" + axis) : null;
  }

  function updateRadar() {
    var radar = document.querySelector(".ptz-radar");
    if (!radar) return;

    var panSlider = document.getElementById("slider-pan");
    var tiltSlider = document.getElementById("slider-tilt");
    var zoomSlider = document.getElementById("slider-zoom");
    if (!panSlider || !tiltSlider || !zoomSlider) return;

    var pan = parseInt(panSlider.value, 10) || 0;
    var tilt = parseInt(tiltSlider.value, 10) || 0;
    var zoom = parseInt(zoomSlider.value, 10) || 0;
    var panMin = parseInt(panSlider.min, 10);
    var panMax = parseInt(panSlider.max, 10);
    var tiltMin = parseInt(tiltSlider.min, 10);
    var tiltMax = parseInt(tiltSlider.max, 10);
    var zoomMin = parseInt(zoomSlider.min, 10);
    var zoomMax = parseInt(zoomSlider.max, 10);

    var panPct = ((pan - panMin) / (panMax - panMin)) * 100;
    var tiltPct = ((tiltMax - tilt) / (tiltMax - tiltMin)) * 100;
    var zoomPct = ((zoom - zoomMin) / (zoomMax - zoomMin)) * 100;

    radar.style.setProperty("--pan-x", panPct.toFixed(1) + "%");
    radar.style.setProperty("--pan-y", tiltPct.toFixed(1) + "%");
    radar.style.setProperty("--zoom-pct", zoomPct.toFixed(1) + "%");
  }

  /* -------------------------------------------------------------------- */
  /*  HTMX request lifecycle                                              */
  /* -------------------------------------------------------------------- */

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
    if (valEl) {
      var suffix = axis === "zoom" ? "x" : "\u00b0";
      valEl.textContent = e.target.value + suffix;
    }
    updateRadar();
  });

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
        updateRadar();
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

  /* -------------------------------------------------------------------- */
  /*  Focus preservation across panel swaps                               */
  /* -------------------------------------------------------------------- */

  var preSwapFocusId = null;

  document.addEventListener("htmx:beforeRequest", function (e) {
    var target = e.detail.target;
    if (target && target.id === "status-panel") {
      var active = document.activeElement;
      if (active && active.id && target.contains(active)) {
        preSwapFocusId = active.id;
      }
    }
  });

  document.addEventListener("htmx:afterSettle", function () {
    if (!preSwapFocusId) return;
    var el = document.getElementById(preSwapFocusId);
    if (el) {
      el.focus();
    }
    preSwapFocusId = null;
  });

  /* -------------------------------------------------------------------- */
  /*  Toast & offline banner                                              */
  /* -------------------------------------------------------------------- */

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
    banner.setAttribute("role", "status");
    var dot = document.createElement("span");
    dot.className = "offline-dot";
    banner.appendChild(dot);
    banner.appendChild(document.createTextNode(" Daemon unreachable \u2014 reconnecting\u2026"));
    panel.insertBefore(banner, panel.firstChild);
  }

  /* -------------------------------------------------------------------- */
  /*  SSE — live state updates                                            */
  /* -------------------------------------------------------------------- */

  var evtSource = null;
  var evtRetryDelay = STREAM_RETRY_INITIAL_MS;

  function connectEvents() {
    if (evtSource) return;

    evtSource = new EventSource("/api/events");

    evtSource.onopen = function () {
      evtRetryDelay = STREAM_RETRY_INITIAL_MS;
      updateSSEIndicator("connected");
    };

    evtSource.addEventListener("refresh", function () {
      htmx.trigger("#status-panel", "refresh");
    });

    evtSource.onerror = function () {
      if (evtSource) {
        evtSource.close();
        evtSource = null;
      }

      updateSSEIndicator("disconnected");
      setTimeout(connectEvents, evtRetryDelay);
      evtRetryDelay = Math.min(evtRetryDelay * 2, STREAM_RETRY_MAX_MS);
    };
  }

  function updateSSEIndicator(state) {
    var dot = document.getElementById("sse-indicator");
    if (!dot) return;
    dot.className = "sse-indicator " + state;
    dot.setAttribute("aria-label", "Live updates " + state);
  }

  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible") {
      streamRetryDelay = STREAM_RETRY_INITIAL_MS;
      connectEvents();
    } else if (evtSource) {
      evtSource.close();
      evtSource = null;
    }
  });

  connectEvents();

  /* -------------------------------------------------------------------- */
  /*  Keyboard shortcuts                                                  */
  /* -------------------------------------------------------------------- */

  document.addEventListener("keydown", function (e) {
    if (
      e.target.tagName === "INPUT" ||
      e.target.tagName === "TEXTAREA" ||
      e.target.tagName === "SELECT"
    )
      return;

    if (e.key === "?") {
      e.preventDefault();
      toggleLegend();
      return;
    }

    if (e.key === "Escape") {
      toggleLegend(false);
    }

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
    htmx.trigger(slider, "input");
  });

  /* -------------------------------------------------------------------- */
  /*  Shortcut legend toggle                                              */
  /* -------------------------------------------------------------------- */

  function toggleLegend(force) {
    var legend = document.getElementById("shortcut-legend");
    var fab = document.getElementById("shortcut-fab");
    if (!legend) return;
    var shouldShow = force !== undefined ? force : legend.hidden;
    legend.hidden = !shouldShow;
    if (fab) fab.setAttribute("aria-expanded", String(shouldShow));
  }

  (function () {
    var fab = document.getElementById("shortcut-fab");
    var closeBtn = document.getElementById("legend-close");
    if (fab) {
      fab.setAttribute("aria-expanded", "false");
      fab.addEventListener("click", function () {
        toggleLegend();
      });
    }
    if (closeBtn) {
      closeBtn.addEventListener("click", function () {
        toggleLegend(false);
      });
    }
  })();

  /* -------------------------------------------------------------------- */
  /*  Snapshot button                                                     */
  /* -------------------------------------------------------------------- */

  (function () {
    var snapshotBtn = document.getElementById("snapshot-btn");
    if (!snapshotBtn) return;

    snapshotBtn.addEventListener("click", function () {
      snapshotBtn.classList.add("flash");
      setTimeout(function () {
        snapshotBtn.classList.remove("flash");
      }, 400);

      fetch("/api/snapshot?ts=" + Date.now())
        .then(function (resp) {
          if (!resp.ok) throw new Error("HTTP " + resp.status);
          return resp.blob();
        })
        .then(function (blob) {
          if (blob.size === 0) throw new Error("empty");
          var url = URL.createObjectURL(blob);
          var a = document.createElement("a");
          a.href = url;
          a.download =
            "pixy-" + new Date().toISOString().slice(0, 19).replace(/[:T]/g, "-") + ".jpg";
          document.body.appendChild(a);
          a.click();
          a.remove();
          URL.revokeObjectURL(url);
          showToast("Snapshot saved", "success");
        })
        .catch(function () {
          showToast("Snapshot unavailable", "error");
        });
    });
  })();

  /* -------------------------------------------------------------------- */
  /*  Preview stream error recovery                                       */
  /* -------------------------------------------------------------------- */

  (function () {
    var img = document.getElementById("preview-img");
    if (!img) return;
    var STREAM_MAX_RETRIES = 10;
    var retryTimer = null;
    var streamRetryCount = 0;

    img.addEventListener("load", function () {
      streamRetryDelay = STREAM_RETRY_INITIAL_MS;
      streamRetryCount = 0;
    });

    img.addEventListener("error", function () {
      if (retryTimer) return;
      streamRetryCount++;

      if (streamRetryCount > STREAM_MAX_RETRIES) {
        this.style.display = "none";
        var fallback = document.getElementById("preview-fallback");
        if (fallback) {
          fallback.style.display = "flex";
          var label = fallback.querySelector("div:last-child");
          if (label) label.textContent = "Stream unavailable \u2014 reload page to retry";
        }
        streamRetryDelay = STREAM_RETRY_INITIAL_MS;
        streamRetryCount = 0;
        return;
      }

      this.style.display = "none";
      var fallback = document.getElementById("preview-fallback");
      if (fallback) {
        fallback.style.display = "flex";
        var label = fallback.querySelector("div:last-child");
        if (label)
          label.textContent =
            "Reconnecting\u2026 (" + streamRetryCount + "/" + STREAM_MAX_RETRIES + ")";
      }
      var delay = streamRetryDelay;
      retryTimer = setTimeout(function () {
        retryTimer = null;
        img.src = "/api/stream?" + Date.now();
        img.style.display = "";
        if (fallback) fallback.style.display = "none";
      }, delay);
      streamRetryDelay = Math.min(streamRetryDelay * 2, STREAM_RETRY_MAX_MS);
    });
  })();

  /* -------------------------------------------------------------------- */
  /*  Preset save (delegated — input lives inside swapped panel)          */
  /* -------------------------------------------------------------------- */

  function savePreset() {
    var input = document.getElementById("preset-name-input");
    if (!input) return;
    var name = input.value.trim();
    if (!name) {
      showToast("Enter a preset name", "error");
      input.focus();
      return;
    }
    input.value = "";
    htmx.ajax("POST", "/api/preset/save/" + encodeURIComponent(name), {
      target: "#status-panel",
      swap: "outerHTML",
    });
  }

  document.addEventListener("click", function (e) {
    if (e.target && e.target.id === "preset-save-btn") {
      e.preventDefault();
      savePreset();
    }
  });

  document.addEventListener("keydown", function (e) {
    if (e.target && e.target.id === "preset-name-input" && e.key === "Enter") {
      e.preventDefault();
      savePreset();
    }
  });
})();
