(function () {
  var timer = document.getElementById("timer");
  if (timer) {
    var start = Date.parse(timer.getAttribute("data-start"));
    var paint = function () {
      if (Number.isNaN(start)) return;
      var sec = Math.max(0, Math.floor((Date.now() - start) / 1000));
      var m = Math.floor(sec / 60);
      var s = sec % 60;
      timer.textContent = m + ":" + String(s).padStart(2, "0");
    };
    paint();
    setInterval(paint, 1000);
  }

  if (document.documentElement.getAttribute("data-script-done") === "true") return;
  var script = document.getElementById("script-form");
  if (!script) return;

  var pump = function () {
    var active = document.activeElement;
    var typing = active && (active.tagName === "INPUT" || active.tagName === "TEXTAREA");
    if (typing) {
      setTimeout(pump, 700);
      return;
    }
    script.requestSubmit();
  };
  setTimeout(pump, 1600);
})();
