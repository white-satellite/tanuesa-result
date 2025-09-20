package main

import (
    "bufio"
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os/exec"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "net/url"
    "time"
)

const version = "0.1.0"

type Flags struct {
    Illust bool `json:"illust"`
    Gif    bool `json:"gif"`
}

type User struct {
    Name    string `json:"name"`
    Hit     int    `json:"hit"`
    Jackpot int    `json:"jackpot"`
    Flags   Flags  `json:"flags"`
    Done    bool   `json:"done"`
    Order   int    `json:"order"`
    Status  string `json:"status"`
    Present string `json:"present"`
    HasReference bool `json:"hasReference"`
}

type State struct {
    Users     []User `json:"users"`
    UpdatedAt string `json:"updatedAt"`
}

// Discord mapping: winner name -> last message ID
type DiscordMap map[string]string

type Settings struct {
    EventJSONLog bool `json:"eventJsonLog"`
    AutoServe    bool `json:"autoServe"`
    ServerPort   int  `json:"serverPort"`
    DiscordEnabled bool `json:"discordEnabled"`
    DiscordNewMessagePerSession bool `json:"discordNewMessagePerSession"`
    DiscordArchiveOldSummary bool `json:"discordArchiveOldSummary"`
    DiscordArchiveLabel string `json:"discordArchiveLabel"`
    DiscordTitle string `json:"discordTitle"`
    DiscordEmojiDone string `json:"discordEmojiDone"`
    DiscordEmojiProgress string `json:"discordEmojiProgress"`
    DiscordEmojiNone string `json:"discordEmojiNone"`
    DiscordHeaderGif string `json:"discordHeaderGif"`
    DiscordHeaderIllustration string `json:"discordHeaderIllustration"`
    DiscordRefLabelYes string `json:"discordRefLabelYes"`
    DiscordRefLabelNo  string `json:"discordRefLabelNo"`
}

type Event struct {
    At        string `json:"at"`
    Winner    string `json:"winner"`
    HitFlag   int    `json:"hitFlag"` // 0: 当たり, 1: 大当たり
    Operation string `json:"operation"`
}

type Session struct {
    ID        string `json:"id"`
    StartedAt string `json:"startedAt"`
}

// Discord embed payloads
type EmbedField struct {
    Name   string `json:"name"`
    Value  string `json:"value"`
    Inline bool   `json:"inline"`
}

type EmbedFooter struct {
    Text string `json:"text"`
}

type DiscordEmbed struct {
    Title       string       `json:"title,omitempty"`
    Description string       `json:"description,omitempty"`
    Color       int          `json:"color,omitempty"`
    Fields      []EmbedField `json:"fields,omitempty"`
    Timestamp   string       `json:"timestamp,omitempty"`
    Footer      *EmbedFooter `json:"footer,omitempty"`
}

