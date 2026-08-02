(function () {
  "use strict";

  var TOAST_DISPLAY_MS = 2500;
  var TOAST_FADE_MS = 300;
  var STREAM_RETRY_INITIAL_MS = 3000;
  var STREAM_RETRY_MAX_MS = 30000;

  /* -------------------------------------------------------------------- */
  /*  Toast — invoked by server ExecuteScript and local callers           */
  /* -------------------------------------------------------------------- */

  window.__showToast = showToast;

  function showToast(msg, type) {
    type = type || "success";
    var container = document.getElementById("toast-container");
    if (!container) return;
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

  /* -------------------------------------------------------------------- */
  /*  SSE connection state indicator + offline banner                     */
  /*  DataStar dispatches 'datastar-fetch' custom events on document.     */
  /*  We listen to all events (no element filter) because:                */
  /*  - datastar-patch-elements = data flowing (green)                    */
  /*  - started = a fetch is in flight (yellow)                           */
  /*  - retries-failed = server unreachable (red + banner)               */
  /*  Button clicks also trigger these events, which is fine — they       */
  /*  prove the server is reachable.                                      */
  /* -------------------------------------------------------------------- */

  (function () {
    var indicator = document.getElementById("sse-indicator");
    var banner = document.getElementById("offline-banner");
    if (!indicator && !banner) return;

    function setSSEState(state) {
      if (indicator) {
        indicator.className = "sse-indicator " + state;
        indicator.setAttribute(
          "aria-label",
          state === "connected"
            ? "Live updates connected"
            : state === "disconnected"
              ? "Live updates disconnected"
              : "Live updates connecting",
        );
      }
      if (banner) banner.style.display = state === "disconnected" ? "flex" : "none";
    }

    window.setSSEState = setSSEState;

    document.addEventListener("datastar-fetch", function (e) {
      var type = e.detail && e.detail.type;
      if (!type) return;

      if (type === "datastar-patch-elements" || type === "datastar-patch-signals") {
        setSSEState("connected");
      } else if (type === "retrying" || type === "retries-failed") {
        setSSEState("disconnected");
      } else if (type === "started") {
        setSSEState("");
      }
    });
  })();

  /* -------------------------------------------------------------------- */
  /*  Keyboard shortcuts                                                  */
  /* -------------------------------------------------------------------- */

  function clickDataStarButton(selector) {
    var btn = document.querySelector(selector);
    if (btn && !btn.disabled) btn.click();
  }

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
      return;
    }

    var modeMap = {
      t: ".mode-track",
      i: ".mode-idle",
      p: ".mode-privacy",
    };
    var selector = modeMap[e.key.toLowerCase()];
    if (selector) {
      e.preventDefault();
      clickDataStarButton(selector);
      return;
    }

    if (e.key.toLowerCase() === "c") {
      e.preventDefault();
      var centerBtn = document.querySelector('[aria-label="Center camera"]');
      if (centerBtn && !centerBtn.disabled) centerBtn.click();
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
    var min = parseInt(slider.min, 10);
    var max = parseInt(slider.max, 10);
    next = Math.max(min, Math.min(max, next));
    slider.value = String(next);
    slider.dispatchEvent(new Event("input", { bubbles: true }));
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
    var streamRetryDelay = STREAM_RETRY_INITIAL_MS;

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
})();
