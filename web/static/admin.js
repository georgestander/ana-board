const form = document.querySelector(".compose");
const clearForm = document.querySelector('.history-head form[action="/admin/clear"]');
const notice = document.querySelector("#admin-notice");
const errorBox = document.querySelector("#admin-error");
const rawSound = document.querySelector("#raw-sound");
const history = document.querySelector("#message-list");
const emptyHistory = document.querySelector("#empty-history");
const quickSendButtons = document.querySelectorAll(".quick-send button");
const insertButtons = document.querySelectorAll("[data-insert]");

function showNotice(message) {
  notice.textContent = message;
  notice.hidden = false;
  errorBox.hidden = true;
}

function showError(message) {
  errorBox.textContent = message;
  errorBox.hidden = false;
}

function playRawSound() {
  if (!rawSound) {
    return;
  }

  rawSound.pause();
  rawSound.currentTime = 0;
  rawSound.volume = 1;
  rawSound.playbackRate = 1;
  rawSound.play().catch((error) => {
    console.error(error);
  });
}

function appendMessage(message) {
  if (!history) {
    window.location.reload();
    return;
  }

  if (emptyHistory) {
    emptyHistory.remove();
  }

  const item = document.createElement("li");
  const text = document.createElement("div");
  const meta = document.createElement("div");
  const created = new Date(message.created_at);

  text.className = "message-text";
  text.textContent = message.text;
  meta.className = "message-meta";
  [created.toLocaleString(), message.source, message.kind, message.color, message.status].forEach((value) => {
    const span = document.createElement("span");
    span.textContent = value;
    meta.appendChild(span);
  });

  item.append(text, meta);
  history.prepend(item);
}

async function sendMessage(message, button) {
  button.disabled = true;
  playRawSound();

  try {
    const response = await fetch("/api/messages", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(message),
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `request failed: ${response.status}`);
    }

    appendMessage(payload.message);
    showNotice("Message sent");
  } catch (error) {
    showError(error.message);
  } finally {
    button.disabled = false;
  }
}

if (form) {
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!form.reportValidity()) {
      return;
    }

    const submit = form.querySelector('button[type="submit"]');
    const data = new FormData(form);
    await sendMessage(
      {
        text: data.get("text"),
        source: data.get("source") || "admin",
        animation: "row",
        kind: data.get("kind") || "info",
        color: data.get("color") || "white",
      },
      submit,
    );
  });
}

quickSendButtons.forEach((button) => {
  button.addEventListener("click", async () => {
    const data = new FormData(form);
    await sendMessage(
      {
        text: button.dataset.message,
        source: data.get("source") || "admin",
        animation: "row",
        kind: button.dataset.kind || data.get("kind") || "info",
        color: button.dataset.color || data.get("color") || "white",
      },
      button,
    );
  });
});

function insertToken(token) {
  const input = form?.querySelector('textarea[name="text"]');
  if (!input) {
    return;
  }

  const start = input.selectionStart;
  const end = input.selectionEnd;
  const before = input.value.slice(0, start);
  const after = input.value.slice(end);
  const prefix = before && !/\s$/.test(before) ? " " : "";
  const suffix = after && !/^\s/.test(after) ? " " : "";
  input.value = `${before}${prefix}${token}${suffix}${after}`;
  const cursor = before.length + prefix.length + token.length + suffix.length;
  input.focus();
  input.setSelectionRange(cursor, cursor);
}

insertButtons.forEach((button) => {
  button.addEventListener("click", () => {
    insertToken(button.dataset.insert || "");
  });
});

if (clearForm) {
  clearForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!window.confirm("Clear the board?")) {
      return;
    }

    try {
      const response = await fetch("/api/clear", { method: "POST" });
      if (!response.ok) {
        throw new Error(`clear failed: ${response.status}`);
      }

      showNotice("Board cleared");
    } catch (error) {
      showError(error.message);
    }
  });
}