type DiscordMessage struct {
    Content string         `json:"content,omitempty"`
    Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

// Escape Discord markdown meta characters to avoid unintended formatting
// Escapes: \ * _ ~ ` > | ( ) [ ]
var discordMarkdownEscaper = strings.NewReplacer(
    "\\", "\\\\",
    "*", "\\*",
    "_", "\\_",
    "~", "\\~",
    "`", "\\`",
    ">", "\\>",
    "|", "\\|",
    "(", "\\(",
    ")", "\\)",
    "[", "\\[",
    "]", "\\]",
)

func escapeDiscordMarkdown(s string) string {
    if s == "" { return s }
    return discordMarkdownEscaper.Replace(s)
}

func main() {
    base := baseDir()
    // Load .env.local if present (simple dotenv)
    loadDotenv(base)
    if err := ensureDirs(base); err != nil {
        fatal(err)
    }
    // Migrate current.json to remove legacy Japanese keys
    if err := migrateCurrentJSON(base); err != nil {
        _ = appendAppLog(base, "warn: migrate current.json failed: "+err.Error())
    }
    if err := ensureDiscordMapExists(base); err != nil {
        _ = appendAppLog(base, "warn: ensure discord_map.json failed: "+err.Error())
    }
    if err := ensureSettingsExists(base); err != nil {
        _ = appendAppLog(base, "warn: ensure setting.json failed: "+err.Error())
    }
    // Upgrade settings for backward-compatibility (add new fields with defaults)
    if err := ensureSettingsUpgraded(base); err != nil {
        _ = appendAppLog(base, "warn: upgrade setting.json failed: "+err.Error())
    }

    args := os.Args[1:]
    if len(args) == 0 {
        usage()
        os.Exit(2)
    }

    cmd := strings.ToLower(args[0])
    // Auto-serve: spawn API server if enabled and not serving now
    if s := loadSettings(base); s.AutoServe && cmd != "serve" {
        ensureAPISpawned(base, s.ServerPort)
    }
    switch cmd {
    case "-h", "--help", "help":
        usage()
        return
    case "-v", "--version", "version":
        fmt.Println(version)
        return
    case "reset":
        if err := doReset(base); err != nil {
            fatal(err)
        }
        fmt.Println("reset: completed")
        return
    case "gen-datajs":
        if err := genDataJS(base); err != nil {
            fatal(err)
        }
        fmt.Println("gen-datajs: completed")
        return
    case "backup":
        if _, err := doBackup(base); err != nil {
            fatal(err)
        }
        fmt.Println("backup: completed")
        return
    case "serve":
        port := 3010
        if len(args) >= 2 {
            if p, err := strconv.Atoi(args[1]); err == nil && p > 0 {
                port = p
            }
        }
        if err := serve(base, port); err != nil {
            fatal(err)
        }
        return
    case "gen-backup-index":
        if err := genBackupIndex(base); err != nil {
            fatal(err)
        }
        fmt.Println("gen-backup-index: completed")
        return
    case "restore":
        if len(args) < 2 {
            fatal(errors.New("usage: gacha restore <backupName(.json|.js)>"))
        }
        if err := doRestore(base, args[1]); err != nil {
            fatal(err)
        }
        fmt.Println("restore: completed")
        return
    default:
        // Update mode: expect 2 args: <winnerName> <hitFlag>
        if len(args) == 2 {
            winner := strings.TrimSpace(args[0])
            hitFlagStr := strings.TrimSpace(args[1])
            if err := handleUpdate(base, winner, hitFlagStr); err != nil {
                fatal(err)
            }
            fmt.Println("update: completed")
            return
        }
        usage()
        os.Exit(2)
    }
}

func usage() {
    fmt.Println(`gacha ` + version + `

Usage:
  gacha <winnerName> <hitFlag>   # hitFlag: 0=当たり, 1=大当たり
  gacha reset                    # バックアップ作成→初期化
  gacha gen-datajs               # data/data.js を再生成
  gacha backup                   # 現在値のバックアップのみ
  gacha restore <backupName>     # backups の JSON/JS を現在値へ復元
  gacha gen-backup-index         # backups/index.js を再生成
  gacha serve [port]             # ローカルAPIサーバーを起動

Notes:
  - 名前に空白/日本語がある場合は二重引用符で囲んでください。
`)
}

func handleUpdate(base, winner, hitFlagStr string) error {
    if err := validateWinner(winner); err != nil {
        return err
    }
    flag, err := parseHitFlag(hitFlagStr)
    if err != nil {
        return err
    }

    st, _ := loadState(base)
    // update
    idx := -1
    for i, u := range st.Users {
        if u.Name == winner {
            idx = i
            break
        }
    }
    if idx == -1 {
        st.Users = append(st.Users, User{Name: winner})
        idx = len(st.Users) - 1
    }

    if flag == 0 {
        st.Users[idx].Hit++
    } else {
        st.Users[idx].Jackpot++
    }
    // assign order on first win
    if st.Users[idx].Order == 0 {
        max := 0
        for _, u := range st.Users {
            if u.Order > max { max = u.Order }
        }
        st.Users[idx].Order = max + 1
    }
    // recompute flags per rule
    st.Users[idx].Flags.Illust = st.Users[idx].Hit >= 1 // 当たり>=1
    st.Users[idx].Flags.Gif = st.Users[idx].Hit >= 3 || st.Users[idx].Jackpot >= 1

    // set Present from flags (優先: Gif > イラスト)
    if st.Users[idx].Flags.Gif {
        st.Users[idx].Present = "Gif"
    } else if st.Users[idx].Flags.Illust {
        st.Users[idx].Present = "Illustration"
    } else {
        st.Users[idx].Present = ""
    }

    st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

    if err := saveState(base, st); err != nil {
        return err
    }

    if err := genDataJS(base); err != nil {
        return err
    }

    // Discord notify (optional)
    cfg := loadSettings(base)
    enabled := cfg.DiscordEnabled || isTruthy(os.Getenv("DISCORD_NOTIFY"))
    if enabled {
        // ensure session exists (used for per-session summary mapping)
        sess, _ := ensureSession(base)
        embed := buildLatestSummaryEmbed(st, cfg)
        payload := DiscordMessage{Embeds: []DiscordEmbed{embed}}
        token := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN"))
        channelID := strings.TrimSpace(os.Getenv("DISCORD_CHANNEL_ID"))
        summaryKey := "__SUMMARY__"
        if cfg.DiscordNewMessagePerSession && sess.ID != "" {
            summaryKey = summaryKey + "::" + sess.ID
        }
        if token != "" && channelID != "" {
            if err := discordBotUpsertEmbed(base, token, channelID, summaryKey, payload); err != nil {
                _ = appendAppLog(base, "warn: discord bot notify failed: "+err.Error())
            } else {
                _ = appendAppLog(base, "info: discord bot upsert ok (summary)")
            }
        } else if url := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")); url != "" {
            if err := discordUpsertEmbed(base, url, summaryKey, payload); err != nil {
                _ = appendAppLog(base, "warn: discord webhook notify failed: "+err.Error())
            } else {
                _ = appendAppLog(base, "info: discord webhook upsert ok (summary)")
            }
        } else {
            _ = appendAppLog(base, "info: discord enabled but no credentials; skipping")
        }
    }

    // write per-event JSON log if enabled
    if loadSettings(base).EventJSONLog {
        if err := writeEvent(base, Event{
            At:        time.Now().UTC().Format(time.RFC3339),
            Winner:    winner,
            HitFlag:   flag,
            Operation: "update",
        }); err != nil {
            // non-fatal
            _ = appendAppLog(base, "warn: writeEvent failed: "+err.Error())
        }
    }

    return appendAppLog(base, fmt.Sprintf("update: winner=%q flag=%d", winner, flag))
}

func validateWinner(name string) error {
    if name == "" {
        return errors.New("winnerName is empty")
    }
    if len(name) > 100 {
        return errors.New("winnerName too long (>100)")
    }
    // reject control chars
    for _, r := range name {
        if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
            return errors.New("winnerName contains control characters")
        }
    }
    return nil
}

func parseHitFlag(s string) (int, error) {
    n, err := strconv.Atoi(s)
    if err != nil {
        return 0, errors.New("hitFlag must be 0 or 1")
    }
    if n != 0 && n != 1 {
        return 0, errors.New("hitFlag must be 0 or 1")
    }
    return n, nil
}

func baseDir() string {
    exe, err := os.Executable()
    if err == nil {
        if d := filepath.Dir(exe); d != "" {
            return d
        }
    }
    return "."
}

func ensureDirs(base string) error {
    for _, d := range []string{"data", "logs", "backups"} {
        if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
            return err
        }
    }
    return nil
}

func statePath(base string) string   { return filepath.Join(base, "data", "current.json") }
func dataJSPath(base string) string  { return filepath.Join(base, "data", "data.js") }
func appLogPath(base string) string  { return filepath.Join(base, "logs", "app.log") }
func eventDir(base string) string    { return filepath.Join(base, "logs") }
func backupDir(base string) string   { return filepath.Join(base, "backups") }
func discordMapPath(base string) string { return filepath.Join(base, "data", "discord_map.json") }
// setting.json is placed at project root (same dir as executable)
func settingsPath(base string) string   { return filepath.Join(base, "setting.json") }
func sessionPath(base string) string    { return filepath.Join(base, "data", "session.json") }

