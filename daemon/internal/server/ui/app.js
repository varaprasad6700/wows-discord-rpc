document.addEventListener("DOMContentLoaded", () => {
  // 1. Tab Switching
  const tabBtns = document.querySelectorAll(".tab-btn");
  const tabPanes = document.querySelectorAll(".tab-pane");

  tabBtns.forEach(btn => {
    btn.addEventListener("click", () => {
      tabBtns.forEach(b => b.classList.remove("active"));
      tabPanes.forEach(p => p.classList.remove("active"));

      btn.classList.add("active");
      const target = document.getElementById(btn.dataset.tab);
      if (target) target.classList.add("active");
    });
  });

  // 2. Fetch and render status
  async function updateStatus() {
    try {
      const res = await fetch("/api/status");
      if (!res.ok) return;
      const data = await res.json();

      const gameEl = document.getElementById("gameStatusText");
      const discordEl = document.getElementById("discordStatusText");
      const badgeEl = document.getElementById("statusBadge");

      if (data.game_running) {
        gameEl.textContent = "● WorldOfWarships64.exe (Running)";
        gameEl.style.color = "#81c784";
        badgeEl.textContent = "Active";
        badgeEl.className = "badge active";
      } else {
        gameEl.textContent = "○ World of Warships Closed";
        gameEl.style.color = "#90a4ae";
        badgeEl.textContent = "Idle";
        badgeEl.className = "badge idle";
      }

      discordEl.textContent = data.discord_state;
      discordEl.style.color = data.discord_state.includes("Connected") ? "#81c784" : "#e57373";

      document.getElementById("presenceDetails").textContent = data.details || "In Port";
      document.getElementById("presenceState").textContent = data.state || "Inspecting Fleet & Armory";
    } catch (e) {
      console.error("Status fetch error:", e);
    }
  }

  // 3. Load Config
  async function loadConfig() {
    try {
      const res = await fetch("/api/config");
      if (!res.ok) return;
      const cfg = await res.json();

      document.getElementById("installPathInput").value = cfg.install_path || "";
      document.getElementById("backendUrlInput").value = cfg.backend_url || "";
      document.getElementById("autoStartCheckbox").checked = !!cfg.auto_start;
      document.getElementById("fileLoggingCheckbox").checked = !!cfg.file_logging;
    } catch (e) {
      console.error("Config load error:", e);
    }
  }

  // 4. Save Config
  document.getElementById("saveConfigBtn").addEventListener("click", async () => {
    const payload = {
      install_path: document.getElementById("installPathInput").value.trim(),
      backend_url: document.getElementById("backendUrlInput").value.trim(),
      auto_start: document.getElementById("autoStartCheckbox").checked,
      file_logging: document.getElementById("fileLoggingCheckbox").checked,
    };

    const notice = document.getElementById("saveNotice");
    try {
      const res = await fetch("/api/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        notice.textContent = "Settings saved & applied live!";
        setTimeout(() => { notice.textContent = ""; }, 3000);
      } else {
        notice.textContent = "Failed to save settings.";
        notice.style.color = "#e57373";
      }
    } catch (e) {
      notice.textContent = "Error saving settings.";
      notice.style.color = "#e57373";
    }
  });

  // 5. Browse Folder Bridge
  document.getElementById("browseFolderBtn").addEventListener("click", async () => {
    try {
      const res = await fetch("/api/browse-folder", { method: "POST" });
      if (!res.ok) return;
      const data = await res.json();
      if (data.path) {
        document.getElementById("installPathInput").value = data.path;
      }
    } catch (e) {
      console.error("Folder browse error:", e);
    }
  });

  // 6. Logs Tab
  async function refreshLogs() {
    try {
      const res = await fetch("/api/logs");
      if (!res.ok) return;
      const data = await res.json();
      const consoleEl = document.getElementById("logConsoleOutput");
      if (data.lines && data.lines.length > 0) {
        consoleEl.textContent = data.lines.join("\n");
      } else {
        consoleEl.textContent = "No log entries recorded yet.";
      }
      consoleEl.scrollTop = consoleEl.scrollHeight;
    } catch (e) {
      console.error("Logs fetch error:", e);
    }
  }

  document.getElementById("refreshLogsBtn").addEventListener("click", refreshLogs);

  document.getElementById("openLogFileBtn").addEventListener("click", () => {
    fetch("/api/logs/open?type=file", { method: "POST" });
  });

  document.getElementById("openLogFolderBtn").addEventListener("click", () => {
    fetch("/api/logs/open?type=folder", { method: "POST" });
  });

  document.getElementById("fileLoggingCheckbox").addEventListener("change", async (e) => {
    await fetch("/api/logs/toggle", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled: e.target.checked }),
    });
  });

  // 7. Updates Tab
  let updateDownloadUrl = "";

  document.getElementById("checkUpdateBtn").addEventListener("click", async () => {
    const btn = document.getElementById("checkUpdateBtn");
    const notesBox = document.getElementById("releaseNotesBox");
    const latestEl = document.getElementById("latestVersionText");
    const applyBtn = document.getElementById("applyUpdateBtn");

    btn.disabled = true;
    btn.textContent = "Checking GitHub...";
    try {
      const res = await fetch("/api/update/check", { method: "POST" });
      if (res.ok) {
        const info = await res.json();
        latestEl.textContent = info.latest_version;
        notesBox.textContent = info.release_notes || "No release notes available.";

        if (info.update_ready) {
          updateDownloadUrl = info.download_url;
          applyBtn.style.display = "inline-block";
          applyBtn.textContent = `Install ${info.latest_version} Now`;
        } else {
          notesBox.textContent += "\nYou are running the latest version.";
          applyBtn.style.display = "none";
        }
      }
    } catch (e) {
      notesBox.textContent = "Error checking for updates: " + e;
    } finally {
      btn.disabled = false;
      btn.textContent = "Check for Updates";
    }
  });

  document.getElementById("applyUpdateBtn").addEventListener("click", async () => {
    const applyBtn = document.getElementById("applyUpdateBtn");
    applyBtn.disabled = true;
    applyBtn.textContent = "Downloading & Updating...";
    try {
      const res = await fetch("/api/update/apply", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ download_url: updateDownloadUrl }),
      });
      if (res.ok) {
        applyBtn.textContent = "Update complete! Relaunching...";
      } else {
        applyBtn.textContent = "Update failed.";
      }
    } catch (e) {
      applyBtn.textContent = "Update failed: " + e;
    }
  });

  // Initial load & polling
  updateStatus();
  loadConfig();
  refreshLogs();

  setInterval(updateStatus, 3000);
});
