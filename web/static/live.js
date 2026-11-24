(function () {
  var timer = document.getElementById("timer");
  var clock = null;
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
    clock = setInterval(paint, 1000);
  }

  var stopForm = document.getElementById("stop-form");
  var stopping = false;

  function endMeeting() {
    if (stopping || !stopForm) return;
    stopping = true;
    if (clock) clearInterval(clock);
    stopForm.submit();
  }

  if (stopForm) {
    stopForm.addEventListener("submit", function () {
      stopping = true;
      if (clock) clearInterval(clock);
    });
  }

  if (document.documentElement.getAttribute("data-script-done") === "true") {
    endMeeting();
    return;
  }

  var script = document.getElementById("script-form");
  if (!script) return;

  var pump = function () {
    if (stopping) return;
    var active = document.activeElement;
    var typing = active && (active.tagName === "INPUT" || active.tagName === "TEXTAREA");
    if (typing) {
      setTimeout(pump, 700);
      return;
    }
    var body = new URLSearchParams();
    body.set("source", "script");
    fetch(script.action, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body.toString(),
      redirect: "follow",
    })
      .then(function (res) {
        return res.text();
      })
      .then(function (html) {
        if (stopping) return;
        var doc = new DOMParser().parseFromString(html, "text/html");
        var next = doc.getElementById("transcript");
        var current = document.getElementById("transcript");
        if (next && current) current.innerHTML = next.innerHTML;
        if (doc.documentElement.getAttribute("data-script-done") === "true") {
          endMeeting();
          return;
        }
        setTimeout(pump, 1600);
      })
      .catch(function () {
        if (!stopping) setTimeout(pump, 1600);
      });
  };
  setTimeout(pump, 1600);
})();