func loadState(base string) (State, error) {
    var st State
    p := statePath(base)
    b, err := os.ReadFile(p)
    if err != nil {
        if os.IsNotExist(err) {
            return State{Users: []User{}, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
        }
        return st, err
    }
    if err := json.Unmarshal(b, &st); err != nil {
        return st, err
    }
    return st, nil
}

func saveState(base string, st State) error {
    b, err := json.MarshalIndent(st, "", "  ")
    if err != nil {
        return err
    }
    return writeFileAtomic(statePath(base), b)
}

func genDataJS(base string) error {
    st, err := loadState(base)
    if err != nil {
        return err
    }
    // ensure Present fields are populated for all users (migration safety)
    for i := range st.Users {
        if strings.TrimSpace(st.Users[i].Present) == "" {
            if st.Users[i].Flags.Gif {
                st.Users[i].Present = "Gif"
            } else if st.Users[i].Flags.Illust {
                st.Users[i].Present = "Illustration"
            }
        }
    }
    // Build ASCII-only export structure for data.js
    type userOut struct {
        Name         string `json:"name"`
        Hit          int    `json:"hit"`
        Jackpot      int    `json:"jackpot"`
        Flags        Flags  `json:"flags"`
        Done         bool   `json:"done"`
        Order        int    `json:"order"`
        Status       string `json:"status"`
        Present      string `json:"present"`
        HasReference bool   `json:"hasReference"`
    }
    type out struct {
        Users     []userOut `json:"users"`
        UpdatedAt string    `json:"updatedAt"`
    }
    o := out{Users: make([]userOut, 0, len(st.Users)), UpdatedAt: st.UpdatedAt}
    for _, u := range st.Users {
        po := u.Present
        if po == "" {
            if u.Flags.Gif { po = "Gif" } else if u.Flags.Illust { po = "Illustration" }
        }
        o.Users = append(o.Users, userOut{
            Name: u.Name, Hit: u.Hit, Jackpot: u.Jackpot, Flags: u.Flags,
            Done: u.Done, Order: u.Order, Status: u.Status, Present: po,
            HasReference: u.HasReference,
        })
    }
    payload, err := json.Marshal(o)
    if err != nil {
        return err
    }
    js := []byte("window.__GACHA_DATA__ = " + string(payload) + ";\n")
    return writeFileAtomic(dataJSPath(base), js)
}

func genBackupIndex(base string) error {
    dir := backupDir(base)
    entries, err := os.ReadDir(dir)
    if err != nil {
        if os.IsNotExist(err) {
            return nil
        }
        return err
    }
    type item struct{ Name, JS string }
    var files []string
    for _, e := range entries {
        if e.IsDir() { continue }
        name := e.Name()
        if strings.HasSuffix(strings.ToLower(name), ".json") {
            files = append(files, name)
        }
    }
    sort.Slice(files, func(i, j int) bool { return files[i] > files[j] })
    var items []item
    for _, f := range files {
        baseName := strings.TrimSuffix(f, filepath.Ext(f))
        jsName := baseName + ".js"
        // ensure wrapper js exists (best-effort)
        _ = ensureBackupJS(base, filepath.Join(dir, f))
        items = append(items, item{ Name: f, JS: jsName })
    }
    b, err := json.Marshal(items)
    if err != nil { return err }
    js := []byte("window.__GACHA_BACKUPS__ = " + string(b) + ";\n")
    p := filepath.Join(dir, "index.js")
    return writeFileAtomic(p, js)
}

func ensureBackupJS(base, jsonPath string) error {
    // read json
    data, err := os.ReadFile(jsonPath)
    if err != nil { return err }
    // write wrapper JS next to it
    bn := strings.TrimSuffix(filepath.Base(jsonPath), filepath.Ext(jsonPath))
    jsPath := filepath.Join(filepath.Dir(jsonPath), bn+".js")
    js := append([]byte("window.__GACHA_BACKUP__ = "), data...)
    js = append(js, []byte(";\n")...)
    return writeFileAtomic(jsPath, js)
}

func doRestore(base, name string) error {
    // Accept JS or JSON file name (with or without extension)
    fname := name
    // If JS, map to JSON
    low := strings.ToLower(fname)
    if strings.HasSuffix(low, ".js") {
        fname = strings.TrimSuffix(fname, filepath.Ext(fname)) + ".json"
    } else if !strings.HasSuffix(low, ".json") {
        // try with .json
        fname = fname + ".json"
    }
    p := filepath.Join(backupDir(base), filepath.Base(fname))
    b, err := os.ReadFile(p)
    if err != nil { return err }
    var st State
    if err := json.Unmarshal(b, &st); err != nil { return err }
    // Overwrite current.json
    if err := saveState(base, st); err != nil { return err }
    if err := genDataJS(base); err != nil { return err }
    return appendAppLog(base, "restore: "+filepath.Base(p))
}

func writeEvent(base string, ev Event) error {
    ts := time.Now().Format("2006-01-02T150405") // Windows-safe (no colon)
    fname := fmt.Sprintf("%s.json", ts)
    p := filepath.Join(eventDir(base), fname)
    b, err := json.MarshalIndent(ev, "", "  ")
    if err != nil {
        return err
    }
    return writeFileAtomic(p, b)
}

func doBackup(base string) (string, error) {
    st, err := loadState(base)
    if err != nil {
        return "", err
    }
    ts := time.Now().Format("2006-01-02_150405")
    p := filepath.Join(backupDir(base), fmt.Sprintf("%s.json", ts))
    b, err := json.MarshalIndent(st, "", "  ")
    if err != nil {
        return "", err
    }
    if err := writeFileAtomic(p, b); err != nil {
        return "", err
    }
    // Generate JS wrapper and update index (best-effort)
    if err := ensureBackupJS(base, p); err != nil {
        _ = appendAppLog(base, "warn: ensureBackupJS failed: "+err.Error())
    }
    if err := genBackupIndex(base); err != nil {
        _ = appendAppLog(base, "warn: genBackupIndex failed: "+err.Error())
    }
    if err := appendAppLog(base, "backup: "+filepath.Base(p)); err != nil {
        return p, err
    }
    return p, nil
}

func doReset(base string) error {
    if _, err := doBackup(base); err != nil {
        return err
    }
    st := State{Users: []User{}, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
    if err := saveState(base, st); err != nil {
        return err
    }
    if err := genDataJS(base); err != nil {
        return err
    }
    if _, err := newSession(base); err != nil {
        _ = appendAppLog(base, "warn: newSession failed: "+err.Error())
    }
    return appendAppLog(base, "reset: completed")
}

func writeFileAtomic(path string, data []byte) error {
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return err
    }
    tmp := filepath.Join(dir, fmt.Sprintf(".%s.tmp", filepath.Base(path)))
    f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
    if err != nil {
        return err
    }
    if _, err := f.Write(data); err != nil {
        f.Close()
        _ = os.Remove(tmp)
        return err
    }
    if err := f.Sync(); err != nil { // flush
        f.Close()
        _ = os.Remove(tmp)
        return err
    }
    if err := f.Close(); err != nil {
        _ = os.Remove(tmp)
        return err
    }
    // On Windows, need to remove target before rename if it exists
    if _, err := os.Stat(path); err == nil {
        _ = os.Remove(path)
    }
    return os.Rename(tmp, path)
}

func appendAppLog(base, line string) error {
    p := appLogPath(base)
    if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
        return err
    }
    f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return err
    }
    defer f.Close()
    w := bufio.NewWriter(f)
    ts := time.Now().Format(time.RFC3339)
    if _, err := io.WriteString(w, fmt.Sprintf("%s %s\n", ts, line)); err != nil {
        return err
    }
    return w.Flush()
}

