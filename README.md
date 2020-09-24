# STORD URL Shortener Exercise
![Codecov](https://img.shields.io/codecov/c/github/ocularminds/url-shortener?style=for-the-badge)
The goal of this exercise is to create a URL shortener web application in the same vein as [bitly](https://bitly.com/), [TinyURL](https://tinyurl.com/), or the now defunct [Google URL Shortener](https://goo.gl/). It is intentionally open-ended and you are welcome to implement your solution using the language and tech stack of your choice, but the core functionality of the application should be expressed through your own original code. This is your opportunity to show off your design and development strengths to our engineering team.

## Application Requirements

- When navigating to the root path (e.g. `http://localhost:8080/`) of the app in a browser a user should be presented with a form that allows them to paste in a (presumably long) URL (e.g. `https://www.google.com/search?q=url+shortener&oq=google+u&aqs=chrome.0.69i59j69i60l3j0j69i57.1069j0j7&sourceid=chrome&ie=UTF-8`).
- When a user submits the form they should be presented with a simplified URL of the form `http://{domain}/{slug}` (e.g. `http://localhost:8080/h40Xg2`). The format and method of generation of the slug is up to your discretion.
- When a user navigates to a shortened URL that they have been provided by the app (e.g. `http://localhost:8080/h40Xg2`) they should be redirected to the original URL that yielded that short URL (e.g `https://www.google.com/search?q=url+shortener&oq=google+u&aqs=chrome.0.69i59j69i60l3j0j69i57.1069j0j7&sourceid=chrome&ie=UTF-8`).


## Deliverables

- Source Code. 
- Makefile.
- Other Notes: I was trying to automate database installation but could not complete due to time constraint. Please follow these process to build and run the application.

1. Install MySQL 8 if its not already installed. Create database BLOGS. 
2. Run the attached script: shortener.sql to create the ShortLink table.
3. Edit the provided config.json file with database credentials.
4. Test by running:
```
make test
```
5. Run the application by executing command:
```
make server
```

## Tech Stack
- Golang
- MySQL
- Vue.js
- Html, CSS, Javascript
- Technical design - API REST/JSON Driven. Persistence, Service, Model, Commons, modules
-- Note on the Database-  Requires MySQL Server 8 to be installed on the host machine as well as Go 1.15
The test contains both unit and integration tests involving database.
Test coverage is 68.1% with no db. 78% with database configured.

