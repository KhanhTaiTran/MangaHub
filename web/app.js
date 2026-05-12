const apiBase = window.location.origin.startsWith("http")
  ? window.location.origin
  : "http://localhost:8080";
const apiBaseEl = document.getElementById("api-base");
if (apiBaseEl) {
  apiBaseEl.textContent = apiBase;
}

const state = {
  token: localStorage.getItem("mh_token") || "",
  ws: null,
  notificationSource: null,
};

const logJson = (el, data) => {
  if (!el) {
    return;
  }
  el.textContent = JSON.stringify(data, null, 2);
};

const apiFetch = async (path, options = {}) => {
  const headers = options.headers || {};
  if (state.token) {
    headers.Authorization = `Bearer ${state.token}`;
  }
  const res = await fetch(`${apiBase}${path}`, { ...options, headers });
  const payload = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(payload.error || "Request failed");
  }
  return payload;
};

const updateTokenDisplay = () => {
  const tokenEl = document.getElementById("token");
  if (tokenEl) {
    tokenEl.value = state.token;
  }
  const tokenShortEl = document.getElementById("token-short");
  if (tokenShortEl) {
    tokenShortEl.textContent = state.token
      ? `${state.token.slice(0, 16)}...`
      : "none";
  }
};

const notificationLog = document.getElementById("notification-stream");

const appendNotification = (text, cls) => {
  if (!notificationLog) {
    return;
  }
  const line = document.createElement("div");
  line.textContent = text;
  if (cls) {
    line.className = cls;
  }
  notificationLog.appendChild(line);
  notificationLog.scrollTop = notificationLog.scrollHeight;
};

const stopNotificationStream = () => {
  if (state.notificationSource) {
    state.notificationSource.close();
    state.notificationSource = null;
  }
};

const startNotificationStream = () => {
  if (!notificationLog) {
    return;
  }

  stopNotificationStream();
  notificationLog.textContent = "";

  if (!state.token) {
    appendNotification("Login required to receive notifications", "system");
    return;
  }

  const url = `${apiBase}/api/notifications/stream?token=${encodeURIComponent(state.token)}`;
  const source = new EventSource(url);
  state.notificationSource = source;

  source.onopen = () => {
    appendNotification("Notification stream connected", "system");
  };
  source.addEventListener("notification", (event) => {
    try {
      const note = JSON.parse(event.data);
      const timeLabel = new Date(note.timestamp * 1000).toLocaleTimeString();
      appendNotification(
        `${timeLabel} | ${note.manga_id} | ${note.message}`,
        "received",
      );
    } catch {
      appendNotification(event.data, "received");
    }
  });
  source.onerror = () => {
    appendNotification("Notification stream error", "system");
  };
};

const setToken = (token) => {
  state.token = token;
  if (token) {
    localStorage.setItem("mh_token", token);
  } else {
    localStorage.removeItem("mh_token");
  }
  updateTokenDisplay();
  startNotificationStream();
};

const bindClick = (id, handler) => {
  const el = document.getElementById(id);
  if (el) {
    el.addEventListener("click", handler);
  }
};

bindClick("btn-set-token", () => {
  const tokenEl = document.getElementById("token");
  if (tokenEl) {
    setToken(tokenEl.value.trim());
  }
});

bindClick("btn-logout", () => {
  setToken("");
  if (state.ws) {
    state.ws.close();
    state.ws = null;
  }
});

const loginLog = document.getElementById("login-log");
bindClick("btn-login", async () => {
  const username = document.getElementById("login-username")?.value.trim();
  const password = document.getElementById("login-password")?.value.trim();
  if (!username || !password) {
    logJson(loginLog, { error: "username and password required" });
    return;
  }
  try {
    const payload = await apiFetch("/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    setToken(payload.token || "");
    logJson(loginLog, payload);
  } catch (err) {
    logJson(loginLog, { error: err.message });
  }
});

const registerLog = document.getElementById("register-log");
bindClick("btn-register", async () => {
  const username = document.getElementById("register-username")?.value.trim();
  const password = document.getElementById("register-password")?.value.trim();
  if (!username || !password) {
    logJson(registerLog, { error: "username and password required" });
    return;
  }
  try {
    const payload = await apiFetch("/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    logJson(registerLog, payload);
  } catch (err) {
    logJson(registerLog, { error: err.message });
  }
});

const profileLog = document.getElementById("profile-log");
bindClick("btn-profile", async () => {
  try {
    const payload = await apiFetch("/api/profile");
    logJson(profileLog, payload);
  } catch (err) {
    logJson(profileLog, { error: err.message });
  }
});

const searchLog = document.getElementById("search-log");
bindClick("btn-search", async () => {
  const query = document.getElementById("search-query")?.value.trim();
  const author = document.getElementById("search-author")?.value.trim();
  const genre = document.getElementById("search-genre")?.value.trim();
  const status = document.getElementById("search-status")?.value.trim();
  const limit = document.getElementById("search-limit")?.value.trim();
  const offset = document.getElementById("search-offset")?.value.trim();

  const params = new URLSearchParams();
  if (query) params.set("q", query);
  if (author) params.set("author", author);
  if (genre) params.set("genre", genre);
  if (status) params.set("status", status);
  if (limit) params.set("limit", limit);
  if (offset) params.set("offset", offset);

  const path = params.toString() ? `/manga?${params.toString()}` : "/manga";
  try {
    const payload = await apiFetch(path);
    logJson(searchLog, payload);
  } catch (err) {
    logJson(searchLog, { error: err.message });
  }
});

bindClick("btn-clear", () => {
  [
    "search-query",
    "search-author",
    "search-genre",
    "search-status",
    "search-limit",
    "search-offset",
  ].forEach((id) => {
    const el = document.getElementById(id);
    if (el) {
      el.value = "";
    }
  });
  if (searchLog) {
    searchLog.textContent = "";
  }
});

const mangaLog = document.getElementById("manga-log");
bindClick("btn-get-manga", async () => {
  const mangaId = document.getElementById("manga-id")?.value.trim();
  if (!mangaId) {
    logJson(mangaLog, { error: "manga id required" });
    return;
  }
  try {
    const payload = await apiFetch(`/manga/${encodeURIComponent(mangaId)}`);
    logJson(mangaLog, payload);
  } catch (err) {
    logJson(mangaLog, { error: err.message });
  }
});

const addLog = document.getElementById("libadd-log");
bindClick("btn-libadd", async () => {
  const mangaId = document.getElementById("libadd-manga")?.value.trim();
  const listName = document.getElementById("libadd-list")?.value.trim();
  const status = document.getElementById("libadd-status")?.value.trim();
  const chapter = Number(
    document.getElementById("libadd-chapter")?.value || "0",
  );
  if (!mangaId) {
    logJson(addLog, { error: "manga id required" });
    return;
  }
  try {
    const payload = await apiFetch("/users/library", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        manga_id: mangaId,
        list_name: listName,
        status,
        current_chapter: chapter,
      }),
    });
    logJson(addLog, payload);
  } catch (err) {
    logJson(addLog, { error: err.message });
  }
});