func fatal(err error) {
    fmt.Fprintln(os.Stderr, "error:", err)
    os.Exit(1)
}

// HTTP API server (optional) for UI-driven operations
func serve(base string, port int) error {
    mux := http.NewServeMux()
    setCORS := func(w http.ResponseWriter, r *http.Request) {
        // 開発ローカル用途のため、常にワイルドカード許可（credentials不使用）
        w.Header().Set("Access-Control-Allow-Origin", "*")
        // プリフライトで明示されたヘッダを尊重しつつ、デフォルトで Content-Type を許可
        reqH := r.Header.Get("Access-Control-Request-Headers")
        if strings.TrimSpace(reqH) == "" { reqH = "Content-Type" }
        w.Header().Set("Access-Control-Allow-Headers", reqH)
        // 許可メソッドを固定で提示（POST/GET/PATCH/OPTIONS）
        w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
        w.Header().Set("Access-Control-Max-Age", "600")
    }
    writeJSON := func(w http.ResponseWriter, r *http.Request, v interface{}, code int) {
        setCORS(w, r)
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.WriteHeader(code)
        _ = json.NewEncoder(w).Encode(v)
    }
    mux.HandleFunc("/api/restore", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        name := r.URL.Query().Get("name")
        if name == "" { writeJSON(w, r, map[string]any{"ok": false, "error": "missing name"}, 400); return }
        if err := doRestore(base, name); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        writeJSON(w, r, map[string]any{"ok": true}, 200)
    })
    mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        writeJSON(w, r, map[string]any{"ok": true, "time": time.Now().Format(time.RFC3339)}, 200)
    })
    mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        st, err := loadState(base)
        if err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        writeJSON(w, r, st, 200)
    })
    mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        cfg := loadSettings(base)
        // sanitize and provide fallbacks
        eNone := strings.TrimSpace(cfg.DiscordEmojiNone); if eNone == "" { eNone = "⏳" }
        eProg := strings.TrimSpace(cfg.DiscordEmojiProgress); if eProg == "" { eProg = "🎨" }
        eDone := strings.TrimSpace(cfg.DiscordEmojiDone); if eDone == "" { eDone = "✅" }
        refYes := strings.TrimSpace(cfg.DiscordRefLabelYes); if refYes == "" { refYes = "参考画像あり" }
        refNo := strings.TrimSpace(cfg.DiscordRefLabelNo); if refNo == "" { refNo = "参考画像なし" }
        writeJSON(w, r, map[string]any{
            "emojiNone": eNone,
            "emojiProgress": eProg,
            "emojiDone": eDone,
            "refLabelYes": refYes,
            "refLabelNo": refNo,
        }, 200)
    })
    mux.HandleFunc("/api/reset", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        if err := doReset(base); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        writeJSON(w, r, map[string]any{"ok": true}, 200)
    })
    mux.HandleFunc("/api/user/done", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        if r.Method != http.MethodPost { writeJSON(w, r, map[string]any{"ok": false, "error": "method"}, 405); return }
        var req struct{ Name string `json:"name"`; Done bool `json:"done"` }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": "bad json"}, 400); return }
        if strings.TrimSpace(req.Name) == "" { writeJSON(w, r, map[string]any{"ok": false, "error": "missing name"}, 400); return }
        st, err := loadState(base)
        if err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        found := false
        for i := range st.Users {
            if st.Users[i].Name == req.Name {
                st.Users[i].Done = req.Done
                if req.Done { st.Users[i].Status = "done" } else { st.Users[i].Status = "none" }
                found = true
                break
            }
        }
        if !found { writeJSON(w, r, map[string]any{"ok": false, "error": "not found"}, 404); return }
        st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
        if err := saveState(base, st); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        if err := genDataJS(base); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        // update discord summary as well
        cfg := loadSettings(base)
        if cfg.DiscordEnabled || isTruthy(os.Getenv("DISCORD_NOTIFY")) {
            // no session change, just replace summary
            embed := buildLatestSummaryEmbed(st, cfg)
            payload := DiscordMessage{Embeds: []DiscordEmbed{embed}}
            // build key
            sess, _ := ensureSession(base)
            key := "__SUMMARY__"
            if cfg.DiscordNewMessagePerSession && sess.ID != "" { key = key+"::"+sess.ID }
            if t := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")); t != "" && strings.TrimSpace(os.Getenv("DISCORD_CHANNEL_ID")) != "" {
                _ = discordBotUpsertEmbed(base, t, os.Getenv("DISCORD_CHANNEL_ID"), key, payload)
            } else if u := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")); u != "" {
                _ = discordUpsertEmbed(base, u, key, payload)
            }
        }
        writeJSON(w, r, map[string]any{"ok": true}, 200)
    })
    // toggle reference image flag
    mux.HandleFunc("/api/user/ref", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        if r.Method != http.MethodPost { writeJSON(w, r, map[string]any{"ok": false, "error": "method"}, 405); return }
        var req struct{ Name string `json:"name"`; HasReference bool `json:"hasReference"` }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": "bad json"}, 400); return }
        st, err := loadState(base)
        if err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        found := false
        for i := range st.Users {
            if st.Users[i].Name == req.Name {
                st.Users[i].HasReference = req.HasReference
                found = true
                break
            }
        }
        if !found { writeJSON(w, r, map[string]any{"ok": false, "error": "not found"}, 404); return }
        st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
        if err := saveState(base, st); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        if err := genDataJS(base); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        // refresh discord summary
        cfg := loadSettings(base)
        if cfg.DiscordEnabled || isTruthy(os.Getenv("DISCORD_NOTIFY")) {
            embed := buildLatestSummaryEmbed(st, cfg)
            payload := DiscordMessage{Embeds: []DiscordEmbed{embed}}
            sess, _ := ensureSession(base)
            key := "__SUMMARY__"
            if cfg.DiscordNewMessagePerSession && sess.ID != "" { key = key+"::"+sess.ID }
            if t := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")); t != "" && strings.TrimSpace(os.Getenv("DISCORD_CHANNEL_ID")) != "" {
                _ = discordBotUpsertEmbed(base, t, os.Getenv("DISCORD_CHANNEL_ID"), key, payload)
            } else if u := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")); u != "" {
                _ = discordUpsertEmbed(base, u, key, payload)
            }
        }
        writeJSON(w, r, map[string]any{"ok": true}, 200)
    })
    mux.HandleFunc("/api/gen-backup-index", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        if err := genBackupIndex(base); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        writeJSON(w, r, map[string]any{"ok": true}, 200)
    })
    mux.HandleFunc("/api/backups", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        dir := backupDir(base)
        entries, err := os.ReadDir(dir)
        if err != nil {
            if os.IsNotExist(err) { writeJSON(w, r, []string{}, 200); return }
            writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return
        }
        var files []string
        for _, e := range entries {
            if e.IsDir() { continue }
            name := e.Name()
            if strings.HasSuffix(strings.ToLower(name), ".json") { files = append(files, name) }
        }
        sort.Slice(files, func(i, j int) bool { return files[i] > files[j] })
        writeJSON(w, r, files, 200)
    })
    mux.HandleFunc("/api/user/status", func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodOptions { setCORS(w, r); w.WriteHeader(204); return }
        if r.Method != http.MethodPost { writeJSON(w, r, map[string]any{"ok": false, "error": "method"}, 405); return }
        var req struct{ Name string `json:"name"`; Status string `json:"status"` }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": "bad json"}, 400); return }
        status := strings.ToLower(strings.TrimSpace(req.Status))
        if status != "none" && status != "progress" && status != "done" { writeJSON(w, r, map[string]any{"ok": false, "error": "bad status"}, 400); return }
        st, err := loadState(base)
        if err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        found := false
        for i := range st.Users {
            if st.Users[i].Name == req.Name {
                st.Users[i].Status = status
                st.Users[i].Done = (status == "done")
                found = true
                break
            }
        }
        if !found { writeJSON(w, r, map[string]any{"ok": false, "error": "not found"}, 404); return }
        st.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
        if err := saveState(base, st); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        if err := genDataJS(base); err != nil { writeJSON(w, r, map[string]any{"ok": false, "error": err.Error()}, 500); return }
        cfg := loadSettings(base)
        if cfg.DiscordEnabled || isTruthy(os.Getenv("DISCORD_NOTIFY")) {
            embed := buildLatestSummaryEmbed(st, cfg)
            payload := DiscordMessage{Embeds: []DiscordEmbed{embed}}
            sess, _ := ensureSession(base)
            key := "__SUMMARY__"
            if cfg.DiscordNewMessagePerSession && sess.ID != "" { key = key+"::"+sess.ID }
            if t := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN")); t != "" && strings.TrimSpace(os.Getenv("DISCORD_CHANNEL_ID")) != "" {
                _ = discordBotUpsertEmbed(base, t, os.Getenv("DISCORD_CHANNEL_ID"), key, payload)
            } else if u := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL")); u != "" {
                _ = discordUpsertEmbed(base, u, key, payload)
            }
        }
        writeJSON(w, r, map[string]any{"ok": true}, 200)
    })

    addr := fmt.Sprintf("127.0.0.1:%d", port)
    fmt.Println("serve: listening on http://" + addr)
    // すべてのリクエストにCORSヘッダを適用し、未登録パスでもプリフライトに応答
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        setCORS(w, r)
        if r.Method == http.MethodOptions {
            w.WriteHeader(204)
            return
        }
        mux.ServeHTTP(w, r)
    })
    return http.ListenAndServe(addr, handler)
}

