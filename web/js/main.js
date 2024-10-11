const baseURL = "http://127.0.0.1:8080/";

function loadLibrary() {
  const url = `${baseURL}api/get_movies`;

  fetch(url)
      .then(response => response.json())
      .then(data => { showLibrary(data); })
      .catch(error => { console.error('Error:', error); });
}

function performSearch() {
  const query = document.querySelector("#search-input").value;
  const url = `${baseURL}api/search?query=${encodeURIComponent(query)}`;

  fetch(url)
      .then(response => response.json())
      .then(data => { showSearchResults(data); })
      .catch(error => { console.error('Error:', error); });
}

document.querySelector("#search-input")
    .addEventListener("keydown", function(e) {
      if (e.key === "Enter") {
        e.preventDefault();
        performSearch();
      }
    });

document.querySelector("#search-button")
    .addEventListener("click", function() { performSearch(); });

function showSearchResults(results) {
  const container = document.querySelector("#search_result");
  container.innerHTML = "";

  results.forEach(result => {
    const box = document.createElement("div");
    box.classList.add("box");

    const imageWrapper = document.createElement("div");
    imageWrapper.classList.add("image-wrapper");
    imageWrapper.style.backgroundImage =
        `url(https://image.tmdb.org/t/p/original${result.poster_path})`;

    const addButton = document.createElement("div");
    addButton.classList.add("add-button");
    addButton.innerHTML = "✚";

    addButton.addEventListener("click", function() {
      const addUrl = `${baseURL}api/add_movie?id=${result.id}`;
      fetch(addUrl, {method : 'POST'})
          .then(response => response.json())
          .then(data => {
            console.log('Movie added:', data);
            loadLibrary();
          })
          .catch(error => { console.error('Error adding movie:', error); });
    });

    const title = document.createElement("p");
    title.classList.add("centered-text");
    title.textContent = result.title;

    imageWrapper.appendChild(addButton);
    box.appendChild(imageWrapper);
    box.appendChild(title);
    container.appendChild(box);
  });
}

function showLibrary(movies) {
  const container = document.querySelector("#library");
  container.innerHTML = "";

  movies.forEach(movie => {
    const box = document.createElement("div");
    box.classList.add("box");

    const imageWrapper = document.createElement("div");
    imageWrapper.classList.add("image-wrapper");
    imageWrapper.style.backgroundImage = "url(img/" + movie.ID + ".jpg)";

    const title = document.createElement("p");
    title.classList.add("centered-text");
    title.textContent = movie.name;

    box.appendChild(imageWrapper);
    box.appendChild(title);
    container.appendChild(box);
  });
}

loadLibrary();
