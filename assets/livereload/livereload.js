(function () {
  var es = new EventSource('/__livereload');
  es.onmessage = function () { window.location.reload(); };
  es.onerror = function () {
    es.close();
    setTimeout(function () { window.location.reload(); }, 2000);
  };
})();