func ensureAPISpawned(base string, port int) {
    // check health
    client := &http.Client{Timeout: 800 * time.Millisecond}
    resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/health", port))
    if err == nil {
        resp.Body.Close()
        return
    }
    exe, err := os.Executable()
    if err != nil { return }
    cmd := exec.Command(exe, "serve", strconv.Itoa(port))
    // Detach from console
    _ = cmd.Start()
    _ = appendAppLog(base, fmt.Sprintf("info: spawned api server on port %d", port))
}

// ---------- Discord integration (Webhook) ----------

func buildLatestSummaryEmbed(st State, cfg Settings) DiscordEmbed {
    // Build fields for [Gif] and [Ilst]
    var gifs, ilsts []string
    for _, u := range st.Users {
        present := strings.TrimSpace(u.Present)
        if present == "" {
            if u.Flags.Gif { present = "Gif" } else if u.Flags.Illust { present = "Illustration" }
        }
        // status emoji from settings (fallback to defaults)
        eNone := cfg.DiscordEmojiNone
        if strings.TrimSpace(eNone) == "" { eNone = "⏳" }
        eProg := cfg.DiscordEmojiProgress
        if strings.TrimSpace(eProg) == "" { eProg = "🎨" }
        eDone := cfg.DiscordEmojiDone
        if strings.TrimSpace(eDone) == "" { eDone = "✅" }
        // reference note (先に表示) ※行フォーマット: 絵文字 + [参考画像○] + ユーザー名
        yes := strings.TrimSpace(cfg.DiscordRefLabelYes)
        no := strings.TrimSpace(cfg.DiscordRefLabelNo)
        if yes == "" { yes = "参考画像あり" }
        if no == "" { no = "参考画像なし" }
        refTag := "[" + no + "]"
        if u.HasReference { refTag = "[" + yes + "]" }
        // Escape user-visible fragments to avoid Discord markdown effects
        safeRef := escapeDiscordMarkdown(refTag)
        safeName := escapeDiscordMarkdown(u.Name)

        if present == "Gif" {
            prefix := eNone + " "
            if u.Status == "progress" { prefix = eProg + " " }
            if u.Done || u.Status == "done" { prefix = eDone + " " }
            line := prefix + safeRef + " " + safeName
            gifs = append(gifs, line)
        } else if present == "Illustration" {
            prefix := eNone + " "
            if u.Status == "progress" { prefix = eProg + " " }
            if u.Done || u.Status == "done" { prefix = eDone + " " }
            line := prefix + safeRef + " " + safeName
            ilsts = append(ilsts, line)
        }
    }
    sort.Strings(gifs)
    sort.Strings(ilsts)
    gh := cfg.DiscordHeaderGif
    if strings.TrimSpace(gh) == "" { gh = "---大当たり（Gif）---" }
    ih := cfg.DiscordHeaderIllustration
    if strings.TrimSpace(ih) == "" { ih = "---当たり（イラスト）---" }
    fieldGif := EmbedField{Name: escapeDiscordMarkdown(gh), Value: "なし", Inline: false}
    fieldIlst := EmbedField{Name: escapeDiscordMarkdown(ih), Value: "なし", Inline: false}
    if len(gifs) > 0 { fieldGif.Value = strings.Join(gifs, "\n") }
    if len(ilsts) > 0 { fieldIlst.Value = strings.Join(ilsts, "\n") }
    // Tailwind emerald-500
    green := 0x10B981
    title := cfg.DiscordTitle
    if strings.TrimSpace(title) == "" { title = "集計（最新）" }
    title = escapeDiscordMarkdown(title)
    ts := st.UpdatedAt
    return DiscordEmbed{
        Title: title,
        Color: green,
        Fields: []EmbedField{fieldGif, fieldIlst},
        Timestamp: ts,
        Footer: &EmbedFooter{Text: escapeDiscordMarkdown("最終更新")},
    }
}

