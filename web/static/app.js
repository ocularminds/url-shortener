(() => {
  "use strict";

  const form = document.querySelector("#shorten-form");
  const input = document.querySelector("#url");
  const submit = document.querySelector("#submit");
  const message = document.querySelector("#message");
  const result = document.querySelector("#result");
  const shortLink = document.querySelector("#short-link");
  const copy = document.querySelector("#copy");

  const showMessage = (text, isError = false) => {
    message.textContent = text;
    message.classList.toggle("error", isError);
  };

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    result.hidden = true;
    showMessage("");

    if (!input.checkValidity()) {
      input.reportValidity();
      return;
    }

    submit.disabled = true;
    submit.textContent = "Shortening…";
    try {
      const response = await fetch("/", {
        method: "POST",
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ url: input.value }),
      });
      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.Message || "Unable to shorten this URL.");
      }

      shortLink.href = payload.ShortLink;
      shortLink.textContent = payload.ShortLink;
      result.hidden = false;
      showMessage(payload.Message);
    } catch (error) {
      showMessage(error instanceof Error ? error.message : "Something went wrong.", true);
    } finally {
      submit.disabled = false;
      submit.textContent = "Shorten";
    }
  });

  copy.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(shortLink.href);
      showMessage("Copied to your clipboard.");
    } catch {
      shortLink.focus();
      showMessage("Select the link and copy it manually.", true);
    }
  });
})();
