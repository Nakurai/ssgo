(function () {
  var data = null;

  function loadIndex() {
    return fetch('/search-index.json')
      .then(function (r) { return r.json(); })
      .then(function (d) { data = d; });
  }

  function trigrams(s) {
    var out = [];
    for (var i = 0; i <= s.length - 3; i++) {
      out.push(s.substring(i, i + 3));
    }
    return out;
  }

  function search(query) {
    if (!data) return [];
    query = query.toLowerCase().trim();
    if (query.length < 3) return [];

    var grams = trigrams(query);
    var scores = {};
    for (var i = 0; i < grams.length; i++) {
      var ids = data.index[grams[i]] || [];
      for (var j = 0; j < ids.length; j++) {
        var id = ids[j];
        scores[id] = (scores[id] || 0) + 1;
      }
    }

    return Object.keys(scores)
      .sort(function (a, b) { return scores[b] - scores[a]; })
      .slice(0, 10)
      .map(function (id) { return data.docs[parseInt(id, 10)]; });
  }

  function escapeHTML(s) {
    return s
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

  function render(results) {
    var el = document.getElementById('ssgo-search-results');
    if (!el) return;
    if (!results.length) { el.innerHTML = ''; return; }
    el.innerHTML = results.map(function (r) {
      return '<div class="search-result">' +
        '<a href="' + escapeHTML(r.url) + '">' + escapeHTML(r.title) + '</a>' +
        '<p class="search-snippet">' + escapeHTML(r.snippet) + '</p>' +
        '</div>';
    }).join('');
  }

  document.addEventListener('DOMContentLoaded', function () {
    var input = document.getElementById('ssgo-search-input');
    if (!input) return;
    loadIndex().then(function () {
      input.addEventListener('input', function () {
        render(search(input.value));
      });
    });
    document.addEventListener('click', function (e) {
      var wrap = input.closest('.nav-search');
      if (wrap && !wrap.contains(e.target)) {
        render([]);
      }
    });
  });
})();