// legacy (unused): kept for reference
func buildDiscordMessage(u User) string {
    if u.Flags.Gif { return "[Gif] "+u.Name }
    if u.Flags.Illust { return "[Ilst] "+u.Name }
    return u.Name
}

func loadDotenv(base string) {
    p := filepath.Join(base, ".env.local")
    b, err := os.ReadFile(p)
    if err != nil {
        return
    }
    content := string(b)
    // Trim UTF-8 BOM if present
    content = strings.TrimPrefix(content, "\uFEFF")
    lines := strings.Split(content, "\n")
    for _, ln := range lines {
        s := strings.TrimSpace(ln)
        if s == "" || strings.HasPrefix(s, "#") || !strings.Contains(s, "=") {
            continue
        }
        idx := strings.Index(s, "=")
        key := strings.TrimSpace(s[:idx])
        val := strings.TrimSpace(s[idx+1:])
        if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) || (strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
            val = val[1:len(val)-1]
        }
        if _, ok := os.LookupEnv(key); !ok {
            _ = os.Setenv(key, val)
        }
    }
}

func isTruthy(v string) bool {
    s := strings.ToLower(strings.TrimSpace(v))
    return s == "1" || s == "true" || s == "yes" || s == "on"
}

func loadDiscordMap(base string) (DiscordMap, error) {
    p := discordMapPath(base)
    b, err := os.ReadFile(p)
    if err != nil {
        if os.IsNotExist(err) {
            return DiscordMap{}, nil
        }
        return nil, err
    }
    var m DiscordMap
    if err := json.Unmarshal(b, &m); err != nil {
        return nil, err
    }
    return m, nil
}

func saveDiscordMap(base string, m DiscordMap) error {
    b, err := json.MarshalIndent(m, "", "  ")
    if err != nil { return err }
    return writeFileAtomic(discordMapPath(base), b)
}

func ensureDiscordMapExists(base string) error {
    p := discordMapPath(base)
    if _, err := os.Stat(p); os.IsNotExist(err) {
        empty := DiscordMap{}
        return saveDiscordMap(base, empty)
    }
    return nil
}

func ensureSession(base string) (Session, error) {
    p := sessionPath(base)
    b, err := os.ReadFile(p)
    if err == nil {
        var s Session
        if json.Unmarshal(b, &s) == nil && s.ID != "" {
            return s, nil
        }
    }
    return newSession(base)
}

func newSession(base string) (Session, error) {
    // reload env to ensure webhook/Bot creds are visible to API server
    loadDotenv(base)
    s := Session{ID: time.Now().Format("20060102_150405"), StartedAt: time.Now().UTC().Format(time.RFC3339)}
    b, err := json.MarshalIndent(s, "", "  ")
    if err != nil { return Session{}, err }
    if err := writeFileAtomic(sessionPath(base), b); err != nil { return Session{}, err }
    // Archive old summaries if enabled
    if err := archiveOldSummaryMessages(base, s.ID); err != nil {
        _ = appendAppLog(base, "warn: archive old summaries failed: "+err.Error())
    }
    return s, nil
}

