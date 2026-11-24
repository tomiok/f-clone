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
  var stopBtn = document.getElementById("stop");
  var resumeBtn = document.getElementById("resume");
  var state = document.getElementById("live-state");
  var paused = false;
  var stopping = false;
  var held = false;

  function endMeeting() {
    if (stopping || !stopForm) return;
    stopping = true;
    if (clock) clearInterval(clock);
    stopForm.submit();
  }

  function pause() {
    paused = true;
    if (clock) clearInterval(clock);
    if (stopBtn) stopBtn.hidden = true;
    if (resumeBtn) resumeBtn.hidden = false;
    if (state) state.lastChild.textContent = " Paused";
  }

  if (stopBtn) {
    stopBtn.addEventListener("click", pause);
  }
  if (resumeBtn) {
    resumeBtn.addEventListener("click", function () {
      if (stopping) return;
      paused = false;
      held = true;
      if (resumeBtn) resumeBtn.hidden = true;
      if (stopBtn) stopBtn.hidden = false;
      if (state) state.lastChild.textContent = " Live";
      pump();
    });
  }

  if (document.documentElement.getAttribute("data-script-done") === "true") {
    endMeeting();
    return;
  }

  var script = document.getElementById("script-form");
  if (!script) return;

  function pump() {
    if (stopping || paused) return;
    var body = new URLSearchParams();
    body.set("source", "script");
    if (held) body.set("held", "1");
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
        if (stopping || paused) return;
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
        if (!stopping && !paused) setTimeout(pump, 1600);
      });
  }
  setTimeout(pump, 1600);
})();