const listLog = document.getElementById("liblist-log");
bindClick("btn-liblist", async () => {
  const listName = document.getElementById("liblist-name")?.value.trim();
  const path = listName
    ? `/users/library?list_name=${encodeURIComponent(listName)}`
    : "/users/library";
  try {
    const payload = await apiFetch(path);
    logJson(listLog, payload);
  } catch (err) {
    logJson(listLog, { error: err.message });
  }
});

const progressLog = document.getElementById("progress-log");
bindClick("btn-progress", async () => {
  const mangaId = document.getElementById("progress-manga")?.value.trim();
  const listName = document.getElementById("progress-list")?.value.trim();
  const status = document.getElementById("progress-status")?.value.trim();
  const chapter = Number(
    document.getElementById("progress-chapter")?.value || "0",
  );
  if (!mangaId) {
    logJson(progressLog, { error: "manga id required" });
    return;
  }
  try {
    const payload = await apiFetch("/users/progress", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        manga_id: mangaId,
        list_name: listName,
        status,
        current_chapter: chapter,
      }),
    });
    logJson(progressLog, payload);
  } catch (err) {
    logJson(progressLog, { error: err.message });
  }
});

const notifyLog = document.getElementById("notify-log");
bindClick("btn-notify", async () => {
  const mangaId = document.getElementById("notify-manga")?.value.trim();
  const message = document.getElementById("notify-message")?.value.trim();
  if (!mangaId || !message) {
    logJson(notifyLog, { error: "manga id and message required" });
    return;
  }
  try {
    const payload = await apiFetch(
      `/api/manga/${encodeURIComponent(mangaId)}/notify`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message }),
      },
    );
    logJson(notifyLog, payload);
  } catch (err) {
    logJson(notifyLog, { error: err.message });
  }
});

const chatLog = document.getElementById("chat-log");
const chatStatus = document.getElementById("chat-status");

const appendChat = (text, cls) => {
  if (!chatLog) {
    return;
  }
  const line = document.createElement("div");
  line.textContent = text;
  if (cls) {
    line.className = cls;
  }
  chatLog.appendChild(line);
  chatLog.scrollTop = chatLog.scrollHeight;
};

const connectChat = () => {
  const room = document.getElementById("chat-room")?.value.trim();
  if (!room) {
    appendChat("Room is required", "system");
    return;
  }
  if (!state.token) {
    appendChat("Login required", "system");
    return;
  }

  if (state.ws) {
    state.ws.close();
  }

  const wsUrl =
    apiBase.replace("http", "ws") +
    `/ws/chat?token=${encodeURIComponent(state.token)}&room=${encodeURIComponent(room)}`;
  const ws = new WebSocket(wsUrl);
  state.ws = ws;

  ws.onopen = () => {
    if (chatStatus) {
      chatStatus.textContent = "Connected";
    }
    appendChat("Connected", "system");
  };
  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      appendChat(
        `${msg.username || "user"}: ${msg.message || event.data}`,
        "received",
      );
    } catch {
      appendChat(event.data, "received");
    }
  };
  ws.onclose = (event) => {
    if (chatStatus) {
      chatStatus.textContent = "Disconnected";
    }
    appendChat(`Disconnected (code: ${event.code})`, "system");
  };
  ws.onerror = () => {
    appendChat("Error occurred", "system");
  };
};

const sendChat = () => {
  const input = document.getElementById("chat-message");
  const message = input?.value.trim();
  if (!message || !state.ws || state.ws.readyState !== WebSocket.OPEN) {
    return;
  }
  state.ws.send(JSON.stringify({ message }));
  appendChat(`-> ${message}`, "sent");
  if (input) {
    input.value = "";
  }
};

bindClick("btn-connect", connectChat);
bindClick("btn-send", sendChat);
bindClick("btn-disconnect", () => {
  if (state.ws) {
    state.ws.close();
    state.ws = null;
  }
});

const chatMessageEl = document.getElementById("chat-message");
if (chatMessageEl) {
  chatMessageEl.addEventListener("keypress", (event) => {
    if (event.key === "Enter") {
      sendChat();
    }
  });
}

window.addEventListener("beforeunload", () => {
  if (state.ws) {
    state.ws.close();
  }
  stopNotificationStream();
});

updateTokenDisplay();
startNotificationStream();