func archiveOldSummaryMessages(base, newSessionID string) error {
    cfg := loadSettings(base)
    if !cfg.DiscordEnabled || !cfg.DiscordArchiveOldSummary { return nil }
    // detect creds
    token := strings.TrimSpace(os.Getenv("DISCORD_BOT_TOKEN"))
    channelID := strings.TrimSpace(os.Getenv("DISCORD_CHANNEL_ID"))
    webhookURL := strings.TrimSpace(os.Getenv("DISCORD_WEBHOOK_URL"))
    if token == "" && webhookURL == "" { return nil }
    m, _ := loadDiscordMap(base)
    header := buildArchiveHeader(cfg)
    newKey := "__SUMMARY__"+"::"+newSessionID
    for key, mid := range m {
        if !strings.HasPrefix(key, "__SUMMARY__") { continue }
        if key == newKey { continue }
        if mid == "" { continue }
        var content string
        var embeds []DiscordEmbed
        var err error
        if token != "" && channelID != "" {
            content, embeds, err = discordBotGetMessage(token, channelID, mid)
            if err == nil {
                // if there is an embed, just retitle/color it; otherwise prepend header to content as fallback
                if len(embeds) > 0 {
                    e := embeds[0]
                    e.Title = header
                    e.Color = 0x9CA3AF // gray
                    payload := DiscordMessage{Embeds: []DiscordEmbed{e}}
                    _ = discordBotEditEmbed(token, channelID, mid, payload)
                } else if !hasArchiveHeader(content, cfg) {
                    nc := header+"\n"+content
                    if len(nc) > 1900 { nc = nc[:1900] }
                    payload := DiscordMessage{Embeds: []DiscordEmbed{{Title: header, Description: content, Color: 0x9CA3AF}}}
                    _ = discordBotEditEmbed(token, channelID, mid, payload)
                }
                continue
            }
        }
        if webhookURL != "" {
            info, perr := parseWebhook(webhookURL)
            if perr == nil {
                content, embeds, err = discordWebhookGetMessage(info, mid)
                if err == nil {
                    if len(embeds) > 0 {
                        e := embeds[0]
                        e.Title = header
                        e.Color = 0x9CA3AF
                        payload := DiscordMessage{Embeds: []DiscordEmbed{e}}
                        _ = discordWebhookEditEmbed(info, mid, payload)
                    } else if !hasArchiveHeader(content, cfg) {
                        payload := DiscordMessage{Embeds: []DiscordEmbed{{Title: header, Description: content, Color: 0x9CA3AF}}}
                        _ = discordWebhookEditEmbed(info, mid, payload)
                    }
                }
            }
        }
    }
    return nil
}

func buildArchiveHeader(cfg Settings) string {
    base := strings.TrimSpace(cfg.DiscordArchiveLabel)
    if base == "" { base = "アーカイブ" }
    // remove surrounding brackets to avoid duplicate
    base = strings.Trim(base, "[] 　")
    ts := time.Now().In(time.Local).Format("2006/01/02 15:04")
    return fmt.Sprintf("[%s %s]", base, ts)
}

func hasArchiveHeader(content string, cfg Settings) bool {
    first := content
    if i := strings.Index(content, "\n"); i >= 0 { first = content[:i] }
    base := strings.TrimSpace(cfg.DiscordArchiveLabel)
    if base == "" { base = "アーカイブ" }
    base = strings.Trim(base, "[] 　")
    if !strings.HasPrefix(first, "[") { return false }
    // check base label occurrence within first 32 chars
    if len(first) > 64 { first = first[:64] }
    return strings.Contains(first, base)
}

func defaultSettings() Settings {
    return Settings{
        EventJSONLog: false,
        AutoServe: true,
        ServerPort: 3010,
        DiscordEnabled: true,
        DiscordNewMessagePerSession: true,
        DiscordArchiveOldSummary: true,
        DiscordArchiveLabel: "アーカイブ",
        DiscordTitle: "集計（最新）",
        DiscordEmojiDone: "✅",
        DiscordEmojiProgress: "🎨",
        DiscordEmojiNone: "⏳",
        DiscordHeaderGif: "---大当たり（Gif）---",
        DiscordHeaderIllustration: "---当たり（イラスト）---",
        DiscordRefLabelYes: "参考画像あり",
        DiscordRefLabelNo:  "参考画像なし",
    }
}

func ensureSettingsExists(base string) error {
    p := settingsPath(base)
    if _, err := os.Stat(p); os.IsNotExist(err) {
        s := defaultSettings()
        return saveSettings(base, s)
    }
    return nil
}

func loadSettings(base string) Settings {
    p := settingsPath(base)
    b, err := os.ReadFile(p)
    if err != nil {
        return defaultSettings()
    }
    var s Settings
    if err := json.Unmarshal(b, &s); err != nil {
        return defaultSettings()
    }
    if s.ServerPort == 0 { s.ServerPort = 3010 }
    return s
}

func saveSettings(base string, s Settings) error {
    b, err := json.MarshalIndent(s, "", "  ")
    if err != nil { return err }
    return writeFileAtomic(settingsPath(base), b)
}

func ensureSettingsUpgraded(base string) error {
    p := settingsPath(base)
    b, err := os.ReadFile(p)
    if err != nil { return nil }
    var raw map[string]interface{}
    if err := json.Unmarshal(b, &raw); err != nil { return nil }
    changed := false
    if _, ok := raw["discordEnabled"]; !ok {
        raw["discordEnabled"] = true
        changed = true
    }
    if _, ok := raw["discordNewMessagePerSession"]; !ok {
        raw["discordNewMessagePerSession"] = true
        changed = true
    }
    if _, ok := raw["discordArchiveOldSummary"]; !ok {
        raw["discordArchiveOldSummary"] = true
        changed = true
    }
    if _, ok := raw["discordArchiveLabel"]; !ok {
        raw["discordArchiveLabel"] = "アーカイブ"
        changed = true
    }
    if _, ok := raw["discordEmojiDone"]; !ok {
        raw["discordEmojiDone"] = "✅"
        changed = true
    }
    if _, ok := raw["discordEmojiProgress"]; !ok {
        raw["discordEmojiProgress"] = "🎨"
        changed = true
    }
    if _, ok := raw["discordEmojiNone"]; !ok {
        raw["discordEmojiNone"] = "⏳"
        changed = true
    }
    if _, ok := raw["discordHeaderGif"]; !ok {
        raw["discordHeaderGif"] = "---大当たり（Gif）---"
        changed = true
    }
    if _, ok := raw["discordHeaderIllustration"]; !ok {
        raw["discordHeaderIllustration"] = "---当たり（イラスト）---"
        changed = true
    }
    if _, ok := raw["discordRefLabelYes"]; !ok {
        raw["discordRefLabelYes"] = "参考画像あり"
        changed = true
    }
    if _, ok := raw["discordRefLabelNo"]; !ok {
        raw["discordRefLabelNo"] = "参考画像なし"
        changed = true
    }
    if changed {
        nb, err := json.MarshalIndent(raw, "", "  ")
        if err != nil { return err }
        return writeFileAtomic(p, nb)
    }
    return nil
}

// Remove legacy Japanese keys from current.json and normalize present
func migrateCurrentJSON(base string) error {
    p := statePath(base)
    b, err := os.ReadFile(p)
    if err != nil { return nil }
    var root map[string]interface{}
    if err := json.Unmarshal(b, &root); err != nil { return nil }
    changed := false
    if arr, ok := root["users"].([]interface{}); ok {
        for _, it := range arr {
            m, ok := it.(map[string]interface{})
            if !ok { continue }
            if _, ok := m["プレゼント"]; ok { delete(m, "プレゼント"); changed = true }
            // normalize present to ASCII
            if pv, ok := m["present"].(string); ok {
                if pv == "イラスト" { m["present"] = "Illustration"; changed = true }
            }
            // ensure hasReference exists (default false)
            if _, ok := m["hasReference"]; !ok { m["hasReference"] = false; changed = true }
        }
    }
    if changed {
        nb, err := json.MarshalIndent(root, "", "  ")
        if err != nil { return err }
        return writeFileAtomic(p, nb)
    }
    return nil
}

type webhookInfo struct { ID, Token string; Base string }

func parseWebhook(raw string) (webhookInfo, error) {
    u, err := url.Parse(raw)
    if err != nil { return webhookInfo{}, err }
    parts := strings.Split(strings.Trim(u.Path, "/"), "/")
    // expect: api/webhooks/{id}/{token}
    var id, token string
    for i := 0; i+2 <= len(parts); i++ {
        if parts[i] == "api" && parts[i+1] == "webhooks" && i+3 <= len(parts) {
            id = parts[i+2]
            if i+3 < len(parts) { token = parts[i+3] }
            break
        }
    }
    if id == "" || token == "" {
        return webhookInfo{}, errors.New("invalid webhook URL")
    }
    base := fmt.Sprintf("%s://%s/api/webhooks/%s/%s", u.Scheme, u.Host, id, token)
    return webhookInfo{ID: id, Token: token, Base: base}, nil
}

func discordUpsertEmbed(base, webhookURL, key string, payload DiscordMessage) error {
    info, err := parseWebhook(webhookURL)
    if err != nil { return err }
    m, _ := loadDiscordMap(base)
    msgID := m[key]
    if msgID == "" {
        id, err := discordWebhookPost(info, payload)
        if err != nil { return err }
        if id != "" {
            m[key] = id
            _ = saveDiscordMap(base, m)
        }
        return nil
    }
    if err := discordWebhookEditEmbed(info, msgID, payload); err != nil {
        // If edit fails (e.g., not found), try to post again and update map
        id, perr := discordWebhookPost(info, payload)
        if perr == nil && id != "" {
            m[key] = id
            _ = saveDiscordMap(base, m)
            return nil
        }
        return err
    }
    return nil
}

func discordWebhookPost(info webhookInfo, payload DiscordMessage) (string, error) {
    // use wait=true to get message payload back
    u := info.Base + "?wait=true"
    b, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", u, bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return "", err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", fmt.Errorf("webhook post failed: %s", resp.Status)
    }
    var res struct{ ID string `json:"id"` }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        // Some webhooks may still return no body; tolerate
        return "", nil
    }
    return res.ID, nil
}

func discordWebhookEditEmbed(info webhookInfo, messageID string, payload DiscordMessage) error {
    u := fmt.Sprintf("%s/messages/%s", info.Base, messageID)
    b, _ := json.Marshal(payload)
    req, _ := http.NewRequest("PATCH", u, bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("webhook edit failed: %s", resp.Status)
    }
    return nil
}

func discordWebhookGetMessage(info webhookInfo, messageID string) (string, []DiscordEmbed, error) {
    u := fmt.Sprintf("%s/messages/%s", info.Base, messageID)
    req, _ := http.NewRequest("GET", u, nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return "", nil, err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", nil, fmt.Errorf("webhook get failed: %s", resp.Status)
    }
    var res struct{
        Content string          `json:"content"`
        Embeds  []DiscordEmbed  `json:"embeds"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil { return "", nil, err }
    return res.Content, res.Embeds, nil
}

// Bot (OAuth2 token) message operations
func discordBotUpsertEmbed(base, token, channelID, key string, payload DiscordMessage) error {
    m, _ := loadDiscordMap(base)
    msgID := m[key]
    if msgID == "" {
        id, err := discordBotPost(token, channelID, payload)
        if err != nil { return err }
        if id != "" {
            m[key] = id
            _ = saveDiscordMap(base, m)
        }
        return nil
    }
    if err := discordBotEditEmbed(token, channelID, msgID, payload); err != nil {
        id, perr := discordBotPost(token, channelID, payload)
        if perr == nil && id != "" {
            m[key] = id
            _ = saveDiscordMap(base, m)
            return nil
        }
        return err
    }
    return nil
}

func discordBotPost(token, channelID string, payload DiscordMessage) (string, error) {
    u := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages", channelID)
    b, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", u, bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bot "+token)
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return "", err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", fmt.Errorf("bot post failed: %s", resp.Status)
    }
    var res struct{ ID string `json:"id"` }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        return "", nil
    }
    return res.ID, nil
}

func discordBotEditEmbed(token, channelID, messageID string, payload DiscordMessage) error {
    u := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", channelID, messageID)
    b, _ := json.Marshal(payload)
    req, _ := http.NewRequest("PATCH", u, bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bot "+token)
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("bot edit failed: %s", resp.Status)
    }
    return nil
}

func discordBotGetMessage(token, channelID, messageID string) (string, []DiscordEmbed, error) {
    u := fmt.Sprintf("https://discord.com/api/v10/channels/%s/messages/%s", channelID, messageID)
    req, _ := http.NewRequest("GET", u, nil)
    req.Header.Set("Authorization", "Bot "+token)
    resp, err := http.DefaultClient.Do(req)
    if err != nil { return "", nil, err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return "", nil, fmt.Errorf("bot get failed: %s", resp.Status)
    }
    var res struct{
        Content string          `json:"content"`
        Embeds  []DiscordEmbed  `json:"embeds"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil { return "", nil, err }
    return res.Content, res.Embeds, nil
}
